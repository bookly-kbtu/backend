package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
)

var errInvalidCode = fmt.Errorf("%w: invalid or expired code", domain.ErrUnauthorized)

type RequestOTPInput struct {
	Phone   string
	Channel string // empty = sms
	IP      string
}

type RequestOTPOutput struct {
	ExpiresAt         time.Time
	ResendAvailableAt time.Time
}

// RequestOTP creates a challenge and sends the code. Same flow for login and register.
func (s *Service) RequestOTP(ctx context.Context, in RequestOTPInput) (*RequestOTPOutput, error) {
	phone, err := domain.NormalizePhone(in.Phone)
	if err != nil {
		return nil, err
	}

	channel := domain.OTPChannelSMS
	if in.Channel != "" {
		if channel, err = domain.ParseOTPChannel(in.Channel); err != nil {
			return nil, err
		}
	}

	now := s.now()

	latest, err := s.deps.OTPs.GetLatest(ctx, phone)
	switch {
	case errors.Is(err, domain.ErrNotFound):
	case err != nil:
		return nil, fmt.Errorf("get latest otp: %w", err)
	case !latest.CanResend(now):
		return nil, fmt.Errorf("%w: resend available at %s", domain.ErrTooManyRequests, latest.ResendAvailableAt.Format(time.RFC3339))
	}

	allowed, err := s.deps.Limiter.Allow(ctx, "otp:phone:"+phone, s.cfg.OTPRequestsPerHour, time.Hour)
	if err != nil {
		return nil, fmt.Errorf("rate limit: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("%w: too many code requests, try later", domain.ErrTooManyRequests)
	}

	code, err := s.deps.Codes.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate otp: %w", err)
	}

	challenge := &domain.OTPChallenge{
		ID:                uuid.New(),
		Destination:       phone,
		Channel:           channel,
		CodeHash:          s.hashCode(phone, code),
		ExpiresAt:         now.Add(s.cfg.OTPTTL),
		MaxAttempts:       s.cfg.OTPMaxAttempts,
		ResendAvailableAt: now.Add(s.cfg.OTPResendCooldown),
		CreatedAt:         now,
	}
	if in.IP != "" {
		challenge.IPAddress = &in.IP
	}

	if err = s.deps.OTPs.Create(ctx, challenge); err != nil {
		return nil, fmt.Errorf("create otp: %w", err)
	}

	if err = s.deps.Sender.Send(ctx, phone, channel, code); err != nil {
		return nil, fmt.Errorf("send otp: %w", err)
	}

	return &RequestOTPOutput{
		ExpiresAt:         challenge.ExpiresAt,
		ResendAvailableAt: challenge.ResendAvailableAt,
	}, nil
}

// consumeOTP checks the code and marks the challenge used in its own transaction,
// so a failed attempt is counted even though the caller returns an error.
// identityID is nil on register: the identity does not exist yet.
func (s *Service) consumeOTP(ctx context.Context, phone, code string, identityID *uuid.UUID) (uuid.UUID, error) {
	var challengeID uuid.UUID

	err := s.deps.Tx.WithinTx(ctx, func(ctx context.Context) error {
		challenge, err := s.deps.OTPs.GetLatestUnconsumedForUpdate(ctx, phone)
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("get otp: %w", err)
		}

		now := s.now()
		if !challenge.Usable(now) {
			return nil
		}

		if !s.codeMatches(challenge, code) {
			return s.deps.OTPs.IncrementAttempts(ctx, challenge.ID)
		}

		challengeID = challenge.ID
		return s.deps.OTPs.Consume(ctx, challenge.ID, identityID, now)
	})
	if err != nil {
		return uuid.Nil, err
	}
	if challengeID == uuid.Nil {
		return uuid.Nil, errInvalidCode
	}
	return challengeID, nil
}

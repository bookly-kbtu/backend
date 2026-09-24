package domain

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type IdentityType string

const (
	IdentityPhone    IdentityType = "phone"
	IdentityTelegram IdentityType = "telegram"
	IdentityEmail    IdentityType = "email"
)

var identityTypes = []IdentityType{IdentityPhone, IdentityTelegram, IdentityEmail}

func ParseIdentityType(s string) (IdentityType, error) {
	return parseEnum("identity type", s, identityTypes)
}

func (t IdentityType) Valid() bool { return slices.Contains(identityTypes, t) }

// OTPChannel is the delivery channel for a phone verification code.
// Telegram means Telegram Gateway API: code goes to the phone number's Telegram account.
type OTPChannel string

const (
	OTPChannelSMS      OTPChannel = "sms"
	OTPChannelWhatsApp OTPChannel = "whatsapp"
	OTPChannelTelegram OTPChannel = "telegram"
)

var otpChannels = []OTPChannel{OTPChannelSMS, OTPChannelWhatsApp, OTPChannelTelegram}

func ParseOTPChannel(s string) (OTPChannel, error) {
	return parseEnum("otp channel", s, otpChannels)
}

func (c OTPChannel) Valid() bool { return slices.Contains(otpChannels, c) }

// NormalizePhone converts user input to E.164 (+77011234567).
// Kazakhstan local formats are accepted: "8 701 123 45 67", "7011234567".
func NormalizePhone(raw string) (string, error) {
	var digits strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	d := digits.String()

	switch {
	case len(d) == 11 && d[0] == '8':
		d = "7" + d[1:]
	case len(d) == 10 && d[0] == '7':
		d = "7" + d
	}

	if len(d) < 8 || len(d) > 15 || d[0] == '0' {
		return "", fmt.Errorf("%w: invalid phone number", ErrValidation)
	}
	return "+" + d, nil
}

type AuthIdentity struct {
	ID         uuid.UUID    `db:"id"`
	UserID     uuid.UUID    `db:"user_id"`
	Type       IdentityType `db:"identity_type"`
	Identifier string       `db:"identifier"`
	VerifiedAt *time.Time   `db:"verified_at"`
	CreatedAt  time.Time    `db:"created_at"`
}

type OTPChallenge struct {
	ID                uuid.UUID  `db:"id"`
	AuthIdentityID    *uuid.UUID `db:"auth_identity_id"`
	Destination       string     `db:"destination"`
	Channel           OTPChannel `db:"delivery_channel"`
	CodeHash          string     `db:"code_hash"`
	ExpiresAt         time.Time  `db:"expires_at"`
	AttemptCount      int        `db:"attempt_count"`
	MaxAttempts       int        `db:"max_attempts"`
	ResendAvailableAt time.Time  `db:"resend_available_at"`
	ConsumedAt        *time.Time `db:"consumed_at"`
	IPAddress         *string    `db:"ip_address"`
	CreatedAt         time.Time  `db:"created_at"`
}

// Usable reports whether the code can still be checked.
func (c *OTPChallenge) Usable(now time.Time) bool {
	return c.ConsumedAt == nil && now.Before(c.ExpiresAt) && c.AttemptCount < c.MaxAttempts
}

func (c *OTPChallenge) CanResend(now time.Time) bool {
	return !now.Before(c.ResendAvailableAt)
}

type Session struct {
	ID               uuid.UUID  `db:"id"`
	UserID           uuid.UUID  `db:"user_id"`
	RefreshTokenHash string     `db:"refresh_token_hash"`
	DeviceInfo       *string    `db:"device_info"`
	IPAddress        *string    `db:"ip_address"`
	ExpiresAt        time.Time  `db:"expires_at"`
	RevokedAt        *time.Time `db:"revoked_at"`
	CreatedAt        time.Time  `db:"created_at"`
	LastUsedAt       time.Time  `db:"last_used_at"`
}

func (s *Session) Active(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.ExpiresAt)
}

type AuthIdentityRepository interface {
	Create(ctx context.Context, identity *AuthIdentity) error
	GetByIdentifier(ctx context.Context, t IdentityType, identifier string) (*AuthIdentity, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]AuthIdentity, error)
}

type OTPChallengeRepository interface {
	Create(ctx context.Context, challenge *OTPChallenge) error
	// GetLatest returns the newest challenge for destination, consumed or not.
	GetLatest(ctx context.Context, destination string) (*OTPChallenge, error)
	// GetLatestUnconsumedForUpdate locks the newest unconsumed challenge. Call inside a transaction.
	GetLatestUnconsumedForUpdate(ctx context.Context, destination string) (*OTPChallenge, error)
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
	Consume(ctx context.Context, id uuid.UUID, identityID *uuid.UUID, at time.Time) error
	AttachIdentity(ctx context.Context, id, identityID uuid.UUID) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	// GetByTokenHashForUpdate locks the session row. Call inside a transaction.
	GetByTokenHashForUpdate(ctx context.Context, hash string) (*Session, error)
	Revoke(ctx context.Context, id uuid.UUID, at time.Time) error
	RevokeByTokenHash(ctx context.Context, hash string, at time.Time) error
	RevokeAllByUser(ctx context.Context, userID uuid.UUID, at time.Time) error
}

// OTPCodeGenerator creates the code sent to the user.
type OTPCodeGenerator interface {
	Generate() (string, error)
}

// OTPSender delivers the code through sms, whatsapp or telegram.
type OTPSender interface {
	Send(ctx context.Context, destination string, channel OTPChannel, code string) error
}

// RateLimiter allows at most limit hits per key within window.
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error)
}

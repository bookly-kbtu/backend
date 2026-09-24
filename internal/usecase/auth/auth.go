package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/pkg/token"
)

// ClientMeta describes the device that owns a session.
type ClientMeta struct {
	IP        string
	UserAgent string
}

type TokenPair struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

type AuthOutput struct {
	Tokens TokenPair
	User   *MeOutput
}

type RegisterInput struct {
	Phone string
	Name  string
	Code  string
	Meta  ClientMeta
}

// Register creates user + client role + client profile + verified phone identity.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*AuthOutput, error) {
	phone, err := domain.NormalizePhone(in.Phone)
	if err != nil {
		return nil, err
	}
	name, err := domain.RequireText("name", in.Name)
	if err != nil {
		return nil, err
	}

	// Checked before the code is consumed, so a registered user keeps the code for login.
	_, err = s.deps.Identities.GetByIdentifier(ctx, domain.IdentityPhone, phone)
	if err == nil {
		return nil, fmt.Errorf("%w: phone already registered", domain.ErrConflict)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("get identity: %w", err)
	}

	userID := uuid.New()
	identityID := uuid.New()

	challengeID, err := s.consumeOTP(ctx, phone, in.Code, nil)
	if err != nil {
		return nil, err
	}

	now := s.now()
	var tokens TokenPair

	err = s.deps.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.deps.Users.Create(ctx, &domain.User{ID: userID, Status: domain.UserStatusActive}); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		if err := s.deps.Users.AddRole(ctx, userID, domain.RoleClient); err != nil {
			return fmt.Errorf("add role: %w", err)
		}
		if err := s.deps.Users.CreateClientProfile(ctx, &domain.ClientProfile{UserID: userID, FirstName: &name}); err != nil {
			return fmt.Errorf("create client profile: %w", err)
		}

		// Unique (identity_type, identifier) rejects a concurrent registration with ErrConflict.
		identity := &domain.AuthIdentity{
			ID:         identityID,
			UserID:     userID,
			Type:       domain.IdentityPhone,
			Identifier: phone,
			VerifiedAt: &now,
		}
		if err := s.deps.Identities.Create(ctx, identity); err != nil {
			return fmt.Errorf("create identity: %w", err)
		}
		if err := s.deps.OTPs.AttachIdentity(ctx, challengeID, identityID); err != nil {
			return fmt.Errorf("attach identity to otp: %w", err)
		}

		tokens, err = s.issueTokens(ctx, userID, []domain.Role{domain.RoleClient}, in.Meta)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "user registered", "user_id", userID)

	return s.authOutput(ctx, userID, tokens)
}

type LoginInput struct {
	Phone string
	Code  string
	Meta  ClientMeta
}

func (s *Service) Login(ctx context.Context, in LoginInput) (*AuthOutput, error) {
	phone, err := domain.NormalizePhone(in.Phone)
	if err != nil {
		return nil, err
	}

	identity, err := s.deps.Identities.GetByIdentifier(ctx, domain.IdentityPhone, phone)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("%w: phone not registered", domain.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get identity: %w", err)
	}

	user, err := s.deps.Users.GetByID(ctx, identity.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if !user.Status.CanSignIn() {
		return nil, fmt.Errorf("%w: account is %s", domain.ErrForbidden, user.Status)
	}

	if _, err = s.consumeOTP(ctx, phone, in.Code, &identity.ID); err != nil {
		return nil, err
	}

	roles, err := s.deps.Users.ListRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	tokens, err := s.issueTokens(ctx, user.ID, roles, in.Meta)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "user logged in", "user_id", user.ID)

	return s.authOutput(ctx, user.ID, tokens)
}

type RefreshInput struct {
	RefreshToken string
	Meta         ClientMeta
}

// Refresh rotates the refresh token: the old session is revoked, a new one is created.
// Reuse of a revoked token means it leaked, so every session of the user is revoked.
func (s *Service) Refresh(ctx context.Context, in RefreshInput) (*TokenPair, error) {
	if in.RefreshToken == "" {
		return nil, domain.ErrUnauthorized
	}

	var (
		tokens TokenPair
		reused bool
	)

	err := s.deps.Tx.WithinTx(ctx, func(ctx context.Context) error {
		session, err := s.deps.Sessions.GetByTokenHashForUpdate(ctx, hashRefreshToken(in.RefreshToken))
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrUnauthorized
		}
		if err != nil {
			return fmt.Errorf("get session: %w", err)
		}

		now := s.now()

		if session.RevokedAt != nil {
			reused = true
			return s.deps.Sessions.RevokeAllByUser(ctx, session.UserID, now)
		}
		if !session.Active(now) {
			return domain.ErrUnauthorized
		}

		user, err := s.deps.Users.GetByID(ctx, session.UserID)
		if err != nil {
			return fmt.Errorf("get user: %w", err)
		}
		if !user.Status.CanSignIn() {
			return fmt.Errorf("%w: account is %s", domain.ErrForbidden, user.Status)
		}

		if err = s.deps.Sessions.Revoke(ctx, session.ID, now); err != nil {
			return fmt.Errorf("revoke session: %w", err)
		}

		roles, err := s.deps.Users.ListRoles(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("list roles: %w", err)
		}

		tokens, err = s.issueTokens(ctx, user.ID, roles, in.Meta)
		return err
	})
	if err != nil {
		return nil, err
	}

	if reused {
		s.log.WarnContext(ctx, "revoked refresh token reused, all sessions revoked")
		return nil, domain.ErrUnauthorized
	}

	return &tokens, nil
}

// Logout revokes the session of this refresh token. Unknown tokens are ignored.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.deps.Sessions.RevokeByTokenHash(ctx, hashRefreshToken(refreshToken), s.now())
}

func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.deps.Sessions.RevokeAllByUser(ctx, userID, s.now())
}

// ParseAccessToken is used by the auth middleware.
func (s *Service) ParseAccessToken(raw string) (token.Claims, error) {
	return s.deps.Tokens.Parse(raw)
}

func (s *Service) issueTokens(ctx context.Context, userID uuid.UUID, roles []domain.Role, meta ClientMeta) (TokenPair, error) {
	refresh, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}

	now := s.now()
	session := &domain.Session{
		ID:               uuid.New(),
		UserID:           userID,
		RefreshTokenHash: hashRefreshToken(refresh),
		ExpiresAt:        now.Add(s.cfg.RefreshTokenTTL),
		CreatedAt:        now,
		LastUsedAt:       now,
	}
	if meta.UserAgent != "" {
		session.DeviceInfo = &meta.UserAgent
	}
	if meta.IP != "" {
		session.IPAddress = &meta.IP
	}

	if err = s.deps.Sessions.Create(ctx, session); err != nil {
		return TokenPair{}, fmt.Errorf("create session: %w", err)
	}

	access, accessExpiresAt, err := s.deps.Tokens.Issue(token.Claims{
		UserID:    userID,
		SessionID: session.ID,
		Roles:     roles,
	}, now)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:           access,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshToken:          refresh,
		RefreshTokenExpiresAt: session.ExpiresAt,
	}, nil
}

func (s *Service) authOutput(ctx context.Context, userID uuid.UUID, tokens TokenPair) (*AuthOutput, error) {
	me, err := s.Me(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &AuthOutput{Tokens: tokens, User: me}, nil
}

package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
)

const issuer = "bookly"

// Claims of the access token. sid ties the token to a sessions row.
type Claims struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Roles     []domain.Role
}

type jwtClaims struct {
	SessionID string        `json:"sid"`
	Roles     []domain.Role `json:"roles"`
	jwt.RegisteredClaims
}

// Manager issues and parses HS256 access tokens.
type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

func (m *Manager) Issue(c Claims, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(m.ttl)

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims{
		SessionID: c.SessionID.String(),
		Roles:     c.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   c.UserID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	signed, err := t.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

// Parse validates signature, algorithm, issuer and expiry.
// Every failure is reported as domain.ErrUnauthorized.
func (m *Manager) Parse(raw string) (Claims, error) {
	var c jwtClaims

	_, err := jwt.ParseWithClaims(raw, &c, func(*jwt.Token) (any, error) {
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Claims{}, errors.Join(domain.ErrUnauthorized, err)
	}

	userID, err := uuid.Parse(c.Subject)
	if err != nil {
		return Claims{}, domain.ErrUnauthorized
	}
	sessionID, err := uuid.Parse(c.SessionID)
	if err != nil {
		return Claims{}, domain.ErrUnauthorized
	}

	return Claims{UserID: userID, SessionID: sessionID, Roles: c.Roles}, nil
}

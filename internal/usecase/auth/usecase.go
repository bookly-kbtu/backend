package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/pkg/token"
)

type Config struct {
	RefreshTokenTTL   time.Duration
	OTPTTL            time.Duration
	OTPResendCooldown time.Duration
	OTPMaxAttempts    int
	// OTPRequestsPerHour limits code requests per phone number.
	OTPRequestsPerHour int64
	// Secret keys the HMAC of OTP codes.
	Secret string
}

type Deps struct {
	Tx         domain.TxManager
	Users      domain.UserRepository
	Identities domain.AuthIdentityRepository
	OTPs       domain.OTPChallengeRepository
	Sessions   domain.SessionRepository
	Codes      domain.OTPCodeGenerator
	Sender     domain.OTPSender
	Limiter    domain.RateLimiter
	Tokens     *token.Manager
}

type Service struct {
	log  *slog.Logger
	cfg  Config
	deps Deps
	now  func() time.Time
}

func New(log *slog.Logger, cfg Config, deps Deps) *Service {
	return &Service{log: log, cfg: cfg, deps: deps, now: time.Now}
}

// hashCode binds the code to its destination so a hash leak of one phone
// does not help with another.
func (s *Service) hashCode(destination, code string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.Secret))
	mac.Write([]byte(destination + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) codeMatches(c *domain.OTPChallenge, code string) bool {
	return hmac.Equal([]byte(c.CodeHash), []byte(s.hashCode(c.Destination, code)))
}

// Refresh tokens are 256-bit random values, so plain SHA-256 is enough for storage.
func hashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

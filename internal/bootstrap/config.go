package bootstrap

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var otpCodePattern = regexp.MustCompile(`^[0-9]{4}$`)

type Config struct {
	AppName         string
	AppEnv          string // "dev" | "prod"
	HTTPAddr        string
	ShutdownTimeout time.Duration

	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string
	MigrationsDir    string

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	Auth   AuthConfig
	Docs   DocsConfig
	Import ImportConfig
}

// ImportConfig is used by cmd/command to pull external catalogues.
type ImportConfig struct {
	ZapisBaseURL      string
	ZapisAssetBaseURL string
	UserAgent         string
	RequestDelay      time.Duration
}

// DocsConfig controls Scalar UI at /docs. In prod basic auth is mandatory.
type DocsConfig struct {
	Enabled  bool
	SpecPath string
	User     string
	Password string
}

type AuthConfig struct {
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	OTPTTL             time.Duration
	OTPResendCooldown  time.Duration
	OTPMaxAttempts     int
	OTPRequestsPerHour int64
	// OTPStaticCode is the MVP stub: every challenge gets this code.
	OTPStaticCode string
}

// LoadConfig loads and validates config for the HTTP API.
func LoadConfig() (Config, error) {
	cfg, err := loadConfig()
	if err != nil {
		return Config{}, err
	}
	if err = cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// LoadCommandConfig loads config for CLI jobs: API secrets (JWT, OTP, docs) are not required.
func LoadCommandConfig() (Config, error) {
	cfg, err := loadConfig()
	if err != nil {
		return Config{}, err
	}
	if cfg.AppEnv != "dev" && cfg.AppEnv != "prod" {
		return Config{}, fmt.Errorf("APP_ENV must be dev or prod, got %q", cfg.AppEnv)
	}
	return cfg, nil
}

func loadConfig() (Config, error) {
	_ = godotenv.Load()

	var p envParser

	cfg := Config{
		AppName:         p.str("APP_NAME", "bookly"),
		AppEnv:          p.str("APP_ENV", "dev"),
		HTTPAddr:        p.str("HTTP_ADDR", ":8080"),
		ShutdownTimeout: p.duration("SHUTDOWN_TIMEOUT", 10*time.Second),

		PostgresHost:     p.str("POSTGRES_HOST", "localhost"),
		PostgresPort:     p.str("POSTGRES_PORT", "5432"),
		PostgresUser:     p.str("POSTGRES_USER", "bookly"),
		PostgresPassword: p.str("POSTGRES_PASSWORD", "bookly"),
		PostgresDB:       p.str("POSTGRES_DB", "bookly"),
		PostgresSSLMode:  p.str("POSTGRES_SSLMODE", "disable"),
		MigrationsDir:    p.str("MIGRATIONS_DIR", "migrations"),

		RedisHost:     p.str("REDIS_HOST", "localhost"),
		RedisPort:     p.str("REDIS_PORT", "6379"),
		RedisPassword: p.str("REDIS_PASSWORD", ""),
		RedisDB:       p.int("REDIS_DB", 0),

		Auth: AuthConfig{
			JWTSecret:       p.str("JWT_SECRET", ""),
			AccessTokenTTL:  p.duration("ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: p.duration("REFRESH_TOKEN_TTL", 30*24*time.Hour),

			OTPTTL:             p.duration("OTP_TTL", 5*time.Minute),
			OTPResendCooldown:  p.duration("OTP_RESEND_COOLDOWN", time.Minute),
			OTPMaxAttempts:     p.int("OTP_MAX_ATTEMPTS", 5),
			OTPRequestsPerHour: int64(p.int("OTP_REQUESTS_PER_HOUR", 5)),
			OTPStaticCode:      p.str("OTP_STATIC_CODE", ""),
		},
	}

	cfg.Docs = DocsConfig{
		Enabled:  p.bool("DOCS_ENABLED", true),
		SpecPath: p.str("DOCS_SPEC_PATH", "docs/swagger.json"),
		User:     p.str("DOCS_USER", ""),
		Password: p.str("DOCS_PASSWORD", ""),
	}

	cfg.Import = ImportConfig{
		ZapisBaseURL:      p.str("ZAPIS_BASE_URL", "https://zapis.kz/rest/clients-app/v1"),
		ZapisAssetBaseURL: p.str("ZAPIS_ASSET_BASE_URL", "https://zapis.kz"),
		UserAgent:         p.str("IMPORT_USER_AGENT", "BooklyImporter/0.1"),
		RequestDelay:      p.duration("IMPORT_REQUEST_DELAY", 500*time.Millisecond),
	}

	if p.err != nil {
		return Config{}, p.err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.AppEnv != "dev" && c.AppEnv != "prod" {
		return fmt.Errorf("APP_ENV must be dev or prod, got %q", c.AppEnv)
	}
	if len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if !otpCodePattern.MatchString(c.Auth.OTPStaticCode) {
		return fmt.Errorf("OTP_STATIC_CODE must be 4 digits")
	}
	if c.Docs.Enabled && c.IsProd() && (c.Docs.User == "" || len(c.Docs.Password) < 12) {
		return fmt.Errorf("DOCS_USER and DOCS_PASSWORD (12+ characters) are required in prod, or set DOCS_ENABLED=false")
	}
	if c.Docs.User != "" && c.Docs.Password == "" {
		return fmt.Errorf("DOCS_PASSWORD is required when DOCS_USER is set")
	}
	if c.Auth.OTPRequestsPerHour < 1 {
		return fmt.Errorf("OTP_REQUESTS_PER_HOUR must be positive")
	}
	if c.Auth.OTPMaxAttempts < 1 || c.Auth.OTPMaxAttempts > 20 {
		return fmt.Errorf("OTP_MAX_ATTEMPTS must be between 1 and 20")
	}
	return nil
}

func (c Config) IsProd() bool {
	return c.AppEnv == "prod"
}

func (c Config) PostgresDSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.PostgresUser, c.PostgresPassword),
		Host:   c.PostgresHost + ":" + c.PostgresPort,
		Path:   c.PostgresDB,
	}

	q := u.Query()
	q.Set("sslmode", c.PostgresSSLMode)
	u.RawQuery = q.Encode()

	return u.String()
}

func (c Config) RedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}

// envParser keeps the first parse error so LoadConfig can report it once.
type envParser struct {
	err error
}

func (p *envParser) str(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func (p *envParser) int(key string, fallback int) int {
	raw := p.str(key, "")
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		p.fail(fmt.Errorf("%s: invalid int %q", key, raw))
		return fallback
	}
	return value
}

func (p *envParser) bool(key string, fallback bool) bool {
	raw := p.str(key, "")
	if raw == "" {
		return fallback
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		p.fail(fmt.Errorf("%s: invalid bool %q", key, raw))
		return fallback
	}
	return value
}

// duration accepts Go duration strings: "15m", "720h", "10s".
func (p *envParser) duration(key string, fallback time.Duration) time.Duration {
	raw := p.str(key, "")
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		p.fail(fmt.Errorf("%s: invalid duration %q", key, raw))
		return fallback
	}
	return value
}

func (p *envParser) fail(err error) {
	if p.err == nil {
		p.err = err
	}
}

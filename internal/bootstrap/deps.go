package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/bookly-kbtu/backend/internal/infrastructure/otp"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	authrepo "github.com/bookly-kbtu/backend/internal/infrastructure/postgres/auth"
	catalogrepo "github.com/bookly-kbtu/backend/internal/infrastructure/postgres/catalog"
	userrepo "github.com/bookly-kbtu/backend/internal/infrastructure/postgres/user"
	redisstorage "github.com/bookly-kbtu/backend/internal/infrastructure/redis"
	"github.com/bookly-kbtu/backend/internal/pkg/token"
	authuc "github.com/bookly-kbtu/backend/internal/usecase/auth"
	cataloguc "github.com/bookly-kbtu/backend/internal/usecase/catalog"
)

type Deps struct {
	Logger *slog.Logger
	DB     *postgres.DB
	Redis  *redisstorage.Client

	AuthUseCase    *authuc.Service
	CatalogUseCase *cataloguc.Service
}

func NewDeps(ctx context.Context, cfg Config) (*Deps, error) {
	logger := newLogger(cfg.AppEnv)

	db, err := postgres.NewDB(ctx, cfg.PostgresDSN())
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}

	redis, err := redisstorage.NewClient(ctx, redisstorage.Config{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init redis: %w", err)
	}

	if cfg.IsProd() {
		logger.Warn("OTP uses static code from OTP_STATIC_CODE and log sender; do not expose to real users")
	}

	authUseCase := authuc.New(logger, authuc.Config{
		RefreshTokenTTL:    cfg.Auth.RefreshTokenTTL,
		OTPTTL:             cfg.Auth.OTPTTL,
		OTPResendCooldown:  cfg.Auth.OTPResendCooldown,
		OTPMaxAttempts:     cfg.Auth.OTPMaxAttempts,
		OTPRequestsPerHour: cfg.Auth.OTPRequestsPerHour,
		Secret:             cfg.Auth.JWTSecret,
	}, authuc.Deps{
		Tx:         db,
		Users:      userrepo.NewRepository(db),
		Identities: authrepo.NewIdentityRepository(db),
		OTPs:       authrepo.NewOTPRepository(db),
		Sessions:   authrepo.NewSessionRepository(db),
		Codes:      otp.NewStaticGenerator(cfg.Auth.OTPStaticCode),
		Sender:     otp.NewLogSender(logger),
		Limiter:    redis,
		Tokens:     token.NewManager(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL),
	})
	catalogUseCase := cataloguc.New(catalogrepo.NewRepository(db))

	return &Deps{
		Logger: logger,
		DB:     db,
		Redis:  redis,

		AuthUseCase:    authUseCase,
		CatalogUseCase: catalogUseCase,
	}, nil
}

func (d *Deps) Close() {
	if err := d.Redis.Close(); err != nil {
		d.Logger.Error("close redis", "error", err)
	}
	if err := d.DB.Close(); err != nil {
		d.Logger.Error("close db", "error", err)
	}
}

func newLogger(env string) *slog.Logger {
	if env == "prod" {
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

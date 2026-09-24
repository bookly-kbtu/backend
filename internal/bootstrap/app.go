package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/transport/rest/docs"
	"github.com/bookly-kbtu/backend/internal/transport/rest/v1/middleware"
	"github.com/bookly-kbtu/backend/internal/transport/rest/v1/response"
)

type App struct {
	cfg   Config
	deps  *Deps
	fiber *fiber.App
}

func NewApp(cfg Config, deps *Deps) *App {
	app := &App{
		cfg:  cfg,
		deps: deps,
		fiber: fiber.New(fiber.Config{
			AppName:      cfg.AppName,
			// Room for one image upload plus multipart overhead.
			BodyLimit: domain.MaxImageBytes + 1<<20,
			ErrorHandler: response.ErrorHandler(deps.Logger),
		}),
	}

	app.setupRoutes()

	return app
}

func (a *App) setupRoutes() {
	a.setupMiddleware()
	a.setupSystemRoutes()
	a.setupAPIRoutes()
}

func (a *App) setupMiddleware() {
	a.fiber.Use(middleware.Recover(a.deps.Logger))
	a.fiber.Use(middleware.RequestID())
	a.fiber.Use(middleware.Logger(a.deps.Logger))
}

func (a *App) setupSystemRoutes() {
	a.fiber.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(fmt.Sprintf("Добро пожаловать в %s", a.cfg.AppName))
	})

	a.fiber.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	a.fiber.Get("/readyz", func(c *fiber.Ctx) error {
		if err := a.deps.DB.PingContext(c.UserContext()); err != nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "postgres unavailable")
		}
		if err := a.deps.Redis.Ping(c.UserContext()).Err(); err != nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "redis unavailable")
		}
		return c.SendString("ok")
	})

	if a.cfg.Docs.Enabled {
		docs.Register(a.fiber, docs.Config{
			SpecPath: a.cfg.Docs.SpecPath,
			User:     a.cfg.Docs.User,
			Password: a.cfg.Docs.Password,
		})
	}
}

// Run blocks until ctx is cancelled or the server fails, then shuts down gracefully.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		a.deps.Logger.Info("starting server", "addr", a.cfg.HTTPAddr)
		errCh <- a.fiber.Listen(a.cfg.HTTPAddr)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	a.deps.Logger.Info("shutting down server")
	if err := a.fiber.ShutdownWithTimeout(a.cfg.ShutdownTimeout); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

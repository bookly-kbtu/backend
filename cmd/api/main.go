package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/bookly-kbtu/backend/internal/bootstrap"
)

// @title						Bookly API
// @version					1.0
// @description				Booking service: clients, masters, services, schedule, bookings.
// @BasePath					/api/v1
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Access token: "Bearer <access_token>"
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	deps, err := bootstrap.NewDeps(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to init deps: %v", err)
	}
	defer deps.Close()

	app := bootstrap.NewApp(cfg, deps)

	if err = app.Run(ctx); err != nil {
		log.Fatalf("application error: %v", err)
	}
}

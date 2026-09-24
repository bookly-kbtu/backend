package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	importerrepo "github.com/bookly-kbtu/backend/internal/infrastructure/postgres/importer"
	"github.com/bookly-kbtu/backend/internal/infrastructure/sources/zapis"
	importeruc "github.com/bookly-kbtu/backend/internal/usecase/importer"
)

// CommandDeps is the CLI dependency set: DB and import sources, no Redis/HTTP.
type CommandDeps struct {
	Logger   *slog.Logger
	DB       *postgres.DB
	Importer *importeruc.Service
}

func NewCommandDeps(ctx context.Context, cfg Config) (*CommandDeps, error) {
	logger := newLogger(cfg.AppEnv)

	db, err := postgres.NewDB(ctx, cfg.PostgresDSN())
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}

	// New sources: implement domain.CatalogSource and register here.
	zapisSource, err := zapis.New(zapis.Config{
		BaseURL:      cfg.Import.ZapisBaseURL,
		AssetBaseURL: cfg.Import.ZapisAssetBaseURL,
		UserAgent:    cfg.Import.UserAgent,
		Delay:        cfg.Import.RequestDelay,
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init zapis source: %w", err)
	}

	return &CommandDeps{
		Logger:   logger,
		DB:       db,
		Importer: importeruc.New(logger, db, importerrepo.NewRepository(db), zapisSource),
	}, nil
}

func (d *CommandDeps) Close() {
	if err := d.DB.Close(); err != nil {
		d.Logger.Error("close db", "error", err)
	}
}

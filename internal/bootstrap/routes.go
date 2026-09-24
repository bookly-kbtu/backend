package bootstrap

import (
	v1 "github.com/bookly-kbtu/backend/internal/transport/rest/v1"
	authhandler "github.com/bookly-kbtu/backend/internal/transport/rest/v1/auth"
	cataloghandler "github.com/bookly-kbtu/backend/internal/transport/rest/v1/catalog"
	"github.com/bookly-kbtu/backend/internal/transport/rest/v1/middleware"
)

func (a *App) setupAPIRoutes() {
	v1.NewRouter(
		a.fiber.Group(v1.Prefix),
		&v1.Handlers{
			Auth:    authhandler.New(a.deps.AuthUseCase),
			Catalog: cataloghandler.New(a.deps.CatalogUseCase),
		},
		middleware.Auth(a.deps.AuthUseCase),
	).SetupRoutes()
}

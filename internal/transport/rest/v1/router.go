package v1

import (
	"github.com/gofiber/fiber/v2"

	authhandler "github.com/bookly-kbtu/backend/internal/transport/rest/v1/auth"
	cataloghandler "github.com/bookly-kbtu/backend/internal/transport/rest/v1/catalog"
	"github.com/bookly-kbtu/backend/internal/transport/rest/v1/middleware"
	platformhandler "github.com/bookly-kbtu/backend/internal/transport/rest/v1/platform"
)

const Prefix = "/api/v1"

// Handlers groups every v1 handler. Fields are added per module.
type Handlers struct {
	Auth     *authhandler.Handler
	Catalog  *cataloghandler.Handler
	Platform *platformhandler.Handler
}

type Router struct {
	api      fiber.Router
	handlers *Handlers
	authMW   fiber.Handler
}

func NewRouter(api fiber.Router, handlers *Handlers, authMW fiber.Handler) *Router {
	return &Router{api: api, handlers: handlers, authMW: authMW}
}

func (r *Router) SetupRoutes() {
	r.setupAuthRoutes()
	r.setupCatalogRoutes()
	r.setupPlatformRoutes()
}

func (r *Router) setupAuthRoutes() {
	r.handlers.Auth.Register(r.api.Group("/auth"), r.authMW)
}

func (r *Router) setupCatalogRoutes() {
	r.handlers.Catalog.Register(r.api.Group("/market"))
}

func (r *Router) setupPlatformRoutes() {
	r.handlers.Platform.Register(r.api, r.authMW, middleware.RequireRole("admin"))
}

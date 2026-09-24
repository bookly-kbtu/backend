package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/pkg/ctxuser"
	"github.com/bookly-kbtu/backend/internal/pkg/token"
)

type tokenParser interface {
	ParseAccessToken(raw string) (token.Claims, error)
}

// Auth requires a valid "Authorization: Bearer <access token>" header.
func Auth(tokens tokenParser) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
		if !ok || raw == "" {
			return domain.ErrUnauthorized
		}

		claims, err := tokens.ParseAccessToken(raw)
		if err != nil {
			return domain.ErrUnauthorized
		}

		c.SetUserContext(ctxuser.With(c.UserContext(), ctxuser.User{
			ID:        claims.UserID,
			SessionID: claims.SessionID,
			Roles:     claims.Roles,
		}))
		return c.Next()
	}
}

// RequireRole passes when the user has at least one of roles. Use after Auth.
func RequireRole(roles ...domain.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := ctxuser.From(c.UserContext())
		if !ok {
			return domain.ErrUnauthorized
		}
		for _, role := range roles {
			if user.HasRole(role) {
				return c.Next()
			}
		}
		return domain.ErrForbidden
	}
}

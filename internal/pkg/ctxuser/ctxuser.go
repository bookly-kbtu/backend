package ctxuser

import (
	"context"

	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
)

// User is the authenticated caller extracted from the access token.
type User struct {
	ID        uuid.UUID
	SessionID uuid.UUID
	Roles     []domain.Role
}

func (u User) HasRole(role domain.Role) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

type ctxKey struct{}

func With(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, ctxKey{}, user)
}

func From(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(ctxKey{}).(User)
	return user, ok
}

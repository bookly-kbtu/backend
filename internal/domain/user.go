package domain

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusActive  UserStatus = "active"
	UserStatusBlocked UserStatus = "blocked"
	UserStatusDeleted UserStatus = "deleted"
)

var userStatuses = []UserStatus{UserStatusActive, UserStatusBlocked, UserStatusDeleted}

func ParseUserStatus(s string) (UserStatus, error) {
	return parseEnum("user status", s, userStatuses)
}

func (s UserStatus) Valid() bool { return slices.Contains(userStatuses, s) }

// CanSignIn reports whether a user with this status may authenticate.
func (s UserStatus) CanSignIn() bool { return s == UserStatusActive }

type User struct {
	ID        uuid.UUID  `db:"id"`
	Status    UserStatus `db:"status"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)

	AddRole(ctx context.Context, userID uuid.UUID, role Role) error
	ListRoles(ctx context.Context, userID uuid.UUID) ([]Role, error)

	CreateClientProfile(ctx context.Context, profile *ClientProfile) error
	GetClientProfile(ctx context.Context, userID uuid.UUID) (*ClientProfile, error)
}

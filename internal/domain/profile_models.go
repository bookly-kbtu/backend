package domain

import (
	"time"

	"github.com/google/uuid"
)

type ClientProfile struct {
	UserID    uuid.UUID `db:"user_id"`
	Phone     *string   `db:"phone"`
	FirstName *string   `db:"first_name"`
	LastName  *string   `db:"last_name"`
	AvatarURL *string   `db:"avatar_url"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type PatchClientProfileInput struct {
	FirstName *string
	LastName  *string
	AvatarURL *string
}

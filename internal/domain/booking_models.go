package domain

import (
	"time"

	"github.com/google/uuid"
)

type BookingFilter struct {
	MasterID, ClientID *uuid.UUID
	Status             string
	From, To           *time.Time
	Limit, Offset      int
}

type Booking struct {
	ID                  uuid.UUID `db:"id"`
	ClientID            uuid.UUID `db:"client_id"`
	MasterID            uuid.UUID `db:"master_id"`
	MasterServiceID     uuid.UUID `db:"master_service_id"`
	MasterLocationID    uuid.UUID `db:"master_location_id"`
	StartsAt            time.Time `db:"starts_at"`
	EndsAt              time.Time `db:"ends_at"`
	Status              string    `db:"status"`
	ServiceNameSnapshot string    `db:"service_name_snapshot"`
	PriceAmount         int64     `db:"price_amount"`
	Currency            string    `db:"currency"`
	DurationMinutes     int       `db:"duration_minutes"`
	ClientComment       *string   `db:"client_comment"`
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
}

type CreateBookingInput struct {
	MasterID         uuid.UUID
	MasterServiceID  uuid.UUID
	MasterLocationID uuid.UUID
	StartsAt         time.Time
	ClientComment    *string
}

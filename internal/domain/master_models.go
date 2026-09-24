package domain

import (
	"time"

	"github.com/google/uuid"
)

type MasterFilter struct {
	Query         string
	CategoryID    *uuid.UUID
	Active        *bool
	Limit, Offset int
}

type CategoryInput struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	IsActive *bool  `json:"is_active"`
}

type MasterProfile struct {
	UserID      uuid.UUID `db:"user_id"`
	DisplayName string    `db:"display_name"`
	Description *string   `db:"description"`
	AvatarURL   *string   `db:"avatar_url"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type UpsertMasterProfileInput struct {
	DisplayName string
	Description *string
}

type MasterLocation struct {
	ID          uuid.UUID `db:"id"`
	MasterID    uuid.UUID `db:"master_id"`
	Name        string    `db:"name"`
	AddressText string    `db:"address_text"`
	Latitude    *float64  `db:"latitude"`
	Longitude   *float64  `db:"longitude"`
	TimeZone    string    `db:"time_zone"`
	IsPrimary   bool      `db:"is_primary"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type LocationInput struct {
	Name        string
	AddressText string
	Latitude    *float64
	Longitude   *float64
	TimeZone    string
	IsPrimary   bool
}

type Category struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	Slug      string    `db:"slug"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ServiceItem struct {
	ID              uuid.UUID `db:"id"`
	MasterID        uuid.UUID `db:"master_id"`
	CategoryID      uuid.UUID `db:"category_id"`
	CategoryName    string    `db:"category_name"`
	Name            string    `db:"name"`
	Description     *string   `db:"description"`
	PriceAmount     int64     `db:"price_amount"`
	Currency        string    `db:"currency"`
	DurationMinutes int       `db:"duration_minutes"`
	IsActive        bool      `db:"is_active"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

type ServiceInput struct {
	CategoryID      uuid.UUID
	Name            string
	Description     *string
	PriceAmount     int64
	Currency        string
	DurationMinutes int
	IsActive        bool
}

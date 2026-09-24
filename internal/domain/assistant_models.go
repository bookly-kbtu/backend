package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AssistantIntent struct {
	Query      string     `json:"query"`
	CategoryID *uuid.UUID `json:"category_id"`
	MaxPrice   *int64     `json:"max_price"`
}

type AssistantRequest struct {
	ID                uuid.UUID       `db:"id" json:"id"`
	Transcript        string          `db:"transcript" json:"transcript"`
	ParsedIntent      json.RawMessage `db:"parsed_intent" json:"parsed_intent" swaggertype:"object"`
	Status            string          `db:"status" json:"status"`
	SelectedMasterID  *uuid.UUID      `db:"selected_master_id" json:"selected_master_id"`
	SelectedServiceID *uuid.UUID      `db:"selected_service_id" json:"selected_service_id"`
	SelectedBookingID *uuid.UUID      `db:"selected_booking_id" json:"selected_booking_id"`
	CreatedAt         time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at" json:"updated_at"`
}

type AssistantCandidate struct {
	MasterID        uuid.UUID `db:"master_id" json:"master_id"`
	DisplayName     string    `db:"display_name" json:"display_name"`
	ServiceID       uuid.UUID `db:"service_id" json:"service_id"`
	ServiceName     string    `db:"service_name" json:"service_name"`
	PriceAmount     int64     `db:"price_amount" json:"price_amount"`
	Currency        string    `db:"currency" json:"currency"`
	DurationMinutes int       `db:"duration_minutes" json:"duration_minutes"`
}

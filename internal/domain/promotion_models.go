package domain

import (
	"github.com/google/uuid"
)

type PromoteService struct {
	SourceServiceID uuid.UUID `json:"source_service_id"`
	CategoryID      uuid.UUID `json:"category_id"`
	PriceAmount     *int64    `json:"price_amount"`
	DurationMinutes *int      `json:"duration_minutes"`
}

type PromoteFirmInput struct {
	UserID   uuid.UUID        `json:"user_id"`
	TimeZone string           `json:"time_zone"`
	Services []PromoteService `json:"services"`
}

type Promotion struct {
	MasterID   uuid.UUID `db:"master_id" json:"master_id"`
	LocationID uuid.UUID `db:"location_id" json:"location_id"`
}

package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SourceCity struct {
	ID         uuid.UUID `db:"id"`
	SourceID   uuid.UUID `db:"source_id"`
	SourceCode string    `db:"source_code"`
	ExternalID string    `db:"external_id"`
	Name       string    `db:"name"`
	Slug       *string   `db:"slug"`
	Latitude   *float64  `db:"latitude"`
	Longitude  *float64  `db:"longitude"`
	LastSeenAt time.Time `db:"last_seen_at"`
}

type SourceCatalogCategory struct {
	ID               uuid.UUID          `db:"id"`
	SourceID         uuid.UUID          `db:"source_id"`
	SourceCode       string             `db:"source_code"`
	Kind             SourceCategoryKind `db:"kind"`
	ExternalID       string             `db:"external_id"`
	ParentExternalID *string            `db:"parent_external_id"`
	Name             string             `db:"name"`
	IconURL          *string            `db:"icon_url"`
	LastSeenAt       time.Time          `db:"last_seen_at"`
}

type SourceFirmSummary struct {
	ID             uuid.UUID  `db:"id"`
	SourceID       uuid.UUID  `db:"source_id"`
	SourceCode     string     `db:"source_code"`
	CityID         *uuid.UUID `db:"source_city_id"`
	CityName       *string    `db:"city_name"`
	ExternalID     string     `db:"external_id"`
	Name           string     `db:"name"`
	Category       *string    `db:"category"`
	EntityType     *string    `db:"entity_type"`
	URLKey         *string    `db:"url_key"`
	AddressText    *string    `db:"address_text"`
	AvatarURL      *string    `db:"avatar_url"`
	AverageRating  *float64   `db:"average_rating"`
	RatingsCount   *int       `db:"ratings_count"`
	ReviewsCount   *int       `db:"reviews_count"`
	WorkStartTime  *string    `db:"work_start_time"`
	WorkEndTime    *string    `db:"work_end_time"`
	IsOnline       *bool      `db:"is_online"`
	IsPromoted     *bool      `db:"is_promoted"`
	Latitude       *float64   `db:"latitude"`
	Longitude      *float64   `db:"longitude"`
	DistanceMeters *float64   `db:"distance_meters"`
	ServicesCount  int        `db:"services_count"`
	MastersCount   int        `db:"masters_count"`
	LastSeenAt     time.Time  `db:"last_seen_at"`
}

type SourceFirm struct {
	SourceFirmSummary
	Description *string `db:"description"`
	MapProvider *string `db:"map_provider"`
}

type SourceFirmPhoto struct {
	FirmID     uuid.UUID `db:"source_firm_id"`
	PhotoURL   string    `db:"photo_url"`
	LastSeenAt time.Time `db:"last_seen_at"`
}

type SourceService struct {
	ID                    uuid.UUID `db:"id"`
	FirmID                uuid.UUID `db:"source_firm_id"`
	ExternalID            string    `db:"external_id"`
	CategoryExternalID    *string   `db:"category_external_id"`
	SubcategoryExternalID *string   `db:"subcategory_external_id"`
	Name                  string    `db:"name"`
	Description           *string   `db:"description"`
	PriceMinAmount        *int64    `db:"price_min_amount"`
	PriceMaxAmount        *int64    `db:"price_max_amount"`
	Currency              *string   `db:"currency"`
	DurationMinutes       *int      `db:"duration_minutes"`
	IsExpress             *bool     `db:"is_express"`
	LastSeenAt            time.Time `db:"last_seen_at"`
}

type SourceMaster struct {
	ID             uuid.UUID `db:"id"`
	SourceID       uuid.UUID `db:"source_id"`
	SourceCode     string    `db:"source_code"`
	ExternalID     string    `db:"external_id"`
	DisplayName    string    `db:"display_name"`
	Profession     *string   `db:"profession"`
	ExperienceText *string   `db:"experience_text"`
	AvatarURL      *string   `db:"avatar_url"`
	AverageRating  *float64  `db:"average_rating"`
	RatingsCount   *int      `db:"ratings_count"`
	IsOnline       *bool     `db:"is_online"`
	LastSeenAt     time.Time `db:"last_seen_at"`
}

type SourceFirmFilter struct {
	CityID     *uuid.UUID
	CategoryID *uuid.UUID
	Query      string
	Latitude   *float64
	Longitude  *float64
	RadiusM    *int
	Limit      int
	Offset     int
}

type SourceCatalogRepository interface {
	ListSourceCities(ctx context.Context) ([]SourceCity, error)
	ListSourceCategories(ctx context.Context, kind *SourceCategoryKind, query string, limit int) ([]SourceCatalogCategory, error)
	ListSourceFirms(ctx context.Context, filter SourceFirmFilter) ([]SourceFirmSummary, error)
	GetSourceFirm(ctx context.Context, id uuid.UUID, lat, lng *float64) (*SourceFirm, error)
	ListSourceFirmPhotos(ctx context.Context, firmID uuid.UUID) ([]SourceFirmPhoto, error)
	ListSourceFirmServices(ctx context.Context, firmID uuid.UUID) ([]SourceService, error)
	ListSourceFirmMasters(ctx context.Context, firmID uuid.UUID) ([]SourceMaster, error)
}

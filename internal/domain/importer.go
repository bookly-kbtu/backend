package domain

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/google/uuid"
)

// External catalogue import (zapis.kz and future sources).
// Imported rows are market research data, never Bookly's source of truth.

type ImportRunStatus string

const (
	ImportRunRunning   ImportRunStatus = "running"
	ImportRunCompleted ImportRunStatus = "completed"
	ImportRunFailed    ImportRunStatus = "failed"
	ImportRunCancelled ImportRunStatus = "cancelled"
)

var importRunStatuses = []ImportRunStatus{ImportRunRunning, ImportRunCompleted, ImportRunFailed, ImportRunCancelled}

func (s ImportRunStatus) Valid() bool { return slices.Contains(importRunStatuses, s) }

type SourceCategoryKind string

const (
	SourceCategory    SourceCategoryKind = "category"
	SourceSubcategory SourceCategoryKind = "subcategory"
)

type ImportRun struct {
	ID                uuid.UUID       `db:"id"`
	SourceID          uuid.UUID       `db:"source_id"`
	SourceCode        string          `db:"source_code"`
	Status            ImportRunStatus `db:"status"`
	Params            json.RawMessage `db:"params"`
	StartedAt         time.Time       `db:"started_at"`
	FinishedAt        *time.Time      `db:"finished_at"`
	CitiesProcessed   int             `db:"cities_processed"`
	FirmsProcessed    int             `db:"firms_processed"`
	MastersProcessed  int             `db:"masters_processed"`
	ServicesProcessed int             `db:"services_processed"`
	ErrorsCount       int             `db:"errors_count"`
	ErrorSummary      *string         `db:"error_summary"`
}

// External* types are the normalized shape every source adapter returns.

type ExternalCity struct {
	ExternalID string
	Name       string
	Slug       string
	Latitude   *float64
	Longitude  *float64
}

type ExternalFirm struct {
	ExternalID    string
	Name          string
	Category      string // source type code, e.g. SALON, MASTER, CLINIC
	EntityType    string // human readable, e.g. "Салон красоты"
	URLKey        string
	Address       string
	Description   string
	AvatarURL     string
	AverageRating *float64
	RatingsCount  *int
	ReviewsCount  *int
	WorkStartTime *string // "HH:MM" local time
	WorkEndTime   *string
	IsOnline      *bool
	IsPromoted    *bool
	Latitude      *float64
	Longitude     *float64
	MapProvider   string
	PhotoURLs     []string
	Categories    []ExternalCategory
	Services      []ExternalService
}

type ExternalCategory struct {
	Kind             SourceCategoryKind
	ExternalID       string
	ParentExternalID string
	Name             string
	IconURL          string
}

type ExternalService struct {
	ExternalID            string
	CategoryExternalID    string
	SubcategoryExternalID string
	Name                  string
	Description           string
	PriceMinAmount        *int64 // minor units
	PriceMaxAmount        *int64
	Currency              string
	DurationMinutes       *int
	IsExpress             *bool
}

type ExternalMaster struct {
	ExternalID    string
	DisplayName   string
	Profession    string
	Experience    string
	AvatarURL     string
	AverageRating *float64
	RatingsCount  *int
	IsOnline      *bool
}

// RawPayload is the unparsed response, stored as a snapshot when enabled.
type RawPayload struct {
	Kind        string
	ExternalID  string
	RequestPath string
	Body        json.RawMessage
}

// CatalogSource is implemented by each external catalogue adapter.
type CatalogSource interface {
	Code() string
	Name() string
	BaseURL() string
	Cities(ctx context.Context) ([]ExternalCity, RawPayload, error)
	FirmIDs(ctx context.Context, cityExternalID string) ([]string, RawPayload, error)
	Firm(ctx context.Context, cityExternalID, firmExternalID string) (*ExternalFirm, RawPayload, error)
	FirmMasters(ctx context.Context, cityExternalID, firmExternalID string) ([]ExternalMaster, RawPayload, error)
}

type ImportRepository interface {
	EnsureSource(ctx context.Context, code, name, baseURL string) (uuid.UUID, error)
	CreateRun(ctx context.Context, run *ImportRun) error
	FinishRun(ctx context.Context, run *ImportRun) error
	ListRuns(ctx context.Context, limit int) ([]ImportRun, error)

	UpsertCity(ctx context.Context, sourceID uuid.UUID, city ExternalCity) (uuid.UUID, error)
	// SaveFirm upserts firm with photos, categories and services. Returns firm row ID.
	SaveFirm(ctx context.Context, sourceID, cityID uuid.UUID, firm *ExternalFirm) (uuid.UUID, error)
	SaveFirmMasters(ctx context.Context, sourceID, firmID uuid.UUID, masters []ExternalMaster) error
	SaveSnapshot(ctx context.Context, runID, sourceID uuid.UUID, payload RawPayload) error
}

package importer

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
)

type Repository struct {
	db *postgres.DB
}

func NewRepository(db *postgres.DB) *Repository {
	return &Repository{db: db}
}

// geographyPoint builds a point from nullable $lng, $lat parameters.
func geographyPoint(lng, lat int) string {
	return fmt.Sprintf(`CASE WHEN $%[1]d::float8 IS NULL OR $%[2]d::float8 IS NULL THEN NULL
		ELSE ST_SetSRID(ST_MakePoint($%[1]d::float8, $%[2]d::float8), 4326)::geography END`, lng, lat)
}

func (r *Repository) EnsureSource(ctx context.Context, code, name, baseURL string) (uuid.UUID, error) {
	const query = `
		INSERT INTO data_sources (code, name, base_url)
		VALUES ($1, $2, $3)
		ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, base_url = EXCLUDED.base_url
		RETURNING id`

	var id uuid.UUID
	err := r.db.Q(ctx).QueryRowxContext(ctx, query, code, name, baseURL).Scan(&id)
	return id, err
}

func (r *Repository) CreateRun(ctx context.Context, run *domain.ImportRun) error {
	const query = `
		INSERT INTO import_runs (id, source_id, status, params, started_at)
		VALUES ($1, $2, $3, $4, $5)`

	return postgres.Exec(ctx, r.db.Q(ctx), query, run.ID, run.SourceID, run.Status, run.Params, run.StartedAt)
}

func (r *Repository) FinishRun(ctx context.Context, run *domain.ImportRun) error {
	const query = `
		UPDATE import_runs
		SET status = $2, finished_at = $3,
			cities_processed = $4, firms_processed = $5, masters_processed = $6,
			services_processed = $7, errors_count = $8, error_summary = $9
		WHERE id = $1`

	return postgres.Exec(ctx, r.db.Q(ctx), query,
		run.ID, run.Status, run.FinishedAt,
		run.CitiesProcessed, run.FirmsProcessed, run.MastersProcessed,
		run.ServicesProcessed, run.ErrorsCount, run.ErrorSummary,
	)
}

func (r *Repository) ListRuns(ctx context.Context, limit int) ([]domain.ImportRun, error) {
	const query = `
		SELECT r.id, r.source_id, s.code AS source_code, r.status, r.params, r.started_at, r.finished_at,
			r.cities_processed, r.firms_processed, r.masters_processed, r.services_processed,
			r.errors_count, r.error_summary
		FROM import_runs r
		JOIN data_sources s ON s.id = r.source_id
		ORDER BY r.started_at DESC
		LIMIT $1`

	return postgres.List[domain.ImportRun](ctx, r.db.Q(ctx), query, limit)
}

func (r *Repository) UpsertCity(ctx context.Context, sourceID uuid.UUID, city domain.ExternalCity) (uuid.UUID, error) {
	query := `
		INSERT INTO source_cities (source_id, external_id, name, slug, location)
		VALUES ($1, $2, $3, NULLIF($4, ''), ` + geographyPoint(5, 6) + `)
		ON CONFLICT (source_id, external_id) DO UPDATE SET
			name = EXCLUDED.name, slug = EXCLUDED.slug, location = EXCLUDED.location, last_seen_at = now()
		RETURNING id`

	var id uuid.UUID
	err := r.db.Q(ctx).QueryRowxContext(ctx, query,
		sourceID, city.ExternalID, city.Name, city.Slug, city.Longitude, city.Latitude,
	).Scan(&id)
	return id, err
}

// SaveFirm must run inside a transaction: firm, photos, categories and services are one unit.
func (r *Repository) SaveFirm(ctx context.Context, sourceID, cityID uuid.UUID, f *domain.ExternalFirm) (uuid.UUID, error) {
	query := `
		INSERT INTO source_firms (
			source_id, source_city_id, external_id, name, category, entity_type, url_key,
			address_text, description, avatar_url, average_rating, ratings_count, reviews_count,
			work_start_time, work_end_time, is_online, is_promoted, map_provider, location
		)
		VALUES (
			$1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''),
			NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''), $11, $12, $13,
			$14::time, $15::time, $16, $17, NULLIF($18, ''), ` + geographyPoint(19, 20) + `
		)
		ON CONFLICT (source_id, external_id) DO UPDATE SET
			source_city_id = EXCLUDED.source_city_id, name = EXCLUDED.name,
			category = EXCLUDED.category, entity_type = EXCLUDED.entity_type, url_key = EXCLUDED.url_key,
			address_text = EXCLUDED.address_text, description = EXCLUDED.description,
			avatar_url = EXCLUDED.avatar_url, average_rating = EXCLUDED.average_rating,
			ratings_count = EXCLUDED.ratings_count, reviews_count = EXCLUDED.reviews_count,
			work_start_time = EXCLUDED.work_start_time, work_end_time = EXCLUDED.work_end_time,
			is_online = EXCLUDED.is_online, is_promoted = EXCLUDED.is_promoted,
			map_provider = EXCLUDED.map_provider, location = EXCLUDED.location,
			last_seen_at = now()
		RETURNING id`

	q := r.db.Q(ctx)

	var firmID uuid.UUID
	err := q.QueryRowxContext(ctx, query,
		sourceID, cityID, f.ExternalID, f.Name, f.Category, f.EntityType, f.URLKey,
		f.Address, f.Description, f.AvatarURL, f.AverageRating, f.RatingsCount, f.ReviewsCount,
		f.WorkStartTime, f.WorkEndTime, f.IsOnline, f.IsPromoted, f.MapProvider, f.Longitude, f.Latitude,
	).Scan(&firmID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert firm: %w", err)
	}

	for _, url := range f.PhotoURLs {
		const photoQuery = `
			INSERT INTO source_firm_photos (source_firm_id, photo_url)
			VALUES ($1, $2)
			ON CONFLICT (source_firm_id, photo_url) DO UPDATE SET last_seen_at = now()`
		if err = postgres.Exec(ctx, q, photoQuery, firmID, url); err != nil {
			return uuid.Nil, fmt.Errorf("upsert photo: %w", err)
		}
	}

	for _, c := range f.Categories {
		const categoryQuery = `
			INSERT INTO source_categories (source_id, kind, external_id, parent_external_id, name, icon_url)
			VALUES ($1, $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''))
			ON CONFLICT (source_id, kind, external_id) DO UPDATE SET
				parent_external_id = COALESCE(EXCLUDED.parent_external_id, source_categories.parent_external_id),
				name = EXCLUDED.name,
				icon_url = COALESCE(EXCLUDED.icon_url, source_categories.icon_url),
				last_seen_at = now()`
		if err = postgres.Exec(ctx, q, categoryQuery,
			sourceID, c.Kind, c.ExternalID, c.ParentExternalID, c.Name, c.IconURL,
		); err != nil {
			return uuid.Nil, fmt.Errorf("upsert category: %w", err)
		}
	}

	for _, s := range f.Services {
		const serviceQuery = `
			INSERT INTO source_services (
				source_firm_id, external_id, category_external_id, subcategory_external_id,
				name, description, price_min_amount, price_max_amount, currency, duration_minutes, is_express
			)
			VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, NULLIF($6, ''), $7, $8, NULLIF($9, ''), $10, $11)
			ON CONFLICT (source_firm_id, external_id) DO UPDATE SET
				category_external_id = EXCLUDED.category_external_id,
				subcategory_external_id = EXCLUDED.subcategory_external_id,
				name = EXCLUDED.name, description = EXCLUDED.description,
				price_min_amount = EXCLUDED.price_min_amount, price_max_amount = EXCLUDED.price_max_amount,
				currency = EXCLUDED.currency, duration_minutes = EXCLUDED.duration_minutes,
				is_express = EXCLUDED.is_express, last_seen_at = now()`
		if err = postgres.Exec(ctx, q, serviceQuery,
			firmID, s.ExternalID, s.CategoryExternalID, s.SubcategoryExternalID,
			s.Name, s.Description, s.PriceMinAmount, s.PriceMaxAmount, s.Currency, s.DurationMinutes, s.IsExpress,
		); err != nil {
			return uuid.Nil, fmt.Errorf("upsert service %s: %w", s.ExternalID, err)
		}
	}

	return firmID, nil
}

// SaveFirmMasters must run inside a transaction.
func (r *Repository) SaveFirmMasters(ctx context.Context, sourceID, firmID uuid.UUID, masters []domain.ExternalMaster) error {
	const masterQuery = `
		INSERT INTO source_masters (
			source_id, external_id, display_name, profession, experience_text,
			avatar_url, average_rating, ratings_count, is_online
		)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9)
		ON CONFLICT (source_id, external_id) DO UPDATE SET
			display_name = EXCLUDED.display_name, profession = EXCLUDED.profession,
			experience_text = EXCLUDED.experience_text, avatar_url = EXCLUDED.avatar_url,
			average_rating = EXCLUDED.average_rating, ratings_count = EXCLUDED.ratings_count,
			is_online = EXCLUDED.is_online, last_seen_at = now()
		RETURNING id`

	const linkQuery = `
		INSERT INTO source_firm_masters (source_firm_id, source_master_id)
		VALUES ($1, $2)
		ON CONFLICT (source_firm_id, source_master_id) DO UPDATE SET last_seen_at = now()`

	q := r.db.Q(ctx)
	for _, m := range masters {
		var masterID uuid.UUID
		err := q.QueryRowxContext(ctx, masterQuery,
			sourceID, m.ExternalID, m.DisplayName, m.Profession, m.Experience,
			m.AvatarURL, m.AverageRating, m.RatingsCount, m.IsOnline,
		).Scan(&masterID)
		if err != nil {
			return fmt.Errorf("upsert master %s: %w", m.ExternalID, err)
		}

		if err = postgres.Exec(ctx, q, linkQuery, firmID, masterID); err != nil {
			return fmt.Errorf("link master %s: %w", m.ExternalID, err)
		}
	}
	return nil
}

func (r *Repository) SaveSnapshot(ctx context.Context, runID, sourceID uuid.UUID, p domain.RawPayload) error {
	const query = `
		INSERT INTO source_snapshots (import_run_id, source_id, entity_kind, external_id, request_path, payload)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6)`

	return postgres.Exec(ctx, r.db.Q(ctx), query, runID, sourceID, p.Kind, p.ExternalID, p.RequestPath, []byte(p.Body))
}

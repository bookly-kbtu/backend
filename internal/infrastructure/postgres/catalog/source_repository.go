package catalog

import (
	"context"
	"fmt"
	"strings"

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

func nullablePoint(lng, lat int) string {
	return fmt.Sprintf(`ST_SetSRID(ST_MakePoint($%d::float8, $%d::float8), 4326)::geography`, lng, lat)
}

func firmSummarySelect(distanceExpr string) string {
	if distanceExpr == "" {
		distanceExpr = "NULL::float8"
	}
	return `
		SELECT
			f.id, f.source_id, ds.code AS source_code, f.source_city_id, c.name AS city_name,
			f.external_id, f.name, f.category, f.entity_type, f.url_key, f.address_text, f.avatar_url,
			f.average_rating::float8 AS average_rating, f.ratings_count, f.reviews_count,
			f.work_start_time::text AS work_start_time, f.work_end_time::text AS work_end_time,
			f.is_online, f.is_promoted,
			ST_Y(f.location::geometry) AS latitude, ST_X(f.location::geometry) AS longitude,
			` + distanceExpr + ` AS distance_meters,
			(SELECT count(*) FROM source_services s WHERE s.source_firm_id = f.id) AS services_count,
			(SELECT count(*) FROM source_firm_masters fm WHERE fm.source_firm_id = f.id) AS masters_count,
			f.last_seen_at
		FROM source_firms f
		JOIN data_sources ds ON ds.id = f.source_id
		LEFT JOIN source_cities c ON c.id = f.source_city_id`
}

func (r *Repository) ListSourceCities(ctx context.Context) ([]domain.SourceCity, error) {
	const query = `
		SELECT c.id, c.source_id, ds.code AS source_code, c.external_id, c.name, c.slug,
			ST_Y(c.location::geometry) AS latitude, ST_X(c.location::geometry) AS longitude,
			c.last_seen_at
		FROM source_cities c
		JOIN data_sources ds ON ds.id = c.source_id
		ORDER BY c.name`

	return postgres.List[domain.SourceCity](ctx, r.db.Q(ctx), query)
}

func (r *Repository) ListSourceCategories(
	ctx context.Context,
	kind *domain.SourceCategoryKind,
	search string,
	limit int,
) ([]domain.SourceCatalogCategory, error) {
	args := []any{}
	conditions := []string{}

	if kind != nil {
		args = append(args, *kind)
		conditions = append(conditions, fmt.Sprintf("cat.kind = $%d", len(args)))
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf("cat.name ILIKE $%d", len(args)))
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	args = append(args, limit)
	query := fmt.Sprintf(`
		SELECT cat.id, cat.source_id, ds.code AS source_code, cat.kind, cat.external_id,
			cat.parent_external_id, cat.name, cat.icon_url, cat.last_seen_at
		FROM source_categories cat
		JOIN data_sources ds ON ds.id = cat.source_id
		%s
		ORDER BY cat.kind, cat.name
		LIMIT $%d`, where, len(args))

	return postgres.List[domain.SourceCatalogCategory](ctx, r.db.Q(ctx), query, args...)
}

func (r *Repository) ListSourceFirms(ctx context.Context, filter domain.SourceFirmFilter) ([]domain.SourceFirmSummary, error) {
	args := []any{}
	conditions := []string{}
	distanceExpr := ""

	if filter.Latitude != nil && filter.Longitude != nil {
		args = append(args, *filter.Longitude, *filter.Latitude)
		point := nullablePoint(len(args)-1, len(args))
		distanceExpr = "ST_Distance(f.location, " + point + ")"
		conditions = append(conditions, "f.location IS NOT NULL")
		if filter.RadiusM != nil {
			args = append(args, *filter.RadiusM)
			conditions = append(conditions, fmt.Sprintf("ST_DWithin(f.location, %s, $%d)", point, len(args)))
		}
	}
	if filter.CityID != nil {
		args = append(args, *filter.CityID)
		conditions = append(conditions, fmt.Sprintf("f.source_city_id = $%d", len(args)))
	}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		conditions = append(conditions, fmt.Sprintf("(f.name ILIKE $%d OR f.address_text ILIKE $%d OR f.description ILIKE $%d)", len(args), len(args), len(args)))
	}
	if filter.CategoryID != nil {
		args = append(args, *filter.CategoryID)
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1
				FROM source_categories cat
				JOIN source_services srv ON srv.source_firm_id = f.id
				WHERE cat.id = $%d
					AND cat.source_id = f.source_id
					AND (srv.category_external_id = cat.external_id OR srv.subcategory_external_id = cat.external_id)
			)`, len(args)))
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	orderBy := "f.is_promoted DESC NULLS LAST, f.average_rating DESC NULLS LAST, f.name"
	if distanceExpr != "" {
		orderBy = "distance_meters ASC NULLS LAST, " + orderBy
	}

	args = append(args, filter.Limit, filter.Offset)
	query := firmSummarySelect(distanceExpr) + where + fmt.Sprintf(`
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, orderBy, len(args)-1, len(args))

	return postgres.List[domain.SourceFirmSummary](ctx, r.db.Q(ctx), query, args...)
}

func (r *Repository) GetSourceFirm(ctx context.Context, id uuid.UUID, lat, lng *float64) (*domain.SourceFirm, error) {
	args := []any{id}
	distanceExpr := "NULL::float8"
	if lat != nil && lng != nil {
		args = append(args, *lng, *lat)
		distanceExpr = "ST_Distance(f.location, " + nullablePoint(2, 3) + ")"
	}

	query := `
		SELECT
			f.id, f.source_id, ds.code AS source_code, f.source_city_id, c.name AS city_name,
			f.external_id, f.name, f.category, f.entity_type, f.url_key, f.address_text, f.avatar_url,
			f.average_rating::float8 AS average_rating, f.ratings_count, f.reviews_count,
			f.work_start_time::text AS work_start_time, f.work_end_time::text AS work_end_time,
			f.is_online, f.is_promoted,
			ST_Y(f.location::geometry) AS latitude, ST_X(f.location::geometry) AS longitude,
			` + distanceExpr + ` AS distance_meters,
			(SELECT count(*) FROM source_services s WHERE s.source_firm_id = f.id) AS services_count,
			(SELECT count(*) FROM source_firm_masters fm WHERE fm.source_firm_id = f.id) AS masters_count,
			f.last_seen_at,
			f.description, f.map_provider
		FROM source_firms f
		JOIN data_sources ds ON ds.id = f.source_id
		LEFT JOIN source_cities c ON c.id = f.source_city_id
		WHERE f.id = $1`

	var firm domain.SourceFirm
	if err := postgres.Get(ctx, r.db.Q(ctx), &firm, query, args...); err != nil {
		return nil, err
	}
	return &firm, nil
}

func (r *Repository) ListSourceFirmPhotos(ctx context.Context, firmID uuid.UUID) ([]domain.SourceFirmPhoto, error) {
	const query = `
		SELECT source_firm_id, photo_url, last_seen_at
		FROM source_firm_photos
		WHERE source_firm_id = $1
		ORDER BY first_seen_at`

	return postgres.List[domain.SourceFirmPhoto](ctx, r.db.Q(ctx), query, firmID)
}

func (r *Repository) ListSourceFirmServices(ctx context.Context, firmID uuid.UUID) ([]domain.SourceService, error) {
	const query = `
		SELECT id, source_firm_id, external_id, category_external_id, subcategory_external_id,
			name, description, price_min_amount, price_max_amount, currency,
			duration_minutes, is_express, last_seen_at
		FROM source_services
		WHERE source_firm_id = $1
		ORDER BY name`

	return postgres.List[domain.SourceService](ctx, r.db.Q(ctx), query, firmID)
}

func (r *Repository) ListSourceFirmMasters(ctx context.Context, firmID uuid.UUID) ([]domain.SourceMaster, error) {
	const query = `
		SELECT m.id, m.source_id, ds.code AS source_code, m.external_id, m.display_name,
			m.profession, m.experience_text, m.avatar_url,
			m.average_rating::float8 AS average_rating, m.ratings_count, m.is_online,
			m.last_seen_at
		FROM source_firm_masters fm
		JOIN source_masters m ON m.id = fm.source_master_id
		JOIN data_sources ds ON ds.id = m.source_id
		WHERE fm.source_firm_id = $1
		ORDER BY m.average_rating DESC NULLS LAST, m.display_name`

	return postgres.List[domain.SourceMaster](ctx, r.db.Q(ctx), query, firmID)
}

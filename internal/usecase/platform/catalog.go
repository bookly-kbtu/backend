package platform

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

func page(limit, offset int) (int, int, error) {
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 || offset < 0 {
		return 0, 0, fmt.Errorf("%w: limit must be 1..100 and offset nonnegative", domain.ErrValidation)
	}
	return limit, offset, nil
}

func (s *Service) ListMasters(ctx context.Context, f domain.MasterFilter, public bool) ([]domain.MasterProfile, error) {
	limit, offset, err := page(f.Limit, f.Offset)
	if err != nil {
		return nil, err
	}
	query := `SELECT m.user_id, m.display_name, m.description, m.avatar_url, m.is_active, m.created_at, m.updated_at
 FROM master_profiles m JOIN users u ON u.id=m.user_id
 WHERE ($1='' OR m.display_name ILIKE '%' || $1 || '%')
 AND ($2::uuid IS NULL OR EXISTS (SELECT 1 FROM master_services ms JOIN service_categories c ON c.id=ms.category_id WHERE ms.master_id=m.user_id AND ms.category_id=$2 AND ms.is_active AND c.is_active))
 AND ($3::bool IS NULL OR m.is_active=$3) AND (NOT $4 OR (m.is_active AND u.status='active'))
 ORDER BY m.created_at DESC, m.user_id LIMIT $5 OFFSET $6`
	return postgres.List[domain.MasterProfile](ctx, s.db.Q(ctx), query, strings.TrimSpace(f.Query), f.CategoryID, f.Active, public, limit, offset)
}

func (s *Service) GetPublicMaster(ctx context.Context, id uuid.UUID) (*domain.MasterProfile, error) {
	var p domain.MasterProfile
	err := postgres.Get(ctx, s.db.Q(ctx), &p, `SELECT m.user_id, m.display_name, m.description, m.avatar_url, m.is_active, m.created_at, m.updated_at FROM master_profiles m JOIN users u ON u.id=m.user_id WHERE m.user_id=$1 AND m.is_active AND u.status='active'`, id)
	return &p, err
}

func (s *Service) SetMasterActive(ctx context.Context, id uuid.UUID, active bool) error {
	return s.execOne(ctx, `UPDATE master_profiles SET is_active=$2 WHERE user_id=$1`, id, active)
}

func (s *Service) execOne(ctx context.Context, query string, args ...any) error {
	result, err := s.db.Q(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return mapDBError(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (s *Service) AdminCategories(ctx context.Context, limit, offset int) ([]domain.Category, error) {
	limit, offset, err := page(limit, offset)
	if err != nil {
		return nil, err
	}
	return postgres.List[domain.Category](ctx, s.db.Q(ctx), `SELECT id,name,slug,is_active,created_at,updated_at FROM service_categories ORDER BY name,id LIMIT $1 OFFSET $2`, limit, offset)
}

func (s *Service) SaveCategory(ctx context.Context, id uuid.UUID, in domain.CategoryInput) (*domain.Category, error) {
	name, err := domain.RequireText("name", in.Name)
	if err != nil {
		return nil, err
	}
	if !slugPattern.MatchString(in.Slug) {
		return nil, fmt.Errorf("%w: slug must contain lowercase letters, digits and hyphens", domain.ErrValidation)
	}
	if id == uuid.Nil {
		id = uuid.New()
		err = postgres.Exec(ctx, s.db.Q(ctx), `INSERT INTO service_categories (id,name,slug,is_active) VALUES ($1,$2,$3,COALESCE($4,true))`, id, name, in.Slug, in.IsActive)
	} else {
		err = s.execOne(ctx, `UPDATE service_categories SET name=$2,slug=$3,is_active=COALESCE($4,is_active) WHERE id=$1`, id, name, in.Slug, in.IsActive)
	}
	if err != nil {
		return nil, err
	}
	var result domain.Category
	err = postgres.Get(ctx, s.db.Q(ctx), &result, `SELECT id,name,slug,is_active,created_at,updated_at FROM service_categories WHERE id=$1`, id)
	return &result, err
}

func (s *Service) DisableCategory(ctx context.Context, id uuid.UUID) error {
	return s.execOne(ctx, `UPDATE service_categories SET is_active=false WHERE id=$1`, id)
}

func (s *Service) AdminBookings(ctx context.Context, f domain.BookingFilter) ([]domain.Booking, error) {
	limit, offset, err := page(f.Limit, f.Offset)
	if err != nil {
		return nil, err
	}
	if f.Status != "" {
		if _, err := domain.ParseBookingStatus(f.Status); err != nil {
			return nil, err
		}
	}
	if f.From != nil && f.To != nil && !f.To.After(*f.From) {
		return nil, fmt.Errorf("%w: to must be after from", domain.ErrValidation)
	}
	return postgres.List[domain.Booking](ctx, s.db.Q(ctx), `SELECT id,client_id,master_id,master_service_id,master_location_id,starts_at,ends_at,status,service_name_snapshot,price_amount,currency,duration_minutes,client_comment,created_at,updated_at FROM bookings WHERE ($1::uuid IS NULL OR master_id=$1) AND ($2::uuid IS NULL OR client_id=$2) AND ($3='' OR status=$3) AND ($4::timestamptz IS NULL OR starts_at >= $4) AND ($5::timestamptz IS NULL OR starts_at < $5) ORDER BY starts_at DESC,id LIMIT $6 OFFSET $7`, f.MasterID, f.ClientID, f.Status, f.From, f.To, limit, offset)
}

func (s *Service) GetImportRun(ctx context.Context, id uuid.UUID) (*domain.ImportRun, error) {
	var result domain.ImportRun
	err := postgres.Get(ctx, s.db.Q(ctx), &result, `SELECT r.id,r.source_id,d.code AS source_code,r.status,r.params,r.started_at,r.finished_at,r.cities_processed,r.firms_processed,r.masters_processed,r.services_processed,r.errors_count,r.error_summary FROM import_runs r JOIN data_sources d ON d.id=r.source_id WHERE r.id=$1`, id)
	return &result, err
}

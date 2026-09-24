package platform

import (
	"context"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

func (s *Service) ListCategories(ctx context.Context) ([]domain.Category, error) {
	const query = `SELECT id, name, slug, is_active, created_at, updated_at FROM service_categories WHERE is_active ORDER BY name`
	return postgres.List[domain.Category](ctx, s.db.Q(ctx), query)
}

func (s *Service) ListMasterServices(ctx context.Context, masterID uuid.UUID, activeOnly bool) ([]domain.ServiceItem, error) {
	query := `
		SELECT ms.id, ms.master_id, ms.category_id, c.name AS category_name,
			ms.name, ms.description, ms.price_amount, ms.currency, ms.duration_minutes,
			ms.is_active, ms.created_at, ms.updated_at
		FROM master_services ms
		JOIN service_categories c ON c.id = ms.category_id
		WHERE ms.master_id = $1`
	if activeOnly {
		query += " AND ms.is_active AND c.is_active"
	}
	query += " ORDER BY ms.created_at DESC"
	return postgres.List[domain.ServiceItem](ctx, s.db.Q(ctx), query, masterID)
}

func (s *Service) CreateMasterService(ctx context.Context, masterID uuid.UUID, in domain.ServiceInput) (*domain.ServiceItem, error) {
	name, err := domain.RequireText("name", in.Name)
	if err != nil {
		return nil, err
	}
	if in.Currency == "" {
		in.Currency = domain.DefaultCurrency
	}
	if err := domain.ValidatePrice(in.PriceAmount); err != nil {
		return nil, err
	}
	if err := domain.ValidateCurrency(in.Currency); err != nil {
		return nil, err
	}
	if err := domain.ValidateServiceDuration(in.DurationMinutes); err != nil {
		return nil, err
	}

	id := uuid.New()
	const query = `
		INSERT INTO master_services (id, master_id, category_id, name, description, price_amount, currency, duration_minutes, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)`
	if err := postgres.Exec(ctx, s.db.Q(ctx), query, id, masterID, in.CategoryID, name, in.Description, in.PriceAmount, in.Currency, in.DurationMinutes); err != nil {
		return nil, err
	}
	return s.GetMasterService(ctx, masterID, id)
}

func (s *Service) UpdateMasterService(ctx context.Context, masterID, id uuid.UUID, in domain.ServiceInput) (*domain.ServiceItem, error) {
	if in.Currency == "" {
		in.Currency = domain.DefaultCurrency
	}
	if _, err := domain.RequireText("name", in.Name); err != nil {
		return nil, err
	}
	if err := domain.ValidatePrice(in.PriceAmount); err != nil {
		return nil, err
	}
	if err := domain.ValidateCurrency(in.Currency); err != nil {
		return nil, err
	}
	if err := domain.ValidateServiceDuration(in.DurationMinutes); err != nil {
		return nil, err
	}
	const query = `
		UPDATE master_services
		SET category_id = $3, name = $4, description = $5, price_amount = $6, currency = $7, duration_minutes = $8, is_active = $9
		WHERE master_id = $1 AND id = $2`
	if err := postgres.Exec(ctx, s.db.Q(ctx), query, masterID, id, in.CategoryID, in.Name, in.Description, in.PriceAmount, in.Currency, in.DurationMinutes, in.IsActive); err != nil {
		return nil, err
	}
	return s.GetMasterService(ctx, masterID, id)
}

func (s *Service) DisableMasterService(ctx context.Context, masterID, id uuid.UUID) error {
	return postgres.Exec(ctx, s.db.Q(ctx), `UPDATE master_services SET is_active = false WHERE master_id = $1 AND id = $2`, masterID, id)
}

func (s *Service) GetMasterService(ctx context.Context, masterID, id uuid.UUID) (*domain.ServiceItem, error) {
	const query = `
		SELECT ms.id, ms.master_id, ms.category_id, c.name AS category_name,
			ms.name, ms.description, ms.price_amount, ms.currency, ms.duration_minutes,
			ms.is_active, ms.created_at, ms.updated_at
		FROM master_services ms
		JOIN service_categories c ON c.id = ms.category_id
		WHERE ms.master_id = $1 AND ms.id = $2`
	var item domain.ServiceItem
	if err := postgres.Get(ctx, s.db.Q(ctx), &item, query, masterID, id); err != nil {
		return nil, err
	}
	return &item, nil
}

package platform

import (
	"context"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

func (s *Service) ListMasterLocations(ctx context.Context, masterID uuid.UUID) ([]domain.MasterLocation, error) {
	const query = `
		SELECT id, master_id, name, address_text,
			ST_Y(location::geometry) AS latitude, ST_X(location::geometry) AS longitude,
			time_zone, is_primary, is_active, created_at, updated_at
		FROM master_locations
		WHERE master_id = $1 AND is_active
		ORDER BY is_primary DESC, created_at`
	return postgres.List[domain.MasterLocation](ctx, s.db.Q(ctx), query, masterID)
}

func (s *Service) CreateMasterLocation(ctx context.Context, masterID uuid.UUID, in domain.LocationInput) (*domain.MasterLocation, error) {
	name, err := domain.RequireText("name", in.Name)
	if err != nil {
		return nil, err
	}
	address, err := domain.RequireText("address_text", in.AddressText)
	if err != nil {
		return nil, err
	}
	if in.TimeZone == "" {
		in.TimeZone = "Asia/Almaty"
	}

	id := uuid.New()
	err = s.db.WithinTx(ctx, func(ctx context.Context) error {
		if in.IsPrimary {
			if err := postgres.Exec(ctx, s.db.Q(ctx), `UPDATE master_locations SET is_primary = false WHERE master_id = $1`, masterID); err != nil {
				return err
			}
		}
		query := `
			INSERT INTO master_locations (id, master_id, name, address_text, location, time_zone, is_primary)
			VALUES ($1, $2, $3, $4,
				CASE WHEN $5::float8 IS NULL OR $6::float8 IS NULL THEN NULL ELSE ST_SetSRID(ST_MakePoint($5::float8, $6::float8), 4326)::geography END,
				$7, $8)`
		return postgres.Exec(ctx, s.db.Q(ctx), query, id, masterID, name, address, in.Longitude, in.Latitude, in.TimeZone, in.IsPrimary)
	})
	if err != nil {
		return nil, err
	}
	return s.GetMasterLocation(ctx, masterID, id)
}

func (s *Service) GetMasterLocation(ctx context.Context, masterID, id uuid.UUID) (*domain.MasterLocation, error) {
	const query = `
		SELECT id, master_id, name, address_text,
			ST_Y(location::geometry) AS latitude, ST_X(location::geometry) AS longitude,
			time_zone, is_primary, is_active, created_at, updated_at
		FROM master_locations
		WHERE master_id = $1 AND id = $2`
	var loc domain.MasterLocation
	if err := postgres.Get(ctx, s.db.Q(ctx), &loc, query, masterID, id); err != nil {
		return nil, err
	}
	return &loc, nil
}

func (s *Service) UpdateMasterLocation(ctx context.Context, masterID, id uuid.UUID, in domain.LocationInput) (*domain.MasterLocation, error) {
	if in.TimeZone == "" {
		in.TimeZone = "Asia/Almaty"
	}
	err := s.db.WithinTx(ctx, func(ctx context.Context) error {
		if in.IsPrimary {
			if err := postgres.Exec(ctx, s.db.Q(ctx), `UPDATE master_locations SET is_primary = false WHERE master_id = $1`, masterID); err != nil {
				return err
			}
		}
		query := `
			UPDATE master_locations
			SET name = $3, address_text = $4,
				location = CASE WHEN $5::float8 IS NULL OR $6::float8 IS NULL THEN NULL ELSE ST_SetSRID(ST_MakePoint($5::float8, $6::float8), 4326)::geography END,
				time_zone = $7, is_primary = $8
			WHERE master_id = $1 AND id = $2`
		return postgres.Exec(ctx, s.db.Q(ctx), query, masterID, id, in.Name, in.AddressText, in.Longitude, in.Latitude, in.TimeZone, in.IsPrimary)
	})
	if err != nil {
		return nil, err
	}
	return s.GetMasterLocation(ctx, masterID, id)
}

func (s *Service) DisableMasterLocation(ctx context.Context, masterID, id uuid.UUID) error {
	return postgres.Exec(ctx, s.db.Q(ctx), `UPDATE master_locations SET is_active = false, is_primary = false WHERE master_id = $1 AND id = $2`, masterID, id)
}

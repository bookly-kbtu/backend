package platform

import (
	"context"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

func (s *Service) UpsertMasterProfile(ctx context.Context, userID uuid.UUID, in domain.UpsertMasterProfileInput) (*domain.MasterProfile, error) {
	displayName, err := domain.RequireText("display_name", in.DisplayName)
	if err != nil {
		return nil, err
	}

	err = s.db.WithinTx(ctx, func(ctx context.Context) error {
		const upsert = `
			INSERT INTO master_profiles (user_id, display_name, description, is_active)
			VALUES ($1, $2, $3, true)
			ON CONFLICT (user_id) DO UPDATE SET
				display_name = EXCLUDED.display_name,
				description = EXCLUDED.description`
		if err := postgres.Exec(ctx, s.db.Q(ctx), upsert, userID, displayName, in.Description); err != nil {
			return err
		}
		const role = `INSERT INTO user_roles (user_id, role_code) VALUES ($1, 'master') ON CONFLICT DO NOTHING`
		return postgres.Exec(ctx, s.db.Q(ctx), role, userID)
	})
	if err != nil {
		return nil, err
	}
	return s.GetMasterProfile(ctx, userID)
}

func (s *Service) GetMasterProfile(ctx context.Context, userID uuid.UUID) (*domain.MasterProfile, error) {
	const query = `
		SELECT user_id, display_name, description, avatar_url, is_active, created_at, updated_at
		FROM master_profiles
		WHERE user_id = $1`

	var profile domain.MasterProfile
	if err := postgres.Get(ctx, s.db.Q(ctx), &profile, query, userID); err != nil {
		return nil, err
	}
	return &profile, nil
}

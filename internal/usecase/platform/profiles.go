package platform

import (
	"context"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

func (s *Service) GetClientProfile(ctx context.Context, userID uuid.UUID) (*domain.ClientProfile, error) {
	const query = `
		SELECT cp.user_id, ai.identifier AS phone, cp.first_name, cp.last_name, cp.avatar_url, cp.created_at, cp.updated_at
		FROM client_profiles cp
		LEFT JOIN auth_identities ai ON ai.user_id = cp.user_id AND ai.identity_type = 'phone'
		WHERE cp.user_id = $1`

	var profile domain.ClientProfile
	if err := postgres.Get(ctx, s.db.Q(ctx), &profile, query, userID); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *Service) PatchClientProfile(ctx context.Context, userID uuid.UUID, in domain.PatchClientProfileInput) (*domain.ClientProfile, error) {
	const query = `
		INSERT INTO client_profiles (user_id, first_name, last_name, avatar_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = COALESCE($2, client_profiles.first_name),
			last_name = COALESCE($3, client_profiles.last_name),
			avatar_url = COALESCE($4, client_profiles.avatar_url)`

	if err := postgres.Exec(ctx, s.db.Q(ctx), query, userID, in.FirstName, in.LastName, in.AvatarURL); err != nil {
		return nil, err
	}
	return s.GetClientProfile(ctx, userID)
}

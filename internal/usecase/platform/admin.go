package platform

import (
	"context"
	"fmt"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

func (s *Service) ListAdminUsers(ctx context.Context, limit, offset int) ([]domain.AdminUser, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	const query = `
		SELECT u.id, ai.identifier AS phone, u.status, u.created_at, u.updated_at
		FROM users u
		LEFT JOIN auth_identities ai ON ai.user_id = u.id AND ai.identity_type = 'phone'
		ORDER BY u.created_at DESC
		LIMIT $1 OFFSET $2`
	users, err := postgres.List[domain.AdminUser](ctx, s.db.Q(ctx), query, limit, offset)
	if err != nil {
		return nil, err
	}
	for i := range users {
		roles, err := postgres.List[string](ctx, s.db.Q(ctx), `SELECT role_code FROM user_roles WHERE user_id = $1 ORDER BY role_code`, users[i].ID)
		if err != nil {
			return nil, err
		}
		users[i].Roles = roles
	}
	return users, nil
}

func (s *Service) UpdateUserStatus(ctx context.Context, userID uuid.UUID, status string) error {
	parsed, err := domain.ParseUserStatus(status)
	if err != nil {
		return err
	}
	return postgres.Exec(ctx, s.db.Q(ctx), `UPDATE users SET status = $2 WHERE id = $1`, userID, parsed)
}

func (s *Service) ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roles []domain.Role) error {
	return s.db.WithinTx(ctx, func(ctx context.Context) error {
		if err := postgres.Exec(ctx, s.db.Q(ctx), `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
			return err
		}
		for _, role := range roles {
			if !role.Valid() {
				return fmt.Errorf("%w: invalid role %q", domain.ErrValidation, role)
			}
			if err := postgres.Exec(ctx, s.db.Q(ctx), `INSERT INTO user_roles (user_id, role_code) VALUES ($1, $2)`, userID, role); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) MarketStats(ctx context.Context) (*domain.MarketStats, error) {
	const query = `
		SELECT
			(SELECT count(*) FROM source_cities) AS cities_count,
			(SELECT count(*) FROM source_firms) AS firms_count,
			(SELECT count(*) FROM source_services) AS services_count,
			(SELECT count(*) FROM source_masters) AS masters_count`
	var stats domain.MarketStats
	if err := postgres.Get(ctx, s.db.Q(ctx), &stats, query); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *Service) ListImportRuns(ctx context.Context, limit int) ([]domain.ImportRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	const query = `
		SELECT r.id, r.source_id, ds.code AS source_code, r.status, r.params, r.started_at, r.finished_at,
			r.cities_processed, r.firms_processed, r.masters_processed, r.services_processed,
			r.errors_count, r.error_summary
		FROM import_runs r
		JOIN data_sources ds ON ds.id = r.source_id
		ORDER BY r.started_at DESC
		LIMIT $1`
	return postgres.List[domain.ImportRun](ctx, s.db.Q(ctx), query, limit)
}

package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
)

type SessionRepository struct {
	db *postgres.DB
}

func NewSessionRepository(db *postgres.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	const query = `
		INSERT INTO sessions (
			id, user_id, refresh_token_hash, device_info, ip_address,
			expires_at, created_at, last_used_at
		)
		VALUES ($1, $2, $3, $4, $5::inet, $6, $7, $8)`

	return postgres.Exec(ctx, r.db.Q(ctx), query,
		s.ID, s.UserID, s.RefreshTokenHash, s.DeviceInfo, s.IPAddress,
		s.ExpiresAt, s.CreatedAt, s.LastUsedAt,
	)
}

func (r *SessionRepository) GetByTokenHashForUpdate(ctx context.Context, hash string) (*domain.Session, error) {
	const query = `
		SELECT id, user_id, refresh_token_hash, device_info, host(ip_address) AS ip_address,
			expires_at, revoked_at, created_at, last_used_at
		FROM sessions
		WHERE refresh_token_hash = $1
		FOR UPDATE`

	var s domain.Session
	if err := postgres.Get(ctx, r.db.Q(ctx), &s, query, hash); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	const query = `UPDATE sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`

	return postgres.Exec(ctx, r.db.Q(ctx), query, id, at)
}

func (r *SessionRepository) RevokeByTokenHash(ctx context.Context, hash string, at time.Time) error {
	const query = `UPDATE sessions SET revoked_at = $2 WHERE refresh_token_hash = $1 AND revoked_at IS NULL`

	return postgres.Exec(ctx, r.db.Q(ctx), query, hash, at)
}

func (r *SessionRepository) RevokeAllByUser(ctx context.Context, userID uuid.UUID, at time.Time) error {
	const query = `UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`

	return postgres.Exec(ctx, r.db.Q(ctx), query, userID, at)
}

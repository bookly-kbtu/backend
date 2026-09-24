package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
)

type OTPRepository struct {
	db *postgres.DB
}

func NewOTPRepository(db *postgres.DB) *OTPRepository {
	return &OTPRepository{db: db}
}

const otpColumns = `id, auth_identity_id, destination, delivery_channel, code_hash, expires_at,
	attempt_count, max_attempts, resend_available_at, consumed_at,
	host(ip_address) AS ip_address, created_at`

func (r *OTPRepository) Create(ctx context.Context, c *domain.OTPChallenge) error {
	const query = `
		INSERT INTO otp_challenges (
			id, auth_identity_id, destination, delivery_channel, code_hash,
			expires_at, max_attempts, resend_available_at, ip_address, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::inet, $10)`

	return postgres.Exec(ctx, r.db.Q(ctx), query,
		c.ID, c.AuthIdentityID, c.Destination, c.Channel, c.CodeHash,
		c.ExpiresAt, c.MaxAttempts, c.ResendAvailableAt, c.IPAddress, c.CreatedAt,
	)
}

func (r *OTPRepository) GetLatest(ctx context.Context, destination string) (*domain.OTPChallenge, error) {
	const query = `SELECT ` + otpColumns + `
		FROM otp_challenges
		WHERE destination = $1
		ORDER BY created_at DESC
		LIMIT 1`

	var c domain.OTPChallenge
	if err := postgres.Get(ctx, r.db.Q(ctx), &c, query, destination); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *OTPRepository) GetLatestUnconsumedForUpdate(ctx context.Context, destination string) (*domain.OTPChallenge, error) {
	const query = `SELECT ` + otpColumns + `
		FROM otp_challenges
		WHERE destination = $1 AND consumed_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
		FOR UPDATE`

	var c domain.OTPChallenge
	if err := postgres.Get(ctx, r.db.Q(ctx), &c, query, destination); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *OTPRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE otp_challenges SET attempt_count = attempt_count + 1 WHERE id = $1`

	return postgres.Exec(ctx, r.db.Q(ctx), query, id)
}

func (r *OTPRepository) Consume(ctx context.Context, id uuid.UUID, identityID *uuid.UUID, at time.Time) error {
	const query = `
		UPDATE otp_challenges
		SET consumed_at = $2, auth_identity_id = COALESCE($3, auth_identity_id)
		WHERE id = $1`

	return postgres.Exec(ctx, r.db.Q(ctx), query, id, at, identityID)
}

func (r *OTPRepository) AttachIdentity(ctx context.Context, id, identityID uuid.UUID) error {
	const query = `UPDATE otp_challenges SET auth_identity_id = $2 WHERE id = $1`

	return postgres.Exec(ctx, r.db.Q(ctx), query, id, identityID)
}

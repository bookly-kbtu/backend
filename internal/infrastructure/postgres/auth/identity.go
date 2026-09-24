package auth

import (
	"context"

	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
)

type IdentityRepository struct {
	db *postgres.DB
}

func NewIdentityRepository(db *postgres.DB) *IdentityRepository {
	return &IdentityRepository{db: db}
}

const identityColumns = `id, user_id, identity_type, identifier, verified_at, created_at`

func (r *IdentityRepository) Create(ctx context.Context, identity *domain.AuthIdentity) error {
	const query = `
		INSERT INTO auth_identities (id, user_id, identity_type, identifier, verified_at)
		VALUES ($1, $2, $3, $4, $5)`

	return postgres.Exec(ctx, r.db.Q(ctx), query,
		identity.ID, identity.UserID, identity.Type, identity.Identifier, identity.VerifiedAt,
	)
}

func (r *IdentityRepository) GetByIdentifier(ctx context.Context, t domain.IdentityType, identifier string) (*domain.AuthIdentity, error) {
	const query = `SELECT ` + identityColumns + `
		FROM auth_identities
		WHERE identity_type = $1 AND identifier = $2`

	var identity domain.AuthIdentity
	if err := postgres.Get(ctx, r.db.Q(ctx), &identity, query, t, identifier); err != nil {
		return nil, err
	}
	return &identity, nil
}

func (r *IdentityRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.AuthIdentity, error) {
	const query = `SELECT ` + identityColumns + `
		FROM auth_identities
		WHERE user_id = $1
		ORDER BY created_at`

	return postgres.List[domain.AuthIdentity](ctx, r.db.Q(ctx), query, userID)
}

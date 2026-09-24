package user

import (
	"context"

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

func (r *Repository) Create(ctx context.Context, user *domain.User) error {
	const query = `
		INSERT INTO users (id, status)
		VALUES ($1, $2)
		RETURNING created_at, updated_at`

	return r.db.Q(ctx).QueryRowxContext(ctx, query, user.ID, user.Status).
		Scan(&user.CreatedAt, &user.UpdatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const query = `SELECT id, status, created_at, updated_at FROM users WHERE id = $1`

	var user domain.User
	if err := postgres.Get(ctx, r.db.Q(ctx), &user, query, id); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) AddRole(ctx context.Context, userID uuid.UUID, role domain.Role) error {
	const query = `
		INSERT INTO user_roles (user_id, role_code)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING`

	return postgres.Exec(ctx, r.db.Q(ctx), query, userID, role)
}

func (r *Repository) ListRoles(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	const query = `SELECT role_code FROM user_roles WHERE user_id = $1 ORDER BY role_code`

	return postgres.List[domain.Role](ctx, r.db.Q(ctx), query, userID)
}

func (r *Repository) CreateClientProfile(ctx context.Context, profile *domain.ClientProfile) error {
	const query = `
		INSERT INTO client_profiles (user_id, first_name, last_name, avatar_url)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at`

	return r.db.Q(ctx).QueryRowxContext(ctx, query,
		profile.UserID, profile.FirstName, profile.LastName, profile.AvatarURL,
	).Scan(&profile.CreatedAt, &profile.UpdatedAt)
}

func (r *Repository) GetClientProfile(ctx context.Context, userID uuid.UUID) (*domain.ClientProfile, error) {
	const query = `
		SELECT user_id, first_name, last_name, avatar_url, created_at, updated_at
		FROM client_profiles
		WHERE user_id = $1`

	var profile domain.ClientProfile
	if err := postgres.Get(ctx, r.db.Q(ctx), &profile, query, userID); err != nil {
		return nil, err
	}
	return &profile, nil
}

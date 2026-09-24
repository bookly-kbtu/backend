package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
)

type MeOutput struct {
	ID        uuid.UUID
	Phone     string
	FirstName string
	Status    domain.UserStatus
	Roles     []domain.Role
}

func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*MeOutput, error) {
	user, err := s.deps.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	roles, err := s.deps.Users.ListRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	out := &MeOutput{ID: user.ID, Status: user.Status, Roles: roles}

	identities, err := s.deps.Identities.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list identities: %w", err)
	}
	for _, identity := range identities {
		if identity.Type == domain.IdentityPhone {
			out.Phone = identity.Identifier
			break
		}
	}

	profile, err := s.deps.Users.GetClientProfile(ctx, userID)
	switch {
	case errors.Is(err, domain.ErrNotFound):
	case err != nil:
		return nil, fmt.Errorf("get client profile: %w", err)
	case profile.FirstName != nil:
		out.FirstName = *profile.FirstName
	}

	return out, nil
}

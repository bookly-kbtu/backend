package catalog

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type Service struct {
	repo domain.SourceCatalogRepository
}

func New(repo domain.SourceCatalogRepository) *Service {
	return &Service{repo: repo}
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

func (s *Service) ListCities(ctx context.Context) ([]domain.SourceCity, error) {
	return s.repo.ListSourceCities(ctx)
}

func (s *Service) ListCategories(ctx context.Context, kindRaw, query string, limit int) ([]domain.SourceCatalogCategory, error) {
	var kind *domain.SourceCategoryKind
	if strings.TrimSpace(kindRaw) != "" {
		parsed := domain.SourceCategoryKind(kindRaw)
		if parsed != domain.SourceCategory && parsed != domain.SourceSubcategory {
			return nil, fmt.Errorf("%w: invalid category kind %q", domain.ErrValidation, kindRaw)
		}
		kind = &parsed
	}
	return s.repo.ListSourceCategories(ctx, kind, strings.TrimSpace(query), clampLimit(limit))
}

func (s *Service) ListFirms(ctx context.Context, filter domain.SourceFirmFilter) ([]domain.SourceFirmSummary, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Limit = clampLimit(filter.Limit)
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if filter.RadiusM != nil && *filter.RadiusM <= 0 {
		return nil, fmt.Errorf("%w: radius_m must be positive", domain.ErrValidation)
	}
	if (filter.Latitude == nil) != (filter.Longitude == nil) {
		return nil, fmt.Errorf("%w: lat and lng must be passed together", domain.ErrValidation)
	}
	return s.repo.ListSourceFirms(ctx, filter)
}

func (s *Service) GetFirm(ctx context.Context, id uuid.UUID, lat, lng *float64) (*domain.SourceFirm, error) {
	if (lat == nil) != (lng == nil) {
		return nil, fmt.Errorf("%w: lat and lng must be passed together", domain.ErrValidation)
	}
	return s.repo.GetSourceFirm(ctx, id, lat, lng)
}

func (s *Service) ListFirmPhotos(ctx context.Context, firmID uuid.UUID) ([]domain.SourceFirmPhoto, error) {
	return s.repo.ListSourceFirmPhotos(ctx, firmID)
}

func (s *Service) ListFirmServices(ctx context.Context, firmID uuid.UUID) ([]domain.SourceService, error) {
	return s.repo.ListSourceFirmServices(ctx, firmID)
}

func (s *Service) ListFirmMasters(ctx context.Context, firmID uuid.UUID) ([]domain.SourceMaster, error) {
	return s.repo.ListSourceFirmMasters(ctx, firmID)
}

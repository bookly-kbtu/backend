package importer

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/bookly-kbtu/backend/internal/domain"
)

type Service struct {
	log     *slog.Logger
	tx      domain.TxManager
	repo    domain.ImportRepository
	sources map[string]domain.CatalogSource
	now     func() time.Time
}

func New(log *slog.Logger, tx domain.TxManager, repo domain.ImportRepository, sources ...domain.CatalogSource) *Service {
	registry := make(map[string]domain.CatalogSource, len(sources))
	for _, s := range sources {
		registry[s.Code()] = s
	}
	return &Service{log: log, tx: tx, repo: repo, sources: registry, now: time.Now}
}

type SourceInfo struct {
	Code    string
	Name    string
	BaseURL string
}

// Sources lists registered source adapters, sorted by code.
func (s *Service) Sources() []SourceInfo {
	out := make([]SourceInfo, 0, len(s.sources))
	for _, src := range s.sources {
		out = append(out, SourceInfo{Code: src.Code(), Name: src.Name(), BaseURL: src.BaseURL()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// Cities fetches the live city list of a source. Nothing is written.
func (s *Service) Cities(ctx context.Context, sourceCode string) ([]domain.ExternalCity, error) {
	src, err := s.source(sourceCode)
	if err != nil {
		return nil, err
	}

	cities, _, err := src.Cities(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch cities: %w", err)
	}
	return cities, nil
}

func (s *Service) Runs(ctx context.Context, limit int) ([]domain.ImportRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListRuns(ctx, limit)
}

func (s *Service) source(code string) (domain.CatalogSource, error) {
	src, ok := s.sources[code]
	if !ok {
		return nil, fmt.Errorf("%w: unknown source %q", domain.ErrValidation, code)
	}
	return src, nil
}

package importer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
)

const (
	maxErrorSummaryLines = 20
	progressEvery        = 25
)

type RunInput struct {
	SourceCode string
	// CityIDs are source external IDs. Empty with AllCities=false is invalid.
	CityIDs   []string `json:"city_ids,omitempty"`
	AllCities bool     `json:"all_cities,omitempty"`
	// MaxFirms limits firms per city, 0 = no limit.
	MaxFirms int `json:"max_firms,omitempty"`
	// SaveSnapshots stores raw responses in source_snapshots.
	SaveSnapshots bool `json:"save_snapshots,omitempty"`
}

func (in RunInput) validate() error {
	if !in.AllCities && len(in.CityIDs) == 0 {
		return fmt.Errorf("%w: choose cities or all cities", domain.ErrValidation)
	}
	if in.AllCities && len(in.CityIDs) > 0 {
		return fmt.Errorf("%w: cities and all cities are mutually exclusive", domain.ErrValidation)
	}
	if in.MaxFirms < 0 {
		return fmt.Errorf("%w: max firms must not be negative", domain.ErrValidation)
	}
	return nil
}

// run holds mutable state of one import.
type run struct {
	*domain.ImportRun
	src      domain.CatalogSource
	in       RunInput
	errLines []string
}

func (r *run) fail(err error) {
	r.ErrorsCount++
	if len(r.errLines) < maxErrorSummaryLines {
		r.errLines = append(r.errLines, err.Error())
	}
}

// Run imports cities -> firms -> services/masters from one source.
// Per-firm failures are counted and skipped; source-level failures stop the run.
// The run row is always finished, also on Ctrl+C (status cancelled).
func (s *Service) Run(ctx context.Context, in RunInput) (result *domain.ImportRun, err error) {
	src, err := s.source(in.SourceCode)
	if err != nil {
		return nil, err
	}
	if err = in.validate(); err != nil {
		return nil, err
	}

	sourceID, err := s.repo.EnsureSource(ctx, src.Code(), src.Name(), src.BaseURL())
	if err != nil {
		return nil, fmt.Errorf("ensure source: %w", err)
	}

	params, _ := json.Marshal(in)
	r := &run{
		ImportRun: &domain.ImportRun{
			ID:         uuid.New(),
			SourceID:   sourceID,
			SourceCode: src.Code(),
			Status:     domain.ImportRunRunning,
			Params:     params,
			StartedAt:  s.now(),
		},
		src: src,
		in:  in,
	}
	if err = s.repo.CreateRun(ctx, r.ImportRun); err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	s.log.Info("import started", "run_id", r.ID, "source", src.Code())

	defer func() {
		s.finish(r, err)
		result = r.ImportRun
	}()

	return nil, s.importCities(ctx, r)
}

func (s *Service) finish(r *run, runErr error) {
	now := s.now()
	r.FinishedAt = &now

	switch {
	case runErr == nil:
		r.Status = domain.ImportRunCompleted
	case errors.Is(runErr, context.Canceled):
		r.Status = domain.ImportRunCancelled
	default:
		r.Status = domain.ImportRunFailed
		r.errLines = append([]string{"fatal: " + runErr.Error()}, r.errLines...)
	}

	if len(r.errLines) > 0 {
		summary := strings.Join(r.errLines, "\n")
		r.ErrorSummary = &summary
	}

	// Ctrl+C cancels the run ctx, but the run row must still be closed.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.repo.FinishRun(ctx, r.ImportRun); err != nil {
		s.log.Error("finish import run", "run_id", r.ID, "error", err)
	}

	s.log.Info("import finished",
		"run_id", r.ID, "status", r.Status,
		"cities", r.CitiesProcessed, "firms", r.FirmsProcessed,
		"masters", r.MastersProcessed, "services", r.ServicesProcessed,
		"errors", r.ErrorsCount, "duration", now.Sub(r.StartedAt).Round(time.Second),
	)
}

func (s *Service) importCities(ctx context.Context, r *run) error {
	cities, raw, err := r.src.Cities(ctx)
	if err != nil {
		return fmt.Errorf("fetch cities: %w", err)
	}
	if err = s.snapshot(ctx, r, raw); err != nil {
		return err
	}

	selected := cities
	if !r.in.AllCities {
		selected = slices.DeleteFunc(slices.Clone(cities), func(c domain.ExternalCity) bool {
			return !slices.Contains(r.in.CityIDs, c.ExternalID)
		})
		if len(selected) != len(r.in.CityIDs) {
			return fmt.Errorf("%w: unknown city id in %v", domain.ErrValidation, r.in.CityIDs)
		}
	}

	for _, city := range selected {
		if err = s.importCity(ctx, r, city); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) importCity(ctx context.Context, r *run, city domain.ExternalCity) error {
	cityID, err := s.repo.UpsertCity(ctx, r.SourceID, city)
	if err != nil {
		return fmt.Errorf("upsert city %s: %w", city.ExternalID, err)
	}

	firmIDs, raw, err := r.src.FirmIDs(ctx, city.ExternalID)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		r.fail(fmt.Errorf("city %s: search: %w", city.ExternalID, err))
		return nil
	}
	if err = s.snapshot(ctx, r, raw); err != nil {
		return err
	}

	if r.in.MaxFirms > 0 && len(firmIDs) > r.in.MaxFirms {
		firmIDs = firmIDs[:r.in.MaxFirms]
	}

	s.log.Info("city", "city", city.Name, "firms", len(firmIDs))

	for i, firmID := range firmIDs {
		if err = s.importFirm(ctx, r, city.ExternalID, cityID, firmID); err != nil {
			return err
		}
		if (i+1)%progressEvery == 0 {
			s.log.Info("progress", "city", city.Name, "done", i+1, "total", len(firmIDs), "errors", r.ErrorsCount)
		}
	}

	r.CitiesProcessed++
	return nil
}

// importFirm returns an error only when the run must stop (context cancelled).
func (s *Service) importFirm(ctx context.Context, r *run, cityExternalID string, cityID uuid.UUID, firmID string) error {
	firm, firmRaw, err := r.src.Firm(ctx, cityExternalID, firmID)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		r.fail(fmt.Errorf("firm %s: %w", firmID, err))
		return nil
	}

	// Firm is still saved when masters fail: services and location are useful alone.
	masters, mastersRaw, mastersErr := r.src.FirmMasters(ctx, cityExternalID, firmID)
	if mastersErr != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		r.fail(fmt.Errorf("firm %s masters: %w", firmID, mastersErr))
	}

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		id, err := s.repo.SaveFirm(ctx, r.SourceID, cityID, firm)
		if err != nil {
			return err
		}
		if err = s.snapshot(ctx, r, firmRaw); err != nil {
			return err
		}

		if mastersErr != nil {
			return nil
		}
		if err = s.repo.SaveFirmMasters(ctx, r.SourceID, id, masters); err != nil {
			return err
		}
		return s.snapshot(ctx, r, mastersRaw)
	})
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		r.fail(fmt.Errorf("firm %s: save: %w", firmID, err))
		return nil
	}

	r.FirmsProcessed++
	r.ServicesProcessed += len(firm.Services)
	if mastersErr == nil {
		r.MastersProcessed += len(masters)
	}
	return nil
}

func (s *Service) snapshot(ctx context.Context, r *run, raw domain.RawPayload) error {
	if !r.in.SaveSnapshots || len(raw.Body) == 0 {
		return nil
	}
	if err := s.repo.SaveSnapshot(ctx, r.ID, r.SourceID, raw); err != nil {
		return fmt.Errorf("save snapshot %s: %w", raw.Kind, err)
	}
	return nil
}

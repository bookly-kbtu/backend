package platform

import (
	"context"
	"fmt"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

func (s *Service) ListWorkingHours(ctx context.Context, masterID uuid.UUID) ([]domain.WorkingHour, error) {
	const query = `
		SELECT id, master_id, location_id, weekday, start_time::text AS start_time, end_time::text AS end_time, is_active
		FROM working_hours
		WHERE master_id = $1 AND is_active
		ORDER BY weekday, start_time`
	return postgres.List[domain.WorkingHour](ctx, s.db.Q(ctx), query, masterID)
}

func (s *Service) ReplaceWorkingHours(ctx context.Context, masterID uuid.UUID, items []domain.WorkingHourInput) ([]domain.WorkingHour, error) {
	err := s.db.WithinTx(ctx, func(ctx context.Context) error {
		if err := postgres.Exec(ctx, s.db.Q(ctx), `UPDATE working_hours SET is_active = false WHERE master_id = $1`, masterID); err != nil {
			return err
		}
		for _, item := range items {
			start, err := clockMinutes(item.StartTime)
			if err != nil {
				return err
			}
			end, err := clockMinutes(item.EndTime)
			if err != nil {
				return err
			}
			if start >= end {
				return fmt.Errorf("%w: start_time must be before end_time", domain.ErrValidation)
			}
			location, err := s.GetMasterLocation(ctx, masterID, item.LocationID)
			if err != nil {
				return err
			}
			if !location.IsActive {
				return domain.ErrNotFound
			}
			if item.Weekday < 1 || item.Weekday > 7 {
				return fmt.Errorf("%w: weekday must be between 1 and 7", domain.ErrValidation)
			}
			const query = `
				INSERT INTO working_hours (master_id, location_id, weekday, start_time, end_time)
				VALUES ($1, $2, $3, $4::time, $5::time)`
			if err := postgres.Exec(ctx, s.db.Q(ctx), query, masterID, item.LocationID, item.Weekday, item.StartTime, item.EndTime); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.ListWorkingHours(ctx, masterID)
}

func (s *Service) ListScheduleExceptions(ctx context.Context, masterID uuid.UUID) ([]domain.ScheduleExceptionRecord, error) {
	const query = `
		SELECT id, master_id, location_id, exception_kind, starts_on, ends_on,
			local_start_time::text AS local_start_time, local_end_time::text AS local_end_time, reason
		FROM schedule_exceptions
		WHERE master_id = $1
		ORDER BY starts_on DESC`
	return postgres.List[domain.ScheduleExceptionRecord](ctx, s.db.Q(ctx), query, masterID)
}

func (s *Service) CreateScheduleException(ctx context.Context, masterID uuid.UUID, in domain.ScheduleExceptionInput) (*domain.ScheduleExceptionRecord, error) {
	kind, err := domain.ParseExceptionKind(in.ExceptionKind)
	if err != nil {
		return nil, err
	}
	if in.EndsOn.Before(in.StartsOn) {
		return nil, fmt.Errorf("%w: ends_on must not be before starts_on", domain.ErrValidation)
	}
	if in.StartsOn.IsZero() || in.EndsOn.IsZero() {
		return nil, domain.ErrValidation
	}
	validation := domain.ScheduleException{Kind: kind, StartsOn: in.StartsOn, EndsOn: in.EndsOn}
	if in.LocalStartTime != nil {
		m, err := clockMinutes(*in.LocalStartTime)
		if err != nil {
			return nil, err
		}
		validation.LocalStartTime = &domain.LocalTime{Hour: m / 60, Minute: m % 60}
	}
	if in.LocalEndTime != nil {
		m, err := clockMinutes(*in.LocalEndTime)
		if err != nil {
			return nil, err
		}
		validation.LocalEndTime = &domain.LocalTime{Hour: m / 60, Minute: m % 60}
	}
	if err := validation.Validate(); err != nil {
		return nil, err
	}
	if in.LocationID != nil {
		if _, err := s.GetMasterLocation(ctx, masterID, *in.LocationID); err != nil {
			return nil, err
		}
	}
	id := uuid.New()
	const query = `
		INSERT INTO schedule_exceptions (id, master_id, location_id, exception_kind, starts_on, ends_on, local_start_time, local_end_time, reason)
		VALUES ($1, $2, $3, $4, $5::date, $6::date, $7::time, $8::time, $9)`
	if err := postgres.Exec(ctx, s.db.Q(ctx), query, id, masterID, in.LocationID, kind, in.StartsOn, in.EndsOn, in.LocalStartTime, in.LocalEndTime, in.Reason); err != nil {
		return nil, err
	}
	return s.GetScheduleException(ctx, masterID, id)
}

func (s *Service) GetScheduleException(ctx context.Context, masterID, id uuid.UUID) (*domain.ScheduleExceptionRecord, error) {
	const query = `
		SELECT id, master_id, location_id, exception_kind, starts_on, ends_on,
			local_start_time::text AS local_start_time, local_end_time::text AS local_end_time, reason
		FROM schedule_exceptions
		WHERE master_id = $1 AND id = $2`
	var item domain.ScheduleExceptionRecord
	if err := postgres.Get(ctx, s.db.Q(ctx), &item, query, masterID, id); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) DeleteScheduleException(ctx context.Context, masterID, id uuid.UUID) error {
	return postgres.Exec(ctx, s.db.Q(ctx), `DELETE FROM schedule_exceptions WHERE master_id = $1 AND id = $2`, masterID, id)
}

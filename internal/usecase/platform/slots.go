package platform

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
	_ "time/tzdata"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// Slots returns one local calendar day on a 15-minute grid anchored to each shift.
func (s *Service) Slots(ctx context.Context, masterID, serviceID, locationID uuid.UUID, date string) ([]domain.Slot, error) {
	if _, err := s.GetPublicMaster(ctx, masterID); err != nil {
		return nil, err
	}
	service, err := s.GetMasterService(ctx, masterID, serviceID)
	if err != nil {
		return nil, err
	}
	if !service.IsActive {
		return nil, domain.ErrNotFound
	}
	var categoryActive bool
	if err = postgres.Get(ctx, s.db.Q(ctx), &categoryActive, `SELECT is_active FROM service_categories WHERE id=$1`, service.CategoryID); err != nil {
		return nil, err
	}
	if !categoryActive {
		return nil, domain.ErrNotFound
	}
	location, err := s.GetMasterLocation(ctx, masterID, locationID)
	if err != nil {
		return nil, err
	}
	if !location.IsActive {
		return nil, domain.ErrNotFound
	}
	zone, err := time.LoadLocation(location.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid location time zone", domain.ErrValidation)
	}
	day, err := time.ParseInLocation(time.DateOnly, date, zone)
	if err != nil {
		return nil, fmt.Errorf("%w: date must be YYYY-MM-DD", domain.ErrValidation)
	}
	now := time.Now()
	if day.Before(now.In(zone).AddDate(0, 0, -1)) || day.After(now.AddDate(1, 0, 0)) {
		return nil, fmt.Errorf("%w: date must be today or within the next year", domain.ErrValidation)
	}
	hours, err := s.ListWorkingHours(ctx, masterID)
	if err != nil {
		return nil, err
	}
	exceptions, err := s.ListScheduleExceptions(ctx, masterID)
	if err != nil {
		return nil, err
	}
	busy, err := postgres.List[domain.Slot](ctx, s.db.Q(ctx), `SELECT starts_at,ends_at FROM bookings WHERE master_id=$1 AND cancelled_at IS NULL AND starts_at < $3 AND ends_at > $2`, masterID, day, day.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	return calculateSlots(day, now, locationID, service.DurationMinutes, hours, exceptions, busy)
}

func clockMinutes(value string) (int, error) {
	for _, layout := range []string{"15:04", "15:04:05"} {
		v, err := time.Parse(layout, value)
		if err == nil && v.Second() == 0 {
			return v.Hour()*60 + v.Minute(), nil
		}
	}
	return 0, fmt.Errorf("%w: time must be HH:MM with minute precision", domain.ErrValidation)
}

func wallTime(day time.Time, value string) (time.Time, error) {
	m, err := clockMinutes(value)
	if err != nil {
		return time.Time{}, err
	}
	t := time.Date(day.Year(), day.Month(), day.Day(), m/60, m%60, 0, 0, day.Location())
	if t.Hour() != m/60 || t.Minute() != m%60 {
		return time.Time{}, fmt.Errorf("%w: nonexistent local time", domain.ErrValidation)
	}
	return t, nil
}

func calculateSlots(day, now time.Time, locationID uuid.UUID, duration int, hours []domain.WorkingHour, exceptions []domain.ScheduleExceptionRecord, busy []domain.Slot) ([]domain.Slot, error) {
	out := make([]domain.Slot, 0)
	if duration <= 0 {
		return nil, fmt.Errorf("%w: invalid duration", domain.ErrValidation)
	}
	windows := make([]domain.Slot, 0)
	custom := make([]domain.Slot, 0)
	blocked := append([]domain.Slot(nil), busy...)
	interval := func(start, end string) (domain.Slot, error) {
		a, e := wallTime(day, start)
		if e != nil {
			return domain.Slot{}, e
		}
		b, e := wallTime(day, end)
		if e != nil {
			return domain.Slot{}, e
		}
		if !b.After(a) {
			return domain.Slot{}, domain.ErrValidation
		}
		return domain.Slot{StartsAt: a, EndsAt: b}, nil
	}
	weekday := (int(day.Weekday())+6)%7 + 1
	for _, h := range hours {
		if h.IsActive && h.LocationID == locationID && h.Weekday == weekday {
			v, e := interval(h.StartTime, h.EndTime)
			if e != nil {
				return nil, e
			}
			windows = append(windows, v)
		}
	}
	for _, e := range exceptions {
		if e.LocationID != nil && *e.LocationID != locationID {
			continue
		}
		date := day.Format(time.DateOnly)
		if date < e.StartsOn.Format(time.DateOnly) || date > e.EndsOn.Format(time.DateOnly) {
			continue
		}
		if e.ExceptionKind == string(domain.ExceptionUnavailable) {
			return out, nil
		}
		if e.LocalStartTime == nil || e.LocalEndTime == nil {
			return nil, fmt.Errorf("%w: incomplete schedule exception", domain.ErrValidation)
		}
		v, err := interval(*e.LocalStartTime, *e.LocalEndTime)
		if err != nil {
			return nil, err
		}
		switch domain.ExceptionKind(e.ExceptionKind) {
		case domain.ExceptionCustomHours:
			custom = append(custom, v)
		case domain.ExceptionBlockedInterval:
			blocked = append(blocked, v)
		default:
			return nil, domain.ErrValidation
		}
	}
	if len(custom) > 0 {
		windows = custom
	}
	sort.Slice(windows, func(i, j int) bool { return windows[i].StartsAt.Before(windows[j].StartsAt) })
	merged := make([]domain.Slot, 0, len(windows))
	for _, w := range windows {
		n := len(merged)
		if n > 0 && !w.StartsAt.After(merged[n-1].EndsAt) {
			if w.EndsAt.After(merged[n-1].EndsAt) {
				merged[n-1].EndsAt = w.EndsAt
			}
		} else {
			merged = append(merged, w)
		}
	}
	for _, w := range merged {
		for start := w.StartsAt; !start.Add(time.Duration(duration) * time.Minute).After(w.EndsAt); start = start.Add(15 * time.Minute) {
			end := start.Add(time.Duration(duration) * time.Minute)
			if !start.After(now) {
				continue
			}
			free := true
			for _, b := range blocked {
				if start.Before(b.EndsAt) && end.After(b.StartsAt) {
					free = false
					break
				}
			}
			if free {
				out = append(out, domain.Slot{StartsAt: start, EndsAt: end})
			}
		}
	}
	return out, nil
}

func (s *Service) validateBookingSlot(ctx context.Context, in domain.CreateBookingInput) error {
	if !in.StartsAt.After(time.Now()) {
		return fmt.Errorf("%w: starts_at must be in the future", domain.ErrValidation)
	}
	location, err := s.GetMasterLocation(ctx, in.MasterID, in.MasterLocationID)
	if err != nil {
		return err
	}
	zone, err := time.LoadLocation(location.TimeZone)
	if err != nil {
		return domain.ErrValidation
	}
	slots, err := s.Slots(ctx, in.MasterID, in.MasterServiceID, in.MasterLocationID, in.StartsAt.In(zone).Format(time.DateOnly))
	if err != nil {
		return err
	}
	for _, slot := range slots {
		if slot.StartsAt.Equal(in.StartsAt) {
			return nil
		}
	}
	return fmt.Errorf("%w: requested slot is unavailable", domain.ErrConflict)
}

func mapDBError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505", "23P01":
			return fmt.Errorf("%w: resource already exists or time is occupied", domain.ErrConflict)
		case "23503", "23514", "22007", "22008":
			return fmt.Errorf("%w: invalid reference or value", domain.ErrValidation)
		}
	}
	return err
}

package domain

import (
	"fmt"
	"slices"
	"time"
)

type ExceptionKind string

const (
	// ExceptionUnavailable closes whole days (vacation, sick day).
	ExceptionUnavailable ExceptionKind = "unavailable"
	// ExceptionCustomHours replaces regular working hours for one day.
	ExceptionCustomHours ExceptionKind = "custom_hours"
	// ExceptionBlockedInterval removes one interval from one day (lunch, personal errand).
	ExceptionBlockedInterval ExceptionKind = "blocked_interval"
)

var exceptionKinds = []ExceptionKind{ExceptionUnavailable, ExceptionCustomHours, ExceptionBlockedInterval}

func ParseExceptionKind(s string) (ExceptionKind, error) {
	return parseEnum("exception kind", s, exceptionKinds)
}

func (k ExceptionKind) Valid() bool { return slices.Contains(exceptionKinds, k) }

// NeedsTimeRange reports whether the kind requires local start/end time on a single day.
func (k ExceptionKind) NeedsTimeRange() bool {
	return k == ExceptionCustomHours || k == ExceptionBlockedInterval
}

// LocalTime is a wall-clock time of day in the location's time zone.
type LocalTime struct {
	Hour, Minute int
}

func (t LocalTime) Before(other LocalTime) bool {
	return t.Hour*60+t.Minute < other.Hour*60+other.Minute
}

// ScheduleException input as the usecase receives it.
type ScheduleException struct {
	Kind           ExceptionKind
	StartsOn       time.Time // date only
	EndsOn         time.Time // date only
	LocalStartTime *LocalTime
	LocalEndTime   *LocalTime
}

func (e ScheduleException) Validate() error {
	if !e.Kind.Valid() {
		return fmt.Errorf("%w: invalid exception kind %q", ErrValidation, e.Kind)
	}
	if e.EndsOn.Before(e.StartsOn) {
		return fmt.Errorf("%w: ends_on must not be before starts_on", ErrValidation)
	}

	hasTime := e.LocalStartTime != nil || e.LocalEndTime != nil

	if !e.Kind.NeedsTimeRange() {
		if hasTime {
			return fmt.Errorf("%w: %s must not have time range", ErrValidation, e.Kind)
		}
		return nil
	}

	if !e.StartsOn.Equal(e.EndsOn) {
		return fmt.Errorf("%w: %s must be a single day", ErrValidation, e.Kind)
	}
	if e.LocalStartTime == nil || e.LocalEndTime == nil {
		return fmt.Errorf("%w: %s requires start and end time", ErrValidation, e.Kind)
	}
	if !e.LocalStartTime.Before(*e.LocalEndTime) {
		return fmt.Errorf("%w: start time must be before end time", ErrValidation)
	}
	return nil
}

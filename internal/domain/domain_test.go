package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"+7 701 123 45 67":  "+77011234567",
		"8 (701) 123-45-67": "+77011234567",
		"7011234567":        "+77011234567",
		"77011234567":       "+77011234567",
	}
	for in, want := range cases {
		got, err := NormalizePhone(in)
		if err != nil || got != want {
			t.Errorf("NormalizePhone(%q) = %q, %v; want %q", in, got, err, want)
		}
	}

	for _, in := range []string{"", "123", "abc", "0123456789"} {
		if _, err := NormalizePhone(in); !errors.Is(err, ErrValidation) {
			t.Errorf("NormalizePhone(%q) error = %v; want ErrValidation", in, err)
		}
	}
}

func TestBookingTransitions(t *testing.T) {
	allowed := [][2]BookingStatus{
		{BookingPending, BookingConfirmed},
		{BookingPending, BookingCancelledByClient},
		{BookingConfirmed, BookingCompleted},
		{BookingConfirmed, BookingNoShow},
	}
	for _, tr := range allowed {
		if err := TransitionBooking(tr[0], tr[1]); err != nil {
			t.Errorf("%s -> %s: %v", tr[0], tr[1], err)
		}
	}

	denied := [][2]BookingStatus{
		{BookingPending, BookingCompleted},
		{BookingCompleted, BookingCancelledByClient},
		{BookingCancelledByMaster, BookingConfirmed},
		{BookingPending, BookingPending},
	}
	for _, tr := range denied {
		if err := TransitionBooking(tr[0], tr[1]); !errors.Is(err, ErrConflict) {
			t.Errorf("%s -> %s: error = %v; want ErrConflict", tr[0], tr[1], err)
		}
	}

	if err := TransitionBooking(BookingPending, "unknown"); !errors.Is(err, ErrValidation) {
		t.Errorf("unknown status error = %v; want ErrValidation", err)
	}
}

func TestParseEnum(t *testing.T) {
	if s, err := ParseBookingStatus("confirmed"); err != nil || s != BookingConfirmed {
		t.Errorf("ParseBookingStatus(confirmed) = %q, %v", s, err)
	}
	if _, err := ParseOTPChannel("pigeon"); !errors.Is(err, ErrValidation) {
		t.Errorf("ParseOTPChannel(pigeon) error = %v; want ErrValidation", err)
	}
}

func TestScheduleExceptionValidate(t *testing.T) {
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	nine, six := &LocalTime{9, 0}, &LocalTime{18, 0}

	valid := []ScheduleException{
		{Kind: ExceptionUnavailable, StartsOn: day, EndsOn: day.AddDate(0, 0, 7)},
		{Kind: ExceptionCustomHours, StartsOn: day, EndsOn: day, LocalStartTime: nine, LocalEndTime: six},
	}
	for _, e := range valid {
		if err := e.Validate(); err != nil {
			t.Errorf("%+v: %v", e, err)
		}
	}

	invalid := []ScheduleException{
		{Kind: ExceptionUnavailable, StartsOn: day, EndsOn: day, LocalStartTime: nine, LocalEndTime: six},
		{Kind: ExceptionBlockedInterval, StartsOn: day, EndsOn: day.AddDate(0, 0, 1), LocalStartTime: nine, LocalEndTime: six},
		{Kind: ExceptionBlockedInterval, StartsOn: day, EndsOn: day, LocalStartTime: six, LocalEndTime: nine},
		{Kind: "holiday", StartsOn: day, EndsOn: day},
	}
	for _, e := range invalid {
		if err := e.Validate(); !errors.Is(err, ErrValidation) {
			t.Errorf("%+v: error = %v; want ErrValidation", e, err)
		}
	}
}

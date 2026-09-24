package domain

import (
	"fmt"
	"slices"
)

type BookingStatus string

const (
	BookingPending           BookingStatus = "pending"
	BookingConfirmed         BookingStatus = "confirmed"
	BookingCompleted         BookingStatus = "completed"
	BookingCancelledByClient BookingStatus = "cancelled_by_client"
	BookingCancelledByMaster BookingStatus = "cancelled_by_master"
	BookingNoShow            BookingStatus = "no_show"
)

var bookingStatuses = []BookingStatus{
	BookingPending, BookingConfirmed, BookingCompleted,
	BookingCancelledByClient, BookingCancelledByMaster, BookingNoShow,
}

// bookingTransitions is the booking state machine. Statuses absent as keys are terminal.
var bookingTransitions = map[BookingStatus][]BookingStatus{
	BookingPending: {
		BookingConfirmed, BookingCancelledByClient, BookingCancelledByMaster,
	},
	BookingConfirmed: {
		BookingCompleted, BookingNoShow, BookingCancelledByClient, BookingCancelledByMaster,
	},
}

func ParseBookingStatus(s string) (BookingStatus, error) {
	return parseEnum("booking status", s, bookingStatuses)
}

func (s BookingStatus) Valid() bool { return slices.Contains(bookingStatuses, s) }

// IsCancelled reports whether the status releases the time slot.
// The usecase must set bookings.cancelled_at exactly when this is true:
// the DB exclusion constraint ignores rows with cancelled_at set.
func (s BookingStatus) IsCancelled() bool {
	return s == BookingCancelledByClient || s == BookingCancelledByMaster
}

func (s BookingStatus) IsTerminal() bool {
	_, ok := bookingTransitions[s]
	return !ok
}

func (s BookingStatus) CanTransitionTo(next BookingStatus) bool {
	return slices.Contains(bookingTransitions[s], next)
}

// TransitionBooking validates a status change and returns an error the API maps to 409.
func TransitionBooking(from, to BookingStatus) error {
	if !to.Valid() {
		return fmt.Errorf("%w: invalid booking status %q", ErrValidation, to)
	}
	if !from.CanTransitionTo(to) {
		return fmt.Errorf("%w: booking cannot move from %s to %s", ErrConflict, from, to)
	}
	return nil
}

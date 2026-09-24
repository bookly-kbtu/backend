package platform

import (
	"context"
	"fmt"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

func (s *Service) CreateBooking(ctx context.Context, clientID uuid.UUID, in domain.CreateBookingInput) (*domain.Booking, error) {
	if clientID == in.MasterID {
		return nil, fmt.Errorf("%w: cannot book yourself", domain.ErrValidation)
	}
	id := uuid.New()
	err := s.db.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.validateBookingSlot(ctx, in); err != nil {
			return err
		}
		const insert = `
			INSERT INTO bookings (
				id, client_id, master_id, master_service_id, master_location_id,
				starts_at, ends_at, status, service_name_snapshot, price_amount, currency, duration_minutes, client_comment
			)
			SELECT $1, $2, ms.master_id, ms.id, ml.id,
				$5::timestamptz, $5::timestamptz + make_interval(mins => ms.duration_minutes),
				'pending', ms.name, ms.price_amount, ms.currency, ms.duration_minutes, $6
			FROM master_services ms
			JOIN master_locations ml ON ml.id = $4 AND ml.master_id = ms.master_id AND ml.is_active
			WHERE ms.id = $3 AND ms.master_id = $7 AND ms.is_active`
		if err := s.execOne(ctx, insert, id, clientID, in.MasterServiceID, in.MasterLocationID, in.StartsAt, in.ClientComment, in.MasterID); err != nil {
			return err
		}
		const hist = `INSERT INTO booking_status_history (booking_id, to_status, changed_by_user_id) VALUES ($1, 'pending', $2)`
		if err := postgres.Exec(ctx, s.db.Q(ctx), hist, id, clientID); err != nil {
			return err
		}
		return s.notifyBooking(ctx, id, string(domain.NotificationBookingCreated))
	})
	if err != nil {
		return nil, err
	}
	return s.GetBookingForUser(ctx, id, clientID)
}

func (s *Service) ListClientBookings(ctx context.Context, clientID uuid.UUID) ([]domain.Booking, error) {
	const query = `
		SELECT id, client_id, master_id, master_service_id, master_location_id, starts_at, ends_at,
			status, service_name_snapshot, price_amount, currency, duration_minutes, client_comment, created_at, updated_at
		FROM bookings
		WHERE client_id = $1
		ORDER BY starts_at DESC`
	return postgres.List[domain.Booking](ctx, s.db.Q(ctx), query, clientID)
}

func (s *Service) ListMasterBookings(ctx context.Context, masterID uuid.UUID) ([]domain.Booking, error) {
	const query = `
		SELECT id, client_id, master_id, master_service_id, master_location_id, starts_at, ends_at,
			status, service_name_snapshot, price_amount, currency, duration_minutes, client_comment, created_at, updated_at
		FROM bookings
		WHERE master_id = $1
		ORDER BY starts_at DESC`
	return postgres.List[domain.Booking](ctx, s.db.Q(ctx), query, masterID)
}

func (s *Service) GetBookingForUser(ctx context.Context, id, userID uuid.UUID) (*domain.Booking, error) {
	const query = `
		SELECT id, client_id, master_id, master_service_id, master_location_id, starts_at, ends_at,
			status, service_name_snapshot, price_amount, currency, duration_minutes, client_comment, created_at, updated_at
		FROM bookings
		WHERE id = $1 AND (client_id = $2 OR master_id = $2)`
	var booking domain.Booking
	if err := postgres.Get(ctx, s.db.Q(ctx), &booking, query, id, userID); err != nil {
		return nil, err
	}
	return &booking, nil
}

func (s *Service) TransitionBooking(ctx context.Context, id, userID uuid.UUID, next domain.BookingStatus) (*domain.Booking, error) {
	err := s.db.WithinTx(ctx, func(ctx context.Context) error {
		var locked uuid.UUID
		if err := postgres.Get(ctx, s.db.Q(ctx), &locked, `SELECT id FROM bookings WHERE id=$1 AND (client_id=$2 OR master_id=$2) FOR UPDATE`, id, userID); err != nil {
			return err
		}
		booking, err := s.GetBookingForUser(ctx, id, userID)
		if err != nil {
			return err
		}
		if err := authorizeTransition(*booking, userID, next); err != nil {
			return err
		}
		current, err := domain.ParseBookingStatus(booking.Status)
		if err != nil {
			return err
		}
		if err := domain.TransitionBooking(current, next); err != nil {
			return err
		}
		const update = `
			UPDATE bookings
			SET status = $2, cancelled_at = CASE WHEN $3 THEN now() ELSE cancelled_at END
			WHERE id = $1`
		if err := postgres.Exec(ctx, s.db.Q(ctx), update, id, next, next.IsCancelled()); err != nil {
			return err
		}
		const hist = `INSERT INTO booking_status_history (booking_id, from_status, to_status, changed_by_user_id) VALUES ($1, $2, $3, $4)`
		if err := postgres.Exec(ctx, s.db.Q(ctx), hist, id, current, next, userID); err != nil {
			return err
		}
		if next.IsCancelled() {
			return s.notifyBooking(ctx, id, string(domain.NotificationBookingCancelled))
		}
		if next == domain.BookingConfirmed {
			return s.notifyBooking(ctx, id, string(domain.NotificationBookingConfirmed))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetBookingForUser(ctx, id, userID)
}

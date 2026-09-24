package platform

import (
	"context"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

func (s *Service) Notifications(ctx context.Context, userID uuid.UUID, unread bool, limit, offset int) ([]domain.Notification, error) {
	limit, offset, err := page(limit, offset)
	if err != nil {
		return nil, err
	}
	return postgres.List[domain.Notification](ctx, s.db.Q(ctx), `SELECT id,booking_id,notification_type,payload,read_at,created_at FROM notifications WHERE user_id=$1 AND (NOT $2 OR read_at IS NULL) ORDER BY created_at DESC,id LIMIT $3 OFFSET $4`, userID, unread, limit, offset)
}

func (s *Service) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := postgres.Get(ctx, s.db.Q(ctx), &count, `SELECT count(*) FROM notifications WHERE user_id=$1 AND read_at IS NULL`, userID)
	return count, err
}

func (s *Service) ReadNotification(ctx context.Context, userID, id uuid.UUID) error {
	return s.execOne(ctx, `UPDATE notifications SET read_at=COALESCE(read_at,now()) WHERE user_id=$1 AND id=$2`, userID, id)
}

func (s *Service) ReadAllNotifications(ctx context.Context, userID uuid.UUID) error {
	return postgres.Exec(ctx, s.db.Q(ctx), `UPDATE notifications SET read_at=now() WHERE user_id=$1 AND read_at IS NULL`, userID)
}

func (s *Service) notifyBooking(ctx context.Context, id uuid.UUID, kind string) error {
	return postgres.Exec(ctx, s.db.Q(ctx), `INSERT INTO notifications(user_id,booking_id,notification_type,delivery_channel,scheduled_at,sent_at,status,payload)
 SELECT recipient,$1,$2,'in_app',now(),now(),'sent',jsonb_build_object('booking_id',b.id,'status',b.status,'starts_at',b.starts_at,'service_name',b.service_name_snapshot)
 FROM bookings b CROSS JOIN LATERAL (SELECT b.client_id AS recipient UNION SELECT b.master_id) recipients WHERE b.id=$1`, id, kind)
}

func authorizeTransition(b domain.Booking, userID uuid.UUID, next domain.BookingStatus) error {
	if next == domain.BookingCancelledByClient {
		if b.ClientID != userID {
			return domain.ErrForbidden
		}
	} else if b.MasterID != userID {
		return domain.ErrForbidden
	}
	return nil
}

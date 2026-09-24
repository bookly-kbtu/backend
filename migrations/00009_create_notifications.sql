-- +goose Up
-- +goose StatementBegin
CREATE TABLE notifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    booking_id uuid REFERENCES bookings (id) ON DELETE RESTRICT,
    notification_type text NOT NULL,
    delivery_channel text NOT NULL,
    scheduled_at timestamptz NOT NULL,
    sent_at timestamptz,
    status text NOT NULL DEFAULT 'pending',
    attempt_count smallint NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    last_error text,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Worker queue: SELECT ... WHERE status = 'pending' AND scheduled_at <= now() FOR UPDATE SKIP LOCKED.
CREATE INDEX notifications_dispatch_idx ON notifications (scheduled_at) WHERE status = 'pending';
CREATE INDEX notifications_user_created_idx ON notifications (user_id, created_at DESC);
CREATE INDEX notifications_booking_idx ON notifications (booking_id) WHERE booking_id IS NOT NULL;

CREATE TRIGGER notifications_set_updated_at BEFORE UPDATE ON notifications
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notifications;
-- +goose StatementEnd

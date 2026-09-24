-- +goose Up
-- +goose StatementBegin
CREATE TABLE bookings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id uuid NOT NULL REFERENCES client_profiles (user_id) ON DELETE RESTRICT,
    master_id uuid NOT NULL REFERENCES master_profiles (user_id) ON DELETE RESTRICT,
    master_service_id uuid NOT NULL,
    master_location_id uuid NOT NULL,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    booking_period tstzrange GENERATED ALWAYS AS (tstzrange(starts_at, ends_at, '[)')) STORED,
    status text NOT NULL DEFAULT 'pending',
    -- Snapshot: later service price changes must not rewrite past bookings.
    service_name_snapshot text NOT NULL,
    price_amount bigint NOT NULL CHECK (price_amount >= 0),
    currency char(3) NOT NULL,
    duration_minutes integer NOT NULL CHECK (duration_minutes > 0),
    client_comment text,
    cancelled_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (ends_at = starts_at + make_interval(mins => duration_minutes)),
    FOREIGN KEY (master_service_id, master_id) REFERENCES master_services (id, master_id) ON DELETE RESTRICT,
    FOREIGN KEY (master_location_id, master_id) REFERENCES master_locations (id, master_id) ON DELETE RESTRICT,
    -- Final guard against double booking. Keyed by master: one person, one place at a time.
    -- Usecase sets cancelled_at on any cancel status; that row stops blocking the slot.
    -- Keyed on cancelled_at, not status values, so new statuses need no migration.
    CONSTRAINT bookings_no_master_overlap EXCLUDE USING gist (
        master_id WITH =,
        booking_period WITH &&
    ) WHERE (cancelled_at IS NULL)
);

CREATE TABLE booking_status_history (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id uuid NOT NULL REFERENCES bookings (id) ON DELETE RESTRICT,
    from_status text,
    to_status text NOT NULL,
    changed_by_user_id uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (from_status IS NULL OR from_status <> to_status)
);

CREATE INDEX bookings_master_starts_at_idx ON bookings (master_id, starts_at);
CREATE INDEX bookings_client_starts_at_idx ON bookings (client_id, starts_at DESC);
CREATE INDEX booking_status_history_booking_created_idx ON booking_status_history (booking_id, created_at);

CREATE TRIGGER bookings_set_updated_at BEFORE UPDATE ON bookings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS booking_status_history;
DROP TABLE IF EXISTS bookings;
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin
-- Weekly rules in the location's local wall-clock time. Slots are computed, never stored.
CREATE TABLE working_hours (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    master_id uuid NOT NULL REFERENCES master_profiles (user_id) ON DELETE RESTRICT,
    location_id uuid NOT NULL,
    weekday smallint NOT NULL CHECK (weekday BETWEEN 1 AND 7), -- ISO: Monday = 1
    start_time time NOT NULL,
    end_time time NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (start_time < end_time),
    FOREIGN KEY (location_id, master_id) REFERENCES master_locations (id, master_id) ON DELETE RESTRICT
);

-- location_id NULL = applies to every location (vacation, sick day).
CREATE TABLE schedule_exceptions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    master_id uuid NOT NULL REFERENCES master_profiles (user_id) ON DELETE RESTRICT,
    location_id uuid,
    exception_kind text NOT NULL,
    starts_on date NOT NULL,
    ends_on date NOT NULL,
    local_start_time time,
    local_end_time time,
    reason text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (ends_on >= starts_on),
    -- Rules per exception_kind live in domain.ScheduleException.Validate.
    CHECK (local_start_time IS NULL OR local_end_time IS NULL OR local_start_time < local_end_time),
    FOREIGN KEY (location_id, master_id) REFERENCES master_locations (id, master_id) ON DELETE RESTRICT
);

CREATE INDEX working_hours_lookup_idx ON working_hours (master_id, location_id, weekday) WHERE is_active;
CREATE INDEX schedule_exceptions_lookup_idx ON schedule_exceptions (master_id, location_id, starts_on, ends_on);

CREATE TRIGGER working_hours_set_updated_at BEFORE UPDATE ON working_hours
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER schedule_exceptions_set_updated_at BEFORE UPDATE ON schedule_exceptions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS schedule_exceptions;
DROP TABLE IF EXISTS working_hours;
-- +goose StatementEnd

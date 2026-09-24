-- +goose Up
-- +goose StatementBegin
ALTER TABLE notifications ADD COLUMN read_at timestamptz;
CREATE INDEX notifications_unread_idx ON notifications (user_id, created_at DESC) WHERE read_at IS NULL;

CREATE TABLE source_firm_promotions (
    source_firm_id uuid PRIMARY KEY REFERENCES source_firms(id) ON DELETE RESTRICT,
    master_id uuid NOT NULL REFERENCES master_profiles(user_id) ON DELETE RESTRICT,
    location_id uuid NOT NULL REFERENCES master_locations(id) ON DELETE RESTRICT,
    promoted_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE source_firm_promotions;
DROP INDEX notifications_unread_idx;
ALTER TABLE notifications DROP COLUMN read_at;
-- +goose StatementEnd

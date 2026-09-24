-- +goose Up
-- +goose StatementBegin
-- AI assistant audit. The LLM only fills parsed_intent (JSON);
-- search and booking stay deterministic backend code.
CREATE TABLE assistant_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    transcript text NOT NULL,
    parsed_intent jsonb,
    status text NOT NULL DEFAULT 'received',
    selected_master_id uuid REFERENCES master_profiles (user_id) ON DELETE SET NULL,
    selected_service_id uuid REFERENCES master_services (id) ON DELETE SET NULL,
    selected_booking_id uuid UNIQUE REFERENCES bookings (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX assistant_requests_user_created_idx ON assistant_requests (user_id, created_at DESC);

CREATE TRIGGER assistant_requests_set_updated_at BEFORE UPDATE ON assistant_requests
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS assistant_requests;
-- +goose StatementEnd

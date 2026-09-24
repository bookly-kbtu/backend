-- +goose Up
-- +goose StatementBegin
-- One user can own both profiles: book services and provide them.
CREATE TABLE client_profiles (
    user_id uuid PRIMARY KEY REFERENCES users (id) ON DELETE RESTRICT,
    first_name text,
    last_name text,
    avatar_url text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE master_profiles (
    user_id uuid PRIMARY KEY REFERENCES users (id) ON DELETE RESTRICT,
    display_name text NOT NULL,
    description text,
    avatar_url text,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER client_profiles_set_updated_at BEFORE UPDATE ON client_profiles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER master_profiles_set_updated_at BEFORE UPDATE ON master_profiles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS master_profiles;
DROP TABLE IF EXISTS client_profiles;
-- +goose StatementEnd

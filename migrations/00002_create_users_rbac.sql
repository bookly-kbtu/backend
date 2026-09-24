-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    status text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Small static lookup: text codes are clearer than UUIDs in authorization checks.
CREATE TABLE roles (
    code text PRIMARY KEY,
    name text NOT NULL UNIQUE
);

INSERT INTO roles (code, name) VALUES
    ('client', 'Client'),
    ('master', 'Master'),
    ('admin', 'Administrator');

CREATE TABLE user_roles (
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    role_code text NOT NULL REFERENCES roles (code) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_code)
);

CREATE INDEX user_roles_role_code_idx ON user_roles (role_code);

CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd

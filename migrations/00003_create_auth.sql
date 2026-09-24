-- +goose Up
-- +goose StatementBegin
-- One user can sign in by several identities: phone, Telegram account, email.
CREATE TABLE auth_identities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    identity_type text NOT NULL,
    -- phone: E.164 (+77011234567); telegram: numeric user ID; email: lower-cased.
    identifier text NOT NULL,
    verified_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (identity_type, identifier)
);

CREATE INDEX auth_identities_user_idx ON auth_identities (user_id);

-- Phone verification code. The code itself is never stored, only its hash.
CREATE TABLE otp_challenges (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- NULL on first sign-in: identity is created only after successful verification.
    auth_identity_id uuid REFERENCES auth_identities (id) ON DELETE SET NULL,
    destination text NOT NULL,
    -- telegram = Telegram Gateway API, sends code to the phone number owner.
    delivery_channel text NOT NULL,
    code_hash text NOT NULL,
    expires_at timestamptz NOT NULL,
    attempt_count smallint NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    max_attempts smallint NOT NULL DEFAULT 5,
    resend_available_at timestamptz NOT NULL,
    consumed_at timestamptz,
    ip_address inet,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (expires_at > created_at),
    CHECK (resend_available_at >= created_at)
);

CREATE INDEX otp_challenges_destination_created_idx ON otp_challenges (destination, created_at DESC);

-- Only the refresh token hash is stored. Access tokens are stateless JWT.
CREATE TABLE sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    refresh_token_hash text NOT NULL UNIQUE,
    device_info text,
    ip_address inet,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz NOT NULL DEFAULT now(),
    CHECK (expires_at > created_at)
);

CREATE INDEX sessions_active_user_idx ON sessions (user_id, expires_at) WHERE revoked_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS otp_challenges;
DROP TABLE IF EXISTS auth_identities;
-- +goose StatementEnd

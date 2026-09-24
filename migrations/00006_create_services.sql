-- +goose Up
-- +goose StatementBegin
CREATE TABLE service_categories (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    slug text NOT NULL UNIQUE,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Price and duration belong to the master's service, not to the shared category.
CREATE TABLE master_services (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    master_id uuid NOT NULL REFERENCES master_profiles (user_id) ON DELETE RESTRICT,
    category_id uuid NOT NULL REFERENCES service_categories (id) ON DELETE RESTRICT,
    name text NOT NULL,
    description text,
    price_amount bigint NOT NULL CHECK (price_amount >= 0), -- minor units (tiyn for KZT)
    currency char(3) NOT NULL DEFAULT 'KZT',
    duration_minutes integer NOT NULL CHECK (duration_minutes > 0),
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, master_id)
);

CREATE INDEX master_services_category_price_idx ON master_services (category_id, price_amount) WHERE is_active;
CREATE INDEX master_services_active_master_idx ON master_services (master_id) WHERE is_active;

CREATE TRIGGER service_categories_set_updated_at BEFORE UPDATE ON service_categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER master_services_set_updated_at BEFORE UPDATE ON master_services
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS master_services;
DROP TABLE IF EXISTS service_categories;
-- +goose StatementEnd

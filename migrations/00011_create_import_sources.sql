-- +goose Up
-- +goose StatementBegin
-- Research/import schema for external catalogues (zapis.kz, ...).
-- Raw market data only: never Bookly's source of truth, never joined into bookings.
CREATE TABLE data_sources (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code text NOT NULL UNIQUE,
    name text NOT NULL,
    base_url text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE import_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id uuid NOT NULL REFERENCES data_sources (id) ON DELETE RESTRICT,
    status text NOT NULL,
    params jsonb NOT NULL DEFAULT '{}'::jsonb,
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    cities_processed integer NOT NULL DEFAULT 0 CHECK (cities_processed >= 0),
    firms_processed integer NOT NULL DEFAULT 0 CHECK (firms_processed >= 0),
    masters_processed integer NOT NULL DEFAULT 0 CHECK (masters_processed >= 0),
    services_processed integer NOT NULL DEFAULT 0 CHECK (services_processed >= 0),
    errors_count integer NOT NULL DEFAULT 0 CHECK (errors_count >= 0),
    error_summary text
);

CREATE TABLE source_cities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id uuid NOT NULL REFERENCES data_sources (id) ON DELETE CASCADE,
    external_id text NOT NULL,
    name text NOT NULL,
    slug text,
    location geography(Point, 4326),
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, external_id)
);

-- Firm = salon, clinic or individual master's storefront (see category).
CREATE TABLE source_firms (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id uuid NOT NULL REFERENCES data_sources (id) ON DELETE CASCADE,
    source_city_id uuid REFERENCES source_cities (id) ON DELETE SET NULL,
    external_id text NOT NULL,
    name text NOT NULL,
    category text,
    entity_type text,
    url_key text,
    address_text text,
    description text,
    avatar_url text,
    average_rating numeric(4, 2),
    ratings_count integer,
    reviews_count integer,
    work_start_time time,
    work_end_time time,
    is_online boolean,
    is_promoted boolean,
    location geography(Point, 4326),
    map_provider text,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, external_id)
);

CREATE TABLE source_firm_photos (
    source_firm_id uuid NOT NULL REFERENCES source_firms (id) ON DELETE CASCADE,
    photo_url text NOT NULL,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (source_firm_id, photo_url)
);

-- kind separates category and subcategory ID spaces of the source.
CREATE TABLE source_categories (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id uuid NOT NULL REFERENCES data_sources (id) ON DELETE CASCADE,
    kind text NOT NULL,
    external_id text NOT NULL,
    parent_external_id text,
    name text NOT NULL,
    icon_url text,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, kind, external_id)
);

CREATE TABLE source_services (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_firm_id uuid NOT NULL REFERENCES source_firms (id) ON DELETE CASCADE,
    external_id text NOT NULL,
    category_external_id text,
    subcategory_external_id text,
    name text NOT NULL,
    description text,
    price_min_amount bigint CHECK (price_min_amount >= 0), -- minor units
    price_max_amount bigint CHECK (price_max_amount >= 0),
    currency char(3),
    duration_minutes integer CHECK (duration_minutes > 0),
    is_express boolean,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_firm_id, external_id)
);

CREATE TABLE source_masters (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id uuid NOT NULL REFERENCES data_sources (id) ON DELETE CASCADE,
    external_id text NOT NULL,
    display_name text NOT NULL,
    profession text,
    experience_text text,
    avatar_url text,
    average_rating numeric(4, 2),
    ratings_count integer,
    is_online boolean,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_id, external_id)
);

CREATE TABLE source_firm_masters (
    source_firm_id uuid NOT NULL REFERENCES source_firms (id) ON DELETE CASCADE,
    source_master_id uuid NOT NULL REFERENCES source_masters (id) ON DELETE CASCADE,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (source_firm_id, source_master_id)
);

-- Raw responses (optional, --snapshots): lets us re-parse without hitting the source again.
CREATE TABLE source_snapshots (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    import_run_id uuid NOT NULL REFERENCES import_runs (id) ON DELETE CASCADE,
    source_id uuid NOT NULL REFERENCES data_sources (id) ON DELETE CASCADE,
    entity_kind text NOT NULL,
    external_id text,
    request_path text NOT NULL,
    payload jsonb NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX import_runs_source_started_idx ON import_runs (source_id, started_at DESC);
CREATE INDEX source_firms_city_idx ON source_firms (source_city_id, name);
CREATE INDEX source_firms_geo_gist ON source_firms USING gist (location);
CREATE INDEX source_services_firm_idx ON source_services (source_firm_id);
CREATE INDEX source_firm_masters_master_idx ON source_firm_masters (source_master_id);
CREATE INDEX source_snapshots_run_idx ON source_snapshots (import_run_id, fetched_at);

CREATE TRIGGER data_sources_set_updated_at BEFORE UPDATE ON data_sources
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS source_snapshots;
DROP TABLE IF EXISTS source_firm_masters;
DROP TABLE IF EXISTS source_masters;
DROP TABLE IF EXISTS source_services;
DROP TABLE IF EXISTS source_categories;
DROP TABLE IF EXISTS source_firm_photos;
DROP TABLE IF EXISTS source_firms;
DROP TABLE IF EXISTS source_cities;
DROP TABLE IF EXISTS import_runs;
DROP TABLE IF EXISTS data_sources;
-- +goose StatementEnd

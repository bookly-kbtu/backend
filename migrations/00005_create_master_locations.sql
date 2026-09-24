-- +goose Up
-- +goose StatementBegin
CREATE TABLE master_locations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    master_id uuid NOT NULL REFERENCES master_profiles (user_id) ON DELETE RESTRICT,
    name text NOT NULL,
    address_text text NOT NULL,
    -- Search "near me": ST_DWithin(location, ST_MakePoint(:lng, :lat)::geography, :meters)
    location geography(Point, 4326),
    time_zone text NOT NULL DEFAULT 'Asia/Almaty',
    is_primary boolean NOT NULL DEFAULT false,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    -- Target for composite FKs: a child row cannot reference another master's location.
    UNIQUE (id, master_id)
);

CREATE UNIQUE INDEX master_locations_one_primary_per_master ON master_locations (master_id) WHERE is_primary;
CREATE INDEX master_locations_geo_gist ON master_locations USING gist (location);
CREATE INDEX master_locations_active_master_idx ON master_locations (master_id) WHERE is_active;

CREATE TRIGGER master_locations_set_updated_at BEFORE UPDATE ON master_locations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS master_locations;
-- +goose StatementEnd

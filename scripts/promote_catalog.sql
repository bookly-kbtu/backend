-- Promote every imported catalogue firm into a bookable Bookly master:
-- profile, primary location with coordinates, priced services mapped to
-- categories, and weekly hours from the firm's working time.
--
-- Idempotent: reuses source row UUIDs as entity ids + ON CONFLICT DO NOTHING.
-- Safe to re-run after a fresh import.

BEGIN;

-- Masters -------------------------------------------------------------------
INSERT INTO users (id)
SELECT f.id FROM source_firms f
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_roles (user_id, role_code)
SELECT f.id, 'master' FROM source_firms f
ON CONFLICT DO NOTHING;

INSERT INTO master_profiles (user_id, display_name, description, avatar_url)
SELECT f.id, f.name, f.description, f.avatar_url
FROM source_firms f
ON CONFLICT (user_id) DO NOTHING;

-- Locations -------------------------------------------------------------------
INSERT INTO master_locations (id, master_id, name, address_text, location, is_primary)
SELECT f.id, f.id, f.name, COALESCE(NULLIF(f.address_text, ''), 'Алматы'), f.location, true
FROM source_firms f
ON CONFLICT (id) DO NOTHING;

-- Categories: carry source category names over, reusing same-name ones --------
INSERT INTO service_categories (name, slug)
SELECT DISTINCT sc.name, 'cat-' || substr(md5(sc.name), 1, 12)
FROM source_categories sc
WHERE sc.kind = 'category'
  AND NOT EXISTS (SELECT 1 FROM service_categories c WHERE c.name = sc.name)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO service_categories (name, slug)
SELECT 'Другое', 'cat-other'
WHERE NOT EXISTS (SELECT 1 FROM service_categories c WHERE c.name = 'Другое')
ON CONFLICT (slug) DO NOTHING;

-- Services: only priced ones, so zero-price rows don't flood the ranking ------
INSERT INTO master_services
    (id, master_id, category_id, name, description, price_amount, currency, duration_minutes)
SELECT
    ss.id,
    ss.source_firm_id,
    c.id,
    ss.name,
    ss.description,
    ss.price_min_amount,
    COALESCE(ss.currency, 'KZT'),
    COALESCE(ss.duration_minutes, 60)
FROM source_services ss
JOIN source_firms f ON f.id = ss.source_firm_id
LEFT JOIN source_categories sc
    ON sc.source_id = f.source_id AND sc.kind = 'category'
   AND sc.external_id = ss.category_external_id
JOIN service_categories c ON c.name = COALESCE(sc.name, 'Другое')
WHERE ss.price_min_amount > 0
ON CONFLICT (id) DO NOTHING;

-- Weekly hours: the firm's own working time, daily; 10:00-20:00 fallback ------
INSERT INTO working_hours (id, master_id, location_id, weekday, start_time, end_time)
SELECT
    md5(f.id::text || '-wh-' || d)::uuid,
    f.id,
    f.id,
    d,
    CASE WHEN f.work_start_time IS NOT NULL AND f.work_start_time < f.work_end_time
         THEN f.work_start_time ELSE '10:00'::time END,
    CASE WHEN f.work_start_time IS NOT NULL AND f.work_start_time < f.work_end_time
         THEN f.work_end_time ELSE '20:00'::time END
FROM source_firms f
CROSS JOIN generate_series(1, 7) AS d
ON CONFLICT (id) DO NOTHING;

COMMIT;

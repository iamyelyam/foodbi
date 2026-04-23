-- iikoWeb integration (third API surface, distinct from iiko Server and iiko Cloud).
-- Used for tenants on *.iikoweb.ru that authenticate via /api/auth/login with
-- session cookies (no apiLogin UUID, no SHA1 password hash).
ALTER TABLE companies ADD COLUMN IF NOT EXISTS iikoweb_url TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS iikoweb_login TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS iikoweb_password TEXT;

-- Per-location store mapping (mirror of iiko_org_id / numier_tpv_id).
ALTER TABLE locations ADD COLUMN IF NOT EXISTS iikoweb_store_id TEXT;

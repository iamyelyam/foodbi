-- Add NUMIER POS system credentials to companies table
ALTER TABLE companies ADD COLUMN IF NOT EXISTS numier_api_key TEXT;

-- Add NUMIER TPV ID to locations for mapping locations to NUMIER establishments
ALTER TABLE locations ADD COLUMN IF NOT EXISTS numier_tpv_id TEXT;

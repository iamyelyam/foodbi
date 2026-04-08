-- Add iiko Server API credentials to companies table
ALTER TABLE companies ADD COLUMN IF NOT EXISTS iiko_server_url TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS iiko_login TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS iiko_password TEXT;

-- Migrate: if iiko_api_key was used as login, copy it
UPDATE companies SET iiko_login = iiko_api_key WHERE iiko_api_key IS NOT NULL AND iiko_login IS NULL;

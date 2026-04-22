-- Add iiko Cloud API credentials to companies
ALTER TABLE companies ADD COLUMN IF NOT EXISTS iiko_cloud_api_login TEXT;

-- Add iiko Cloud organization ID to locations for mapping locations to iiko Cloud organizations
ALTER TABLE locations ADD COLUMN IF NOT EXISTS iiko_cloud_org_id TEXT;

-- Per-location currency support (previously company-level only)
ALTER TABLE locations
  ADD COLUMN IF NOT EXISTS currency_symbol VARCHAR(5),
  ADD COLUMN IF NOT EXISTS locale VARCHAR(10);

-- Backfill from company defaults
UPDATE locations l
SET currency_symbol = COALESCE(c.currency_symbol, '₸'),
    locale = COALESCE(c.locale, 'ru-KZ')
FROM companies c
WHERE l.company_id = c.id
  AND l.currency_symbol IS NULL;

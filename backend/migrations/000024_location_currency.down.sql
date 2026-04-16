ALTER TABLE locations
  DROP COLUMN IF EXISTS currency_symbol,
  DROP COLUMN IF EXISTS locale;

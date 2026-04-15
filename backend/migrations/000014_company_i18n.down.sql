ALTER TABLE companies
    DROP COLUMN IF EXISTS country,
    DROP COLUMN IF EXISTS currency_code,
    DROP COLUMN IF EXISTS currency_symbol,
    DROP COLUMN IF EXISTS locale;

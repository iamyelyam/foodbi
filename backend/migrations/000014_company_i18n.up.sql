-- Add i18n / locale columns to companies (already present in dev DB, missing in prod).

ALTER TABLE companies
    ADD COLUMN IF NOT EXISTS country VARCHAR(8),
    ADD COLUMN IF NOT EXISTS currency_code VARCHAR(8),
    ADD COLUMN IF NOT EXISTS currency_symbol VARCHAR(8),
    ADD COLUMN IF NOT EXISTS locale VARCHAR(16);

UPDATE companies SET
    country = COALESCE(country, 'KZ'),
    currency_code = COALESCE(currency_code, 'KZT'),
    currency_symbol = COALESCE(currency_symbol, '₸'),
    locale = COALESCE(locale, 'ru-KZ')
WHERE country IS NULL OR currency_code IS NULL OR currency_symbol IS NULL OR locale IS NULL;

-- Seed minimal data into Railway production DB.
-- Run via Railway → Postgres → Data → Query, or `psql $DATABASE_URL -f seed_production.sql`.
-- Safe to re-run.

INSERT INTO companies (id, name, iiko_server_url, iiko_login, iiko_password, country, currency_code, currency_symbol, locale)
VALUES (
    '91c69e0a-d0d1-46d6-a558-9a370e5bfce4',
    'Test Restaurant',
    'https://palaushy-co.iiko.it',
    'Choco',
    '12345',
    'KZ',
    'KZT',
    '₸',
    'ru-KZ'
)
ON CONFLICT (id) DO UPDATE SET
    iiko_server_url = EXCLUDED.iiko_server_url,
    iiko_login = EXCLUDED.iiko_login,
    iiko_password = EXCLUDED.iiko_password,
    country = EXCLUDED.country,
    currency_code = EXCLUDED.currency_code,
    currency_symbol = EXCLUDED.currency_symbol,
    locale = EXCLUDED.locale;

INSERT INTO locations (id, company_id, name, address, iiko_org_id)
VALUES (
    '5895f231-8a99-4833-95fe-99564ebf9e88',
    '91c69e0a-d0d1-46d6-a558-9a370e5bfce4',
    'Палаушы',
    'Алматы',
    '5895f231-8a99-4833-95fe-99564ebf9e88'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, company_id, email, password_hash, first_name, last_name, phone, role, is_active)
VALUES (
    '3321191a-731d-4365-9356-32a65fd6cd67',
    '91c69e0a-d0d1-46d6-a558-9a370e5bfce4',
    'yelyam@choco.kz',
    '$2a$12$5exftFKJ9ZxLzJODziLDXeMunrTGTcYDuR66W/PVWZwyX7.fD1/dq',
    'John',
    'Doe',
    '',
    'owner',
    true
)
ON CONFLICT (id) DO UPDATE SET
    email = EXCLUDED.email,
    password_hash = EXCLUDED.password_hash,
    role = EXCLUDED.role;

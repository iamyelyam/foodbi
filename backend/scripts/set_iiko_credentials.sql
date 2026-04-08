-- Set iiko Server API credentials for the test company
-- Run: psql -U foodbi -d foodbi -f scripts/set_iiko_credentials.sql

UPDATE companies
SET iiko_server_url = 'https://palaushy-co.iiko.it',
    iiko_login = 'Choco',
    iiko_password = '12345'
WHERE id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4';

-- Verify
SELECT id, name, iiko_server_url, iiko_login FROM companies WHERE iiko_server_url IS NOT NULL;

DROP INDEX IF EXISTS idx_companies_bot_link_token;
ALTER TABLE companies DROP COLUMN IF EXISTS bot_link_token;
ALTER TABLE payment_attempts DROP CONSTRAINT IF EXISTS payment_attempts_idempotency_uniq;

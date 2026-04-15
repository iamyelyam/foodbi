-- Idempotency: prevent duplicate payment attempts (at-least-once webhook delivery)
-- A given (company, order, status, exact timestamp) can only land once.
ALTER TABLE payment_attempts
    ADD CONSTRAINT payment_attempts_idempotency_uniq
    UNIQUE (company_id, order_id, status, attempt_at);

-- Separate bot link token from webhook signing secret.
-- webhook_secret = HMAC key for payment systems (high-trust, server-to-server)
-- bot_link_token = short-lived API key for Telegram /start (lower trust, can be rotated)
ALTER TABLE companies ADD COLUMN IF NOT EXISTS bot_link_token VARCHAR(255);
CREATE UNIQUE INDEX IF NOT EXISTS idx_companies_bot_link_token
    ON companies(bot_link_token) WHERE bot_link_token IS NOT NULL;

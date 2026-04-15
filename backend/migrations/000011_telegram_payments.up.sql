-- Payment attempts from external payment system webhooks
CREATE TABLE IF NOT EXISTS payment_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id),
    terminal_id VARCHAR(255) NOT NULL,
    order_id VARCHAR(255) NOT NULL,
    table_number VARCHAR(50) NOT NULL,
    guest_name VARCHAR(255),
    guest_phone VARCHAR(50),
    amount NUMERIC(12, 0) NOT NULL CHECK (amount >= 0),
    status VARCHAR(20) NOT NULL CHECK (status IN ('failed', 'success')),
    attempt_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payment_attempts_company ON payment_attempts(company_id);
CREATE INDEX idx_payment_attempts_terminal ON payment_attempts(company_id, terminal_id);
CREATE INDEX idx_payment_attempts_order ON payment_attempts(company_id, order_id, status);
CREATE INDEX idx_payment_attempts_time ON payment_attempts(company_id, attempt_at DESC);

ALTER TABLE payment_attempts ENABLE ROW LEVEL SECURITY;

CREATE POLICY payment_attempts_tenant ON payment_attempts
    USING (company_id = current_setting('app.current_tenant')::uuid);

-- Telegram bot links: maps telegram chat to a company
CREATE TABLE IF NOT EXISTS telegram_bot_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id),
    telegram_chat_id BIGINT NOT NULL,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(telegram_chat_id)
);

CREATE INDEX idx_telegram_bot_links_company ON telegram_bot_links(company_id);

ALTER TABLE telegram_bot_links ENABLE ROW LEVEL SECURITY;

CREATE POLICY telegram_bot_links_tenant ON telegram_bot_links
    USING (company_id = current_setting('app.current_tenant')::uuid);

-- Telegram subscriptions: which chats receive notifications for which terminals
CREATE TABLE IF NOT EXISTS telegram_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id),
    telegram_chat_id BIGINT NOT NULL,
    terminal_ids TEXT[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(company_id, telegram_chat_id)
);

CREATE INDEX idx_telegram_subs_company ON telegram_subscriptions(company_id);
CREATE INDEX idx_telegram_subs_active ON telegram_subscriptions(company_id, is_active) WHERE is_active = true;

ALTER TABLE telegram_subscriptions ENABLE ROW LEVEL SECURITY;

CREATE POLICY telegram_subscriptions_tenant ON telegram_subscriptions
    USING (company_id = current_setting('app.current_tenant')::uuid);

-- Webhook secrets per company for HMAC verification
ALTER TABLE companies ADD COLUMN IF NOT EXISTS webhook_secret VARCHAR(255);

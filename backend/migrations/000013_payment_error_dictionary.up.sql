-- Dictionary of payment error messages seen in webhooks/events.
-- Auto-populated as new errors arrive; `message_ru` is filled in manually
-- by admins over time to provide localized text in Telegram notifications.
-- Until `message_ru` is filled, the original `message` is shown.
CREATE TABLE IF NOT EXISTS payment_error_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source VARCHAR(100) NOT NULL,       -- e.g. "payment-gateway", "smart-receipt"
    code INTEGER,                        -- numeric code, nullable (some sources don't have one)
    message TEXT NOT NULL,               -- original, e.g. "Insufficient funds"
    message_ru TEXT,                     -- manual translation; NULL = show original
    occurrences_count BIGINT NOT NULL DEFAULT 1,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(source, code, message)
);

CREATE INDEX idx_payment_error_messages_missing_translation
    ON payment_error_messages (last_seen_at DESC)
    WHERE message_ru IS NULL;

CREATE INDEX idx_payment_error_messages_lookup
    ON payment_error_messages (source, code, message);

-- NOT tenant-scoped: the dictionary is shared across all companies.
-- All companies benefit from one admin filling translations.
-- RLS is NOT enabled on this table.

COMMENT ON TABLE payment_error_messages IS
    'Dictionary of payment error messages. Auto-populated by event consumer. Translations filled manually.';
COMMENT ON COLUMN payment_error_messages.message_ru IS
    'Russian translation. NULL = notifier falls back to original message. Fill this to localize.';

-- Phase 6: Email delivery via Resend + outbox pattern
-- Adds an email_outbox table for transactional email enqueue + async processor.
-- Registration + password-reset + invite flows INSERT into this table inside their
-- existing tx, and a separate goroutine (guarded by pg_try_advisory_lock) drains it.

-- Preferred language for transactional email templates (ru | en). Defaults to ru
-- per target audience (Russian-speaking KZ restaurants).
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS preferred_language VARCHAR(8) NOT NULL DEFAULT 'ru';

CREATE TABLE IF NOT EXISTS email_outbox (
    id                BIGSERIAL PRIMARY KEY,
    company_id        UUID NOT NULL,
    user_id           UUID NULL,
    type              TEXT NOT NULL CHECK (type IN ('otp','password_reset','invite')),
    to_email          TEXT NOT NULL,
    template_key      TEXT NOT NULL,
    params            JSONB NOT NULL DEFAULT '{}'::jsonb,
    status            TEXT NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending','sending','sent','retrying','failed','dry_run_skipped')),
    attempts          INT NOT NULL DEFAULT 0,
    last_error        TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at           TIMESTAMPTZ,
    next_attempt_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Processor polling index: picks up pending + retrying rows whose attempt window is ready.
CREATE INDEX IF NOT EXISTS idx_email_outbox_status_next_attempt
  ON email_outbox(status, next_attempt_at);

-- Row-level security: tenant isolation mirrors existing pattern in 000023.
-- Note: FORCE RLS is NOT enabled here; that is Phase 7's job.
ALTER TABLE email_outbox ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_email_outbox ON email_outbox
  FOR ALL USING (company_id = current_setting('app.current_tenant', true)::uuid);

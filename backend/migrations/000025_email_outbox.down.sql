DROP POLICY IF EXISTS tenant_isolation_email_outbox ON email_outbox;
DROP INDEX IF EXISTS idx_email_outbox_status_next_attempt;
DROP TABLE IF EXISTS email_outbox;

ALTER TABLE users DROP COLUMN IF EXISTS preferred_language;

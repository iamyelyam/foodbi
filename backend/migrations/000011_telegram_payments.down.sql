ALTER TABLE companies DROP COLUMN IF EXISTS webhook_secret;

DROP POLICY IF EXISTS telegram_subscriptions_tenant ON telegram_subscriptions;
DROP TABLE IF EXISTS telegram_subscriptions;

DROP POLICY IF EXISTS telegram_bot_links_tenant ON telegram_bot_links;
DROP TABLE IF EXISTS telegram_bot_links;

DROP POLICY IF EXISTS payment_attempts_tenant ON payment_attempts;
DROP TABLE IF EXISTS payment_attempts;

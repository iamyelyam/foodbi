-- Manual stock overrides — user-entered amount and/or unit price that override
-- iiko-reported numbers in the UI. Used for: physical inventory recounts (real
-- amount differs from iiko), missing/wrong unit prices in iiko nomenclature,
-- and any data-quality fixes the restaurant operator wants to make.
--
-- Both columns are nullable: only the columns the user touched are stored.
-- A NULL value means "fall back to the iiko-reported value".

CREATE TABLE stock_overrides (
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    iiko_product_id VARCHAR(255) NOT NULL,
    manual_amount NUMERIC(18, 6),
    manual_price_per_unit NUMERIC(18, 6),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- 'manual' (user typed it) vs 'iiko_synced' (we pushed to iiko and it's now authoritative there too)
    source VARCHAR(32) NOT NULL DEFAULT 'manual',
    PRIMARY KEY (company_id, iiko_product_id)
);

ALTER TABLE stock_overrides ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_overrides FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_stock_overrides ON stock_overrides
    USING (company_id = current_setting('app.current_tenant', true)::uuid);

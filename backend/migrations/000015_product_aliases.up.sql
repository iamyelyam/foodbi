-- User-editable product display names. Overrides whatever iiko gave us
-- (raw GUIDs from deleted nomenclature, ALL-CAPS Russian, internal codes, etc.).

CREATE TABLE product_aliases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    iiko_product_id VARCHAR(255) NOT NULL,
    display_name VARCHAR(500) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(company_id, iiko_product_id)
);

CREATE INDEX idx_product_aliases_company ON product_aliases(company_id);

ALTER TABLE product_aliases ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_aliases FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_product_aliases ON product_aliases
    USING (company_id = current_setting('app.current_tenant', true)::uuid);

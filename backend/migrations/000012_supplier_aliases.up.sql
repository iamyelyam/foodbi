-- User-editable supplier display names. Overrides iiko GUID/whatever came from the API.

CREATE TABLE supplier_aliases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    iiko_supplier_id VARCHAR(255) NOT NULL,
    display_name VARCHAR(500) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(company_id, iiko_supplier_id)
);

CREATE INDEX idx_supplier_aliases_company ON supplier_aliases(company_id);

ALTER TABLE supplier_aliases ENABLE ROW LEVEL SECURITY;
ALTER TABLE supplier_aliases FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_supplier_aliases ON supplier_aliases
    USING (company_id = current_setting('app.current_tenant', true)::uuid);

CREATE TABLE ai_generated_suggestions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    suggestion_type VARCHAR(100) NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    solution TEXT,
    impact VARCHAR(20) DEFAULT 'medium',
    loss_amount NUMERIC(12, 2) DEFAULT 0,
    gain_amount NUMERIC(12, 2) DEFAULT 0,
    raw_ai_response TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_generated_suggestions_company ON ai_generated_suggestions(company_id);
CREATE INDEX idx_ai_generated_suggestions_location ON ai_generated_suggestions(company_id, location_id);

ALTER TABLE ai_generated_suggestions ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_generated_suggestions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_ai_generated ON ai_generated_suggestions
    USING (company_id = current_setting('app.current_tenant')::uuid);

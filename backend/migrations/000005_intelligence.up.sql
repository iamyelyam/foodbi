CREATE TABLE ai_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'done')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_tasks_company ON ai_tasks(company_id);

CREATE TABLE uploaded_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    uploaded_by UUID NOT NULL REFERENCES users(id),
    filename VARCHAR(500) NOT NULL,
    stored_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100),
    size_bytes BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'uploaded' CHECK (status IN ('uploaded', 'processing', 'processed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_uploaded_files_company ON uploaded_files(company_id);

ALTER TABLE ai_tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_tasks FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_ai_tasks ON ai_tasks
    USING (company_id = current_setting('app.current_tenant', true)::uuid);

ALTER TABLE uploaded_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE uploaded_files FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_uploaded_files ON uploaded_files
    USING (company_id = current_setting('app.current_tenant', true)::uuid);

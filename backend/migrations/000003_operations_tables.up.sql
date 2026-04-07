-- Supply requests and transfer requests

CREATE TABLE supply_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    supplier_name VARCHAR(500) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    total_sum NUMERIC(12, 2) NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_supply_requests_company ON supply_requests(company_id);
CREATE INDEX idx_supply_requests_status ON supply_requests(company_id, status);

CREATE TABLE supply_request_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id UUID NOT NULL REFERENCES supply_requests(id) ON DELETE CASCADE,
    product_name VARCHAR(500) NOT NULL,
    category VARCHAR(255),
    quantity NUMERIC(10, 3) NOT NULL,
    unit VARCHAR(50),
    price_per_unit NUMERIC(12, 2) NOT NULL DEFAULT 0,
    sort_order INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_supply_items_request ON supply_request_items(request_id);

CREATE TABLE transfer_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    from_location_id UUID NOT NULL REFERENCES locations(id),
    to_location_id UUID NOT NULL REFERENCES locations(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transfer_requests_company ON transfer_requests(company_id);

CREATE TABLE transfer_request_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id UUID NOT NULL REFERENCES transfer_requests(id) ON DELETE CASCADE,
    product_name VARCHAR(500) NOT NULL,
    category VARCHAR(255),
    quantity NUMERIC(10, 3) NOT NULL,
    unit VARCHAR(50),
    sort_order INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_transfer_items_request ON transfer_request_items(request_id);

-- RLS
ALTER TABLE supply_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE supply_requests FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_supply ON supply_requests
    USING (company_id = current_setting('app.current_tenant', true)::uuid);

ALTER TABLE transfer_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE transfer_requests FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_transfers ON transfer_requests
    USING (company_id = current_setting('app.current_tenant', true)::uuid);

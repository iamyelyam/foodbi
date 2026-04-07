-- Fact tables for synced iiko data + sync tracking

-- Revenue facts (orders/transactions)
CREATE TABLE revenue_facts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    iiko_order_id VARCHAR(255),
    order_date TIMESTAMPTZ NOT NULL,
    revenue NUMERIC(12, 2) NOT NULL DEFAULT 0,
    discount NUMERIC(12, 2) NOT NULL DEFAULT 0,
    order_type VARCHAR(50),
    status VARCHAR(50) NOT NULL DEFAULT 'closed',
    item_count INT NOT NULL DEFAULT 0,
    waiter_name VARCHAR(255),
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(company_id, iiko_order_id)
);

CREATE INDEX idx_revenue_facts_company ON revenue_facts(company_id);
CREATE INDEX idx_revenue_facts_location ON revenue_facts(location_id);
CREATE INDEX idx_revenue_facts_date ON revenue_facts(order_date);
CREATE INDEX idx_revenue_facts_company_date ON revenue_facts(company_id, order_date);

-- Product sales facts
CREATE TABLE product_sales_facts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    iiko_product_id VARCHAR(255),
    product_name VARCHAR(500) NOT NULL,
    category VARCHAR(255),
    sale_date DATE NOT NULL,
    quantity NUMERIC(10, 3) NOT NULL DEFAULT 0,
    revenue NUMERIC(12, 2) NOT NULL DEFAULT 0,
    cost_price NUMERIC(12, 2) NOT NULL DEFAULT 0,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_product_sales_company_date ON product_sales_facts(company_id, sale_date);
CREATE INDEX idx_product_sales_location ON product_sales_facts(location_id);

-- Purchase facts (invoices from suppliers)
CREATE TABLE purchase_facts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    iiko_invoice_id VARCHAR(255),
    document_number VARCHAR(100),
    supplier_id VARCHAR(255),
    supplier_name VARCHAR(500),
    incoming_date TIMESTAMPTZ NOT NULL,
    status VARCHAR(50),
    total_sum NUMERIC(12, 2) NOT NULL DEFAULT 0,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(company_id, iiko_invoice_id)
);

CREATE INDEX idx_purchase_facts_company_date ON purchase_facts(company_id, incoming_date);
CREATE INDEX idx_purchase_facts_location ON purchase_facts(location_id);

-- Stock snapshots (point-in-time inventory)
CREATE TABLE stock_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
    iiko_product_id VARCHAR(255),
    product_name VARCHAR(500) NOT NULL,
    amount NUMERIC(10, 3) NOT NULL DEFAULT 0,
    unit VARCHAR(50),
    cost_sum NUMERIC(12, 2) NOT NULL DEFAULT 0,
    snapshot_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stock_snapshots_company ON stock_snapshots(company_id);
CREATE INDEX idx_stock_snapshots_location_time ON stock_snapshots(location_id, snapshot_at);

-- Sync log (tracks sync status per company/location)
CREATE TABLE iiko_sync_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    location_id UUID REFERENCES locations(id) ON DELETE CASCADE,
    sync_type VARCHAR(50) NOT NULL, -- 'revenue', 'purchases', 'stock', 'organizations'
    status VARCHAR(20) NOT NULL, -- 'success', 'failed', 'running'
    records_synced INT NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INT
);

CREATE INDEX idx_sync_log_company ON iiko_sync_log(company_id);
CREATE INDEX idx_sync_log_type_status ON iiko_sync_log(sync_type, status);
CREATE INDEX idx_sync_log_started ON iiko_sync_log(started_at);

-- Enable RLS on all new fact tables
ALTER TABLE revenue_facts ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_sales_facts ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_facts ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE iiko_sync_log ENABLE ROW LEVEL SECURITY;

ALTER TABLE revenue_facts FORCE ROW LEVEL SECURITY;
ALTER TABLE product_sales_facts FORCE ROW LEVEL SECURITY;
ALTER TABLE purchase_facts FORCE ROW LEVEL SECURITY;
ALTER TABLE stock_snapshots FORCE ROW LEVEL SECURITY;
ALTER TABLE iiko_sync_log FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_revenue ON revenue_facts
    USING (company_id = current_setting('app.current_tenant', true)::uuid);
CREATE POLICY tenant_isolation_product_sales ON product_sales_facts
    USING (company_id = current_setting('app.current_tenant', true)::uuid);
CREATE POLICY tenant_isolation_purchases ON purchase_facts
    USING (company_id = current_setting('app.current_tenant', true)::uuid);
CREATE POLICY tenant_isolation_stock ON stock_snapshots
    USING (company_id = current_setting('app.current_tenant', true)::uuid);
CREATE POLICY tenant_isolation_sync_log ON iiko_sync_log
    USING (company_id = current_setting('app.current_tenant', true)::uuid);

-- Materialized view for dashboard KPIs (refreshed after each sync)
CREATE MATERIALIZED VIEW IF NOT EXISTS dashboard_daily_revenue AS
SELECT
    company_id,
    location_id,
    DATE(order_date) AS day,
    COUNT(*) AS order_count,
    SUM(revenue) AS total_revenue,
    SUM(discount) AS total_discount,
    SUM(item_count) AS total_items
FROM revenue_facts
GROUP BY company_id, location_id, DATE(order_date);

CREATE UNIQUE INDEX idx_dashboard_daily_revenue_pk
    ON dashboard_daily_revenue(company_id, location_id, day);

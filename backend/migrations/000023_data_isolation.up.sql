-- Phase 1: Data Isolation for SaaS scale-up
-- Fixes critical issues: unique constraints, missing RLS, indexes

-- ============================================================
-- 1.3 Fix unique constraints: include location_id
-- Without this, orders from different locations overwrite each other
-- ============================================================

-- revenue_facts: UNIQUE(company_id, iiko_order_id) → UNIQUE(company_id, location_id, iiko_order_id)
ALTER TABLE revenue_facts DROP CONSTRAINT IF EXISTS revenue_facts_company_id_iiko_order_id_key;
ALTER TABLE revenue_facts ADD CONSTRAINT revenue_facts_unique_order
  UNIQUE(company_id, location_id, iiko_order_id);

-- purchase_facts: same fix
ALTER TABLE purchase_facts DROP CONSTRAINT IF EXISTS purchase_facts_company_id_iiko_invoice_id_key;
ALTER TABLE purchase_facts ADD CONSTRAINT purchase_facts_unique_invoice
  UNIQUE(company_id, location_id, iiko_invoice_id);

-- ============================================================
-- 1.2 Add company_id + RLS to item tables
-- These tables had no tenant isolation — data could leak
-- ============================================================

-- purchase_line_items: add company_id, backfill from parent, add RLS
ALTER TABLE purchase_line_items ADD COLUMN IF NOT EXISTS company_id UUID;
UPDATE purchase_line_items SET company_id = pf.company_id
  FROM purchase_facts pf WHERE purchase_line_items.purchase_id = pf.id
  AND purchase_line_items.company_id IS NULL;
ALTER TABLE purchase_line_items ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE purchase_line_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_purchase_line_items ON purchase_line_items
  FOR ALL USING (company_id = current_setting('app.current_tenant', true)::uuid);

-- supply_request_items: add company_id, backfill, RLS
ALTER TABLE supply_request_items ADD COLUMN IF NOT EXISTS company_id UUID;
UPDATE supply_request_items SET company_id = sr.company_id
  FROM supply_requests sr WHERE supply_request_items.request_id = sr.id
  AND supply_request_items.company_id IS NULL;
ALTER TABLE supply_request_items ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE supply_request_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_supply_request_items ON supply_request_items
  FOR ALL USING (company_id = current_setting('app.current_tenant', true)::uuid);

-- transfer_request_items: add company_id, backfill, RLS
ALTER TABLE transfer_request_items ADD COLUMN IF NOT EXISTS company_id UUID;
UPDATE transfer_request_items SET company_id = tr.company_id
  FROM transfer_requests tr WHERE transfer_request_items.request_id = tr.id
  AND transfer_request_items.company_id IS NULL;
ALTER TABLE transfer_request_items ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE transfer_request_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_transfer_request_items ON transfer_request_items
  FOR ALL USING (company_id = current_setting('app.current_tenant', true)::uuid);

-- ============================================================
-- 1.4 Composite indexes for multi-location queries at scale
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_revenue_company_location_date
  ON revenue_facts(company_id, location_id, order_date);

CREATE INDEX IF NOT EXISTS idx_product_sales_company_location_date
  ON product_sales_facts(company_id, location_id, sale_date);

CREATE INDEX IF NOT EXISTS idx_stock_company_location_time
  ON stock_snapshots(company_id, location_id, snapshot_at);

CREATE INDEX IF NOT EXISTS idx_recipe_company_dish
  ON recipe_components(company_id, dish_iiko_id);

-- Index for sync GetCompaniesToSync query
CREATE INDEX IF NOT EXISTS idx_companies_iiko_configured
  ON companies(id) WHERE iiko_server_url IS NOT NULL AND iiko_server_url != '';

-- NOTE: FORCE ROW LEVEL SECURITY intentionally NOT enabled yet.
-- The foodbi DB user is the table owner; PostgreSQL skips RLS for owners.
-- FORCE will be enabled in a future migration after all handlers use
-- the TenantDB wrapper (SET LOCAL app.current_tenant per transaction).
-- For now, RLS policies exist as defense-in-depth for non-owner roles.

-- Revert data isolation changes

-- Remove indexes
DROP INDEX IF EXISTS idx_revenue_company_location_date;
DROP INDEX IF EXISTS idx_product_sales_company_location_date;
DROP INDEX IF EXISTS idx_stock_company_location_time;
DROP INDEX IF EXISTS idx_recipe_company_dish;
DROP INDEX IF EXISTS idx_companies_iiko_configured;

-- Remove RLS from item tables
DROP POLICY IF EXISTS tenant_isolation_purchase_line_items ON purchase_line_items;
DROP POLICY IF EXISTS tenant_isolation_supply_request_items ON supply_request_items;
DROP POLICY IF EXISTS tenant_isolation_transfer_request_items ON transfer_request_items;
ALTER TABLE purchase_line_items DISABLE ROW LEVEL SECURITY;
ALTER TABLE supply_request_items DISABLE ROW LEVEL SECURITY;
ALTER TABLE transfer_request_items DISABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_line_items DROP COLUMN IF EXISTS company_id;
ALTER TABLE supply_request_items DROP COLUMN IF EXISTS company_id;
ALTER TABLE transfer_request_items DROP COLUMN IF EXISTS company_id;

-- Revert unique constraints
ALTER TABLE revenue_facts DROP CONSTRAINT IF EXISTS revenue_facts_unique_order;
ALTER TABLE revenue_facts ADD CONSTRAINT revenue_facts_company_id_iiko_order_id_key UNIQUE(company_id, iiko_order_id);
ALTER TABLE purchase_facts DROP CONSTRAINT IF EXISTS purchase_facts_unique_invoice;
ALTER TABLE purchase_facts ADD CONSTRAINT purchase_facts_company_id_iiko_invoice_id_key UNIQUE(company_id, iiko_invoice_id);

-- Add order_id column and unique constraint to product_sales_facts
-- Required for ON CONFLICT upsert in sync/service.go SyncProductSales

ALTER TABLE product_sales_facts ADD COLUMN IF NOT EXISTS order_id VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_sales_facts_upsert
    ON product_sales_facts (company_id, iiko_product_id, sale_date, order_id);

DROP INDEX IF EXISTS idx_product_sales_facts_upsert;
ALTER TABLE product_sales_facts DROP COLUMN IF EXISTS order_id;

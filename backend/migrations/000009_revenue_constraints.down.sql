ALTER TABLE revenue_facts DROP CONSTRAINT IF EXISTS chk_revenue_positive;
ALTER TABLE revenue_facts DROP CONSTRAINT IF EXISTS chk_revenue_sane;
ALTER TABLE revenue_facts DROP CONSTRAINT IF EXISTS chk_discount_positive;
ALTER TABLE product_sales_facts DROP CONSTRAINT IF EXISTS chk_psf_revenue_positive;
ALTER TABLE purchase_facts DROP CONSTRAINT IF EXISTS chk_pf_totalsum_positive;

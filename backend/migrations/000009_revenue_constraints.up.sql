-- Sanity constraints on monetary values to prevent bad data at the DB level

ALTER TABLE revenue_facts ADD CONSTRAINT chk_revenue_positive CHECK (revenue >= 0);
ALTER TABLE revenue_facts ADD CONSTRAINT chk_revenue_sane CHECK (revenue < 10000000);
ALTER TABLE revenue_facts ADD CONSTRAINT chk_discount_positive CHECK (discount >= 0);

ALTER TABLE product_sales_facts ADD CONSTRAINT chk_psf_revenue_positive CHECK (revenue >= 0);
ALTER TABLE purchase_facts ADD CONSTRAINT chk_pf_totalsum_positive CHECK (total_sum >= 0);

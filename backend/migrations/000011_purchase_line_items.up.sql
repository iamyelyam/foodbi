-- Line items for purchase invoices (scanned from iiko XML).

CREATE TABLE purchase_line_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    purchase_id UUID NOT NULL REFERENCES purchase_facts(id) ON DELETE CASCADE,
    product_code VARCHAR(100),
    product_name VARCHAR(500) NOT NULL,
    unit VARCHAR(50),
    quantity NUMERIC(12, 3) NOT NULL DEFAULT 0,
    price NUMERIC(12, 2) NOT NULL DEFAULT 0,
    subtotal NUMERIC(12, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_purchase_line_items_purchase ON purchase_line_items(purchase_id);

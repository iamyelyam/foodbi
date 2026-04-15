-- Recipe components: which ingredients (stock items) make up which dishes.
-- Sourced from iiko Server API endpoint /resto/api/v2/assemblyCharts/getPrepared
-- (technological card / технологическая карта). One row per (dish, ingredient) pair.
-- amount = quantity of ingredient required for 1 unit of the dish.

CREATE TABLE recipe_components (
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    dish_iiko_id VARCHAR(255) NOT NULL,
    dish_name VARCHAR(500) NOT NULL,
    ingredient_iiko_id VARCHAR(255) NOT NULL,
    ingredient_name VARCHAR(500) NOT NULL,
    amount NUMERIC(18, 9) NOT NULL,
    unit VARCHAR(50),
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (company_id, dish_iiko_id, ingredient_iiko_id)
);

-- Reverse lookup: "which dishes use this ingredient?" — main API query path.
CREATE INDEX idx_recipe_components_ingredient
    ON recipe_components(company_id, ingredient_iiko_id);

ALTER TABLE recipe_components ENABLE ROW LEVEL SECURITY;
ALTER TABLE recipe_components FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_recipe_components ON recipe_components
    USING (company_id = current_setting('app.current_tenant', true)::uuid);

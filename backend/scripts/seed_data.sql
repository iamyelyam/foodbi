-- FoodBI Synthetic Data Seed
-- Covers ~6 months (Oct 2025 - Apr 2026)
-- Tagged with is_seed=true comment for easy cleanup
-- Run: psql -U foodbi -d foodbi -f scripts/seed_data.sql
-- Cleanup: psql -U foodbi -d foodbi -f scripts/cleanup_seed.sql

BEGIN;

-- Company and user IDs (from existing test account)
DO $$
DECLARE
  v_company_id UUID := '91c69e0a-d0d1-46d6-a558-9a370e5bfce4';
  v_owner_id UUID := '3321191a-731d-4365-9356-32a65fd6cd67';
  v_loc1 UUID;
  v_loc2 UUID;
  v_loc3 UUID;
  v_emp1 UUID;
  v_emp2 UUID;
  v_emp3 UUID;
  v_day DATE;
  v_hour INT;
  v_revenue NUMERIC;
  v_discount NUMERIC;
  v_items INT;
  v_waiter TEXT;
  v_order_type TEXT;
  v_product TEXT;
  v_category TEXT;
  v_qty NUMERIC;
  v_cost NUMERIC;
  v_supplier TEXT;
  v_sup_id TEXT;
  v_amount NUMERIC;
  v_i INT;
  v_loc UUID;
  v_waiters TEXT[] := ARRAY['Anna K.', 'Dmitry S.', 'Maria P.', 'Alex V.', 'Elena T.', 'Ivan M.'];
  v_order_types TEXT[] := ARRAY['dine-in', 'dine-in', 'dine-in', 'takeaway', 'delivery'];
  v_products TEXT[] := ARRAY['Margherita Pizza', 'Caesar Salad', 'Pasta Carbonara', 'Tom Yum Soup', 'Grilled Salmon', 'Beef Burger', 'Tiramisu', 'Espresso', 'Latte', 'Fresh Juice', 'Club Sandwich', 'Greek Salad', 'Risotto', 'Chicken Wings', 'Cheesecake', 'Americano', 'Cappuccino', 'Mojito', 'Beer Draft', 'Wine Glass'];
  v_categories TEXT[] := ARRAY['Pizza', 'Salads', 'Pasta', 'Soups', 'Seafood', 'Burgers', 'Desserts', 'Coffee', 'Coffee', 'Beverages', 'Sandwiches', 'Salads', 'Pasta', 'Appetizers', 'Desserts', 'Coffee', 'Coffee', 'Cocktails', 'Beer', 'Wine'];
  v_suppliers TEXT[] := ARRAY['Fresh Farm LLC', 'Coca Cola Bottlers', 'Metro Cash&Carry', 'Seafood Direct', 'Baker Street Supplies', 'Wine & Spirits Co'];
  v_sup_ids TEXT[] := ARRAY['sup-001', 'sup-002', 'sup-003', 'sup-004', 'sup-005', 'sup-006'];
  v_stock_products TEXT[] := ARRAY['Flour', 'Olive Oil', 'Mozzarella', 'Tomatoes', 'Chicken Breast', 'Salmon Fillet', 'Beef Patty', 'Lettuce', 'Eggs', 'Milk', 'Coffee Beans', 'Sugar', 'Pasta Dry', 'Rice', 'Butter', 'Cream', 'Onions', 'Garlic', 'Potatoes', 'Bread'];
  v_units TEXT[] := ARRAY['kg', 'L', 'kg', 'kg', 'kg', 'kg', 'kg', 'kg', 'pcs', 'L', 'kg', 'kg', 'kg', 'kg', 'kg', 'L', 'kg', 'kg', 'kg', 'pcs'];
BEGIN

  -- Create 3 locations
  v_loc1 := gen_random_uuid();
  v_loc2 := gen_random_uuid();
  v_loc3 := gen_random_uuid();

  INSERT INTO locations (id, company_id, name, address, iiko_org_id, created_at, updated_at) VALUES
    (v_loc1, v_company_id, 'Downtown Branch', '123 Main Street, City Center', 'iiko-org-001', NOW(), NOW()),
    (v_loc2, v_company_id, 'Mall Location', '456 Shopping Ave, Mall Floor 2', 'iiko-org-002', NOW(), NOW()),
    (v_loc3, v_company_id, 'Airport Terminal', '789 Airport Rd, Terminal B', 'iiko-org-003', NOW(), NOW());

  -- Create 3 employees
  v_emp1 := gen_random_uuid();
  v_emp2 := gen_random_uuid();
  v_emp3 := gen_random_uuid();

  INSERT INTO users (id, company_id, email, password_hash, first_name, last_name, phone, role, is_active, created_at, updated_at) VALUES
    (v_emp1, v_company_id, 'anna@foodbi.test', '$2a$12$seed00000000000000000uAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA', 'Anna', 'Kozlova', '+7-900-111-2233', 'employee', true, NOW(), NOW()),
    (v_emp2, v_company_id, 'dmitry@foodbi.test', '$2a$12$seed00000000000000000uAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA', 'Dmitry', 'Smirnov', '+7-900-444-5566', 'employee', true, NOW(), NOW()),
    (v_emp3, v_company_id, 'maria@foodbi.test', '$2a$12$seed00000000000000000uAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA', 'Maria', 'Petrova', '+7-900-777-8899', 'owner', true, NOW(), NOW());

  INSERT INTO user_locations (user_id, location_id) VALUES
    (v_emp1, v_loc1), (v_emp2, v_loc2), (v_emp3, v_loc1), (v_emp3, v_loc2), (v_emp3, v_loc3);

  -- Generate revenue_facts: ~180 days, 15-40 orders per day per location
  FOR v_day IN SELECT generate_series('2025-10-01'::date, '2026-04-07'::date, '1 day')
  LOOP
    FOREACH v_loc IN ARRAY ARRAY[v_loc1, v_loc2, v_loc3]
    LOOP
      -- More orders on weekends
      v_i := CASE WHEN EXTRACT(DOW FROM v_day) IN (0, 6) THEN 25 + (random() * 20)::int ELSE 15 + (random() * 15)::int END;

      FOR v_hour IN 1..v_i
      LOOP
        v_revenue := 15 + (random() * 150)::numeric(10,2);
        v_discount := CASE WHEN random() < 0.15 THEN (random() * v_revenue * 0.2)::numeric(10,2) ELSE 0 END;
        v_items := 1 + (random() * 6)::int;
        v_waiter := v_waiters[1 + (random() * 5)::int];
        v_order_type := v_order_types[1 + (random() * 4)::int];

        INSERT INTO revenue_facts (company_id, location_id, iiko_order_id, order_date, revenue, discount, order_type, status, item_count, waiter_name, synced_at)
        VALUES (v_company_id, v_loc, 'seed-' || gen_random_uuid()::text,
                v_day + (interval '10 hours' + (random() * interval '12 hours')),
                v_revenue, v_discount, v_order_type, 'closed', v_items, v_waiter, NOW());
      END LOOP;
    END LOOP;
  END LOOP;

  -- Generate product_sales_facts
  FOR v_day IN SELECT generate_series('2025-10-01'::date, '2026-04-07'::date, '1 day')
  LOOP
    FOREACH v_loc IN ARRAY ARRAY[v_loc1, v_loc2, v_loc3]
    LOOP
      FOR v_i IN 1..array_length(v_products, 1)
      LOOP
        v_qty := (2 + random() * 20)::numeric(10,1);
        v_revenue := v_qty * (8 + random() * 25)::numeric(10,2);
        v_cost := v_revenue * (0.25 + random() * 0.20)::numeric(10,2);

        INSERT INTO product_sales_facts (company_id, location_id, iiko_product_id, product_name, category, sale_date, quantity, revenue, cost_price, synced_at)
        VALUES (v_company_id, v_loc, 'prod-' || v_i, v_products[v_i], v_categories[v_i], v_day, v_qty, v_revenue, v_cost, NOW());
      END LOOP;
    END LOOP;
  END LOOP;

  -- Generate purchase_facts: 2-5 invoices per week per location
  FOR v_day IN SELECT generate_series('2025-10-01'::date, '2026-04-07'::date, '3 days')
  LOOP
    FOREACH v_loc IN ARRAY ARRAY[v_loc1, v_loc2, v_loc3]
    LOOP
      v_i := 1 + (random() * 5)::int;
      v_supplier := v_suppliers[v_i];
      v_sup_id := v_sup_ids[v_i];
      v_revenue := 500 + (random() * 5000)::numeric(10,2);

      INSERT INTO purchase_facts (company_id, location_id, iiko_invoice_id, document_number, supplier_id, supplier_name, incoming_date, status, total_sum, synced_at)
      VALUES (v_company_id, v_loc, 'seed-inv-' || gen_random_uuid()::text,
              'INV-' || to_char(v_day, 'YYMMDD') || '-' || (100 + (random()*899)::int),
              v_sup_id, v_supplier, v_day + interval '10 hours', 'processed', v_revenue, NOW());
    END LOOP;
  END LOOP;

  -- Generate stock_snapshots (current state)
  FOREACH v_loc IN ARRAY ARRAY[v_loc1, v_loc2, v_loc3]
  LOOP
    FOR v_i IN 1..array_length(v_stock_products, 1)
    LOOP
      v_amount := (1 + random() * 50)::numeric(10,2);
      v_cost := v_amount * (2 + random() * 15)::numeric(10,2);

      INSERT INTO stock_snapshots (company_id, location_id, iiko_product_id, product_name, amount, unit, cost_sum, snapshot_at, synced_at)
      VALUES (v_company_id, v_loc, 'stock-' || v_i, v_stock_products[v_i], v_amount, v_units[v_i], v_cost, NOW(), NOW());
    END LOOP;
  END LOOP;

  -- Generate some supply_requests
  FOR v_i IN 1..15
  LOOP
    v_loc := CASE WHEN v_i % 3 = 0 THEN v_loc1 WHEN v_i % 3 = 1 THEN v_loc2 ELSE v_loc3 END;
    v_supplier := v_suppliers[1 + (random() * 5)::int];
    v_revenue := 200 + (random() * 3000)::numeric(10,2);

    INSERT INTO supply_requests (company_id, location_id, supplier_name, status, total_sum, created_by, created_at)
    VALUES (v_company_id, v_loc, v_supplier,
            (ARRAY['pending', 'approved', 'rejected'])[1 + (random() * 2)::int],
            v_revenue, v_owner_id, NOW() - (random() * interval '30 days'));
  END LOOP;

  -- Generate some transfer_requests
  FOR v_i IN 1..10
  LOOP
    INSERT INTO transfer_requests (company_id, from_location_id, to_location_id, status, created_by, created_at)
    VALUES (v_company_id,
            (ARRAY[v_loc1, v_loc2, v_loc3])[1 + (random() * 2)::int],
            (ARRAY[v_loc1, v_loc2, v_loc3])[1 + (random() * 2)::int],
            (ARRAY['pending', 'completed', 'cancelled'])[1 + (random() * 2)::int],
            v_owner_id, NOW() - (random() * interval '30 days'));
  END LOOP;

  -- Generate notifications
  INSERT INTO notifications (company_id, user_id, type, title, message, is_read, created_at) VALUES
    (v_company_id, v_owner_id, 'low_stock', 'Low stock: Salmon Fillet', 'Salmon Fillet at Downtown Branch is below threshold (2.3 kg remaining)', false, NOW() - interval '1 hour'),
    (v_company_id, v_owner_id, 'low_stock', 'Low stock: Coffee Beans', 'Coffee Beans at Mall Location is below threshold (1.5 kg remaining)', false, NOW() - interval '3 hours'),
    (v_company_id, v_owner_id, 'supply_approved', 'Supply request approved', 'Your supply request to Fresh Farm LLC ($2,450) has been approved', true, NOW() - interval '1 day'),
    (v_company_id, v_owner_id, 'sync_failed', 'Sync failed: Airport Terminal', 'iiko sync failed for Airport Terminal. Will retry in 15 minutes.', false, NOW() - interval '5 hours'),
    (v_company_id, NULL, 'supply_approved', 'New supply delivery', 'Metro Cash&Carry delivery confirmed for Downtown Branch', true, NOW() - interval '2 days'),
    (v_company_id, v_owner_id, 'low_stock', 'Low stock: Butter', 'Butter at Airport Terminal is below threshold (0.8 kg remaining)', false, NOW() - interval '30 minutes'),
    (v_company_id, NULL, 'supply_rejected', 'Supply request rejected', 'Supply request to Wine & Spirits Co was rejected by manager', true, NOW() - interval '3 days');

  -- Sync log entries
  FOREACH v_loc IN ARRAY ARRAY[v_loc1, v_loc2, v_loc3]
  LOOP
    INSERT INTO iiko_sync_log (company_id, location_id, sync_type, status, records_synced, started_at, completed_at, duration_ms) VALUES
      (v_company_id, v_loc, 'revenue', 'success', 25 + (random()*20)::int, NOW() - interval '15 minutes', NOW() - interval '14 minutes', 3500 + (random()*2000)::int),
      (v_company_id, v_loc, 'purchases', 'success', 3 + (random()*5)::int, NOW() - interval '60 minutes', NOW() - interval '59 minutes', 2000 + (random()*1500)::int),
      (v_company_id, v_loc, 'stock', 'success', 20, NOW() - interval '30 minutes', NOW() - interval '29 minutes', 1500 + (random()*1000)::int);
  END LOOP;

  -- Refresh materialized view
  REFRESH MATERIALIZED VIEW dashboard_daily_revenue;

  RAISE NOTICE 'Seed data created successfully!';
  RAISE NOTICE 'Locations: %, %, %', v_loc1, v_loc2, v_loc3;
END $$;

COMMIT;

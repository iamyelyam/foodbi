-- Cleanup all synthetic seed data
-- Removes everything except the original owner account and company
-- Run: psql -U foodbi -d foodbi -f scripts/cleanup_seed.sql

BEGIN;

-- Delete seeded data (identified by seed- prefixes and test emails)
DELETE FROM notifications WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4';
DELETE FROM iiko_sync_log WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4';
DELETE FROM supply_request_items WHERE request_id IN (SELECT id FROM supply_requests WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4');
DELETE FROM supply_requests WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4';
DELETE FROM transfer_request_items WHERE request_id IN (SELECT id FROM transfer_requests WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4');
DELETE FROM transfer_requests WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4';
DELETE FROM stock_snapshots WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4';
DELETE FROM product_sales_facts WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4';
DELETE FROM purchase_facts WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4';
DELETE FROM revenue_facts WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4';
DELETE FROM user_locations WHERE user_id IN (SELECT id FROM users WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4' AND email != 'owner@foodbi.test');
DELETE FROM users WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4' AND email != 'owner@foodbi.test';
DELETE FROM locations WHERE company_id = '91c69e0a-d0d1-46d6-a558-9a370e5bfce4';

-- Refresh materialized view
REFRESH MATERIALIZED VIEW dashboard_daily_revenue;

COMMIT;

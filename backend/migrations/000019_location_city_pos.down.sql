ALTER TABLE locations
    DROP COLUMN IF EXISTS city,
    DROP COLUMN IF EXISTS pos_system;

-- City + POS System for locations. POS System tracks which back-office system a
-- restaurant uses (iiko, r_keeper, Poster, manual). Drives downstream sync logic.
ALTER TABLE locations
    ADD COLUMN IF NOT EXISTS city VARCHAR(255),
    ADD COLUMN IF NOT EXISTS pos_system VARCHAR(50);

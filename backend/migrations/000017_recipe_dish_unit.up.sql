-- Add dish_unit so the UI can show "0.24 л / порц." vs "1.84 л / кг".
-- Dishes sold by weight (PREPARED-type half-products) measure recipes per 1 kg;
-- portioned dishes measure per 1 порция. Stored at sync time.
ALTER TABLE recipe_components
    ADD COLUMN IF NOT EXISTS dish_unit VARCHAR(50);

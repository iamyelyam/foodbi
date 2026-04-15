-- Reverting to the original 2-value enum loses information for any rows that
-- used the new operational roles, so this down-migration normalizes them to
-- 'employee' before reinstating the old check.
UPDATE users SET role = 'employee' WHERE role NOT IN ('owner', 'employee');
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('owner', 'employee'));

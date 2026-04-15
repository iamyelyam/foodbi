-- Extend the user role enum from {owner, employee} to include operational
-- restaurant roles. Authorization in handlers still keys off "owner" only —
-- everything else (general_manager, manager, bartender, waiter, cashier,
-- accountant) is treated as non-owner today; finer permissions can be added later.
-- 'employee' kept for legacy rows (pre-2026 invites used it as default).

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (
    role IN (
        'owner',
        'employee',
        'general_manager',
        'manager',
        'bartender',
        'waiter',
        'cashier',
        'accountant'
    )
);

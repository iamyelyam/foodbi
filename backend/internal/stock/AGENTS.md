# stock — inventory list, overrides, aliases, recipe reverse-lookup

Serves the /api/v1/stock endpoints. Handles iiko-reported stock numbers plus the three user-editable overlays: display name aliases, manual amount/price overrides, and recipe component reverse-lookup.

## Files

| File | What |
|---|---|
| `handler.go` | Chi routes: `GET /`, `GET /low-stock`, `PUT /products/{id}/alias`, `PUT /products/{id}/override`, `GET /products/{id}/used-in`. `stockSelect` constant is the joined SQL used by both list queries. |

## Tables

| Table | Role |
|---|---|
| `stock_snapshots` | iiko-synced stock levels (time series). Read with `DISTINCT ON (iiko_product_id)` + ORDER BY snapshot_at DESC to get latest. |
| `product_aliases` | Per-tenant display-name override. `pa.display_name` wins over `s.product_name` via `COALESCE(NULLIF(...))`. |
| `stock_overrides` | Per-tenant manual values. `manual_amount` and `manual_price_per_unit` are each nullable. When either is set, it wins via `COALESCE(o.manual_*, s.*)`. RLS-enforced (company_id + current_tenant). |
| `recipe_components` | Reverse-lookup source for `/used-in`: filter by `ingredient_iiko_id`. |

## Override semantics — important

- Sending `{manual_amount: 1500}` only updates amount, leaves price untouched. Partial upsert via `ON CONFLICT DO UPDATE SET manual_* = COALESCE(EXCLUDED.manual_*, stock_overrides.manual_*)`.
- Sending `{manual_amount: null, manual_price_per_unit: null}` → DELETE the override row (revert to iiko).
- Both nil in the request body → treated as delete (no-op if row doesn't exist).
- Effective `cost_sum` = `override.manual_amount * override.manual_price_per_unit` when override is set, else iiko's `cost_sum`.
- Effective `price_per_unit` exposed to frontend = `COALESCE(override.manual_price_per_unit, s.cost_sum / NULLIF(s.amount, 0), 0)`.

## Gotchas when editing

1. **DISTINCT ON order matters.** The `stockSelect` query does `DISTINCT ON (s.iiko_product_id) ... ORDER BY s.iiko_product_id, s.snapshot_at DESC`. The ORDER BY prefix MUST start with the DISTINCT ON column.
2. **RLS is enforced on stock_overrides.** The middleware sets `app.current_tenant` — if you bypass middleware (e.g. webhook), the query will return 0 rows.
3. **Negative amount is a real signal.** iiko allows selling more than stock (data entry error). Frontend shows these with red + "Сделать инвентаризацию" warning. Don't filter them out in the handler.
4. **Used-in response includes `dish_unit`.** The frontend renders "0.24 л / порц." vs "1.84 л / кг" based on `dish_unit` from `recipe_components` — populated during `sync/service.go::SyncRecipes` with fallback (`порц.` for DISH, `кг` for PREPARED).
5. **Recipe sync runs every 6h** (not 15 min like revenue). Don't expect fresh-fresh data in `/used-in` if a new dish was just added to iiko.

## When editing

- Adding a new stock-level feature? Route goes in `Routes()`, handler below. Always use `middleware.GetCompanyID(r.Context())` for tenant isolation.
- Modifying `stockSelect`? Run both `/stock` and `/stock/low-stock` endpoints locally — they share the same SELECT + WHERE snippets.
- Changing override shape? Migration goes in `backend/migrations/`, update `OverrideInput` struct + SQL in `SetOverride`.
- Changing list shape? Update `StockItem` struct, both Scan calls (2 in this file), and mirror in `frontend/src/pages/stock/StockPage.tsx` TypeScript interface.

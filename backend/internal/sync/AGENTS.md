# sync — iiko pulls and ticker orchestration

Pulls data from iiko Server API and mirrors it into PostgreSQL. Runs as a separate binary (`backend/cmd/sync/main.go`) on interval tickers plus one immediate sync at boot.

## Files

| File | What |
|---|---|
| `service.go` | All `Sync{Revenue, ProductSales, Purchases, Stock, Recipes}` methods; plus `OrderAgg`, `ResolveUnit`, `NormalizeOrderType`, `AggregateOrdersFromOLAP`, `ValidateRevenueAfterSync`. |
| `../../cmd/sync/main.go` | Ticker config: revenue 15 min, product_sales 15 min, purchases 60 min, stock 30 min, recipes 6 h. Loops over companies from `companies` table. |

## Tables owned / written

| Table | Written by |
|---|---|
| `revenue_facts` | SyncRevenue |
| `product_sales_facts` | SyncProductSales |
| `purchase_facts`, `purchase_line_items` | SyncPurchases |
| `stock_snapshots` | SyncStock (time-series, one row per sync) |
| `recipe_components` | SyncRecipes |
| `iiko_sync_log` | every sync (start/complete/fail records) |

## Gotchas — read before touching

1. **Never divide DishSumInt by 100.** iiko returns KZT amounts as integers already in currency units. Multiplying or dividing destroys data. See `CLAUDE.md` iiko rule 1.
2. **OLAP dish-level aggregation.** Grouping only by `UniqOrderId.Id` returns a random per-order dish value, not the sum. Always group by `UniqOrderId.Id + DishName` and SUM `DishSumInt` in Go (`AggregateOrdersFromOLAP`). See `CLAUDE.md` rule 2.
3. **Date range pattern.** `>= start AND < (end::date + 1)` — never `<= end`. Applies to every SQL filter here.
4. **Kill stale `go run` binaries before rebuilding.** Stale binaries in `/var/folders` can keep running for days and overwrite data. `pkill -9 -f foodbi` before every `go build`.
5. **Unit resolution.** `ResolveUnit(rawUnit, productName, measureUnits)` uses this priority: iiko measureUnit map → string match on GUID length → name heuristic (`КГ`/`Л`/`ГР`) → known GUID fallback → `шт`. Do not skip steps.
6. **Recipe sync filters to DISH + PREPARED only.** These are the only nomenclature types with assembly charts. `dish_unit` defaults to `порц.` for DISH, `кг` for PREPARED when iiko returns a GUID or empty.
7. **`SyncRevenue` post-check.** `ValidateRevenueAfterSync` asserts `MAX(revenue) > 10_000` per day — if a sync produces garbage, this catches it immediately.

## When editing

- Adding a new sync type? Mirror the `startSyncLog` → work → `completeSyncLog` / `failSyncLog` pattern. Add a ticker entry in `cmd/sync/main.go`.
- Adding a new OLAP field to an existing sync? Read the corresponding iiko handler (`../iiko/api.go::GetOLAPReport`), add field to `GroupByRowFields`, extract via `GetString` / `GetFloat`, upsert.
- Changing aggregation shape? Write a unit test in `service_test.go` — there's already `TestAggregateOrdersFromOLAP_*` covering the key invariants.

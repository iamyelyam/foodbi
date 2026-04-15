# revenue — orders and products sold

Read-only over `revenue_facts` and `product_sales_facts` (populated by `../sync/`). Handles the /api/v1/revenue/* endpoints plus the product detail + trend + orders sub-pages.

## Files

| File | What |
|---|---|
| `handler.go` | Routes: `GET /orders`, `GET /orders/{id}`, `POST /orders/{id}/status`, `GET /products`, `GET /products/{id}`, `GET /products/{id}/trend`, `GET /products/{id}/orders`. |

## Tables

| Table | Role |
|---|---|
| `revenue_facts` | One row per order. Populated by `sync.SyncRevenue` after aggregating per-dish OLAP rows per order. |
| `product_sales_facts` | One row per (product, order). Populated by `sync.SyncProductSales`. |

## Request patterns

- `GET /orders?date_from=&date_to=&order_type=&waiter=&sort=` — multi-filter. Date range uses `>= start AND < (end::date + 1)`.
- `GET /products` returns aggregated totals per product over the date window.
- `GET /products/{id}/trend?days=30` returns daily buckets (for `RevenueChart`).
- `POST /orders/{id}/status` is owner-only — used by the "Approve / Reject" BottomSheet on OrderDetailPage.

## Gotchas when editing

1. **Margin math.** `(SUM(revenue) - SUM(cost_price)) / SUM(revenue) * 100`. If `cost_price` is 0 (missing in iiko), margin shoots to 100% — frontend displays with a warning. The AI `suspiciousMargin` suggestion also catches this.
2. **Orders endpoint is NOT paginated by default.** Unlike `/purchases`, it returns everything in the date window. If list grows too big, add `?page=N` param like purchases does.
3. **Date boundaries.** Orders at 23:59:59 of end_date must be included — use the `< (end::date + 1)` pattern, never `<= end`.
4. **product_sales_facts has no stable `iiko_product_id`.** It's a `sha256(product_name)[:8]` hash — see `sync/service.go::SyncProductSales`. Don't use it for joins with iiko nomenclature; only use it as a unique key per product within revenue data.

## When editing

- Adding a new metric? It probably needs a SQL aggregate over `revenue_facts` + `product_sales_facts`. Add a new endpoint, mirror the existing filter signature.
- Adding a new filter? Add a query param, extend the WHERE clause, mirror in `frontend/src/hooks/useApi.ts::useOrders` or similar.
- Changing the order-detail shape? Update `OrderDetail` struct here AND `Order` type in `frontend/src/pages/revenue/OrderDetailPage.tsx`.

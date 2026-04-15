# revenue pages — orders list, order detail, product detail

## Files

| File | What |
|---|---|
| `RevenuePage.tsx` | Main page: 4 metric cards (Revenue/Orders/AOV/MI/T) + Orders vs Products segmented tab + Filters BottomSheet (embeds `<DateRangeBlock>` + price sort + waiter multi-select + order type toggle) + metric detail BottomSheet (trend chart). |
| `OrderDetailPage.tsx` | Order detail with line items + status change sheet (owner-only Approve/Reject). |
| `ProductDetailPage.tsx` | Product detail with trend chart + daily metrics (avg/best/worst) + recent orders list. |

## Data sources

- `useOrders({date_from, date_to, order_type, waiter, ...})` → `{orders, total, ...}` (paginated-ish, but not page-param at the moment).
- `useProducts({date_from, date_to, ...})` → per-product aggregates with margin %.
- `/revenue/orders/{id}` for OrderDetail.
- `/revenue/products/{id}/{,trend,orders}` for ProductDetail.
- Metric trend: `/revenue/trend` or similar — invoked from the metric-detail sheet.

## Shared components

- `<DateRangeBlock>` (from `@/components/ui/date-range-block`) embedded inside the Filters sheet.
- `<RevenueChart>` (from `@/components/charts/RevenueChart`) for trends.
- `<PeriodPills>` for toggling `30 / 60 / 90` day windows on product detail.

## Gotchas when editing

1. **Money formatting** uses `toLocaleString('ru-KZ', {maximumFractionDigits: 0})` — KZT locale stays ru-KZ regardless of UI language. Don't locale-switch the number format.
2. **"Order #" and "Waiter:" are i18n'd with params.** Use `t('revenue.orderNumber', {number: ...})` and `t('revenue.waiterLine', {name: ...})`. Don't build strings with template literals.
3. **Order type translation uses a switch**: `revenue.orderType.dineIn / delivery / takeaway`. The raw value from iiko (`'dine-in' | 'delivery' | 'takeaway'`) is NOT a valid i18n key — map before passing.
4. **"pc." suffix** → `t('common.piecesShort')`. Same value (`'pc.'` in en, `'шт'` in ru) is reused across Stock + Revenue + Product detail.
5. **Filter waiter list** is derived from `rawOrders` (unique waiter_name values) — not from a dedicated `/waiters` endpoint. If a waiter never appeared in the date range, they won't show up. This is intentional.
6. **Metric detail sheet** uses a single `selectedMetric: 'revenue' | 'orders' | 'aov' | 'mit'` state and a labels object — if you add a new metric to the top grid, add it here too.

## When editing

- Adding a new metric card? Add entry in the 4-card grid (will auto-lay to 5 columns — consider narrower format). Extend `labels` and `chartData` blocks in the metric-detail sheet.
- Adding a new filter? Extend the Filters BottomSheet with a new section, add state + pass to `useOrders({...})`. Mirror the filter in `backend/internal/revenue/handler.go::GetOrders`.
- Modifying how orders render in the list? The `<button>` wrapping each row uses `formatOrderDateTime` + `formatPersonName` — already locale-aware.

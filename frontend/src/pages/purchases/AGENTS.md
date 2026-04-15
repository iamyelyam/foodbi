# purchases pages — invoices list, supplier detail

## Files

| File | What |
|---|---|
| `PurchasesPage.tsx` | Invoice list grouped by date + 2 metric cards (Purchases sum / Invoices count) + Filters sheet (supplier multi-select) + per-invoice detail BottomSheet with line items + supplier alias editor. |
| `SupplierDetailPage.tsx` | Supplier detail: total spend + invoices count + contact info + purchase history list. |

## Data sources

- `usePurchases({date_from, date_to})` → `{purchases, total, page, per_page, total_pages}`. Note: without `?page=N` param, backend returns ALL rows (no LIMIT/OFFSET) within the date window.
- `useSuppliers()` → supplier list for the filter.
- `/purchases/{id}` for invoice detail (includes line_items).
- `/purchases/suppliers/{id}` for supplier detail (with purchases history).
- `PUT /purchases/suppliers/{id}/alias` for the inline rename (supplier UUIDs → readable names).

## Shared components

- `<DateRangeSheet>` — full BottomSheet with presets + From/To picker. Reused unchanged from `@/components/layout/DateRangeSheet`.
- `<BottomSheet>`, `<Header>`, `<Tabbar>`.

## Gotchas when editing

1. **Supplier names are often UUIDs.** iiko Server API returns `supplier` as a UUID on this tenant. `formatSupplierName()` from `@/lib/format` replaces raw UUIDs with "Unknown supplier". Use it everywhere supplier names render.
2. **Alias editor uses inline-input pattern** (same as Stock page's product alias). `editingSupplier === supplierId` → show input, else show text + pencil. Save on Enter / close on Escape.
3. **Filter defaults to last 30 days.** `isoDaysAgo(30)` as `dateFrom`, `todayIso()` as `dateTo`. Purchases happen less often than orders so 30d is the useful default.
4. **Line items are from `purchase_line_items` table** which is REPLACED on every sync (60 min interval). Don't cache line-item lists client-side beyond the react-query default.
5. **"Show N results"** in the Filters sheet uses `t('common.showResults', {count: N})` — if you add a new filter dimension, N should reflect the filtered count, not raw `purchases.length`.

## When editing

- Adding a new filter? Extend Filters sheet, add state + pass to `usePurchases`. Mirror query param in `backend/internal/purchases/handler.go::ListPurchases` WHERE clause.
- Adding a column to the list? The row is `<button>` wrapping a 2-line (supplier / date + amount). Keep that — it's the mobile-first pattern.
- Want pagination back? Add a `<InfiniteScroll>` component or a "Load more" button that passes `?page=N` — but verify the backend still supports it (it does, opt-in via `?page=`).

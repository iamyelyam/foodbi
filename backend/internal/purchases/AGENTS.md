# purchases — iiko incoming invoices

Read + lightweight write (supplier alias) over `purchase_facts` and `purchase_line_items`. Populated by `sync.SyncPurchases`.

## Files

| File | What |
|---|---|
| `handler.go` | Routes: `GET /` (list with optional filters), `GET /{id}` (detail with line items), `GET /suppliers` (list suppliers + counts), `GET /suppliers/{id}` (supplier detail), `PUT /suppliers/{id}/alias` (rename UUID-ish names). |

## Tables

| Table | Role |
|---|---|
| `purchase_facts` | One row per iiko incoming invoice. |
| `purchase_line_items` | Items within an invoice (product_name, quantity, unit, price, subtotal). FK to `purchase_facts.id`. Replaced on every sync (DELETE+INSERT by parent id). |
| `supplier_aliases` | Per-tenant display-name override for suppliers whose iiko name is a UUID (common on this tenant — see iiko AGENTS.md). |

## Pagination — read before editing

`GET /api/v1/purchases?page=N` = paginated (20 per page), returns `{purchases, total, page, per_page, total_pages}`.

`GET /api/v1/purchases` without `?page` = ALL rows in the date window, no LIMIT/OFFSET. This is intentional: the frontend shows every invoice within the picked date range at once. The list page filters by `date_from` + `date_to`, so "all in range" is bounded.

Don't add pagination back unconditionally — it would break the frontend list behavior.

## Gotchas when editing

1. **Supplier names are often UUIDs.** iiko Server API on this tenant returns supplier as a UUID for old data, not a readable name. `formatSupplierName()` on the frontend replaces raw UUIDs with "Unknown supplier". The `supplier_aliases` table lets the user type a real name.
2. **Line items are REPLACED on every sync.** If you need a historical line-item table, don't use `purchase_line_items` — it's rebuilt from iiko every 60 min.
3. **XML, not JSON.** Purchase invoices come from `/resto/api/documents/export/incomingInvoice` as XML. `iiko/api.go::GetPurchaseInvoices` parses it.
4. **Date filter uses `incoming_date`**, not `created_at`. The incoming_date is the business date of the invoice, created_at is the sync time.

## When editing

- Adding a supplier feature? Look at `supplier_aliases` pattern — same structure as `product_aliases` and `stock_overrides` (company_id + iiko_id composite key, RLS on `app.current_tenant`).
- Need to export or CSV? The frontend already has `downloadExcel` in `StockPage.tsx` as a template (client-side CSV generation).
- Adding purchase forecast / trend? Probably belongs in `../dashboard/` or a new `../forecast/` module, not here — this handler stays a plain CRUD view.

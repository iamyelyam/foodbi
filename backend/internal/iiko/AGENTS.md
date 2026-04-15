# iiko — Server API client

Low-level HTTP client for the iiko Server REST API. Consumed by `../sync/` tickers. Handles auth, retry, token refresh, and the XML/JSON format quirks of different iiko endpoints.

## Files

| File | What |
|---|---|
| `client.go` | `NewClient()`, `Authenticate()`, `doGet()`, `doPost()` — SHA1 password, 15 min token TTL, 3 retries with backoff, auto-refresh on 401. |
| `api.go` | All endpoints: `GetOLAPReport`, `GetPurchaseInvoices` (XML), `GetStockBalance`, `GetNomenclature` (XML-first + JSON fallback), `GetMeasureUnits` (3-endpoint fallback), `GetSuppliers`, `GetAssemblyChart`. |
| `types.go` | Request/response structs: `OLAPReportRequest`, `PurchaseInvoice`, `StockItem`, `ProductInfo`, etc. |

## Known quirks (Palaushy tenant)

- `/resto/api/v2/entities/suppliers/list` → 404. Fallback to `/resto/api/employees` (XML, supplier roles).
- `/resto/api/v2/entities/measureUnits/list` → 404. Fallback map of 4 known GUID → name + name heuristic.
- `OrderServiceType` OLAP field is null for every order (iiko tenant didn't configure it), so filtering by order type won't work — defaults to `dine-in`.
- `OrderNum` comes as `float64`, not string (`UniqOrderId.Number` returns 400). See `GetFloat` + `fmt.Sprintf("%.0f", n)` in `sync/service.go`.
- `cost_price` = `ProductCostBase.ProductCost` — some dishes return 0 or inflated values (misconfigured cost). The AI module flags these.
- `/resto/api/v2/assemblyCharts/getPrepared?productId=&date=` works (the `date=` param is required — without it returns 400 NullPointerException).

## Write-back (iiko → app → iiko)

- `/resto/api/v2/entities/products/update` (JSON POST, ProductDto shape) — verified to exist, can update product price. Not wired yet.
- `/resto/api/documents/import/incomingInvoice` — exists but creates real invoice documents (pollutes purchase ledger). Use with caution.
- `/resto/api/documents/import/inventory` → 404. No clean write-back for inventory recounts on this iiko version.

See `../../cmd/probe-writeback/main.go` for the probe that discovered these results.

## Gotchas when editing

1. **Don't hardcode endpoints.** Try XML first, JSON second, fail gracefully. That's the pattern `GetNomenclature` follows.
2. **SHA1 password, not plaintext.** `NewClient` does `sha1.Sum([]byte(password))` — if you add a new iiko HTTP interaction outside this module, re-hash yourself.
3. **Token cache is per-Client struct.** 15 min TTL, `authMu` mutex. `doGet`/`doPost` auto-refresh on 401.
4. **iiko returns JSON objects with non-deterministic field presence.** Use `GetString(row, "field")` / `GetFloat(row, "field")` helpers from `sync/service.go` instead of typed structs on OLAP results.
5. **XML dates.** `incomingInvoice` dates come as `2006-01-02`, not RFC3339. Use `time.Parse("2006-01-02", ...)`.

## When editing

- Adding a new endpoint? Add method on `*Client`, use `doGet`/`doPost`. Add a test that would spike the endpoint via `backend/cmd/probe-*/main.go` if you're not sure of the shape.
- Changing the auth flow? `Authenticate()` is the only place — it's called lazily. Keep the mutex + expiry logic.
- Adding a new type? Put it in `types.go`, not `api.go`.

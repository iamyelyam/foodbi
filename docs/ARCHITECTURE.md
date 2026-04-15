<!-- generated-by: gsd-doc-writer -->
# FoodBI — Architecture

## System Overview

FoodBI is a multi-tenant restaurant analytics SaaS for Kazakhstan restaurant groups. It pulls operational data from iiko POS systems via a background sync worker, stores it in a PostgreSQL database with Row Level Security (RLS) tenant isolation, and serves it through a REST API consumed by a React web application and a Capacitor iOS shell. The primary output is actionable business intelligence: revenue trends, product-level profitability, purchase costs, stock levels, recipe ingredient usage, and AI-generated optimization suggestions.

```
┌──────────────────────────────────────────────────────────────────────┐
│                         iiko Server API                              │
│  OLAP /resto/api/v2/reports/olap  (revenue, product sales)           │
│  XML  /resto/api/documents/export/incomingInvoice  (purchases)       │
│  JSON /resto/api/v2/reports/balance/stores  (stock)                  │
│  XML  /resto/api/products  (nomenclature — primary)                  │
│  JSON /resto/api/v2/entities/products/list  (nomenclature — fallback)│
│  JSON /resto/api/v2/assemblyCharts/getPrepared  (recipes)            │
└──────────────────┬───────────────────────────────────────────────────┘
                   │ HTTP (per-company credentials stored in DB)
                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│               Sync Worker  (cmd/sync)                                │
│  Tickers: revenue/product_sales 15 min · purchases 60 min            │
│           stock 30 min · recipes 6 h                                 │
│  Upserts fact rows; refreshes materialized view after each cycle     │
└──────────────────┬───────────────────────────────────────────────────┘
                   │ pgx/v5
                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│               PostgreSQL  (RLS multi-tenant)                         │
│  SET LOCAL app.current_tenant = '<company_id>' per transaction       │
│  Migrations 000001–000020 managed by RunMigrations() on API startup  │
└──────────────────┬───────────────────────────────────────────────────┘
                   │ pgx/v5
                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│               API Server  (cmd/api)  — Chi router, port 8080         │
│  /api/v1/auth  /api/v1/dashboard  /api/v1/revenue  /api/v1/stock    │
│  /api/v1/purchases  /api/v1/ai  /api/v1/employees  /api/v1/...      │
│  JWT Bearer auth → JWTAuth middleware → TenantContext middleware     │
└──────────────────┬───────────────────────────────────────────────────┘
                   │ HTTPS (VITE_API_URL = https://foodbi-production.up.railway.app)
                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│               React Frontend  (Vite + TypeScript + TailwindCSS)      │
│  Zustand stores: useAppStore (location filter, company settings)     │
│                  useAuthStore (JWT tokens, user profile)             │
│  i18n: useT() hook — 4 locales (ru default / kk / en / es)          │
└──────────────────┬───────────────────────────────────────────────────┘
                   │ WKWebView
                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│               Capacitor iOS Shell                                    │
│  Bundle ID: kz.foodbi  (frontend/ios/App/App.xcodeproj)             │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Backend Layout

### Entry Points (`backend/cmd/`)

| Binary | Purpose |
|--------|---------|
| `cmd/api/` | HTTP API server. Registers all route handlers, applies migrations on startup, starts the optional Telegram bot if `TELEGRAM_BOT_TOKEN` is set. |
| `cmd/sync/` | Background sync worker. Runs independent goroutine tickers per sync type; authenticates to iiko per company and calls `internal/sync` service methods. |
| `cmd/probe-recipes/` | One-shot debug tool for probing iiko assembly chart data. |
| `cmd/probe-writeback/` | One-shot debug tool for testing iiko write-back endpoints. |

### Internal Packages (`backend/internal/`)

| Package | Purpose |
|---------|---------|
| `auth` | Registration, OTP email verification, login, JWT token issuance, refresh token rotation, password reset. Uses bcrypt (cost 12) for passwords. |
| `ai` | AI suggestion engine. Returns `Suggestion` structs with `title_key` / `description_key` + `params` rather than rendered strings — frontend resolves locale. Manages `ai_tasks`. |
| `dashboard` | Aggregated KPI queries against `revenue_facts` and `product_sales_facts`; reads the `dashboard_daily_revenue` materialized view. |
| `database` | `pgxpool.Pool` constructor + `RunMigrations()` (applied on API startup from `./migrations/`). |
| `employees` | CRUD for users within a tenant; invite flow via `invites` table. |
| `files` | File upload handling; tracks uploads in `uploaded_files`. |
| `iiko` | iiko Server API client: `Authenticate`, `GetOLAPReport`, `GetPurchaseInvoices` (XML), `GetStockBalance`, `GetNomenclature` (XML v1 first, JSON v2 fallback), `GetAssemblyChart`. |
| `locations` | CRUD for restaurant locations; reads `locations` table with city and pos_system fields. |
| `middleware` | `JWTAuth` (parses Bearer token, populates context), `TenantContext` (validates company_id present), `Logger` (zerolog structured logging), `SecurityHeaders`, `MaxBodySize`. |
| `models` | Shared Go structs and role constants (`RoleOwner`, `RoleEmployee`, `RoleGeneralManager`, etc.). |
| `notifications` | Write and read notifications (low_stock, supply_approved, sync_failed, etc.); supports per-user and company-wide. |
| `payments` | Payment webhook handler (HMAC signature verification per company); integrates with Telegram bot for failure alerts. |
| `profiles` | User profile read/update (name, phone, language preference). |
| `purchases` | API handlers for purchase facts and line items; supplier alias resolution. |
| `revenue` | Revenue facts query handlers; date range filtering with exclusive upper bound pattern. |
| `statistics` | Cross-metric aggregations (revenue vs cost, margin computation using `ProductCostBase.ProductCost` from iiko). |
| `stock` | Stock snapshot queries; merges `stock_overrides` on top of iiko-reported values; recipe component lookup. |
| `supplying` | Supply request (order from supplier) CRUD; `supply_requests` + `supply_request_items`. |
| `sync` | Core sync business logic: `SyncRevenue`, `SyncProductSales`, `SyncPurchases`, `SyncStock`, `SyncRecipes`, `ValidateRevenueAfterSync`, `RefreshDashboardViews`. |
| `telegram` | Telegram bot: subscribes company admins; routes payment failure notifications. |
| `transfers` | Inter-location stock transfer requests; `transfer_requests` + `transfer_request_items`. |

---

## Frontend Layout

### Directory Structure (`frontend/src/`)

```
src/
├── App.tsx                 — Router root; protected route wrapper
├── main.tsx                — Vite entry; mounts React app
├── index.css               — Tailwind base styles
├── pages/                  — One directory per feature area
│   ├── DashboardPage.tsx
│   ├── EmployeeHomePage.tsx
│   ├── auth/               — Login, register, OTP verify
│   ├── ai-suggestions/     — Suggestion list + detail
│   ├── dashboard/          — Dashboard sub-views
│   ├── employees/          — Employee list, add, detail
│   ├── file-upload/        — Invoice/document upload
│   ├── locations/          — Location list and add
│   ├── notifications/      — Notification centre
│   ├── profile/            — Profile edit, language picker
│   ├── purchases/          — Purchase invoice list
│   ├── revenue/            — Revenue charts and table
│   ├── statistics/         — Margin and cost statistics
│   ├── stock/              — Stock snapshot view
│   ├── supplying/          — Supply request creation
│   └── transfers/          — Inter-location transfer requests
├── components/
│   ├── layout/             — Header, LocationSwitcher, BottomSheet, Sidebar
│   ├── charts/             — RevenueChart (Recharts), and other chart wrappers
│   └── ui/                 — Shared primitives (segmented-control, etc.)
├── hooks/                  — Custom React hooks
├── lib/                    — Utilities: format.ts (formatProductName), api client
├── stores/
│   ├── app.ts              — useAppStore (Zustand): location filter, companySettings, uiPrefs
│   └── auth.ts             — useAuthStore (Zustand): JWT tokens, user object
└── i18n/
    ├── index.ts            — useI18nStore (Zustand), useT() hook
    ├── en.json             — Canonical locale (source of truth for keys)
    ├── ru.json             — Russian (default locale)
    ├── kk.json             — Kazakh
    └── es.json             — Spanish
```

### State Management

Two Zustand stores cover all global state:

- **`useAppStore`** — Multi-location filter (`selectedLocationIds`), derived single `activeLocationId` used for API calls, `companySettings` (country, currency, currency_symbol, locale from company row), per-device `uiPrefs` persisted to `localStorage`.
- **`useAuthStore`** — JWT access token, refresh token, decoded user profile (id, email, role, company_id).

`useCurrency()` is a helper selector exported from `app.ts` that returns `companySettings.currency_symbol`.

---

## Database

### Multi-Tenancy via RLS

Every tenant-scoped table has RLS enabled and a policy of the form:

```sql
USING (company_id = current_setting('app.current_tenant', true)::uuid)
```

The Go API sets this at the start of each authenticated request via `SET LOCAL app.current_tenant = '<uuid>'`. The middleware chain is: `JWTAuth` (extracts company_id from token) → handler acquires DB connection → sets `app.current_tenant`.

### Migrations (000001 – 000020)

| Migration | Tables Created / Modified |
|-----------|--------------------------|
| 000001 | `companies`, `locations`, `users`, `user_locations`, `sessions` |
| 000002 | `revenue_facts`, `product_sales_facts`, `purchase_facts`, `stock_snapshots`, `iiko_sync_log`, `dashboard_daily_revenue` (materialized view) |
| 000003 | `supply_requests`, `supply_request_items`, `transfer_requests`, `transfer_request_items` |
| 000004 | `notifications` |
| 000005 | `ai_tasks`, `uploaded_files` |
| 000006 | `invites`; adds `reset_token` / `reset_token_expires` to `users` |
| 000007 | Adds `iiko_server_url`, `iiko_login`, `iiko_password` to `companies` |
| 000008 | Adds `iiko_order_id` column to `product_sales_facts` |
| 000009 | Revenue uniqueness constraints |
| 000010 | Adds `order_number` to `revenue_facts` (sourced from iiko `OrderNum` float64 field) |
| 000011 | `purchase_line_items`; Telegram payment tables |
| 000012 | `supplier_aliases`; Telegram hardening |
| 000013 | Payment error dictionary |
| 000014 | Adds `country`, `currency_code`, `currency_symbol`, `locale` to `companies`; seeds KZ defaults |
| 000015 | `product_aliases` |
| 000016 | `recipe_components` |
| 000017 | Adds dish unit column to recipe data |
| 000018 | `stock_overrides` |
| 000019 | Adds `city`, `pos_system` to `locations` |
| 000020 | Expands `users.role` CHECK to include `general_manager`, `manager`, `bartender`, `waiter`, `cashier`, `accountant` |

### Key Tables Summary

| Table | Description |
|-------|-------------|
| `companies` | Tenants. Holds iiko Server credentials (url, login, password) and i18n settings (country, currency, locale). |
| `locations` | Restaurants within a company. Holds `iiko_org_id`, `city`, `pos_system`. |
| `users` | All users. `role` is one of owner / employee / general_manager / manager / bartender / waiter / cashier / accountant. |
| `user_locations` | M:N assignment of employees to locations. |
| `sessions` | Refresh tokens with expiry. |
| `invites` | Email invite tokens for non-owner onboarding. |
| `revenue_facts` | One row per iiko order. `revenue` is in KZT, no subunits. Unique on `(company_id, iiko_order_id)`. |
| `product_sales_facts` | One row per product per day per location. Includes `cost_price` from `ProductCostBase.ProductCost`. |
| `purchase_facts` | iiko incoming invoices. `total_sum` in KZT. |
| `purchase_line_items` | Line items from iiko invoice XML. |
| `stock_snapshots` | Point-in-time inventory from iiko `/resto/api/v2/reports/balance/stores`. |
| `stock_overrides` | User-entered corrections for stock amounts and unit prices. NULL columns fall back to iiko values. |
| `recipe_components` | Dish→ingredient mapping from iiko assembly charts. PK: `(company_id, dish_iiko_id, ingredient_iiko_id)`. |
| `supply_requests` + `supply_request_items` | Manual supplier order requests. |
| `transfer_requests` + `transfer_request_items` | Inter-location stock transfer requests. |
| `notifications` | In-app notifications (low_stock, supply_approved, sync_failed, etc.). `user_id` NULL = company-wide. |
| `ai_tasks` | User-created analysis tasks tracked for AI suggestions. |
| `uploaded_files` | Uploaded invoice documents (status: uploaded → processing → processed). |
| `supplier_aliases` | User-edited supplier display names overriding iiko GUIDs. |
| `product_aliases` | User-edited product display names overriding iiko raw strings. |
| `iiko_sync_log` | Per-sync-run log: type, status, records_synced, duration_ms. |
| `dashboard_daily_revenue` | Materialized view: daily order count + revenue + discount per company/location. Refreshed after every sync cycle. |

---

## iiko Sync Flow

### Sync Worker Tickers

| Sync Type | Interval | iiko Endpoint(s) |
|-----------|----------|-----------------|
| `revenue` | 15 min | OLAP `/resto/api/v2/reports/olap` |
| `product_sales` | 15 min | OLAP `/resto/api/v2/reports/olap` |
| `purchases` | 60 min | XML `/resto/api/documents/export/incomingInvoice` |
| `stock` | 30 min | JSON `/resto/api/v2/reports/balance/stores` |
| `recipes` | 6 h | JSON `/resto/api/v2/assemblyCharts/getPrepared?productId=&date=` |

An initial `syncType = "all"` run fires immediately on worker startup.

### Revenue OLAP Aggregation (Critical Gotcha)

iiko OLAP does **not** aggregate `DishSumInt` by order when only `UniqOrderId.Id` is in `GroupByRowFields` — it returns a single arbitrary dish value. The correct approach, enforced in `internal/sync`:

1. Group by `UniqOrderId.Id` + `OpenDate.Typed` + `DishName` in the OLAP request.
2. In Go, accumulate `DishSumInt` per `UniqOrderId.Id` using a `map[string]float64`.
3. Upsert the summed total into `revenue_facts`.
4. After sync: call `ValidateRevenueAfterSync` — asserts `MAX(revenue) > 10,000` per day.

`DishSumInt` is **already in KZT** (no subunit division — never divide by 100).

### Nomenclature Fallback

`GetNomenclature` tries XML v1 (`/resto/api/products`) first because it returns `mainUnit` as a human-readable string (e.g., "кг", "шт"). If the XML parse fails or returns 0 products, it falls back to JSON v2 (`/resto/api/v2/entities/products/list`) where `mainUnit` is a GUID. Product GUIDs from stock balance and invoice XML are resolved to names + units via this nomenclature map.

### Assembly Charts (Recipes)

Each dish recipe is fetched individually: `GET /resto/api/v2/assemblyCharts/getPrepared?productId=<dish_id>&date=<YYYY-MM-DD>`. Results are upserted into `recipe_components` as `(company_id, dish_iiko_id, ingredient_iiko_id)` rows. The reverse index `idx_recipe_components_ingredient` supports the primary query: "which dishes consume this ingredient?"

### Post-Sync

After each full company cycle, `RefreshDashboardViews` issues `REFRESH MATERIALIZED VIEW CONCURRENTLY dashboard_daily_revenue`.

---

## i18n Architecture

### Design Principle

Backend handlers never render user-facing text. The `Suggestion` struct (and other i18n-aware responses) carry:

```go
type Suggestion struct {
    TitleKey          string         `json:"title_key"`
    TitleParams       map[string]any `json:"title_params,omitempty"`
    DescriptionKey    string         `json:"description_key"`
    DescriptionParams map[string]any `json:"description_params,omitempty"`
    // ...
}
```

The backend formats param values (e.g., calls `formatProductName()` on product names) so the frontend substitutes them verbatim.

### Frontend Resolution

`useT()` (from `frontend/src/i18n/index.ts`) is a Zustand-backed hook:

```typescript
const t = useT()
t('dashboard.totalRevenue')                      // "Общая выручка"
t('ai.s.topSeller.title', { product: 'Плов' })  // interpolates {product}
```

Resolution order: current locale → English fallback → raw key.

Placeholder syntax: `{name}` replaced by `params[name]` via regex.

### Locales

| Code | Language | Notes |
|------|----------|-------|
| `en` | English | Canonical key source — all new keys added here first |
| `ru` | Russian | Default locale (stored in `localStorage` as `foodbi_locale`) |
| `kk` | Kazakh | |
| `es` | Spanish | |

Locale is persisted per-device in `localStorage` under key `foodbi_locale`. The `useI18nStore` Zustand store reads it on mount.

---

## Auth

### Flow

1. **Register** — POST `/api/v1/auth/register`. Creates company (if owner) + user row. Sends 6-digit OTP to email. User is inactive until OTP verified.
2. **Verify OTP** — POST `/api/v1/auth/verify-otp`. Sets `users.is_active = true`. Returns JWT token pair.
3. **Login** — POST `/api/v1/auth/login`. Validates bcrypt password + active status. Returns access token (short-lived JWT) + refresh token (stored in `sessions`).
4. **Refresh** — POST `/api/v1/auth/refresh`. Validates refresh token from `sessions`, issues new token pair.

### JWT Structure

Claims: `user_id`, `company_id`, `role` (standard HMAC-SHA256 JWT via `golang-jwt/jwt/v5`). Secret from `JWT_SECRET` env var; defaults to a dev placeholder if unset.

### Middleware Chain (authenticated routes)

```
JWTAuth  →  TenantContext  →  handler
```

- `JWTAuth`: parses `Authorization: Bearer <token>`, populates `user_id`, `company_id`, `role` into request context.
- `TenantContext`: validates `company_id` is non-nil.
- Each handler that queries the DB calls `SET LOCAL app.current_tenant = '<company_id>'` at the start of each transaction, activating RLS.

### Roles

Authorization in handlers gates on `role == "owner"` vs everything else. The finer roles (`general_manager`, `manager`, `bartender`, `waiter`, `cashier`, `accountant`) are stored and passed through JWT but no per-role permission matrix is enforced beyond the owner gate as of migration 000020.

---

## Mobile (Capacitor iOS)

The frontend is wrapped as a native iOS app using Capacitor.

- **Project location**: `frontend/ios/App/App.xcodeproj`
- **Bundle ID**: `kz.foodbi`
- **Mechanism**: WKWebView renders the Vite-built React app; Capacitor plugins bridge native iOS APIs where needed.

The web and iOS apps share the same React codebase. The `VITE_API_URL` environment variable points to the Railway production URL in both the web build (`frontend/.env.production`) and the iOS bundle.

---

## Deployment

### Platform

Backend is deployed on **Railway** at `https://foodbi-production.up.railway.app`.

### Build (Dockerfile at repo root)

```
Stage 1 (golang:alpine):
  go build -o /api  ./cmd/api/
  go build -o /sync ./cmd/sync/

Stage 2 (alpine:3.19):
  COPY /api, /sync, backend/migrations/
  EXPOSE 8080
  CMD ["./api"]
```

The sync binary is included in the image but started separately (Railway service or manual `./sync` invocation). <!-- VERIFY: confirm whether sync runs as a separate Railway service or is started manually alongside the API container -->

### railway.json (repo root)

```json
{
  "build":  { "builder": "DOCKERFILE", "dockerfilePath": "Dockerfile" },
  "deploy": {
    "startCommand": "./api",
    "healthcheckPath": "/health",
    "healthcheckTimeout": 30,
    "restartPolicyType": "ON_FAILURE"
  }
}
```

### Required Environment Variables

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `JWT_SECRET` | HMAC secret for JWT signing |
| `TELEGRAM_BOT_TOKEN` | Optional — bot disabled if absent |
| `MIGRATIONS_DIR` | Path to migrations directory (defaults to `./migrations`) |
| `ENV` | Set to `production` for JSON log output |

<!-- VERIFY: confirm full list of required env vars (SMTP credentials for OTP email, any payment webhook secret) -->

---

## Key Architectural Decisions and Gotchas

### iiko Monetary Values
`DishSumInt` from iiko OLAP reports is already in **KZT with no subunits**. Never divide by 100. Never multiply by any factor in the sync pipeline (iiko → sync → DB → API → frontend).

### Number Formatting
All monetary values render with `toLocaleString('ru-KZ', { maximumFractionDigits: 0 })`. Never `toFixed(2)`, never `'en'` locale, no hardcoded currency symbols (`€`, `$`, `₽`). Use `useCurrency()` hook for the symbol.

### SQL Date Ranges
Always `>= start AND < (end::date + 1)` (exclusive upper bound). Never `<= end`.

### Product Name Formatting
iiko returns ALL-CAPS Russian names. Apply `formatProductName()` (first letter uppercase, rest lowercase) everywhere iiko-sourced names are rendered — both in the frontend (`@/lib/format`) and in the backend Go helper in `internal/ai`.

### i18n Canonical Source
`en.json` is the canonical key source. All new translation keys must be added there first, then translated into `ru`, `kk`, `es`.

### schema_migrations Backfill
If `schema_migrations` is empty, `RunMigrations()` fails because migration 000001 is seen as a duplicate when the table was pre-populated by a manual schema import. Backfill the table from migration filenames before adding any new migration file.

### Stale `go run` Processes
`go run` compiles to a temp binary under `/var/folders/...` which persists across reboots. Always kill all old processes before starting new ones: `pkill -9 -f foodbi`. Otherwise stale workers write to the DB from outdated code.

### PostgreSQL Native vs Docker
Using `brew postgresql@16` on macOS while Docker Compose is also running will cause a port 5432 conflict. Stop the Homebrew service (`brew services stop postgresql@16`) when using the Docker Compose database, or vice versa.

### Unit Resolution
The `GetNomenclature` function tries the XML v1 endpoint first to obtain human-readable unit strings. If that fails it falls back to JSON v2 where units are GUIDs. Code consuming nomenclature must handle both forms.

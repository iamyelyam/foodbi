# FoodBI Project Map

**What this is:** a lookup table that tells future Claude sessions (and humans) which files to read when tackling a given task. Instead of skimming the whole repo (~200 KB of source), start here, jump to the relevant section, then open only the 3–8 files that section lists.

Canonical source order for any task:
1. This map (you're reading it).
2. [ARCHITECTURE.md](./ARCHITECTURE.md) when you need a diagram-level view of the full system.
3. The module's `AGENTS.md` (hot modules only — see table below).
4. The specific files cited in AGENTS.md / that cover your task.

## Hot modules with `AGENTS.md`

These six backend packages + four frontend page-groups cover ~80% of day-to-day work. Each has a local `AGENTS.md` with purpose, file list, DB tables, gotchas.

| Module | AGENTS.md |
|---|---|
| Backend sync worker (iiko pulls) | [backend/internal/sync/AGENTS.md](../backend/internal/sync/AGENTS.md) |
| Backend iiko client | [backend/internal/iiko/AGENTS.md](../backend/internal/iiko/AGENTS.md) |
| Backend stock (+ recipes, overrides) | [backend/internal/stock/AGENTS.md](../backend/internal/stock/AGENTS.md) |
| Backend revenue | [backend/internal/revenue/AGENTS.md](../backend/internal/revenue/AGENTS.md) |
| Backend purchases | [backend/internal/purchases/AGENTS.md](../backend/internal/purchases/AGENTS.md) |
| Backend AI suggestions | [backend/internal/ai/AGENTS.md](../backend/internal/ai/AGENTS.md) |
| Frontend Stock page | [frontend/src/pages/stock/AGENTS.md](../frontend/src/pages/stock/AGENTS.md) |
| Frontend Revenue page | [frontend/src/pages/revenue/AGENTS.md](../frontend/src/pages/revenue/AGENTS.md) |
| Frontend Purchases page | [frontend/src/pages/purchases/AGENTS.md](../frontend/src/pages/purchases/AGENTS.md) |
| Frontend AI Suggestions pages | [frontend/src/pages/ai-suggestions/AGENTS.md](../frontend/src/pages/ai-suggestions/AGENTS.md) |

## Task → files lookup

Pick the row that matches what you're about to do. Open the listed files. Don't read anything else up front.

### Backend

| Task | Files to open |
|---|---|
| Add a new API endpoint in existing module | `backend/cmd/api/main.go` (routes), `backend/internal/{module}/handler.go`, `docs/API.md` |
| Create a new backend module | `backend/cmd/api/main.go`, any existing `backend/internal/{similar}/handler.go` as template, `backend/internal/middleware/auth.go` |
| Write a database migration | `backend/migrations/` (pick the latest number and follow the format), `backend/internal/database/migrate.go` |
| Modify iiko sync behavior | `backend/internal/sync/AGENTS.md`, `backend/cmd/sync/main.go` (tickers), `backend/internal/sync/service.go` |
| Try a new iiko endpoint | `backend/internal/iiko/AGENTS.md`, `backend/internal/iiko/api.go`, optionally spike via `backend/cmd/probe-*/main.go` |
| Add stock-level feature (override, alias, recipe lookup) | `backend/internal/stock/AGENTS.md`, `backend/internal/stock/handler.go` |
| Revenue data shape / OLAP change | `backend/internal/revenue/AGENTS.md`, `backend/internal/sync/service.go::SyncRevenue` |
| Add a new AI suggestion type | `backend/internal/ai/AGENTS.md`, `backend/internal/ai/handler.go`, `frontend/src/i18n/*.json` (new `ai.s.{type}.*` keys) |
| Change auth / JWT / OTP / roles | `backend/internal/auth/`, `backend/internal/middleware/auth.go`, `backend/migrations/000020_employee_roles.up.sql`, `frontend/src/lib/employeeRoles.ts` |
| Multi-tenant / RLS behavior | `backend/internal/middleware/auth.go` (TenantContext), `backend/internal/database/pool.go`, any migration with `FORCE ROW LEVEL SECURITY` |
| Add / change notification type | `backend/internal/notifications/handler.go`, `frontend/src/pages/notifications/NotificationsPage.tsx`, `frontend/src/i18n/*.json` (`notifications.*`) |

### Frontend

| Task | Files to open |
|---|---|
| Add a new page | `frontend/src/App.tsx` (routes), a similar existing page as template, `frontend/src/hooks/useApi.ts` (add react-query hook) |
| Edit an existing page | That page's `AGENTS.md` if it's in a hot module, otherwise just the `*.tsx` file + its hooks |
| Add i18n string | `frontend/src/i18n/en.json` (canonical source), then mirror to ru/kk/es — OR invoke `/localize` skill for a new language |
| Add an i18n-interpolated string (`{placeholder}`) | `frontend/src/i18n/AGENTS.md` if present, otherwise the `useT` signature in `frontend/src/i18n/index.ts` |
| Create a new chart | `frontend/src/components/charts/RevenueChart.tsx` as template |
| Add a shared UI component | `frontend/src/components/ui/` |
| Add a shared layout component | `frontend/src/components/layout/` (BottomSheet, Header, Tabbar, DateRangeSheet) |
| Change global store / persistence | `frontend/src/stores/app.ts` (uiPrefs, locations, currency), `frontend/src/stores/auth.ts` (tokens) |
| New language | Run `/localize <lang>` skill (project-scoped at `.claude/skills/localize/`) |
| Dashboard / home changes | `frontend/src/pages/DashboardPage.tsx` |
| Stock (list, overrides, recipes) | [frontend/src/pages/stock/AGENTS.md](../frontend/src/pages/stock/AGENTS.md) |
| Revenue (orders, products, detail) | [frontend/src/pages/revenue/AGENTS.md](../frontend/src/pages/revenue/AGENTS.md) |
| Purchases (invoices, suppliers) | [frontend/src/pages/purchases/AGENTS.md](../frontend/src/pages/purchases/AGENTS.md) |
| AI Suggestions (list, detail, WhatsApp share) | [frontend/src/pages/ai-suggestions/AGENTS.md](../frontend/src/pages/ai-suggestions/AGENTS.md) |

### Mobile / Deployment

| Task | Files to open |
|---|---|
| iOS app change / Bundle ID / TestFlight | `frontend/capacitor.config.ts`, `frontend/ios/App/App.xcodeproj/project.pbxproj`, [DEPLOY_TESTFLIGHT.md](../DEPLOY_TESTFLIGHT.md) |
| Production env / Railway | [docs/DEPLOYMENT.md](./DEPLOYMENT.md), `railway.json`, `Dockerfile`, `frontend/.env.production` |
| Env var changes | [docs/CONFIGURATION.md](./CONFIGURATION.md), `backend/.env.example`, `backend/internal/database/pool.go` (where `DATABASE_URL` is read) |

## Non-hot modules (read only when touching)

These don't have their own `AGENTS.md` yet — open the handler file directly:

- `backend/internal/auth/` — JWT, OTP, register/login
- `backend/internal/profiles/` — user profile
- `backend/internal/employees/` — team management
- `backend/internal/locations/` — company locations
- `backend/internal/dashboard/` — dashboard summary metrics
- `backend/internal/statistics/` — time-series stats
- `backend/internal/transfers/` — inter-location inventory transfers
- `backend/internal/supplying/` — supply requests
- `backend/internal/files/` — file uploads (invoices)
- `backend/internal/notifications/` — notification delivery
- `backend/internal/payments/` — payment webhook
- `backend/internal/telegram/` — Telegram bot

If a task makes you read any of these twice in a month, consider adding an `AGENTS.md` for it.

## Protocol for starting work

1. **Small change (one file, clear scope):** open that file. Skip this map.
2. **Feature change (new endpoint / page / data):** open this map → jump to the right row → open only the files listed.
3. **Cross-cutting change (auth, RLS, i18n, migrations):** open `ARCHITECTURE.md` first, then this map for the specific lookup.
4. **Never read:** `node_modules/`, `frontend/ios/Pods/`, `backend/vendor/`, compiled binaries (`foodbi-api`, `foodbi-sync`), `frontend/dist/`.

When writing or editing an `AGENTS.md`, keep it under ~150 lines. If it grows past that, the module is big enough to deserve its own folder-level split.

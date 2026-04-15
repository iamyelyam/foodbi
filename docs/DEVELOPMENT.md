<!-- generated-by: gsd-doc-writer -->
# Development Guide

Day-to-day reference for engineers working on FoodBI. For environment variables and secrets see [docs/CONFIGURATION.md](CONFIGURATION.md). For running tests see [docs/TESTING.md](TESTING.md).

---

## Daily Commands

### Backend API

Kill any stale process, rebuild, and restart:

```bash
pkill -9 -f foodbi-api || true
cd backend
/opt/homebrew/bin/go build -o foodbi-api ./cmd/api/
nohup ./foodbi-api > /tmp/foodbi-api.log 2>&1 & disown
```

Tail logs:

```bash
tail -F /tmp/foodbi-api.log
```

### Backend Sync

```bash
pkill -9 -f foodbi-sync || true
cd backend
/opt/homebrew/bin/go build -o foodbi-sync ./cmd/sync/
nohup ./foodbi-sync > /tmp/foodbi-sync.log 2>&1 & disown
```

Tail sync logs:

```bash
tail -F /tmp/foodbi-sync.log
```

> **Important:** Always use `/opt/homebrew/bin/go` (Go 1.26.x). Do not use `/usr/local/go/bin/go` (outdated 1.19 install).

### Frontend

```bash
cd frontend
npm run dev          # Vite dev server with HMR
npm run build        # tsc + vite build — catches TypeScript errors
npm run lint:money   # project-specific script that checks KZT formatting rules
```

---

## Adding a Database Migration

Migrations live in `backend/migrations/` and are applied automatically when the API starts.

**Convention:** two paired files per migration.

```
backend/migrations/000NNN_short_description.up.sql
backend/migrations/000NNN_short_description.down.sql
```

The current latest migration number is **000020** (`000020_employee_roles`). Increment to `000021` for the next one.

**Steps:**

1. Create `backend/migrations/000021_your_feature.up.sql` and `backend/migrations/000021_your_feature.down.sql`.
2. Write forward SQL in the `.up.sql` file and reverse SQL in `.down.sql`.
3. Restart the API — `RunMigrations` applies any pending migrations on startup.

**Troubleshooting — empty `schema_migrations` table:**

If `RunMigrations` fails on migration `000001` with a duplicate-key error, the `schema_migrations` table is empty but the schema already exists. Backfill it from the filenames before adding your new migration:

```sql
INSERT INTO schema_migrations (version)
SELECT regexp_replace(filename, '_(.*)', '')
FROM (
  VALUES
    ('000001_init'),
    ('000002_...'),
    -- add all existing migration numbers up to 000020
    ('000020_employee_roles')
) AS t(filename)
ON CONFLICT DO NOTHING;
```

Then restart the API — it will detect only the new migration as pending.

---

## Adding a Backend Handler

Follow the pattern used by every existing module (e.g. `backend/internal/stock/`):

**1. Create the handler file.**

```
backend/internal/{module}/handler.go
```

Minimal structure:

```go
package mymodule

import (
    "github.com/go-chi/chi/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
    db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
    return &Handler{db: db}
}

func (h *Handler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Get("/", h.List)
    return r
}
```

**2. Register the handler in `backend/cmd/api/main.go`.**

Instantiate the handler after the database pool is ready:

```go
myHandler := mymodule.NewHandler(db)
```

Mount it inside the authenticated route group (which already applies `middleware.JWTAuth` and `middleware.TenantContext`):

```go
r.Group(func(r chi.Router) {
    r.Use(middleware.JWTAuth)
    r.Use(middleware.TenantContext)
    // existing mounts ...
    r.Mount("/mymodule", myHandler.Routes())
})
```

**RLS requirement:** All data-access routes must live inside the `r.Group` that uses both `middleware.JWTAuth` and `middleware.TenantContext`. These middleware set the `app.current_tenant` PostgreSQL session variable that Row-Level Security relies on. Never expose tenant data on unauthenticated routes.

**Exception:** Webhook endpoints (e.g. payment callbacks) live outside the authenticated group and use HMAC signature verification instead:

```go
r.Post("/api/v1/webhooks/payment/{companyID}", paymentHandler.HandleWebhook)
```

---

## Adding a Frontend Page

**1. Create the page component.**

```
frontend/src/pages/{feature}/MyFeaturePage.tsx
```

A typical page uses these shared components:

```tsx
import { Header } from '@/components/layout/Header'
import { Tabbar } from '@/components/layout/Tabbar'
import { BottomSheet } from '@/components/layout/BottomSheet'
import { DateRangeSheet } from '@/components/layout/DateRangeSheet'
import { useDashboard } from '@/hooks/useApi'          // react-query hooks
import { useAppStore, useCurrency } from '@/stores/app'
import { useT } from '@/i18n'
```

**2. Register the route in `frontend/src/App.tsx`.**

Add a `<Route>` entry for the new page.

**Shared component rules:**

- `<BottomSheet>` — the only permitted bottom-sheet implementation. Handles 300 ms slide-up/slide-down animation and mount/unmount timing. Do not re-implement.
- `<DateRangeSheet>` / `<DateRangeBlock>` — shared date-range picker. Use it instead of rolling a custom one.
- `useCurrency()` — returns the formatted currency string. Never hardcode `₸`, `$`, `€`, or `₽`.
- `useT()` — returns the translation function. All user-visible strings must go through it.

**Data fetching:** Use react-query hooks from `@/hooks/useApi`. Do not call `axios` directly from page components.

---

## i18n Workflow

The project supports four locales: `en`, `ru`, `kk`, `es`. Russian (`ru`) is the default.

### Adding or updating translation keys

`en.json` is the canonical file. Every other locale file must have a matching key.

```
frontend/src/i18n/en.json   ← add new key here first
frontend/src/i18n/ru.json
frontend/src/i18n/kk.json
frontend/src/i18n/es.json
```

Keys are dot-namespaced objects. Example addition in `en.json`:

```json
{
  "stock": {
    "lowStockAlert": "Low stock: {product}"
  }
}
```

Then add the matching key to `ru.json`, `kk.json`, and `es.json`.

### Using translations in components

```tsx
const t = useT()

// Simple key
t('stock.lowStockAlert')

// With interpolation
t('stock.lowStockAlert', { product: 'Плов классический' })
// → "Low stock: Плов классический"
```

### Adding a new locale (automated)

Use the project-scoped Claude skill:

```
/localize <lang>
```

This invokes `.claude/skills/localize/SKILL.md` which creates `frontend/src/i18n/{lang}.json` and wires it into `index.ts` automatically.

### Backend i18n

Backend handlers return a `{key, params}` shape — they do not render localized text. The frontend's `useT(key, params)` interpolates the final string. The `ai` handler follows this pattern as the reference implementation (`backend/internal/ai/handler.go`).

---

## Code Style

### Go

- Format with `gofmt` (or `goimports`). Run before committing.
- No linter config file is checked in — standard `go vet` is the baseline.

### TypeScript / React

ESLint is present in the project. Run the linter to check for issues.

**Project-specific rules enforced by code review and `npm run lint:money`:**

| Rule | Correct | Wrong |
|------|---------|-------|
| KZT display | `value.toLocaleString('ru-KZ', { maximumFractionDigits: 0 })` | `value.toFixed(2)`, `toLocaleString('en')` |
| Currency symbol | `useCurrency()` hook | Hardcoded `₸`, `$`, `€`, `₽` |
| iiko monetary sums | Use the raw value as-is | Divide or multiply `DishSumInt` |
| SQL date ranges | `>= start AND < (end::date + 1)` | `<= end` |
| Product/dish names | `formatProductName(name)` from `@/lib/format` | Raw uppercase iiko strings |
| Bottom sheets | `<BottomSheet>` component | Custom modal/drawer implementations |

**`formatProductName()`** converts iiko's ALL-CAPS product names to sentence case: `"ПЛОВ КЛАССИЧЕСКИЙ"` → `"Плов классический"`. Apply it everywhere iiko-sourced names are rendered. An equivalent Go helper exists in the backend for AI-generated titles.

---

## iiko Integration Quirks

These rules are critical. Violating them produces silent data corruption.

1. **Never divide `DishSumInt` by 100.** iiko returns monetary values already in KZT (not sub-units).
2. **OLAP does not aggregate by order.** When `GroupByRowFields` contains only `UniqOrderId.Id`, iiko returns one arbitrary dish row — not the order total. Always include `DishName` in `GroupByRowFields` and `SUM(DishSumInt)` per `UniqOrderId.Id` in Go before upserting.
3. **Revenue OLAP grouping:** `UniqOrderId.Id` + `OpenDate.Typed` + `DishName`. Aggregate per order in Go.
4. **Product names stay in Russian** in iiko's nomenclature. Do not translate them at the source — apply `formatProductName()` only at display time.
5. **After each sync** verify `SELECT MAX(revenue) FROM revenue_facts` returns > 10,000 per day as a sanity check.

---

## Debugging

### Backend

```bash
# Live API logs
tail -F /tmp/foodbi-api.log

# Live sync logs
tail -F /tmp/foodbi-sync.log

# Check which migration version the DB thinks it is on
psql $DATABASE_URL -c "SELECT * FROM schema_migrations ORDER BY version;"

# Kill stale go-run processes (processes from /var/folders can persist for days
# and will overwrite the DB with old behaviour)
pkill -9 -f "go-build" || true
pkill -9 -f foodbi-api  || true
pkill -9 -f foodbi-sync || true
```

### Database

The app uses `SET LOCAL app.current_tenant = '<company_id>'` for RLS. Queries run outside an authenticated request context will return no rows (by design). Use a superuser connection or set the variable explicitly when debugging:

```sql
SET app.current_tenant = 'your-company-uuid';
SELECT * FROM revenue_facts LIMIT 5;
```

---

## Git Workflow

- **Main branch** (`main`) is production. Pushes to `main` trigger an automatic Railway deployment.
- Create feature branches from `main`. Merge via pull request.
- Commit messages follow the conventional-commits style used in recent history (e.g. `feat:`, `fix:`, `chore:`).
- Run `npm run build` (type-check) and rebuild the Go binary before opening a PR to catch compile errors early.

---

## Custom Skills

| Skill | Invocation | Effect |
|-------|-----------|--------|
| Add a UI language | `/localize <lang>` | Creates `frontend/src/i18n/{lang}.json`, wires it into `index.ts`, and populates all keys from `en.json` |

Skills are project-scoped and live in `.claude/skills/`.

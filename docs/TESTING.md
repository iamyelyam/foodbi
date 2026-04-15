<!-- generated-by: gsd-doc-writer -->
# Testing

FoodBI does not yet have a comprehensive automated test suite. This document describes what
verification infrastructure exists today, how to run it, and what is explicitly missing.

---

## Current Test Coverage State

### Backend Go Tests

One test file exists in the backend:

**`backend/internal/sync/service_test.go`** — 9 unit tests covering the OLAP row parsing
and per-order aggregation logic in the sync package.

| Test | What it verifies |
|---|---|
| `TestGetFloat_Float64` | `GetFloat` extracts a `float64` from an OLAP row map |
| `TestGetFloat_Nil` | `GetFloat` returns `0` for a `nil` value |
| `TestGetFloat_MissingKey` | `GetFloat` returns `0` for a missing key |
| `TestGetString_Normal` | `GetString` extracts a string value correctly |
| `TestGetString_Nil` | `GetString` returns `""` for a `nil` value |
| `TestAggregateOrders_MultiDish` | Revenue, discount, and item count are summed correctly across dishes in one order |
| `TestAggregateOrders_MultipleOrders` | Rows belonging to different orders are separated correctly |
| `TestAggregateOrders_NoDivision` | Revenue is **never divided by 100** — a 49,000 KZT dish stays 49,000 |
| `TestAggregateOrders_LargeOrder` | A 20-dish banquet order accumulates to the correct total |

The `NoDivision` test is a regression guard for a critical business rule: iiko returns monetary
values in KZT with no subunits and they must never be divided by 100 anywhere in the pipeline.

No test files exist outside this package. All other backend packages (`ai`, `dashboard`,
`locations`, `purchases`, `stock`, `iiko`) have zero unit tests at this time.

### Frontend Tests

No test files exist under `frontend/src/`. The frontend `package.json` has no test runner
configured (no `jest`, `vitest`, or `mocha` in `devDependencies` and no `scripts.test` entry).

---

## Running the Backend Tests

```bash
# From the repo root
cd backend
go test ./...

# Test only the sync package
go test ./internal/sync/...

# With verbose output
go test -v ./internal/sync/...
```

The Go toolchain version required is `>= 1.26.1` (see `backend/go.mod`).

---

## Defense Layers in Production Code

In the absence of broad test coverage, several layers in the codebase catch bad data before
it reaches users.

### Layer 1 — Database CHECK Constraints (`backend/migrations/000009_revenue_constraints.up.sql`)

```sql
ALTER TABLE revenue_facts ADD CONSTRAINT chk_revenue_positive  CHECK (revenue >= 0);
ALTER TABLE revenue_facts ADD CONSTRAINT chk_revenue_sane      CHECK (revenue < 10000000);
ALTER TABLE revenue_facts ADD CONSTRAINT chk_discount_positive CHECK (discount >= 0);
ALTER TABLE product_sales_facts ADD CONSTRAINT chk_psf_revenue_positive CHECK (revenue >= 0);
ALTER TABLE purchase_facts      ADD CONSTRAINT chk_pf_totalsum_positive CHECK (total_sum >= 0);
```

Any write that violates these constraints is rejected by PostgreSQL at the database level,
regardless of which code path produced the value.

### Layer 2 — Row-Level Security (RLS) Tenant Isolation

All core tables have RLS enabled and forced via `ENABLE ROW LEVEL SECURITY` +
`FORCE ROW LEVEL SECURITY`. Policies gate every query on `app.current_tenant`, so one
company's data is never visible to another company's session. Tables covered include
`locations`, `users`, `sessions`, `revenue_facts`, `product_sales_facts`, `purchase_facts`,
`stock_snapshots`, `iiko_sync_log`, `ai_tasks`, `supplier_aliases`, `product_aliases`, and others.

### Layer 3 — Post-Sync Revenue Validation

After every successful revenue sync cycle, `svc.ValidateRevenueAfterSync` is called
(`backend/cmd/sync/main.go`, line 123). The intended check is:

```sql
SELECT MAX(revenue) FROM revenue_facts
WHERE company_id = $1 AND location_id = $2
```

The result should be `> 10,000` KZT for any active restaurant day. If the max is suspiciously
low, it signals that values may have been divided by 100 or that the sync produced no data.
Failures are logged as warnings; the sync worker does not halt.

### Layer 4 — Frontend Money Lint (`frontend/scripts/lint-money.sh`)

Run via `npm run lint:money` from the `frontend/` directory. Catches four classes of KZT
formatting bugs across all `.ts` and `.tsx` files:

| Check | What it catches |
|---|---|
| `.toFixed(2)` | KZT has no subunits; decimal formatting is wrong |
| `toLocaleString('en'` | Must use `'ru-KZ'` locale for Tenge formatting |
| `€`, `₽`, `£`, `¥` | Hardcoded non-KZT currency symbols |
| `/ 100` or `* 0.01` | Division on monetary values, which must never happen |

### Layer 5 — TypeScript Build and `tsc --noEmit`

CI runs `npx tsc --noEmit` on the frontend before the production build. This enforces type
correctness across all React components, hooks, and API client calls without running a test
runner.

---

## CI Pipeline

The GitHub Actions workflow at `.github/workflows/ci.yml` runs on every push and pull request
to `main`.

**Backend job:**
1. `go vet ./...` — static analysis for common Go mistakes
2. `go test ./...` — all Go unit tests

**Frontend job:**
1. `npm ci` — clean dependency install
2. `npx tsc --noEmit` — TypeScript type check
3. `bash scripts/lint-money.sh` — money formatting lint
4. `npm run build` — Vite production build (fails on bundler errors)

There is no separate lint step using ESLint or Prettier — the frontend `package.json` does
not configure either tool.

---

## Manual Verification Routines

When automated tests cannot cover a flow, use these manual checks.

### Post-Sync Database Verification

After triggering a manual sync or deploying a sync change, run these queries directly against
the database:

```sql
-- Revenue sanity: max order revenue per location should be > 10,000 KZT
SELECT location_id, MAX(revenue) AS max_order_revenue
FROM revenue_facts
GROUP BY location_id
ORDER BY max_order_revenue DESC;

-- Check no revenue was divided by 100
SELECT COUNT(*) AS suspicious_low_revenue
FROM revenue_facts
WHERE revenue < 100 AND revenue > 0;

-- Recent sync log
SELECT location_id, sync_type, synced_at, rows_upserted
FROM iiko_sync_log
ORDER BY synced_at DESC
LIMIT 20;
```

### Smoke Test Checklist

Run these checks manually after any significant code change before deploying:

- [ ] **Login** — Email + password login succeeds; session cookie is set; redirect to dashboard
- [ ] **Dashboard loads** — Metric cards (revenue, orders, avg check) show non-zero values for today or the last active day
- [ ] **Revenue page** — `/revenue` lists individual orders with amounts > 1,000 KZT; no orders show decimals
- [ ] **Stock page** — `/stock` shows product items with quantities and units (кг, шт, л)
- [ ] **Purchases page** — `/purchases` shows supplier invoices with positive totals
- [ ] **AI Suggestions** — `/ai-suggestions` returns at least one suggestion card; detail page loads
- [ ] **Location switcher** — Switching location in the header reloads data for the selected location only
- [ ] **Locale switch** — Profile page language selector switches between RU, KK, and EN; all labels update including Dashboard, Stock, Revenue, and Purchases pages
- [ ] **Mobile layout** — Bottom sheet components (filters, detail panels) slide up and down with animation on mobile viewport

---

## Adding a New Unit Test

### Backend (Go)

Create a `*_test.go` file in the package directory you want to test. Use the standard library
`testing` package — no additional framework is needed.

```go
// backend/internal/mypackage/myfile_test.go
package mypackage

import "testing"

func TestMyFunction(t *testing.T) {
    got := MyFunction("input")
    want := "expected"
    if got != want {
        t.Errorf("MyFunction(%q) = %q, want %q", "input", got, want)
    }
}
```

Run:

```bash
go test ./internal/mypackage/...
```

Pure business logic functions (formatters, calculators, aggregators) are the highest-priority
targets for new tests. Avoid testing functions that require a live database connection without
a test database setup.

### Frontend (Vitest — not yet configured)

Vitest is the natural choice given the project uses Vite. To add frontend unit tests when
the team is ready:

1. Install: `npm install --save-dev vitest @testing-library/react @testing-library/user-event jsdom`
2. Add to `vite.config.ts`:
   ```ts
   test: { environment: 'jsdom' }
   ```
3. Add to `package.json` scripts:
   ```json
   "test": "vitest run",
   "test:watch": "vitest"
   ```
4. Create test files as `src/**/*.test.tsx`

Priority targets: `formatProductName()` in `src/lib/format`, the `useCurrency()` hook, and
KZT locale formatting utilities.

---

## Known Gaps

The following test categories are entirely absent from the codebase:

| Gap | Description |
|---|---|
| **Backend integration tests** | No tests that exercise handlers against a real or test database |
| **iiko API contract tests** | No tests verifying OLAP response parsing against real or fixture responses |
| **Frontend component tests** | No React component rendering or interaction tests |
| **Frontend hook tests** | `useCurrency`, `useT` (i18n), and store hooks are untested |
| **End-to-end tests** | No Playwright, Cypress, or similar browser automation |
| **Sync data volume tests** | No tests asserting that a full sync produces the expected row count |
| **RLS enforcement tests** | No automated tests confirming cross-tenant data isolation |
| **Snapshot tests** | No UI snapshot or API response snapshot tests |
| **Performance / load tests** | No benchmarks on sync throughput or API response times |

---

## Future Direction

Recommended additions as the team grows, in priority order:

1. **Backend integration tests with `pgx` test pool** — Spin up a test PostgreSQL instance in
   CI (Docker service) and test sync functions end-to-end against real schema including
   constraints and RLS policies.

2. **Frontend Vitest + React Testing Library** — Unit test formatting utilities and critical
   hooks first (`formatProductName`, `useCurrency`), then add component tests for high-risk
   UI paths (revenue table rendering, locale switching).

3. **iiko fixture-based tests** — Capture real OLAP response JSON as fixtures and write
   parsing tests that do not require a live iiko connection.

4. **Playwright e2e suite** — Cover the login → dashboard → revenue → logout flow as a
   minimum smoke suite that runs in CI against a staging environment.

5. **RLS policy tests** — SQL-level tests (using `pgTAP` or a Go test with two tenant
   connections) that assert one tenant cannot read another tenant's `revenue_facts`.

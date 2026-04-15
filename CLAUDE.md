# FoodBI - Claude Instructions

## iiko Integration Rules

1. **NEVER divide monetary sums from iiko by 100** -- `DishSumInt` is already in restaurant currency (KZT)
2. **iiko OLAP does NOT aggregate DishSumInt by order** -- when GroupByRowFields has only `UniqOrderId.Id`, iiko returns a single arbitrary dish value, NOT the order total. **ALWAYS include `DishName` in GroupByRowFields and aggregate per-order in Go code.**
3. **OLAP for revenue**: group by `UniqOrderId.Id` + `OpenDate.Typed` + `DishName`, then SUM `DishSumInt` per `UniqOrderId.Id` in Go before upserting to `revenue_facts`
4. **OLAP for products**: group by `DishName` + `DishGroup` + `DishCategory` + `UniqOrderId.Id` + `OpenDate.Typed`
5. **After each sync**: auto-verify `SELECT MAX(revenue) FROM revenue_facts` is > 10,000 per day
6. **Before `go build` sync/api**: kill ALL old processes: `pkill -9 -f foodbi`

## Monetary Value Rules

- All monetary values from iiko are in KZT (Kazakhstan Tenge) with no subunits
- **Frontend**: use `toLocaleString('ru-KZ', { maximumFractionDigits: 0 })` -- never `toFixed(2)`, never locale `'en'`
- **Backend SQL date ranges**: always use `>= start AND < (end::date + 1)` pattern (exclusive upper bound) -- never `<= end`
- **No hardcoded currency symbols** (`€`, `$`, `₽`) -- always use `useCurrency()` hook
- **No division/multiplication on monetary values** anywhere in the pipeline (iiko -> sync -> DB -> API -> frontend)

## UI Rules

- **BottomSheet animation**: ALL bottom sheets across the project MUST slide smoothly up on open and slide smoothly down on close (300ms ease-out, overlay fades simultaneously). Always use the shared `<BottomSheet>` component from `@/components/layout/BottomSheet` — never re-implement. The component handles mount/unmount delay so the closing animation completes before the DOM is removed.
- **Product/dish names AND categories**: always format with `formatProductName()` (or alias `formatCategory`) from `@/lib/format` — first letter uppercase, rest lowercase (e.g. "Плов классический", not "ПЛОВ КЛАССИЧЕСКИЙ"; "Плов готовый", not "ПЛОВ ГОТОВЫЙ"). Applies everywhere iiko-sourced names are rendered. In backend use `formatProductName()` helper in Go for AI-generated titles.
- **No card background on white pages**: when page background is white, use light-gray metric cards (`bg-bg`). When page is gray (`bg-bg`), use white cards.

## Architecture

- Backend: Go + Chi router + pgx/v5 (PostgreSQL)
- Frontend: React + TypeScript + Vite + TailwindCSS + Recharts
- DB: PostgreSQL with RLS tenant isolation via `app.current_tenant`
- Sync: iiko Server API (OLAP reports + REST endpoints)

## Session Start — Read Order (token-efficient)

This project has ~200 source files. Reading them all eats ~200 KB of context. To stay under ~20 KB for most tasks, follow this order:

1. **Read [docs/PROJECT-MAP.md](docs/PROJECT-MAP.md) FIRST.** It maps tasks → the specific files to open.
2. **If the task touches a hot module**, read its `AGENTS.md` next. Hot modules (listed in PROJECT-MAP.md):
   - Backend: `backend/internal/{sync,iiko,stock,revenue,purchases,ai}/AGENTS.md`
   - Frontend: `frontend/src/pages/{stock,revenue,purchases,ai-suggestions}/AGENTS.md`
3. **Only then** open the specific `.tsx` / `.go` files cited in the AGENTS.md.
4. For cross-cutting topics (auth, RLS, i18n architecture, migrations), read `docs/ARCHITECTURE.md` FIRST before diving into files.
5. **Never load** `node_modules/`, `frontend/ios/Pods/`, `frontend/dist/`, compiled binaries.

If a non-hot module keeps coming up in your tasks, create an `AGENTS.md` for it (template: 50–150 lines, sections "Purpose / Files / Tables / Gotchas / When editing").

## Documentation Maintenance Protocol

The project has full documentation under `docs/` (README, ARCHITECTURE, GETTING-STARTED, DEVELOPMENT, TESTING, CONFIGURATION, API, DEPLOYMENT). These docs are generated via `/gsd-docs-update` and verified against the live codebase.

**After every commit that meaningfully changes user-visible behavior, infrastructure, or architecture, refresh affected docs:**

| Change type | Docs to refresh |
|---|---|
| New API endpoint / handler / route | `docs/API.md` |
| New migration / schema change | `docs/ARCHITECTURE.md` (Database section), `docs/API.md` if endpoints affected |
| New env var / config setting | `docs/CONFIGURATION.md` |
| New page or major UI flow | `docs/ARCHITECTURE.md` (Frontend section) |
| Deploy infrastructure change | `docs/DEPLOYMENT.md` |
| New dev command / workflow | `docs/DEVELOPMENT.md` |
| New i18n locale | `docs/ARCHITECTURE.md` (i18n section) — use `/localize` skill |

For broad refresh: run `/gsd-docs-update` (regenerates affected docs against live code). For verification only: `/gsd-docs-update --verify-only`.

Trivial commits (typo fixes, single-string i18n adds, dependency bumps, refactors that preserve behavior) do NOT require doc updates.

`CLAUDE.md` (this file) and `DEPLOY_TESTFLIGHT.md` are hand-written and intentionally preserved by the docs workflow.

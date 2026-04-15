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

# ai — suggestions + tasks

Generates AI suggestions on-the-fly from current data (no model call today — rule-based). Exposes `/api/v1/ai/suggestions` and a lightweight task-tracking pair of endpoints.

## Files

| File | What |
|---|---|
| `handler.go` | Routes: `GET /suggestions`, `POST /tasks`, `GET /tasks`. Generates 7 suggestion types; returns `SuggestionsResponse { summary, suggestions[] }`. |

## Suggestion types (as of commit)

| Type | Trigger | Loss/Gain |
|---|---|---|
| `menu_optimization` (top seller) | highest-revenue product in date window | gain ≈ topRevenue × 0.10 |
| `price_adjustment` (low margin) | product with margin < 30% | loss ≈ revenue × (0.30 − margin/100) |
| `cost_configuration` (suspicious margin) | margin > 90% or < 15% (usually misconfigured cost_price in iiko) | — |
| `stock_data_issue/negative` | latest stock_snapshot amount < 0 | loss ≈ Σ\|cost_sum\| |
| `stock_data_issue/zero_cost` | amount > 0 and cost_sum = 0 | — |
| `purchase_recommendation` (top supplier) | highest-spend supplier | gain ≈ spend × 0.05 |
| `collectMore` | fallback when no data | — |

## i18n key+params contract (important)

Backend does NOT return rendered text. Every `Suggestion` has:

```
TitleKey: "ai.s.topSeller.title"       TitleParams: {"product": "Плов классический"}
DescriptionKey: "ai.s.topSeller.description"
SolutionKey: "ai.s.topSeller.solution" (optional)
SolutionParams: {"product": "..."}
```

Frontend calls `t(s.title_key, s.title_params)` with placeholder substitution. This means adding a new suggestion type requires adding keys in all 4 locales (`frontend/src/i18n/en.json` + ru/kk/es) under `ai.s.{newType}.*`. Use `/localize` skill for new languages.

## Gotchas when editing

1. **Never return a raw string as title/description.** Always a key + params. If a new suggestion has no template text, still put it in i18n and reference by key — keeps the locale system consistent.
2. **Loss amounts should be positive.** Frontend displays `-{formatMoney(loss)}` itself. Gain amounts positive too.
3. **The `summary.total_loss` / `total_gain_with_ai` are sums** over `LossAmount` / `GainAmount` across all returned suggestions. Don't compute them on the frontend.
4. **Suggestions are generated fresh per request.** There's no database cache. If generation becomes slow, consider a materialized view or background refresh — but don't return stale data silently.
5. **Task IDs are random UUIDs generated per request.** Don't rely on them for deep links — they won't survive a reload. Frontend passes the full suggestion object via `useLocation().state` instead.

## When editing

- Adding a new suggestion type? 1) Add i18n keys `ai.s.{type}.title/description/solution` in all 4 locales. 2) Add a query block in `GetSuggestions()` that computes the signal and appends the `Suggestion` with key+params. 3) Add an entry to the table above.
- Changing loss/gain math? Update this file's table AND `docs/API.md` AI section so the frontend tooltip ("How is this calculated?") stays accurate.
- Persisting tasks (dedupe, status, assignment)? The `ai_tasks` table is simple (id, title, description, status, created_at). Add fields as a new migration.

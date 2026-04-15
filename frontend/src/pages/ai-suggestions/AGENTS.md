# ai-suggestions pages — list + detail + WhatsApp task share

## Files

| File | What |
|---|---|
| `AISuggestionsPage.tsx` | List: header summary (Total Loss / Total gain with AI) + suggestion cards (title + loss/gain + description + "How to fix it?" button). Each card navigates to detail with the suggestion passed via `useLocation.state`. |
| `AISuggestionDetailPage.tsx` | Detail: summary card + title + Description + Solution sections + "Create task" CTA that opens an assign-to sheet → WhatsApp share (`wa.me/...?text=...`). |

## Data source

- `GET /api/v1/ai/suggestions` → `{summary: {total_loss, total_gain_with_ai, date}, suggestions: Suggestion[]}`. Generated on-the-fly, no persistence.
- Each `Suggestion` carries i18n KEYS + PARAMS, not rendered text. Frontend resolves via `t(key, params)`.

## i18n key+params contract (important)

The backend sends:
```ts
{ title_key: "ai.s.topSeller.title", title_params: { product: "Плов классический" }, ... }
```

The frontend renders:
```tsx
<p>{t(suggestion.title_key, suggestion.title_params)}</p>
```

`useT` interpolates `{placeholder}` tokens from the params map. See `frontend/src/i18n/index.ts::useT`.

If a new suggestion type is added on the backend, add matching keys under `ai.s.{type}.*` in all 4 locale files (`en.json` / `ru.json` / `kk.json` / `es.json`), or run `/localize <lang>` to regenerate a locale.

## WhatsApp share flow

1. User clicks "Create task" on detail.
2. `<BottomSheet>` opens with employee list (with phones) + "Pick contact in WhatsApp" button.
3. Selecting an employee → `window.open(whatsappShareUrl(text, phone), '_blank')`.
4. Without a phone → `https://api.whatsapp.com/send?text=...` (opens contact picker).

Task message built via `buildTaskMessage(suggestion, t)` — concatenates title + solution + loss/gain + "— from FoodBI" signature, all i18n'd.

## Navigation

- List → Detail passes the full suggestion object via `navigate(..., { state: { suggestion, summary } })`. This avoids an extra network roundtrip + keeps IDs stable across the session.
- Detail falls back to `GET /suggestions` + find-by-id if state is empty (direct navigation or page reload).

## Gotchas when editing

1. **Suggestion IDs are random per request.** Don't deep-link by suggestion ID — the next `/suggestions` call will return new UUIDs. That's why navigate-with-state exists.
2. **Notification bell is hidden globally** (see `components/layout/Header.tsx::NOTIFICATIONS_ENABLED`). Don't wire any "open notifications from here" flow until the feature is enabled.
3. **Loss displays as `-{amount}`** in the danger color. Gain displays as `+{amount}` in success. Don't invert the signs.
4. **If the suggestion has no `solution_key`**, the Detail page falls back to `ai.fallbackSolution`. Always include a solution key on the backend for new suggestion types.

## When editing

- Adding a new field to the card? Extend the `<SuggestionCard>` sub-component. Make sure the field is read from the `Suggestion` interface and the field exists on the backend `Suggestion` struct.
- Changing how the info modal renders? See `<BottomSheet isOpen={showInfo}>` — explains Total Loss / Total gain. Keep the wording consistent with what the math actually does.
- New share target (SMS, Telegram)? Build another `shareUrl` helper similar to `whatsappShareUrl`. Make sure it URL-encodes the message.

# stock page — list, filters, overrides, recipes

## Files

| File | What |
|---|---|
| `StockPage.tsx` | List view + BottomSheet detail. Contains `MetricCard`, `EditableMetricCard`, `ChipButton` sub-components + `formatRecipeAmount()` helper. |

## Features on this page

1. **List** of current stock (one row per iiko_product_id, latest snapshot) with cost/amount + red highlight on negative amounts + stale-filter chip (>30 days) + low-stock filter chip.
2. **Download Excel** — client-side CSV export.
3. **BottomSheet detail** with:
   - Product name alias editor (pencil → inline input → save to `/stock/products/{id}/alias`).
   - 3-card override editor (Quantity / Price per unit / Cost value). Price per unit ↔ Cost value are linked: editing price recomputes cost. "Reset to iiko" link appears when `override_at` is present.
   - "Used in dishes" section (from `/stock/products/{id}/used-in`) — renders "0.24 л / порц." format via `formatRecipeAmount` + `dish_unit` suffix.

## Dependencies

- Backend: `backend/internal/stock/` (list + override + alias + used-in endpoints).
- Shared: `formatProductName` from `@/lib/format`, `useT` / `useI18nStore` from `@/i18n`, `BottomSheet` + `Header` + `Tabbar` from `@/components/layout/*`.
- Data: `useQuery` with keys `['stock', locationId]`, `['low-stock', locationId]`, `['stock-used-in', productId]` (enabled when bottom sheet is open).

## i18n keys used (partial)

`stock.title`, `stock.inStock`, `stock.writeOff`, `stock.downloadExcel`, `stock.chipStale`, `stock.chipLowStock`, `stock.noStockMatching`, `stock.productNamePlaceholder`, `stock.editProductName`, `stock.quantity`, `stock.pricePerUnit`, `stock.costValue`, `stock.resetOverride`, `stock.usedInDishes`, `stock.noRecipesForIngredient`, `stock.negativeStockWarning`, `stock.lowStockWarning`, `stock.lastSynced`, `stock.makeInventory`, `common.save`, `common.cancel`, `common.back`, `common.filter`, `common.loading`, `common.piecesShort`.

## Gotchas when editing

1. **Fallback unit.** Display `i.unit && !isUuid(i.unit) ? i.unit : t('common.piecesShort')`. iiko sometimes returns a raw GUID instead of a unit — `isUuid` catches this.
2. **Override optimistic update.** After `overrideMutation.onSuccess`, the local `selectedItem` state is patched so the sheet reflects changes without a re-fetch. The list invalidates via `queryClient.invalidateQueries`.
3. **`formatRecipeAmount`** uses `maximumSignificantDigits: 2` for tiny recipe amounts (e.g. 0.00086 kg per dish) with 6-decimal cap. Keep it — the user rejected `<0.01` sentinel earlier.
4. **Data source for "used in dishes"** is `recipe_components`, synced every 6h. Fresh iiko recipe changes won't appear until the next recipe tick.
5. **UUIDs as product_names** (products deleted from iiko nomenclature but still in stock) are filtered via `isUuid(i.product_name)` in `filtered` — don't show them in the list.

## When editing

- Adding a new metric card in the list header? Copy the `<MetricCard>` pattern. 2-col grid on white pages, 3-col for 3 cards max.
- Adding a new editable field in the detail sheet? Extend `EditableMetricCard` or add another instance. Wire a new backend field in `OverrideInput` struct on the Go side.
- Changing the recipe display format? Edit `formatRecipeAmount` + the `<span>` that renders it. Don't edit the server — `dish_unit` comes from sync, not computed client-side.

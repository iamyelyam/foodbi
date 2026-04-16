package stock

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/foodbi/backend/internal/middleware"
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
	r.Get("/", h.CurrentStock)
	r.Get("/low-stock", h.LowStock)
	r.Put("/products/{id}/alias", h.SetProductAlias)
	r.Put("/products/{id}/override", h.SetOverride)
	r.Get("/products/{id}/used-in", h.UsedInDishes)
	return r
}

type StockItem struct {
	ProductID    string   `json:"product_id"`
	ProductName  string   `json:"product_name"`
	Amount       float64  `json:"amount"`
	Unit         string   `json:"unit"`
	CostSum      float64  `json:"cost_sum"`
	PricePerUnit float64  `json:"price_per_unit"`
	SnapshotAt   string   `json:"snapshot_at"`
	OverrideAt   *string  `json:"override_at,omitempty"`
}

// stockSelect builds the base SELECT joining product_aliases (display name override)
// and stock_overrides (manual amount / unit price corrections). Effective values:
//   amount        = override.manual_amount (when set) else iiko snapshot amount
//   cost_sum      = override.manual_price_per_unit × effective amount (when override set) else iiko cost_sum
//   price_per_unit = override.manual_price_per_unit when set, else cost_sum / amount (or 0 if amount is 0)
const stockSelect = `SELECT DISTINCT ON (s.iiko_product_id)
	s.iiko_product_id,
	COALESCE(NULLIF(pa.display_name, ''), s.product_name) AS product_name,
	COALESCE(o.manual_amount, s.amount) AS amount,
	COALESCE(s.unit,'') AS unit,
	CASE
		WHEN o.manual_price_per_unit IS NOT NULL
			THEN COALESCE(o.manual_amount, s.amount) * o.manual_price_per_unit
		ELSE s.cost_sum
	END AS cost_sum,
	COALESCE(
		o.manual_price_per_unit,
		CASE WHEN s.amount <> 0 THEN s.cost_sum / s.amount ELSE 0 END
	) AS price_per_unit,
	s.snapshot_at,
	o.updated_at AS override_at
FROM stock_snapshots s
LEFT JOIN product_aliases pa ON pa.company_id = s.company_id AND pa.iiko_product_id = s.iiko_product_id
LEFT JOIN stock_overrides o  ON o.company_id  = s.company_id AND o.iiko_product_id  = s.iiko_product_id`

func (h *Handler) CurrentStock(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locIDs := middleware.ParseLocationFilter(r)

	query := stockSelect + ` WHERE s.company_id = $1`
	args := []interface{}{companyID}

	locFilter, args := middleware.AddLocationFilter(args, locIDs)
	query += locFilter
	query += ` ORDER BY s.iiko_product_id, s.snapshot_at DESC`

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch stock")
		return
	}
	defer rows.Close()

	var items []StockItem
	for rows.Next() {
		var s StockItem
		var t time.Time
		var overrideAt *time.Time
		if err := rows.Scan(&s.ProductID, &s.ProductName, &s.Amount, &s.Unit, &s.CostSum, &s.PricePerUnit, &t, &overrideAt); err != nil {
			continue
		}
		s.SnapshotAt = t.Format(time.RFC3339)
		if overrideAt != nil {
			iso := overrideAt.Format(time.RFC3339)
			s.OverrideAt = &iso
		}
		items = append(items, s)
	}
	if items == nil {
		items = []StockItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) LowStock(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locIDs := middleware.ParseLocationFilter(r)
	threshold := 5.0

	query := stockSelect + ` WHERE s.company_id = $1 AND s.amount <= $2`
	args := []interface{}{companyID, threshold}

	locFilter, args := middleware.AddLocationFilter(args, locIDs)
	query += locFilter
	query += ` ORDER BY s.iiko_product_id, s.snapshot_at DESC`

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch low stock")
		return
	}
	defer rows.Close()

	var items []StockItem
	for rows.Next() {
		var s StockItem
		var t time.Time
		var overrideAt *time.Time
		if err := rows.Scan(&s.ProductID, &s.ProductName, &s.Amount, &s.Unit, &s.CostSum, &s.PricePerUnit, &t, &overrideAt); err != nil {
			continue
		}
		s.SnapshotAt = t.Format(time.RFC3339)
		if overrideAt != nil {
			iso := overrideAt.Format(time.RFC3339)
			s.OverrideAt = &iso
		}
		items = append(items, s)
	}
	if items == nil {
		items = []StockItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

type DishUsage struct {
	DishIikoID string  `json:"dish_iiko_id"`
	DishName   string  `json:"dish_name"`
	Amount     float64 `json:"amount"`
	Unit       string  `json:"unit"`      // ingredient unit (л, кг, шт)
	DishUnit   string  `json:"dish_unit"` // per-unit-of-dish (порц. / кг)
}

// UsedInDishes returns the list of dishes whose recipe uses the given stock product.
// Path: GET /api/v1/stock/products/{id}/used-in (where {id} is iiko_product_id, the iiko UUID).
func (h *Handler) UsedInDishes(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	productID := chi.URLParam(r, "id")

	rows, err := h.db.Query(r.Context(),
		`SELECT dish_iiko_id, dish_name, amount, COALESCE(unit, ''), COALESCE(dish_unit, '')
		 FROM recipe_components
		 WHERE company_id = $1 AND ingredient_iiko_id = $2
		 ORDER BY dish_name`,
		companyID, productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch dish usage")
		return
	}
	defer rows.Close()

	out := []DishUsage{}
	for rows.Next() {
		var d DishUsage
		if err := rows.Scan(&d.DishIikoID, &d.DishName, &d.Amount, &d.Unit, &d.DishUnit); err != nil {
			continue
		}
		out = append(out, d)
	}
	writeJSON(w, http.StatusOK, out)
}

// OverrideInput accepts partial updates: any nil field means "leave alone".
// Sending both fields nil deletes the override row entirely (revert to iiko).
type OverrideInput struct {
	ManualAmount       *float64 `json:"manual_amount"`
	ManualPricePerUnit *float64 `json:"manual_price_per_unit"`
}

// SetOverride upserts a manual amount and/or unit price override for a stock product.
// PUT /api/v1/stock/products/{id}/override
//   body: {"manual_amount": 1500, "manual_price_per_unit": 290}
//   either field may be omitted; both null → delete override.
func (h *Handler) SetOverride(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	productID := chi.URLParam(r, "id")

	var input OverrideInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Both nil = delete override (revert to iiko values everywhere).
	if input.ManualAmount == nil && input.ManualPricePerUnit == nil {
		_, err := h.db.Exec(r.Context(),
			`DELETE FROM stock_overrides WHERE company_id = $1 AND iiko_product_id = $2`,
			companyID, productID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to clear override")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
		return
	}

	// Validate non-negative
	if input.ManualAmount != nil && *input.ManualAmount < 0 {
		writeError(w, http.StatusBadRequest, "manual_amount must be >= 0")
		return
	}
	if input.ManualPricePerUnit != nil && *input.ManualPricePerUnit < 0 {
		writeError(w, http.StatusBadRequest, "manual_price_per_unit must be >= 0")
		return
	}

	// Partial UPSERT: only overwrite the provided columns; keep existing values for the others.
	_, err := h.db.Exec(r.Context(),
		`INSERT INTO stock_overrides (company_id, iiko_product_id, manual_amount, manual_price_per_unit, updated_at, source)
		 VALUES ($1, $2, $3, $4, NOW(), 'manual')
		 ON CONFLICT (company_id, iiko_product_id) DO UPDATE SET
		   manual_amount         = COALESCE(EXCLUDED.manual_amount, stock_overrides.manual_amount),
		   manual_price_per_unit = COALESCE(EXCLUDED.manual_price_per_unit, stock_overrides.manual_price_per_unit),
		   updated_at            = NOW(),
		   source                = 'manual'`,
		companyID, productID, input.ManualAmount, input.ManualPricePerUnit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save override")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":                "saved",
		"manual_amount":         input.ManualAmount,
		"manual_price_per_unit": input.ManualPricePerUnit,
	})
}

type SetAliasInput struct {
	DisplayName string `json:"display_name"`
}

// SetProductAlias upserts a user-defined display name for an iiko product.
// Empty display_name removes any existing alias.
func (h *Handler) SetProductAlias(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	productID := chi.URLParam(r, "id")

	var input SetAliasInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(input.DisplayName)
	if name == "" {
		_, err := h.db.Exec(r.Context(),
			`DELETE FROM product_aliases WHERE company_id = $1 AND iiko_product_id = $2`,
			companyID, productID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to remove alias")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
		return
	}
	_, err := h.db.Exec(r.Context(),
		`INSERT INTO product_aliases (company_id, iiko_product_id, display_name)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (company_id, iiko_product_id)
		 DO UPDATE SET display_name = EXCLUDED.display_name, updated_at = NOW()`,
		companyID, productID, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save alias")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"display_name": name})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

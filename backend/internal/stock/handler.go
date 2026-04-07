package stock

import (
	"encoding/json"
	"net/http"
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
	return r
}

type StockItem struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Amount      float64 `json:"amount"`
	Unit        string  `json:"unit"`
	CostSum     float64 `json:"cost_sum"`
	SnapshotAt  string  `json:"snapshot_at"`
}

func (h *Handler) CurrentStock(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locationID := r.URL.Query().Get("location_id")

	query := `SELECT DISTINCT ON (iiko_product_id) iiko_product_id, product_name, amount, COALESCE(unit,''), cost_sum, snapshot_at
		FROM stock_snapshots WHERE company_id = $1`
	args := []interface{}{companyID}

	if locationID != "" {
		query += ` AND location_id = $2`
		args = append(args, locationID)
	}
	query += ` ORDER BY iiko_product_id, snapshot_at DESC`

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
		if err := rows.Scan(&s.ProductID, &s.ProductName, &s.Amount, &s.Unit, &s.CostSum, &t); err != nil {
			continue
		}
		s.SnapshotAt = t.Format(time.RFC3339)
		items = append(items, s)
	}
	if items == nil {
		items = []StockItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) LowStock(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locationID := r.URL.Query().Get("location_id")
	threshold := 5.0

	query := `SELECT DISTINCT ON (iiko_product_id) iiko_product_id, product_name, amount, COALESCE(unit,''), cost_sum, snapshot_at
		FROM stock_snapshots WHERE company_id = $1 AND amount <= $2`
	args := []interface{}{companyID, threshold}
	argIdx := 3

	if locationID != "" {
		query += ` AND location_id = $` + string(rune('0'+argIdx))
		args = append(args, locationID)
	}
	query += ` ORDER BY iiko_product_id, snapshot_at DESC`

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
		if err := rows.Scan(&s.ProductID, &s.ProductName, &s.Amount, &s.Unit, &s.CostSum, &t); err != nil {
			continue
		}
		s.SnapshotAt = t.Format(time.RFC3339)
		items = append(items, s)
	}
	if items == nil {
		items = []StockItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

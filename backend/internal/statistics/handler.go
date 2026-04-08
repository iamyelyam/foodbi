package statistics

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
	r.Get("/revenue", h.RevenueStats)
	r.Get("/profit", h.ProfitStats)
	r.Get("/top-products", h.TopProducts)
	return r
}

type StatPoint struct {
	Date     string  `json:"date"`
	Revenue  float64 `json:"revenue"`
	Cost     float64 `json:"cost"`
	Profit   float64 `json:"profit"`
	Orders   int     `json:"orders"`
}

func (h *Handler) RevenueStats(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locationID := r.URL.Query().Get("location_id")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	if dateFrom == "" {
		dateFrom = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if dateTo == "" {
		dateTo = time.Now().Format("2006-01-02")
	}

	query := `SELECT DATE(order_date), COALESCE(SUM(revenue), 0), COUNT(*)
		FROM revenue_facts WHERE company_id = $1 AND order_date >= $2 AND order_date < ($3::date + 1)`
	args := []interface{}{companyID, dateFrom, dateTo}

	if locationID != "" {
		query += ` AND location_id = $4`
		args = append(args, locationID)
	}
	query += ` GROUP BY DATE(order_date) ORDER BY DATE(order_date)`

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch revenue stats")
		return
	}
	defer rows.Close()

	var points []StatPoint
	for rows.Next() {
		var p StatPoint
		var d time.Time
		if err := rows.Scan(&d, &p.Revenue, &p.Orders); err != nil {
			continue
		}
		p.Date = d.Format("2006-01-02")
		points = append(points, p)
	}
	if points == nil {
		points = []StatPoint{}
	}
	writeJSON(w, http.StatusOK, points)
}

func (h *Handler) ProfitStats(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locationID := r.URL.Query().Get("location_id")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	if dateFrom == "" {
		dateFrom = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if dateTo == "" {
		dateTo = time.Now().Format("2006-01-02")
	}

	// Revenue per day
	revQuery := `SELECT DATE(order_date) as d, COALESCE(SUM(revenue), 0)
		FROM revenue_facts WHERE company_id = $1 AND order_date >= $2 AND order_date < ($3::date + 1)`
	revArgs := []interface{}{companyID, dateFrom, dateTo}
	if locationID != "" {
		revQuery += ` AND location_id = $4`
		revArgs = append(revArgs, locationID)
	}
	revQuery += ` GROUP BY d ORDER BY d`

	// Cost per day
	costQuery := `SELECT DATE(incoming_date) as d, COALESCE(SUM(total_sum), 0)
		FROM purchase_facts WHERE company_id = $1 AND incoming_date >= $2 AND incoming_date < ($3::date + 1)`
	costArgs := []interface{}{companyID, dateFrom, dateTo}
	if locationID != "" {
		costQuery += ` AND location_id = $4`
		costArgs = append(costArgs, locationID)
	}
	costQuery += ` GROUP BY d ORDER BY d`

	revMap := make(map[string]float64)
	rows, err := h.db.Query(r.Context(), revQuery, revArgs...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d time.Time
			var rev float64
			rows.Scan(&d, &rev)
			revMap[d.Format("2006-01-02")] = rev
		}
	}

	costMap := make(map[string]float64)
	rows2, err := h.db.Query(r.Context(), costQuery, costArgs...)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var d time.Time
			var cost float64
			rows2.Scan(&d, &cost)
			costMap[d.Format("2006-01-02")] = cost
		}
	}

	// Merge into points
	allDates := make(map[string]bool)
	for d := range revMap {
		allDates[d] = true
	}
	for d := range costMap {
		allDates[d] = true
	}

	var points []StatPoint
	for d := range allDates {
		rev := revMap[d]
		cost := costMap[d]
		points = append(points, StatPoint{
			Date:    d,
			Revenue: rev,
			Cost:    cost,
			Profit:  rev - cost,
		})
	}
	if points == nil {
		points = []StatPoint{}
	}

	writeJSON(w, http.StatusOK, points)
}

type TopProduct struct {
	ProductName string  `json:"product_name"`
	Category    string  `json:"category"`
	Quantity    float64 `json:"quantity"`
	Revenue     float64 `json:"revenue"`
	Margin      float64 `json:"margin_percent"`
}

func (h *Handler) TopProducts(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locationID := r.URL.Query().Get("location_id")

	query := `SELECT product_name, COALESCE(category, ''), SUM(quantity), SUM(revenue),
		CASE WHEN SUM(revenue) > 0 THEN ((SUM(revenue) - SUM(cost_price)) / SUM(revenue)) * 100 ELSE 0 END
		FROM product_sales_facts WHERE company_id = $1`
	args := []interface{}{companyID}

	if locationID != "" {
		query += ` AND location_id = $2`
		args = append(args, locationID)
	}
	query += ` GROUP BY product_name, category ORDER BY SUM(revenue) DESC LIMIT 20`

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch top products")
		return
	}
	defer rows.Close()

	var products []TopProduct
	for rows.Next() {
		var p TopProduct
		if err := rows.Scan(&p.ProductName, &p.Category, &p.Quantity, &p.Revenue, &p.Margin); err != nil {
			continue
		}
		products = append(products, p)
	}
	if products == nil {
		products = []TopProduct{}
	}
	writeJSON(w, http.StatusOK, products)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

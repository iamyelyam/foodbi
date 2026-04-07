package dashboard

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
	r.Get("/summary", h.Summary)
	r.Get("/revenue-trend", h.RevenueTrend)
	return r
}

type DashboardSummary struct {
	TodayRevenue      float64 `json:"today_revenue"`
	TodayOrders       int     `json:"today_orders"`
	TodayPurchases    float64 `json:"today_purchases"`
	WeekRevenue       float64 `json:"week_revenue"`
	WeekOrders        int     `json:"week_orders"`
	PrevWeekRevenue   float64 `json:"prev_week_revenue"`
	RevenueChangePercent float64 `json:"revenue_change_percent"`
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locationID := r.URL.Query().Get("location_id")

	today := time.Now().Truncate(24 * time.Hour)
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))
	prevWeekStart := weekStart.AddDate(0, 0, -7)

	var summary DashboardSummary

	// Today's revenue
	locFilter := ""
	args := []interface{}{companyID, today}
	if locationID != "" {
		locFilter = " AND location_id = $3"
		args = append(args, locationID)
	}

	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(revenue), 0), COALESCE(COUNT(*), 0)
		 FROM revenue_facts WHERE company_id = $1 AND order_date >= $2`+locFilter, args...).
		Scan(&summary.TodayRevenue, &summary.TodayOrders)

	// Today's purchases
	purchArgs := []interface{}{companyID, today}
	purchLocFilter := ""
	if locationID != "" {
		purchLocFilter = " AND location_id = $3"
		purchArgs = append(purchArgs, locationID)
	}
	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(total_sum), 0)
		 FROM purchase_facts WHERE company_id = $1 AND incoming_date >= $2`+purchLocFilter, purchArgs...).
		Scan(&summary.TodayPurchases)

	// This week revenue
	weekArgs := []interface{}{companyID, weekStart}
	weekLocFilter := ""
	if locationID != "" {
		weekLocFilter = " AND location_id = $3"
		weekArgs = append(weekArgs, locationID)
	}
	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(revenue), 0), COALESCE(COUNT(*), 0)
		 FROM revenue_facts WHERE company_id = $1 AND order_date >= $2`+weekLocFilter, weekArgs...).
		Scan(&summary.WeekRevenue, &summary.WeekOrders)

	// Previous week revenue (for comparison)
	prevArgs := []interface{}{companyID, prevWeekStart, weekStart}
	prevLocFilter := ""
	if locationID != "" {
		prevLocFilter = " AND location_id = $4"
		prevArgs = append(prevArgs, locationID)
	}
	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(revenue), 0)
		 FROM revenue_facts WHERE company_id = $1 AND order_date >= $2 AND order_date < $3`+prevLocFilter, prevArgs...).
		Scan(&summary.PrevWeekRevenue)

	if summary.PrevWeekRevenue > 0 {
		summary.RevenueChangePercent = ((summary.WeekRevenue - summary.PrevWeekRevenue) / summary.PrevWeekRevenue) * 100
	}

	writeJSON(w, http.StatusOK, summary)
}

type TrendPoint struct {
	Date    string  `json:"date"`
	Revenue float64 `json:"revenue"`
	Orders  int     `json:"orders"`
}

func (h *Handler) RevenueTrend(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locationID := r.URL.Query().Get("location_id")
	days := 7

	if r.URL.Query().Get("days") == "30" {
		days = 30
	}

	dateFrom := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	query := `SELECT DATE(order_date) as day, COALESCE(SUM(revenue), 0), COALESCE(COUNT(*), 0)
		FROM revenue_facts WHERE company_id = $1 AND order_date >= $2`
	args := []interface{}{companyID, dateFrom}

	if locationID != "" {
		query += " AND location_id = $3"
		args = append(args, locationID)
	}
	query += " GROUP BY DATE(order_date) ORDER BY day"

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch trend")
		return
	}
	defer rows.Close()

	var points []TrendPoint
	for rows.Next() {
		var p TrendPoint
		var day time.Time
		if err := rows.Scan(&day, &p.Revenue, &p.Orders); err != nil {
			continue
		}
		p.Date = day.Format("2006-01-02")
		points = append(points, p)
	}
	if points == nil {
		points = []TrendPoint{}
	}

	writeJSON(w, http.StatusOK, points)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

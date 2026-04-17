package dashboard

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/foodbi/backend/internal/cache"
	"github.com/foodbi/backend/internal/middleware"
	"github.com/foodbi/backend/internal/timezone"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// cacheTTL controls how long dashboard responses are cached.
// 2 minutes balances freshness with load reduction — sync runs every 15 min
// so data is anyway up to 15 min stale.
const cacheTTL = 2 * time.Minute

type Handler struct {
	db    *pgxpool.Pool
	cache *cache.Cache
}

func NewHandler(db *pgxpool.Pool, c *cache.Cache) *Handler {
	return &Handler{db: db, cache: c}
}

// cacheKeySummary builds a deterministic cache key for dashboard/summary.
func cacheKeySummary(companyID string, locIDs []uuid.UUID, dateFrom, dateTo string) string {
	parts := make([]string, 0, len(locIDs))
	for _, id := range locIDs {
		parts = append(parts, id.String())
	}
	return "dash:sum:" + companyID + ":" + strings.Join(parts, ",") + ":" + dateFrom + ":" + dateTo
}

// cacheKeyTrend builds a cache key for revenue-trend.
func cacheKeyTrend(companyID string, locIDs []uuid.UUID, days int) string {
	parts := make([]string, 0, len(locIDs))
	for _, id := range locIDs {
		parts = append(parts, id.String())
	}
	return "dash:trend:" + companyID + ":" + strings.Join(parts, ",") + ":" + strconv.Itoa(days)
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/summary", h.Summary)
	r.Get("/revenue-trend", h.RevenueTrend)
	return r
}

type SupplierSummary struct {
	SupplierName string  `json:"supplier_name"`
	TotalSum     float64 `json:"total_sum"`
}

type DashboardSummary struct {
	TodayRevenue         float64           `json:"today_revenue"`
	TodayOrders          int               `json:"today_orders"`
	TodayPurchases       float64           `json:"today_purchases"`
	TodayPurchaseCount   int               `json:"today_purchase_count"`
	WeekRevenue          float64           `json:"week_revenue"`
	WeekOrders           int               `json:"week_orders"`
	WeekPurchases        float64           `json:"week_purchases"`
	WeekPurchaseCount    int               `json:"week_purchase_count"`
	TodayProfit          float64           `json:"today_profit"`
	WeekProfit           float64           `json:"week_profit"`
	PrevWeekRevenue      float64           `json:"prev_week_revenue"`
	RevenueChangePercent float64           `json:"revenue_change_percent"`
	TopSuppliers         []SupplierSummary `json:"top_suppliers"`
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locIDs := middleware.ParseLocationFilter(r)
	dateFromParam := r.URL.Query().Get("date_from")
	dateToParam := r.URL.Query().Get("date_to")

	// Cache lookup — all query inputs baked into the key.
	sumKey := cacheKeySummary(companyID.String(), locIDs, dateFromParam, dateToParam)
	if h.cache != nil {
		if cached, ok := h.cache.Get(sumKey); ok {
			if summary, ok := cached.(DashboardSummary); ok {
				writeJSON(w, http.StatusOK, summary)
				return
			}
		}
	}

	// Use Almaty timezone (UTC+5) so "today" matches the restaurant's local day,
	// not UTC midnight (which would cut off orders before 05:00 Almaty).
	now := timezone.Now().Truncate(24 * time.Hour)
	rangeStart := now
	rangeEnd := now.AddDate(0, 0, 1) // exclusive end for >= start AND < end
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	prevWeekStart := weekStart.AddDate(0, 0, -7)

	customRange := false
	if dateFromParam != "" && dateToParam != "" {
		if df, err := time.Parse("2006-01-02", dateFromParam); err == nil {
			rangeStart = df
		}
		if dt, err := time.Parse("2006-01-02", dateToParam); err == nil {
			rangeEnd = dt.AddDate(0, 0, 1)
		}
		weekStart = rangeStart
		prevWeekStart = rangeStart.AddDate(0, 0, -(int(rangeEnd.Sub(rangeStart).Hours()/24)))
		customRange = true
	}
	_ = customRange

	var summary DashboardSummary

	// Revenue for selected range (or today)
	args := []interface{}{companyID, rangeStart, rangeEnd}
	locFilter, args := middleware.AddLocationFilter(args, locIDs)
	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(revenue), 0), COALESCE(COUNT(*), 0)
		 FROM revenue_facts WHERE company_id = $1 AND order_date >= $2 AND order_date < $3`+locFilter, args...).
		Scan(&summary.TodayRevenue, &summary.TodayOrders)

	// Purchases for selected range
	purchArgs := []interface{}{companyID, rangeStart, rangeEnd}
	purchLocFilter, purchArgs := middleware.AddLocationFilter(purchArgs, locIDs)
	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(total_sum), 0), COALESCE(COUNT(*), 0)
		 FROM purchase_facts WHERE company_id = $1 AND incoming_date >= $2 AND incoming_date < $3`+purchLocFilter, purchArgs...).
		Scan(&summary.TodayPurchases, &summary.TodayPurchaseCount)

	// This week purchases
	weekPurchArgs := []interface{}{companyID, weekStart}
	weekPurchLocFilter, weekPurchArgs := middleware.AddLocationFilter(weekPurchArgs, locIDs)
	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(total_sum), 0), COALESCE(COUNT(*), 0)
		 FROM purchase_facts WHERE company_id = $1 AND incoming_date >= $2`+weekPurchLocFilter, weekPurchArgs...).
		Scan(&summary.WeekPurchases, &summary.WeekPurchaseCount)

	// Top suppliers
	topArgs := []interface{}{companyID, weekStart}
	topLocFilter, topArgs := middleware.AddLocationFilter(topArgs, locIDs)
	supRows, supErr := h.db.Query(r.Context(),
		`SELECT COALESCE(supplier_name, 'Unknown'), SUM(total_sum)
		 FROM purchase_facts WHERE company_id = $1 AND incoming_date >= $2`+topLocFilter+`
		 GROUP BY supplier_name ORDER BY SUM(total_sum) DESC LIMIT 5`, topArgs...)
	if supErr == nil {
		defer supRows.Close()
		for supRows.Next() {
			var s SupplierSummary
			supRows.Scan(&s.SupplierName, &s.TotalSum)
			summary.TopSuppliers = append(summary.TopSuppliers, s)
		}
	}
	if summary.TopSuppliers == nil {
		summary.TopSuppliers = []SupplierSummary{}
	}

	// This week revenue
	weekArgs := []interface{}{companyID, weekStart}
	weekLocFilter, weekArgs := middleware.AddLocationFilter(weekArgs, locIDs)
	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(revenue), 0), COALESCE(COUNT(*), 0)
		 FROM revenue_facts WHERE company_id = $1 AND order_date >= $2`+weekLocFilter, weekArgs...).
		Scan(&summary.WeekRevenue, &summary.WeekOrders)

	// Previous week revenue (for comparison)
	prevArgs := []interface{}{companyID, prevWeekStart, weekStart}
	prevLocFilter, prevArgs := middleware.AddLocationFilter(prevArgs, locIDs)
	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(revenue), 0)
		 FROM revenue_facts WHERE company_id = $1 AND order_date >= $2 AND order_date < $3`+prevLocFilter, prevArgs...).
		Scan(&summary.PrevWeekRevenue)

	// Period profit = SUM(revenue) - SUM(cost_price of products sold) for selected range.
	// Uses product_sales_facts (per-dish cost_price from iiko) — represents true COGS,
	// not purchase invoices (which arrive irregularly).
	cogsArgs := []interface{}{companyID, rangeStart, rangeEnd}
	cogsLocFilter, cogsArgs := middleware.AddLocationFilter(cogsArgs, locIDs)
	var periodCOGS float64
	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(cost_price), 0)
		 FROM product_sales_facts
		 WHERE company_id = $1 AND sale_date >= $2 AND sale_date < $3`+cogsLocFilter, cogsArgs...).
		Scan(&periodCOGS)
	summary.TodayProfit = summary.TodayRevenue - periodCOGS

	// Week profit = week revenue - week COGS
	weekCogsArgs := []interface{}{companyID, weekStart}
	weekCogsLocFilter, weekCogsArgs := middleware.AddLocationFilter(weekCogsArgs, locIDs)
	var weekCOGS float64
	h.db.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(cost_price), 0)
		 FROM product_sales_facts
		 WHERE company_id = $1 AND sale_date >= $2`+weekCogsLocFilter, weekCogsArgs...).
		Scan(&weekCOGS)
	summary.WeekProfit = summary.WeekRevenue - weekCOGS

	if summary.PrevWeekRevenue > 0 {
		summary.RevenueChangePercent = ((summary.WeekRevenue - summary.PrevWeekRevenue) / summary.PrevWeekRevenue) * 100
	}

	if h.cache != nil {
		h.cache.Set(sumKey, summary, cacheTTL)
	}
	writeJSON(w, http.StatusOK, summary)
}

type TrendPoint struct {
	Date    string  `json:"date"`
	Revenue float64 `json:"revenue"`
	Orders  int     `json:"orders"`
	Items   int     `json:"items"`
}

func (h *Handler) RevenueTrend(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locIDs := middleware.ParseLocationFilter(r)
	days := 7
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	tkey := cacheKeyTrend(companyID.String(), locIDs, days)
	if h.cache != nil {
		if cached, ok := h.cache.Get(tkey); ok {
			if pts, ok := cached.([]TrendPoint); ok {
				writeJSON(w, http.StatusOK, pts)
				return
			}
		}
	}

	dateFrom := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	args := []interface{}{companyID, dateFrom}
	trendLocFilter, args := middleware.AddLocationFilter(args, locIDs)
	query := `SELECT DATE(order_date) as day, COALESCE(SUM(revenue), 0), COALESCE(COUNT(*), 0), COALESCE(SUM(item_count), 0)
		FROM revenue_facts WHERE company_id = $1 AND order_date >= $2` + trendLocFilter + ` GROUP BY DATE(order_date) ORDER BY day`

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
		if err := rows.Scan(&day, &p.Revenue, &p.Orders, &p.Items); err != nil {
			continue
		}
		p.Date = day.Format("2006-01-02")
		points = append(points, p)
	}
	if points == nil {
		points = []TrendPoint{}
	}

	if h.cache != nil {
		h.cache.Set(tkey, points, cacheTTL)
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

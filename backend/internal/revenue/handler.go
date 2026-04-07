package revenue

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
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
	r.Get("/orders", h.ListOrders)
	r.Get("/orders/{id}", h.GetOrder)
	r.Get("/products", h.ListProducts)
	r.Get("/products/{id}", h.GetProductDetails)
	return r
}

type Order struct {
	ID        string  `json:"id"`
	OrderDate string  `json:"order_date"`
	Revenue   float64 `json:"revenue"`
	Discount  float64 `json:"discount"`
	OrderType string  `json:"order_type"`
	Status    string  `json:"status"`
	ItemCount int     `json:"item_count"`
	Waiter    string  `json:"waiter_name"`
}

type OrdersResponse struct {
	Orders     []Order `json:"orders"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PerPage    int     `json:"per_page"`
	TotalPages int     `json:"total_pages"`
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locationID := r.URL.Query().Get("location_id")
	status := r.URL.Query().Get("status")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 20
	offset := (page - 1) * perPage

	query := `SELECT id, order_date, revenue, discount, order_type, status, item_count, COALESCE(waiter_name, '')
		FROM revenue_facts WHERE company_id = $1`
	countQuery := `SELECT COUNT(*) FROM revenue_facts WHERE company_id = $1`
	args := []interface{}{companyID}
	argIdx := 2

	if locationID != "" {
		query += ` AND location_id = $` + strconv.Itoa(argIdx)
		countQuery += ` AND location_id = $` + strconv.Itoa(argIdx)
		args = append(args, locationID)
		argIdx++
	}
	if status != "" {
		query += ` AND status = $` + strconv.Itoa(argIdx)
		countQuery += ` AND status = $` + strconv.Itoa(argIdx)
		args = append(args, status)
		argIdx++
	}
	if dateFrom != "" {
		query += ` AND order_date >= $` + strconv.Itoa(argIdx)
		countQuery += ` AND order_date >= $` + strconv.Itoa(argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		query += ` AND order_date <= $` + strconv.Itoa(argIdx)
		countQuery += ` AND order_date <= $` + strconv.Itoa(argIdx)
		args = append(args, dateTo)
		argIdx++
	}

	var total int
	h.db.QueryRow(r.Context(), countQuery, args...).Scan(&total)

	query += ` ORDER BY order_date DESC LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, perPage, offset)

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch orders")
		return
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var orderDate time.Time
		if err := rows.Scan(&o.ID, &orderDate, &o.Revenue, &o.Discount, &o.OrderType, &o.Status, &o.ItemCount, &o.Waiter); err != nil {
			continue
		}
		o.OrderDate = orderDate.Format(time.RFC3339)
		orders = append(orders, o)
	}
	if orders == nil {
		orders = []Order{}
	}

	writeJSON(w, http.StatusOK, OrdersResponse{
		Orders:     orders,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	})
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	orderID := chi.URLParam(r, "id")

	var o Order
	var orderDate time.Time
	err := h.db.QueryRow(r.Context(),
		`SELECT id, order_date, revenue, discount, order_type, status, item_count, COALESCE(waiter_name, '')
		 FROM revenue_facts WHERE id = $1 AND company_id = $2`,
		orderID, companyID).Scan(&o.ID, &orderDate, &o.Revenue, &o.Discount, &o.OrderType, &o.Status, &o.ItemCount, &o.Waiter)
	if err != nil {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	o.OrderDate = orderDate.Format(time.RFC3339)
	writeJSON(w, http.StatusOK, o)
}

type ProductSummary struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Category    string  `json:"category"`
	TotalQty    float64 `json:"total_quantity"`
	TotalRev    float64 `json:"total_revenue"`
	TotalCost   float64 `json:"total_cost"`
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locationID := r.URL.Query().Get("location_id")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	query := `SELECT iiko_product_id, product_name, COALESCE(category, ''),
		SUM(quantity), SUM(revenue), SUM(cost_price)
		FROM product_sales_facts WHERE company_id = $1`
	args := []interface{}{companyID}
	argIdx := 2

	if locationID != "" {
		query += ` AND location_id = $` + strconv.Itoa(argIdx)
		args = append(args, locationID)
		argIdx++
	}
	if dateFrom != "" {
		query += ` AND sale_date >= $` + strconv.Itoa(argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		query += ` AND sale_date <= $` + strconv.Itoa(argIdx)
		args = append(args, dateTo)
		argIdx++
	}

	query += ` GROUP BY iiko_product_id, product_name, category ORDER BY SUM(revenue) DESC LIMIT 50`

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch products")
		return
	}
	defer rows.Close()

	var products []ProductSummary
	for rows.Next() {
		var p ProductSummary
		if err := rows.Scan(&p.ProductID, &p.ProductName, &p.Category, &p.TotalQty, &p.TotalRev, &p.TotalCost); err != nil {
			continue
		}
		products = append(products, p)
	}
	if products == nil {
		products = []ProductSummary{}
	}

	writeJSON(w, http.StatusOK, products)
}

func (h *Handler) GetProductDetails(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	productID := chi.URLParam(r, "id")

	rows, err := h.db.Query(r.Context(),
		`SELECT sale_date, SUM(quantity), SUM(revenue), SUM(cost_price)
		 FROM product_sales_facts
		 WHERE company_id = $1 AND iiko_product_id = $2
		 GROUP BY sale_date ORDER BY sale_date DESC LIMIT 30`,
		companyID, productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch product details")
		return
	}
	defer rows.Close()

	type DaySales struct {
		Date     string  `json:"date"`
		Quantity float64 `json:"quantity"`
		Revenue  float64 `json:"revenue"`
		Cost     float64 `json:"cost"`
	}

	var sales []DaySales
	for rows.Next() {
		var s DaySales
		var d time.Time
		if err := rows.Scan(&d, &s.Quantity, &s.Revenue, &s.Cost); err != nil {
			continue
		}
		s.Date = d.Format("2006-01-02")
		sales = append(sales, s)
	}
	if sales == nil {
		sales = []DaySales{}
	}
	writeJSON(w, http.StatusOK, sales)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

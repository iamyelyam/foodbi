package revenue

import (
	"encoding/json"
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
	r.Post("/orders/{id}/status", h.UpdateOrderStatus)
	r.Get("/products", h.ListProducts)
	r.Get("/products/{id}", h.GetProductDetails)
	r.Get("/products/{id}/trend", h.GetProductTrend)
	r.Get("/products/{id}/orders", h.GetProductOrders)
	return r
}

type Order struct {
	ID          string  `json:"id"`
	OrderNumber string  `json:"order_number"`
	OrderDate   string  `json:"order_date"`
	Revenue     float64 `json:"revenue"`
	Discount    float64 `json:"discount"`
	OrderType   string  `json:"order_type"`
	Status      string  `json:"status"`
	ItemCount   int     `json:"item_count"`
	Waiter      string  `json:"waiter_name"`
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
	locIDs := middleware.ParseLocationFilter(r)
	status := r.URL.Query().Get("status")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	// No pagination — always return all orders for the selected period.
	_ = page

	query := `SELECT id, COALESCE(order_number, ''), order_date, revenue, discount, order_type, status, item_count, COALESCE(waiter_name, '')
		FROM revenue_facts WHERE company_id = $1`
	countQuery := `SELECT COUNT(*) FROM revenue_facts WHERE company_id = $1`
	args := []interface{}{companyID}

	locFilter, args := middleware.AddLocationFilter(args, locIDs)
	query += locFilter
	countQuery += locFilter
	argIdx := len(args) + 1

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
		query += ` AND order_date < ($` + strconv.Itoa(argIdx) + `::date + 1)`
		countQuery += ` AND order_date < ($` + strconv.Itoa(argIdx) + `::date + 1)`
		args = append(args, dateTo)
		argIdx++
	}

	var total int
	h.db.QueryRow(r.Context(), countQuery, args...).Scan(&total)

	query += ` ORDER BY order_date DESC`

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
		if err := rows.Scan(&o.ID, &o.OrderNumber, &orderDate, &o.Revenue, &o.Discount, &o.OrderType, &o.Status, &o.ItemCount, &o.Waiter); err != nil {
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
		Page:       1,
		PerPage:    total,
		TotalPages: 1,
	})
}

type OrderItem struct {
	ProductName string  `json:"product_name"`
	Quantity    float64 `json:"quantity"`
	Revenue     float64 `json:"revenue"`
	CostPrice   float64 `json:"cost_price"`
}

type OrderDetail struct {
	Order
	TotalCost float64     `json:"total_cost"`
	Profit    float64     `json:"profit"`
	Items     []OrderItem `json:"items"`
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	orderID := chi.URLParam(r, "id")

	var detail OrderDetail
	var orderDate time.Time
	err := h.db.QueryRow(r.Context(),
		`SELECT id, COALESCE(order_number, ''), order_date, revenue, discount, order_type, status, item_count, COALESCE(waiter_name, '')
		 FROM revenue_facts WHERE id = $1 AND company_id = $2`,
		orderID, companyID).Scan(&detail.ID, &detail.OrderNumber, &orderDate, &detail.Revenue, &detail.Discount, &detail.OrderType, &detail.Status, &detail.ItemCount, &detail.Waiter)
	if err != nil {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	detail.OrderDate = orderDate.Format(time.RFC3339)

	// Fetch line items aggregated per product for this order
	// order_id in product_sales_facts is the iiko UniqOrderId.Id, which equals revenue_facts.iiko_order_id
	rows, err := h.db.Query(r.Context(),
		`SELECT psf.product_name, SUM(psf.quantity), SUM(psf.revenue), SUM(psf.cost_price)
		 FROM product_sales_facts psf
		 JOIN revenue_facts rf ON rf.iiko_order_id = psf.order_id AND rf.company_id = psf.company_id
		 WHERE rf.id = $1 AND rf.company_id = $2
		 GROUP BY psf.product_name
		 ORDER BY SUM(psf.revenue) DESC`,
		orderID, companyID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var it OrderItem
			if err := rows.Scan(&it.ProductName, &it.Quantity, &it.Revenue, &it.CostPrice); err != nil {
				continue
			}
			detail.Items = append(detail.Items, it)
			detail.TotalCost += it.CostPrice
		}
	}
	if detail.Items == nil {
		detail.Items = []OrderItem{}
	}
	detail.Profit = detail.Revenue - detail.TotalCost

	writeJSON(w, http.StatusOK, detail)
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
	locIDs := middleware.ParseLocationFilter(r)
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	query := `SELECT iiko_product_id, product_name, COALESCE(category, ''),
		SUM(quantity), SUM(revenue), SUM(cost_price)
		FROM product_sales_facts WHERE company_id = $1`
	args := []interface{}{companyID}

	locFilter, args := middleware.AddLocationFilter(args, locIDs)
	query += locFilter
	argIdx := len(args) + 1

	if dateFrom != "" {
		query += ` AND sale_date >= $` + strconv.Itoa(argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		query += ` AND sale_date < ($` + strconv.Itoa(argIdx) + `::date + 1)`
		args = append(args, dateTo)
		argIdx++
	}

	query += ` GROUP BY iiko_product_id, product_name, category ORDER BY SUM(revenue) DESC`

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

func (h *Handler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only owners can change order status")
		return
	}

	companyID := middleware.GetCompanyID(r.Context())
	orderID := chi.URLParam(r, "id")

	var input struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Status != "approved" && input.Status != "rejected" {
		writeError(w, http.StatusBadRequest, "status must be 'approved' or 'rejected'")
		return
	}

	tag, err := h.db.Exec(r.Context(),
		`UPDATE revenue_facts SET status = $1 WHERE id = $2 AND company_id = $3`,
		input.Status, orderID, companyID)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": input.Status})
}

func (h *Handler) GetProductTrend(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	productID := chi.URLParam(r, "id")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	query := `SELECT sale_date, SUM(quantity), SUM(revenue), COUNT(DISTINCT order_id)
		FROM product_sales_facts
		WHERE company_id = $1 AND iiko_product_id = $2`
	args := []interface{}{companyID, productID}
	argIdx := 3
	if dateFrom != "" {
		query += ` AND sale_date >= $` + strconv.Itoa(argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		query += ` AND sale_date < ($` + strconv.Itoa(argIdx) + `::date + 1)`
		args = append(args, dateTo)
		argIdx++
	}
	// If no date range supplied, default to last 30 days
	if dateFrom == "" && dateTo == "" {
		query += ` AND sale_date >= NOW() - INTERVAL '30 days'`
	}
	query += ` GROUP BY sale_date ORDER BY sale_date`

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch trend")
		return
	}
	defer rows.Close()

	type TrendPoint struct {
		Date         string  `json:"date"`
		Revenue      float64 `json:"revenue"`
		Quantity     float64 `json:"quantity"`
		Transactions int     `json:"transactions"`
	}

	var points []TrendPoint
	for rows.Next() {
		var p TrendPoint
		var d time.Time
		if err := rows.Scan(&d, &p.Quantity, &p.Revenue, &p.Transactions); err != nil {
			continue
		}
		p.Date = d.Format("2006-01-02")
		points = append(points, p)
	}
	if points == nil {
		points = []TrendPoint{}
	}
	writeJSON(w, http.StatusOK, points)
}

func (h *Handler) GetProductOrders(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	productID := chi.URLParam(r, "id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 10
	}

	rows, err := h.db.Query(r.Context(),
		`SELECT DISTINCT rf.id, rf.order_date, psf.revenue, psf.quantity
		 FROM product_sales_facts psf
		 JOIN revenue_facts rf ON rf.id = psf.order_id AND rf.company_id = psf.company_id
		 WHERE psf.company_id = $1 AND psf.iiko_product_id = $2
		 ORDER BY rf.order_date DESC LIMIT $3`,
		companyID, productID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch orders")
		return
	}
	defer rows.Close()

	type OrderRef struct {
		OrderID   string  `json:"order_id"`
		OrderDate string  `json:"order_date"`
		Revenue   float64 `json:"revenue"`
		Quantity  float64 `json:"quantity"`
	}

	var orders []OrderRef
	for rows.Next() {
		var o OrderRef
		var d time.Time
		if err := rows.Scan(&o.OrderID, &d, &o.Revenue, &o.Quantity); err != nil {
			continue
		}
		o.OrderDate = d.Format(time.RFC3339)
		orders = append(orders, o)
	}
	if orders == nil {
		orders = []OrderRef{}
	}
	writeJSON(w, http.StatusOK, orders)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

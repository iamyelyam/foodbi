package purchases

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
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
	r.Get("/", h.ListPurchases)
	r.Get("/{id}", h.GetPurchase)
	r.Get("/suppliers", h.ListSuppliers)
	r.Get("/suppliers/{id}", h.GetSupplier)
	r.Put("/suppliers/{id}/alias", h.SetSupplierAlias)
	return r
}

type Purchase struct {
	ID             string  `json:"id"`
	DocumentNumber string  `json:"document_number"`
	SupplierID     string  `json:"supplier_id"`
	SupplierName   string  `json:"supplier_name"`
	IncomingDate   string  `json:"incoming_date"`
	Status         string  `json:"status"`
	TotalSum       float64 `json:"total_sum"`
}

type PurchasesResponse struct {
	Purchases  []Purchase `json:"purchases"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PerPage    int        `json:"per_page"`
	TotalPages int        `json:"total_pages"`
}

func (h *Handler) ListPurchases(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locationID := r.URL.Query().Get("location_id")
	supplierID := r.URL.Query().Get("supplier_id")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	// Pagination: only applied when caller explicitly passes ?page=N.
	// Without ?page, return all rows in the date filter window (front-end relies
	// on this — it shows the full list within the picked date range, no paging).
	pageParam := r.URL.Query().Get("page")
	page := 0
	perPage := 0
	offset := 0
	paginated := pageParam != ""
	if paginated {
		page, _ = strconv.Atoi(pageParam)
		if page < 1 {
			page = 1
		}
		perPage = 20
		offset = (page - 1) * perPage
	}

	query := `SELECT pf.id, COALESCE(pf.document_number,''), COALESCE(pf.supplier_id,''),
		       COALESCE(NULLIF(sa.display_name, ''), pf.supplier_name, ''),
		       pf.incoming_date, COALESCE(pf.status,''), pf.total_sum
		FROM purchase_facts pf
		LEFT JOIN supplier_aliases sa ON sa.company_id = pf.company_id AND sa.iiko_supplier_id = pf.supplier_id
		WHERE pf.company_id = $1`
	countQuery := `SELECT COUNT(*) FROM purchase_facts WHERE company_id = $1`
	args := []interface{}{companyID}
	argIdx := 2

	if locationID != "" {
		query += ` AND pf.location_id = $` + strconv.Itoa(argIdx)
		countQuery += ` AND location_id = $` + strconv.Itoa(argIdx)
		args = append(args, locationID)
		argIdx++
	}
	if supplierID != "" {
		query += ` AND pf.supplier_id = $` + strconv.Itoa(argIdx)
		countQuery += ` AND supplier_id = $` + strconv.Itoa(argIdx)
		args = append(args, supplierID)
		argIdx++
	}
	if dateFrom != "" {
		query += ` AND pf.incoming_date >= $` + strconv.Itoa(argIdx)
		countQuery += ` AND incoming_date >= $` + strconv.Itoa(argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		query += ` AND pf.incoming_date < ($` + strconv.Itoa(argIdx) + `::date + 1)`
		countQuery += ` AND incoming_date < ($` + strconv.Itoa(argIdx) + `::date + 1)`
		args = append(args, dateTo)
		argIdx++
	}

	var total int
	h.db.QueryRow(r.Context(), countQuery, args...).Scan(&total)

	query += ` ORDER BY pf.incoming_date DESC`
	if paginated {
		query += ` LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
		args = append(args, perPage, offset)
	}

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch purchases")
		return
	}
	defer rows.Close()

	var purchases []Purchase
	for rows.Next() {
		var p Purchase
		var d time.Time
		if err := rows.Scan(&p.ID, &p.DocumentNumber, &p.SupplierID, &p.SupplierName, &d, &p.Status, &p.TotalSum); err != nil {
			continue
		}
		p.IncomingDate = d.Format(time.RFC3339)
		purchases = append(purchases, p)
	}
	if purchases == nil {
		purchases = []Purchase{}
	}

	respPerPage := perPage
	respPage := page
	respTotalPages := 1
	if paginated && perPage > 0 {
		respTotalPages = int(math.Ceil(float64(total) / float64(perPage)))
	} else {
		// Single-page response when no ?page=N requested.
		respPerPage = total
		respPage = 1
	}
	writeJSON(w, http.StatusOK, PurchasesResponse{
		Purchases:  purchases,
		Total:      total,
		Page:       respPage,
		PerPage:    respPerPage,
		TotalPages: respTotalPages,
	})
}

type PurchaseLineItem struct {
	ProductName string  `json:"product_name"`
	ProductCode string  `json:"product_code"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price"`
	Subtotal    float64 `json:"subtotal"`
}

type PurchaseDetail struct {
	Purchase
	LineItems []PurchaseLineItem `json:"line_items"`
}

func (h *Handler) GetPurchase(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id := chi.URLParam(r, "id")

	var detail PurchaseDetail
	var d time.Time
	err := h.db.QueryRow(r.Context(),
		`SELECT pf.id, COALESCE(pf.document_number,''), COALESCE(pf.supplier_id,''),
		        COALESCE(NULLIF(sa.display_name, ''), pf.supplier_name, ''),
		        pf.incoming_date, COALESCE(pf.status,''), pf.total_sum
		 FROM purchase_facts pf
		 LEFT JOIN supplier_aliases sa ON sa.company_id = pf.company_id AND sa.iiko_supplier_id = pf.supplier_id
		 WHERE pf.id = $1 AND pf.company_id = $2`,
		id, companyID).Scan(&detail.ID, &detail.DocumentNumber, &detail.SupplierID, &detail.SupplierName, &d, &detail.Status, &detail.TotalSum)
	if err != nil {
		writeError(w, http.StatusNotFound, "purchase not found")
		return
	}
	detail.IncomingDate = d.Format(time.RFC3339)

	// Fetch line items linked to this purchase
	rows, err := h.db.Query(r.Context(),
		`SELECT COALESCE(product_name,''), COALESCE(product_code,''), COALESCE(unit,''), quantity, price, subtotal
		 FROM purchase_line_items WHERE purchase_id = $1 ORDER BY created_at`,
		id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var li PurchaseLineItem
			if err := rows.Scan(&li.ProductName, &li.ProductCode, &li.Unit, &li.Quantity, &li.Price, &li.Subtotal); err != nil {
				continue
			}
			detail.LineItems = append(detail.LineItems, li)
		}
	}
	if detail.LineItems == nil {
		detail.LineItems = []PurchaseLineItem{}
	}

	writeJSON(w, http.StatusOK, detail)
}

type Supplier struct {
	SupplierID   string  `json:"supplier_id"`
	SupplierName string  `json:"supplier_name"`
	TotalSum     float64 `json:"total_sum"`
	InvoiceCount int     `json:"invoice_count"`
	LastInvoice  string  `json:"last_invoice"`
}

func (h *Handler) ListSuppliers(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())

	rows, err := h.db.Query(r.Context(),
		`SELECT pf.supplier_id,
		        COALESCE(NULLIF(MAX(sa.display_name), ''), MAX(pf.supplier_name), ''),
		        SUM(pf.total_sum), COUNT(*), MAX(pf.incoming_date)
		 FROM purchase_facts pf
		 LEFT JOIN supplier_aliases sa ON sa.company_id = pf.company_id AND sa.iiko_supplier_id = pf.supplier_id
		 WHERE pf.company_id = $1 AND pf.supplier_id IS NOT NULL
		 GROUP BY pf.supplier_id ORDER BY SUM(pf.total_sum) DESC`,
		companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch suppliers")
		return
	}
	defer rows.Close()

	var suppliers []Supplier
	for rows.Next() {
		var s Supplier
		var lastDate time.Time
		if err := rows.Scan(&s.SupplierID, &s.SupplierName, &s.TotalSum, &s.InvoiceCount, &lastDate); err != nil {
			continue
		}
		s.LastInvoice = lastDate.Format("2006-01-02")
		suppliers = append(suppliers, s)
	}
	if suppliers == nil {
		suppliers = []Supplier{}
	}
	writeJSON(w, http.StatusOK, suppliers)
}

type SetAliasInput struct {
	DisplayName string `json:"display_name"`
}

// SetSupplierAlias upserts a user-defined display name for an iiko supplier.
func (h *Handler) SetSupplierAlias(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	supplierID := chi.URLParam(r, "id")

	var input SetAliasInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(input.DisplayName)
	if name == "" {
		// Empty name = remove alias
		_, err := h.db.Exec(r.Context(),
			`DELETE FROM supplier_aliases WHERE company_id = $1 AND iiko_supplier_id = $2`,
			companyID, supplierID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to remove alias")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
		return
	}
	_, err := h.db.Exec(r.Context(),
		`INSERT INTO supplier_aliases (company_id, iiko_supplier_id, display_name)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (company_id, iiko_supplier_id)
		 DO UPDATE SET display_name = EXCLUDED.display_name, updated_at = NOW()`,
		companyID, supplierID, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save alias")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"display_name": name})
}

func (h *Handler) GetSupplier(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	supplierID := chi.URLParam(r, "id")

	type SupplierDetail struct {
		Supplier
		Purchases []Purchase `json:"purchases"`
	}

	var detail SupplierDetail
	var lastDate time.Time
	err := h.db.QueryRow(r.Context(),
		`SELECT supplier_id, supplier_name, SUM(total_sum), COUNT(*), MAX(incoming_date)
		 FROM purchase_facts WHERE company_id = $1 AND supplier_id = $2
		 GROUP BY supplier_id, supplier_name`,
		companyID, supplierID).Scan(&detail.SupplierID, &detail.SupplierName, &detail.TotalSum, &detail.InvoiceCount, &lastDate)
	if err != nil {
		writeError(w, http.StatusNotFound, "supplier not found")
		return
	}
	detail.LastInvoice = lastDate.Format("2006-01-02")

	rows, err := h.db.Query(r.Context(),
		`SELECT id, COALESCE(document_number,''), COALESCE(supplier_name,''), incoming_date, COALESCE(status,''), total_sum
		 FROM purchase_facts WHERE company_id = $1 AND supplier_id = $2
		 ORDER BY incoming_date DESC LIMIT 20`,
		companyID, supplierID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p Purchase
			var d time.Time
			if err := rows.Scan(&p.ID, &p.DocumentNumber, &p.SupplierName, &d, &p.Status, &p.TotalSum); err != nil {
				continue
			}
			p.IncomingDate = d.Format(time.RFC3339)
			detail.Purchases = append(detail.Purchases, p)
		}
	}
	if detail.Purchases == nil {
		detail.Purchases = []Purchase{}
	}
	writeJSON(w, http.StatusOK, detail)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

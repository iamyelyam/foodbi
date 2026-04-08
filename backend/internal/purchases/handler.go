package purchases

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
	r.Get("/", h.ListPurchases)
	r.Get("/{id}", h.GetPurchase)
	r.Get("/suppliers", h.ListSuppliers)
	r.Get("/suppliers/{id}", h.GetSupplier)
	return r
}

type Purchase struct {
	ID             string  `json:"id"`
	DocumentNumber string  `json:"document_number"`
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
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 20
	offset := (page - 1) * perPage

	query := `SELECT id, COALESCE(document_number,''), COALESCE(supplier_name,''), incoming_date, COALESCE(status,''), total_sum
		FROM purchase_facts WHERE company_id = $1`
	countQuery := `SELECT COUNT(*) FROM purchase_facts WHERE company_id = $1`
	args := []interface{}{companyID}
	argIdx := 2

	if locationID != "" {
		f := ` AND location_id = $` + strconv.Itoa(argIdx)
		query += f
		countQuery += f
		args = append(args, locationID)
		argIdx++
	}
	if supplierID != "" {
		f := ` AND supplier_id = $` + strconv.Itoa(argIdx)
		query += f
		countQuery += f
		args = append(args, supplierID)
		argIdx++
	}
	if dateFrom != "" {
		f := ` AND incoming_date >= $` + strconv.Itoa(argIdx)
		query += f
		countQuery += f
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		f := ` AND incoming_date < ($` + strconv.Itoa(argIdx) + `::date + 1)`
		query += f
		countQuery += f
		args = append(args, dateTo)
		argIdx++
	}

	var total int
	h.db.QueryRow(r.Context(), countQuery, args...).Scan(&total)

	query += ` ORDER BY incoming_date DESC LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, perPage, offset)

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
		if err := rows.Scan(&p.ID, &p.DocumentNumber, &p.SupplierName, &d, &p.Status, &p.TotalSum); err != nil {
			continue
		}
		p.IncomingDate = d.Format(time.RFC3339)
		purchases = append(purchases, p)
	}
	if purchases == nil {
		purchases = []Purchase{}
	}

	writeJSON(w, http.StatusOK, PurchasesResponse{
		Purchases:  purchases,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	})
}

func (h *Handler) GetPurchase(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id := chi.URLParam(r, "id")

	var p Purchase
	var d time.Time
	err := h.db.QueryRow(r.Context(),
		`SELECT id, COALESCE(document_number,''), COALESCE(supplier_name,''), incoming_date, COALESCE(status,''), total_sum
		 FROM purchase_facts WHERE id = $1 AND company_id = $2`,
		id, companyID).Scan(&p.ID, &p.DocumentNumber, &p.SupplierName, &d, &p.Status, &p.TotalSum)
	if err != nil {
		writeError(w, http.StatusNotFound, "purchase not found")
		return
	}
	p.IncomingDate = d.Format(time.RFC3339)
	writeJSON(w, http.StatusOK, p)
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
		`SELECT supplier_id, supplier_name, SUM(total_sum), COUNT(*), MAX(incoming_date)
		 FROM purchase_facts WHERE company_id = $1 AND supplier_id IS NOT NULL
		 GROUP BY supplier_id, supplier_name ORDER BY SUM(total_sum) DESC`,
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

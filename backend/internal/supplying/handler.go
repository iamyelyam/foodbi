package supplying

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/foodbi/backend/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var validate = validator.New()

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListRequests)
	r.Post("/", h.CreateRequest)
	r.Get("/{id}", h.GetRequest)
	r.Post("/{id}/approve", h.ApproveRequest)
	r.Post("/{id}/reject", h.RejectRequest)
	return r
}

type SupplyRequest struct {
	ID           string        `json:"id"`
	CompanyID    string        `json:"company_id"`
	LocationID   string        `json:"location_id"`
	SupplierName string        `json:"supplier_name"`
	Status       string        `json:"status"`
	TotalSum     float64       `json:"total_sum"`
	CreatedBy    string        `json:"created_by"`
	CreatedAt    string        `json:"created_at"`
	Items        []RequestItem `json:"items,omitempty"`
}

type RequestItem struct {
	ProductName string  `json:"product_name"`
	Category    string  `json:"category"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	PricePerUnit float64 `json:"price_per_unit"`
}

type CreateInput struct {
	LocationID   string        `json:"location_id" validate:"required"`
	SupplierName string        `json:"supplier_name" validate:"required"`
	Items        []RequestItem `json:"items" validate:"required,min=1"`
}

func (h *Handler) ListRequests(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	status := r.URL.Query().Get("status")

	query := `SELECT id, company_id, location_id, supplier_name, status, total_sum, created_by, created_at
		FROM supply_requests WHERE company_id = $1`
	args := []interface{}{companyID}

	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch requests")
		return
	}
	defer rows.Close()

	var requests []SupplyRequest
	for rows.Next() {
		var sr SupplyRequest
		var t time.Time
		if err := rows.Scan(&sr.ID, &sr.CompanyID, &sr.LocationID, &sr.SupplierName, &sr.Status, &sr.TotalSum, &sr.CreatedBy, &t); err != nil {
			continue
		}
		sr.CreatedAt = t.Format(time.RFC3339)
		requests = append(requests, sr)
	}
	if requests == nil {
		requests = []SupplyRequest{}
	}
	writeJSON(w, http.StatusOK, requests)
}

func (h *Handler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Struct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation failed: "+err.Error())
		return
	}

	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id := uuid.New()

	var totalSum float64
	for _, item := range input.Items {
		totalSum += item.Quantity * item.PricePerUnit
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(),
		`INSERT INTO supply_requests (id, company_id, location_id, supplier_name, status, total_sum, created_by, created_at)
		 VALUES ($1, $2, $3, $4, 'pending', $5, $6, NOW())`,
		id, companyID, input.LocationID, input.SupplierName, totalSum, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create request")
		return
	}

	for i, item := range input.Items {
		_, err = tx.Exec(r.Context(),
			`INSERT INTO supply_request_items (id, request_id, product_name, category, quantity, unit, price_per_unit, sort_order)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			uuid.New(), id, item.ProductName, item.Category, item.Quantity, item.Unit, item.PricePerUnit, i)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to add item")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id, "status": "pending", "total_sum": totalSum})
}

func (h *Handler) GetRequest(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id := chi.URLParam(r, "id")

	var sr SupplyRequest
	var t time.Time
	err := h.db.QueryRow(r.Context(),
		`SELECT id, company_id, location_id, supplier_name, status, total_sum, created_by, created_at
		 FROM supply_requests WHERE id = $1 AND company_id = $2`, id, companyID).
		Scan(&sr.ID, &sr.CompanyID, &sr.LocationID, &sr.SupplierName, &sr.Status, &sr.TotalSum, &sr.CreatedBy, &t)
	if err != nil {
		writeError(w, http.StatusNotFound, "request not found")
		return
	}
	sr.CreatedAt = t.Format(time.RFC3339)

	rows, err := h.db.Query(r.Context(),
		`SELECT product_name, COALESCE(category,''), quantity, COALESCE(unit,''), price_per_unit
		 FROM supply_request_items WHERE request_id = $1 ORDER BY sort_order`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item RequestItem
			rows.Scan(&item.ProductName, &item.Category, &item.Quantity, &item.Unit, &item.PricePerUnit)
			sr.Items = append(sr.Items, item)
		}
	}
	if sr.Items == nil {
		sr.Items = []RequestItem{}
	}
	writeJSON(w, http.StatusOK, sr)
}

func (h *Handler) ApproveRequest(w http.ResponseWriter, r *http.Request) {
	h.updateStatus(w, r, "approved")
}

func (h *Handler) RejectRequest(w http.ResponseWriter, r *http.Request) {
	h.updateStatus(w, r, "rejected")
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request, status string) {
	role := middleware.GetRole(r.Context())
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only owners can "+status[:len(status)-1]+" requests")
		return
	}

	companyID := middleware.GetCompanyID(r.Context())
	id := chi.URLParam(r, "id")

	tag, err := h.db.Exec(r.Context(),
		`UPDATE supply_requests SET status = $1 WHERE id = $2 AND company_id = $3 AND status = 'pending'`,
		status, id, companyID)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "request not found or already processed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

// unused but keeps the import valid
var _ = strconv.Itoa

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

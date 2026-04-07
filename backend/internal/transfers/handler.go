package transfers

import (
	"encoding/json"
	"net/http"
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
	r.Get("/", h.ListTransfers)
	r.Post("/", h.CreateTransfer)
	r.Get("/{id}", h.GetTransfer)
	return r
}

type Transfer struct {
	ID             string         `json:"id"`
	CompanyID      string         `json:"company_id"`
	FromLocationID string         `json:"from_location_id"`
	ToLocationID   string         `json:"to_location_id"`
	Status         string         `json:"status"`
	CreatedBy      string         `json:"created_by"`
	CreatedAt      string         `json:"created_at"`
	Items          []TransferItem `json:"items,omitempty"`
}

type TransferItem struct {
	ProductName string  `json:"product_name"`
	Category    string  `json:"category"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
}

type CreateInput struct {
	FromLocationID string         `json:"from_location_id" validate:"required"`
	ToLocationID   string         `json:"to_location_id" validate:"required"`
	Items          []TransferItem `json:"items" validate:"required,min=1"`
}

func (h *Handler) ListTransfers(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locationID := r.URL.Query().Get("location_id")
	status := r.URL.Query().Get("status")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	query := `SELECT id, company_id, from_location_id, to_location_id, status, created_by, created_at
		FROM transfer_requests WHERE company_id = $1`
	args := []interface{}{companyID}
	argIdx := 2

	if locationID != "" {
		query += ` AND (from_location_id = $` + itoa(argIdx) + ` OR to_location_id = $` + itoa(argIdx) + `)`
		args = append(args, locationID)
		argIdx++
	}
	if status != "" {
		query += ` AND status = $` + itoa(argIdx)
		args = append(args, status)
		argIdx++
	}
	if dateFrom != "" {
		query += ` AND created_at >= $` + itoa(argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		query += ` AND created_at <= $` + itoa(argIdx)
		args = append(args, dateTo)
		argIdx++
	}
	query += ` ORDER BY created_at DESC`

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch transfers")
		return
	}
	defer rows.Close()

	var transfers []Transfer
	for rows.Next() {
		var t Transfer
		var createdAt time.Time
		if err := rows.Scan(&t.ID, &t.CompanyID, &t.FromLocationID, &t.ToLocationID, &t.Status, &t.CreatedBy, &createdAt); err != nil {
			continue
		}
		t.CreatedAt = createdAt.Format(time.RFC3339)
		transfers = append(transfers, t)
	}
	if transfers == nil {
		transfers = []Transfer{}
	}
	writeJSON(w, http.StatusOK, transfers)
}

func (h *Handler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Struct(input); err != nil {
		writeError(w, http.StatusBadRequest, "validation failed: "+err.Error())
		return
	}
	if input.FromLocationID == input.ToLocationID {
		writeError(w, http.StatusBadRequest, "source and destination must be different")
		return
	}

	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())
	id := uuid.New()

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(),
		`INSERT INTO transfer_requests (id, company_id, from_location_id, to_location_id, status, created_by, created_at)
		 VALUES ($1, $2, $3, $4, 'pending', $5, NOW())`,
		id, companyID, input.FromLocationID, input.ToLocationID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create transfer")
		return
	}

	for i, item := range input.Items {
		_, err = tx.Exec(r.Context(),
			`INSERT INTO transfer_request_items (id, request_id, product_name, category, quantity, unit, sort_order)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			uuid.New(), id, item.ProductName, item.Category, item.Quantity, item.Unit, i)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to add item")
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id, "status": "pending"})
}

func (h *Handler) GetTransfer(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id := chi.URLParam(r, "id")

	var t Transfer
	var createdAt time.Time
	err := h.db.QueryRow(r.Context(),
		`SELECT id, company_id, from_location_id, to_location_id, status, created_by, created_at
		 FROM transfer_requests WHERE id = $1 AND company_id = $2`, id, companyID).
		Scan(&t.ID, &t.CompanyID, &t.FromLocationID, &t.ToLocationID, &t.Status, &t.CreatedBy, &createdAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "transfer not found")
		return
	}
	t.CreatedAt = createdAt.Format(time.RFC3339)

	rows, err := h.db.Query(r.Context(),
		`SELECT product_name, COALESCE(category,''), quantity, COALESCE(unit,'')
		 FROM transfer_request_items WHERE request_id = $1 ORDER BY sort_order`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item TransferItem
			rows.Scan(&item.ProductName, &item.Category, &item.Quantity, &item.Unit)
			t.Items = append(t.Items, item)
		}
	}
	if t.Items == nil {
		t.Items = []TransferItem{}
	}
	writeJSON(w, http.StatusOK, t)
}

func itoa(i int) string {
	return string(rune('0' + i))
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

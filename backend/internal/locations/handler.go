package locations

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
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/sync-status", h.SyncStatus)
	r.Post("/{id}/sync", h.TriggerSync)
	return r
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())

	rows, err := h.db.Query(r.Context(),
		`SELECT id, company_id, name, address, iiko_org_id, created_at FROM locations WHERE company_id = $1 ORDER BY name`,
		companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch locations")
		return
	}
	defer rows.Close()

	type Location struct {
		ID        uuid.UUID `json:"id"`
		CompanyID uuid.UUID `json:"company_id"`
		Name      string    `json:"name"`
		Address   string    `json:"address"`
		IikoOrgID *string   `json:"iiko_org_id,omitempty"`
		CreatedAt string    `json:"created_at"`
	}

	var locations []Location
	for rows.Next() {
		var loc Location
		var iikoOrgID *string
		var createdAt time.Time
		if err := rows.Scan(&loc.ID, &loc.CompanyID, &loc.Name, &loc.Address, &iikoOrgID, &createdAt); err != nil {
			continue
		}
		loc.IikoOrgID = iikoOrgID
		loc.CreatedAt = createdAt.Format(time.RFC3339)
		locations = append(locations, loc)
	}

	if locations == nil {
		locations = []Location{}
	}
	writeJSON(w, http.StatusOK, locations)
}

type CreateInput struct {
	Name      string `json:"name" validate:"required"`
	Address   string `json:"address"`
	IikoOrgID string `json:"iiko_org_id"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only owners can add locations")
		return
	}

	var input CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Struct(input); err != nil {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	companyID := middleware.GetCompanyID(r.Context())
	id := uuid.New()

	_, err := h.db.Exec(r.Context(),
		`INSERT INTO locations (id, company_id, name, address, iiko_org_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`,
		id, companyID, input.Name, input.Address, nilIfEmpty(input.IikoOrgID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create location")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":   id,
		"name": input.Name,
	})
}

func (h *Handler) SyncStatus(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())

	rows, err := h.db.Query(r.Context(),
		`SELECT DISTINCT ON (location_id, sync_type)
		   location_id, sync_type, status, records_synced, started_at, completed_at, error_message
		 FROM iiko_sync_log
		 WHERE company_id = $1
		 ORDER BY location_id, sync_type, started_at DESC`,
		companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch sync status")
		return
	}
	defer rows.Close()

	type SyncEntry struct {
		LocationID   *uuid.UUID `json:"location_id"`
		SyncType     string     `json:"sync_type"`
		Status       string     `json:"status"`
		RecordsSynced int       `json:"records_synced"`
		StartedAt    string     `json:"started_at"`
		CompletedAt  *string    `json:"completed_at"`
		Error        *string    `json:"error,omitempty"`
	}

	var entries []SyncEntry
	for rows.Next() {
		var e SyncEntry
		if err := rows.Scan(&e.LocationID, &e.SyncType, &e.Status, &e.RecordsSynced, &e.StartedAt, &e.CompletedAt, &e.Error); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	if entries == nil {
		entries = []SyncEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (h *Handler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id := chi.URLParam(r, "id")

	// Verify location belongs to company
	var exists bool
	h.db.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM locations WHERE id = $1 AND company_id = $2)`,
		id, companyID).Scan(&exists)
	if !exists {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	locUUID, err := uuid.Parse(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location id")
		return
	}

	// Queue sync for all types
	for _, syncType := range []string{"revenue", "product_sales", "purchases", "stock"} {
		h.db.Exec(r.Context(),
			`INSERT INTO iiko_sync_log (id, company_id, location_id, sync_type, status, started_at)
			 VALUES ($1, $2, $3, $4, 'queued', NOW())`,
			uuid.New(), companyID, locUUID, syncType)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sync_queued", "location_id": id})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

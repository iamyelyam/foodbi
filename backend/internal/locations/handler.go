package locations

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/foodbi/backend/internal/iiko"
	"github.com/foodbi/backend/internal/middleware"
	gosync "github.com/foodbi/backend/internal/sync"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var validate = validator.New()

type Handler struct {
	db          *pgxpool.Pool
	syncService *gosync.Service
}

func NewHandler(db *pgxpool.Pool, syncService *gosync.Service) *Handler {
	return &Handler{db: db, syncService: syncService}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/iiko-config", h.GetIikoConfig)
	r.Put("/iiko-config", h.SetIikoConfig)
	r.Get("/sync-status", h.SyncStatus)
	r.Post("/{id}/sync", h.TriggerSync)
	return r
}

type IikoConfigInput struct {
	ServerURL string `json:"iiko_server_url" validate:"required"`
	Login     string `json:"iiko_login" validate:"required"`
	Password  string `json:"iiko_password" validate:"required"`
}

// GetIikoConfig returns the current iiko credentials (password masked).
// GET /api/v1/locations/iiko-config
func (h *Handler) GetIikoConfig(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())

	var serverURL, login, password *string
	err := h.db.QueryRow(r.Context(),
		`SELECT iiko_server_url, iiko_login, iiko_password FROM companies WHERE id = $1`, companyID).
		Scan(&serverURL, &login, &password)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{})
		return
	}

	result := map[string]string{}
	if serverURL != nil {
		result["iiko_server_url"] = *serverURL
	}
	if login != nil {
		result["iiko_login"] = *login
	}
	if password != nil && len(*password) > 0 {
		result["iiko_password_set"] = "true"
	}
	writeJSON(w, http.StatusOK, result)
}

// SetIikoConfig saves iiko Server API credentials on the company record.
// PUT /api/v1/locations/iiko-config
func (h *Handler) SetIikoConfig(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only owners can configure iiko")
		return
	}

	var input IikoConfigInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.ServerURL == "" || input.Login == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "all iiko fields are required")
		return
	}

	companyID := middleware.GetCompanyID(r.Context())
	_, err := h.db.Exec(r.Context(),
		`UPDATE companies SET iiko_server_url = $1, iiko_login = $2, iiko_password = $3, updated_at = NOW() WHERE id = $4`,
		input.ServerURL, input.Login, input.Password, companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save iiko config")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
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
		var address *string
		var iikoOrgID *string
		var createdAt time.Time
		if err := rows.Scan(&loc.ID, &loc.CompanyID, &loc.Name, &address, &iikoOrgID, &createdAt); err != nil {
			continue
		}
		if address != nil {
			loc.Address = *address
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
	City      string `json:"city"`
	Address   string `json:"address"`
	PosSystem string `json:"pos_system"`
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
		`INSERT INTO locations (id, company_id, name, city, address, pos_system, iiko_org_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
		id, companyID, input.Name, nilIfEmpty(input.City), input.Address, nilIfEmpty(input.PosSystem), nilIfEmpty(input.IikoOrgID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create location")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":   id,
		"name": input.Name,
	})
}

type UpdateInput struct {
	Name    string `json:"name" validate:"required"`
	Address string `json:"address"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only owners can edit locations")
		return
	}

	id := chi.URLParam(r, "id")
	companyID := middleware.GetCompanyID(r.Context())

	var input UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Struct(input); err != nil {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	tag, err := h.db.Exec(r.Context(),
		`UPDATE locations SET name = $1, address = $2, updated_at = NOW() WHERE id = $3 AND company_id = $4`,
		input.Name, input.Address, id, companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update location")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only owners can delete locations")
		return
	}

	id := chi.URLParam(r, "id")
	companyID := middleware.GetCompanyID(r.Context())

	tag, err := h.db.Exec(r.Context(),
		`DELETE FROM locations WHERE id = $1 AND company_id = $2`,
		id, companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete location")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) SyncStatus(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())

	rows, err := h.db.Query(r.Context(),
		`SELECT DISTINCT ON (location_id, sync_type)
		   location_id, sync_type, status, records_synced, started_at, completed_at, error_message
		 FROM iiko_sync_log
		 WHERE company_id = $1 AND status IN ('success', 'failed')
		 ORDER BY location_id, sync_type, started_at DESC`,
		companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch sync status")
		return
	}
	defer rows.Close()

	type SyncEntry struct {
		LocationID    *uuid.UUID `json:"location_id"`
		SyncType      string     `json:"sync_type"`
		Status        string     `json:"status"`
		RecordsSynced int        `json:"records_synced"`
		StartedAt     string     `json:"started_at"`
		CompletedAt   *string    `json:"completed_at"`
		Error         *string    `json:"error,omitempty"`
	}

	var entries []SyncEntry
	for rows.Next() {
		var e SyncEntry
		var startedAt time.Time
		var completedAt *time.Time
		if err := rows.Scan(&e.LocationID, &e.SyncType, &e.Status, &e.RecordsSynced, &startedAt, &completedAt, &e.Error); err != nil {
			continue
		}
		e.StartedAt = startedAt.Format(time.RFC3339)
		if completedAt != nil {
			s := completedAt.Format(time.RFC3339)
			e.CompletedAt = &s
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

	// Verify the placeholder location belongs to company
	var exists bool
	h.db.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM locations WHERE id = $1 AND company_id = $2)`,
		id, companyID).Scan(&exists)
	if !exists {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	placeholderUUID, err := uuid.Parse(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location id")
		return
	}

	// Get iiko credentials from company
	var iikoURL, iikoLogin, iikoPassword string
	err = h.db.QueryRow(r.Context(),
		`SELECT iiko_server_url, iiko_login, iiko_password FROM companies
		 WHERE id = $1 AND iiko_server_url IS NOT NULL AND iiko_server_url != ''`,
		companyID).Scan(&iikoURL, &iikoLogin, &iikoPassword)
	if err != nil {
		writeError(w, http.StatusBadRequest, "iiko not configured for this company")
		return
	}

	// Run sync in background goroutine
	go h.runMultiLocationSync(companyID, placeholderUUID, iikoURL, iikoLogin, iikoPassword)

	writeJSON(w, http.StatusOK, map[string]string{"status": "sync_started", "location_id": id})
}

// runMultiLocationSync discovers departments from iiko, creates a location per department,
// removes the placeholder, and syncs each location with its iiko_org_id.
func (h *Handler) runMultiLocationSync(companyID, placeholderID uuid.UUID, iikoURL, iikoLogin, iikoPassword string) {
	ctx := context.Background()
	logger := log.With().Str("company", companyID.String()).Logger()

	client := iiko.NewClient(iikoURL, iikoLogin, iikoPassword)
	if err := client.Authenticate(ctx); err != nil {
		logger.Error().Err(err).Msg("trigger-sync: iiko auth failed")
		return
	}

	// Fetch departments (restaurants) from iiko
	depts, err := client.GetDepartments(ctx)
	if err != nil || len(depts) == 0 {
		logger.Warn().Err(err).Msg("trigger-sync: no departments found, syncing placeholder as single location")
		h.runSingleLocationSync(ctx, client, companyID, placeholderID, "", logger)
		return
	}

	logger.Info().Int("departments", len(depts)).Msg("trigger-sync: discovered iiko departments")

	// Rename placeholder to first department, create new locations for the rest
	locationIDs := make([]uuid.UUID, len(depts))

	for i, dept := range depts {
		if i == 0 {
			// Reuse placeholder location for first department
			_, _ = h.db.Exec(ctx,
				`UPDATE locations SET name = $1, iiko_org_id = $2, updated_at = NOW() WHERE id = $3`,
				dept.Name, dept.ID, placeholderID)
			locationIDs[0] = placeholderID
		} else {
			// Create new location for each additional department
			newID := uuid.New()
			_, err := h.db.Exec(ctx,
				`INSERT INTO locations (id, company_id, name, address, iiko_org_id, pos_system, created_at, updated_at)
				 VALUES ($1, $2, $3, '', $4, 'iiko', NOW(), NOW())`,
				newID, companyID, dept.Name, dept.ID)
			if err != nil {
				logger.Error().Err(err).Str("dept", dept.Name).Msg("trigger-sync: failed to create location")
				continue
			}
			locationIDs[i] = newID
		}
		logger.Info().Str("dept", dept.Name).Str("iiko_org_id", dept.ID).Msg("trigger-sync: location ready")
	}

	// Sync each location
	for i, dept := range depts {
		if locationIDs[i] == uuid.Nil {
			continue
		}
		h.runSingleLocationSync(ctx, client, companyID, locationIDs[i], dept.ID, logger)
	}

	if err := h.syncService.RefreshDashboardViews(ctx); err != nil {
		logger.Warn().Err(err).Msg("trigger-sync: dashboard refresh failed")
	}

	logger.Info().Msg("trigger-sync: all locations complete")
}

func (h *Handler) runSingleLocationSync(ctx context.Context, client *iiko.Client, companyID, locationID uuid.UUID, iikoOrgID string, logger zerolog.Logger) {
	l := logger.With().Str("location", locationID.String()).Logger()

	if err := h.syncService.SyncRevenue(ctx, client, companyID, locationID, iikoOrgID); err != nil {
		l.Error().Err(err).Msg("trigger-sync: revenue failed")
	} else {
		h.syncService.ValidateRevenueAfterSync(ctx, companyID, locationID)
	}

	if err := h.syncService.SyncProductSales(ctx, client, companyID, locationID, iikoOrgID); err != nil {
		l.Error().Err(err).Msg("trigger-sync: product_sales failed")
	}

	if err := h.syncService.SyncPurchases(ctx, client, companyID, locationID, iikoOrgID); err != nil {
		l.Error().Err(err).Msg("trigger-sync: purchases failed")
	}

	if err := h.syncService.SyncStock(ctx, client, companyID, locationID, iikoOrgID); err != nil {
		l.Error().Err(err).Msg("trigger-sync: stock failed")
	}

	if err := h.syncService.SyncRecipes(ctx, client, companyID, locationID, iikoOrgID); err != nil {
		l.Error().Err(err).Msg("trigger-sync: recipes failed")
	}

	l.Info().Msg("trigger-sync: location sync complete")
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

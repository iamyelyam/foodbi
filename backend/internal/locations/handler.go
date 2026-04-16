package locations

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/foodbi/backend/internal/iiko"
	"github.com/foodbi/backend/internal/middleware"
	"github.com/foodbi/backend/internal/numier"
	"github.com/foodbi/backend/internal/numiersync"
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
	db               *pgxpool.Pool
	syncService      *gosync.Service
	numierSyncService *numiersync.Service
}

func NewHandler(db *pgxpool.Pool, syncService *gosync.Service, numierSyncService *numiersync.Service) *Handler {
	return &Handler{db: db, syncService: syncService, numierSyncService: numierSyncService}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/iiko-config", h.GetIikoConfig)
	r.Put("/iiko-config", h.SetIikoConfig)
	r.Get("/numier-config", h.GetNumierConfig)
	r.Put("/numier-config", h.SetNumierConfig)
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

// GetNumierConfig returns the current NUMIER API key (masked).
// GET /api/v1/locations/numier-config
func (h *Handler) GetNumierConfig(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())

	var apiKey *string
	err := h.db.QueryRow(r.Context(),
		`SELECT numier_api_key FROM companies WHERE id = $1`, companyID).
		Scan(&apiKey)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{})
		return
	}

	result := map[string]string{}
	if apiKey != nil && len(*apiKey) > 0 {
		result["numier_api_key_set"] = "true"
	}
	writeJSON(w, http.StatusOK, result)
}

// SetNumierConfig saves NUMIER API key on the company record.
// PUT /api/v1/locations/numier-config
func (h *Handler) SetNumierConfig(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only owners can configure numier")
		return
	}

	var input struct {
		APIKey string `json:"numier_api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.APIKey == "" {
		writeError(w, http.StatusBadRequest, "numier_api_key is required")
		return
	}

	// Validate the API key by calling NUMIER
	client := numier.NewClient(input.APIKey)
	if err := client.Validate(r.Context()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid NUMIER API key: "+err.Error())
		return
	}

	companyID := middleware.GetCompanyID(r.Context())
	_, err := h.db.Exec(r.Context(),
		`UPDATE companies SET numier_api_key = $1, updated_at = NOW() WHERE id = $2`,
		input.APIKey, companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save numier config")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())

	rows, err := h.db.Query(r.Context(),
		`SELECT id, company_id, name, address, iiko_org_id, pos_system, created_at FROM locations WHERE company_id = $1 ORDER BY name`,
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
		PosSystem string    `json:"pos_system"`
		CreatedAt string    `json:"created_at"`
	}

	var locations []Location
	for rows.Next() {
		var loc Location
		var address *string
		var iikoOrgID *string
		var createdAt time.Time
		if err := rows.Scan(&loc.ID, &loc.CompanyID, &loc.Name, &address, &iikoOrgID, &loc.PosSystem, &createdAt); err != nil {
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

	// Verify the location belongs to company and get its POS system
	var posSystem *string
	err := h.db.QueryRow(r.Context(),
		`SELECT pos_system FROM locations WHERE id = $1 AND company_id = $2`,
		id, companyID).Scan(&posSystem)
	if err != nil {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	locationUUID, err := uuid.Parse(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid location id")
		return
	}

	// Route to the correct POS sync based on location's pos_system
	if posSystem != nil && *posSystem == "numier" {
		// NUMIER sync
		var apiKey string
		err = h.db.QueryRow(r.Context(),
			`SELECT numier_api_key FROM companies
			 WHERE id = $1 AND numier_api_key IS NOT NULL AND numier_api_key != ''`,
			companyID).Scan(&apiKey)
		if err != nil {
			writeError(w, http.StatusBadRequest, "numier not configured for this company")
			return
		}
		go h.runNumierLocationSync(companyID, locationUUID, apiKey)
	} else {
		// iiko sync (default)
		var iikoURL, iikoLogin, iikoPassword string
		err = h.db.QueryRow(r.Context(),
			`SELECT iiko_server_url, iiko_login, iiko_password FROM companies
			 WHERE id = $1 AND iiko_server_url IS NOT NULL AND iiko_server_url != ''`,
			companyID).Scan(&iikoURL, &iikoLogin, &iikoPassword)
		if err != nil {
			writeError(w, http.StatusBadRequest, "iiko not configured for this company")
			return
		}
		go h.runMultiLocationSync(companyID, locationUUID, iikoURL, iikoLogin, iikoPassword)
	}

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

	// Find or create a location for each department. Never overwrite existing locations.
	locationIDs := make([]uuid.UUID, len(depts))
	placeholderUsed := false

	for i, dept := range depts {
		// Check if a location with this iiko_org_id already exists
		var existingID uuid.UUID
		err := h.db.QueryRow(ctx,
			`SELECT id FROM locations WHERE company_id = $1 AND iiko_org_id = $2`,
			companyID, dept.ID).Scan(&existingID)
		if err == nil {
			// Location already exists — just sync it
			locationIDs[i] = existingID
			logger.Info().Str("dept", dept.Name).Msg("trigger-sync: existing location found")
			continue
		}

		if !placeholderUsed {
			// Reuse placeholder for the first new department
			_, _ = h.db.Exec(ctx,
				`UPDATE locations SET name = $1, iiko_org_id = $2, pos_system = 'iiko', updated_at = NOW() WHERE id = $3`,
				dept.Name, dept.ID, placeholderID)
			locationIDs[i] = placeholderID
			placeholderUsed = true
		} else {
			// Create new location for additional departments
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

// runNumierLocationSync runs a full NUMIER sync for a single location.
func (h *Handler) runNumierLocationSync(companyID, locationID uuid.UUID, apiKey string) {
	ctx := context.Background()
	logger := log.With().Str("company", companyID.String()).Str("pos", "numier").Logger()

	client := numier.NewClient(apiKey)

	// Get TPV ID for this location
	var tpvID string
	h.db.QueryRow(ctx,
		`SELECT COALESCE(numier_tpv_id, '') FROM locations WHERE id = $1`,
		locationID).Scan(&tpvID)

	if tpvID == "" {
		// Try to discover and map locales
		locations, err := h.numierSyncService.DiscoverAndMapLocales(ctx, client, companyID)
		if err != nil {
			logger.Error().Err(err).Msg("numier-trigger: discover locales failed")
			return
		}
		for _, loc := range locations {
			if loc.LocationID == locationID {
				tpvID = loc.NumierTpvID
				break
			}
		}
		if tpvID == "" {
			logger.Error().Str("location", locationID.String()).Msg("numier-trigger: no TPV ID found for location")
			return
		}
	}

	l := logger.With().Str("location", locationID.String()).Str("tpv_id", tpvID).Logger()

	if err := h.numierSyncService.SyncRevenue(ctx, client, companyID, locationID, tpvID); err != nil {
		l.Error().Err(err).Msg("numier-trigger: revenue failed")
	}

	if err := h.numierSyncService.SyncProductSales(ctx, client, companyID, locationID, tpvID); err != nil {
		l.Error().Err(err).Msg("numier-trigger: product_sales failed")
	}

	if err := h.numierSyncService.SyncPurchases(ctx, client, companyID, locationID, tpvID); err != nil {
		l.Error().Err(err).Msg("numier-trigger: purchases failed")
	}

	if err := h.numierSyncService.SyncCalculatedStock(ctx, companyID, locationID); err != nil {
		l.Error().Err(err).Msg("numier-trigger: calculated stock failed")
	}

	if err := h.numierSyncService.SyncRecipes(ctx, client, companyID, locationID, tpvID); err != nil {
		l.Error().Err(err).Msg("numier-trigger: recipes failed")
	}

	if err := h.numierSyncService.RefreshDashboardViews(ctx); err != nil {
		l.Warn().Err(err).Msg("numier-trigger: dashboard refresh failed")
	}

	l.Info().Msg("numier-trigger: location sync complete")
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

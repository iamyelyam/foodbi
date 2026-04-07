package employees

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/foodbi/backend/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
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
	r.Get("/{id}", h.Get)
	r.Put("/{id}/role", h.UpdateRole)
	r.Put("/{id}/locations", h.AssignLocations)
	return r
}

type Employee struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Phone     string   `json:"phone"`
	Role      string   `json:"role"`
	IsActive  bool     `json:"is_active"`
	CreatedAt string   `json:"created_at"`
	Locations []string `json:"locations,omitempty"`
}

type CreateInput struct {
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Phone     string `json:"phone"`
	Role      string `json:"role" validate:"required,oneof=owner employee"`
	Password  string `json:"password" validate:"required,min=8"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())

	rows, err := h.db.Query(r.Context(),
		`SELECT id, email, first_name, last_name, COALESCE(phone,''), role, is_active, created_at
		 FROM users WHERE company_id = $1 ORDER BY created_at DESC`, companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch employees")
		return
	}
	defer rows.Close()

	var employees []Employee
	for rows.Next() {
		var e Employee
		var t time.Time
		if err := rows.Scan(&e.ID, &e.Email, &e.FirstName, &e.LastName, &e.Phone, &e.Role, &e.IsActive, &t); err != nil {
			continue
		}
		e.CreatedAt = t.Format(time.RFC3339)
		employees = append(employees, e)
	}
	if employees == nil {
		employees = []Employee{}
	}
	writeJSON(w, http.StatusOK, employees)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only owners can add employees")
		return
	}

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

	var exists bool
	h.db.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", input.Email).Scan(&exists)
	if exists {
		writeError(w, http.StatusConflict, "user with this email already exists")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	id := uuid.New()
	_, err = h.db.Exec(r.Context(),
		`INSERT INTO users (id, company_id, email, password_hash, first_name, last_name, phone, role, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, NOW(), NOW())`,
		id, companyID, input.Email, string(hash), input.FirstName, input.LastName, input.Phone, input.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create employee")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id, "email": input.Email})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id := chi.URLParam(r, "id")

	var e Employee
	var t time.Time
	err := h.db.QueryRow(r.Context(),
		`SELECT id, email, first_name, last_name, COALESCE(phone,''), role, is_active, created_at
		 FROM users WHERE id = $1 AND company_id = $2`, id, companyID).
		Scan(&e.ID, &e.Email, &e.FirstName, &e.LastName, &e.Phone, &e.Role, &e.IsActive, &t)
	if err != nil {
		writeError(w, http.StatusNotFound, "employee not found")
		return
	}
	e.CreatedAt = t.Format(time.RFC3339)

	rows, err := h.db.Query(r.Context(),
		`SELECT l.name FROM user_locations ul JOIN locations l ON l.id = ul.location_id WHERE ul.user_id = $1`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			rows.Scan(&name)
			e.Locations = append(e.Locations, name)
		}
	}
	if e.Locations == nil {
		e.Locations = []string{}
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only owners can change roles")
		return
	}

	companyID := middleware.GetCompanyID(r.Context())
	id := chi.URLParam(r, "id")

	var input struct {
		Role string `json:"role" validate:"required,oneof=owner employee"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tag, err := h.db.Exec(r.Context(),
		`UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2 AND company_id = $3`,
		input.Role, id, companyID)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "employee not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"role": input.Role})
}

func (h *Handler) AssignLocations(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only owners can assign locations")
		return
	}

	companyID := middleware.GetCompanyID(r.Context())
	id := chi.URLParam(r, "id")

	var input struct {
		LocationIDs []string `json:"location_ids" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "transaction failed")
		return
	}
	defer tx.Rollback(r.Context())

	tx.Exec(r.Context(), "DELETE FROM user_locations WHERE user_id = $1", id)

	for _, locID := range input.LocationIDs {
		var valid bool
		tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM locations WHERE id = $1 AND company_id = $2)", locID, companyID).Scan(&valid)
		if valid {
			tx.Exec(r.Context(), "INSERT INTO user_locations (user_id, location_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", id, locID)
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "commit failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "locations assigned"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

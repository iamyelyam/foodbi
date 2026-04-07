package profiles

import (
	"encoding/json"
	"net/http"

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
	r.Get("/me", h.GetProfile)
	r.Put("/me", h.UpdateProfile)
	return r
}

type Profile struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Role      string `json:"role"`
	Company   string `json:"company_name"`
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	companyID := middleware.GetCompanyID(r.Context())

	var p Profile
	err := h.db.QueryRow(r.Context(),
		`SELECT u.id, u.email, u.first_name, u.last_name, COALESCE(u.phone,''), u.role, c.name
		 FROM users u JOIN companies c ON c.id = u.company_id
		 WHERE u.id = $1 AND u.company_id = $2`, userID, companyID).
		Scan(&p.ID, &p.Email, &p.FirstName, &p.LastName, &p.Phone, &p.Role, &p.Company)
	if err != nil {
		writeError(w, http.StatusNotFound, "profile not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Phone     string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tag, err := h.db.Exec(r.Context(),
		`UPDATE users SET first_name = COALESCE(NULLIF($1,''), first_name),
		 last_name = COALESCE(NULLIF($2,''), last_name),
		 phone = $3, updated_at = NOW() WHERE id = $4`,
		input.FirstName, input.LastName, input.Phone, userID)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

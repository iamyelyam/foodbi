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
	r.Put("/company-settings", h.UpdateCompanySettings)
	return r
}

type Profile struct {
	ID             string          `json:"id"`
	Email          string          `json:"email"`
	FirstName      string          `json:"first_name"`
	LastName       string          `json:"last_name"`
	Phone          string          `json:"phone"`
	Role           string          `json:"role"`
	Company        string          `json:"company_name"`
	CompanySettings CompanySettings `json:"company_settings"`
}

type CompanySettings struct {
	Country        string `json:"country"`
	Currency       string `json:"currency"`
	CurrencySymbol string `json:"currency_symbol"`
	Locale         string `json:"locale"`
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	companyID := middleware.GetCompanyID(r.Context())

	var p Profile
	err := h.db.QueryRow(r.Context(),
		`SELECT u.id, u.email, u.first_name, u.last_name, COALESCE(u.phone,''), u.role, c.name,
		 COALESCE(c.country,'KZ'), COALESCE(c.currency_code,'KZT'), COALESCE(c.currency_symbol,'₸'), COALESCE(c.locale,'ru-KZ')
		 FROM users u JOIN companies c ON c.id = u.company_id
		 WHERE u.id = $1 AND u.company_id = $2`, userID, companyID).
		Scan(&p.ID, &p.Email, &p.FirstName, &p.LastName, &p.Phone, &p.Role, &p.Company,
			&p.CompanySettings.Country, &p.CompanySettings.Currency, &p.CompanySettings.CurrencySymbol, &p.CompanySettings.Locale)
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

// Supported currency presets
var currencyPresets = map[string]CompanySettings{
	"KZ": {Country: "KZ", Currency: "KZT", CurrencySymbol: "₸", Locale: "ru-KZ"},
	"RU": {Country: "RU", Currency: "RUB", CurrencySymbol: "₽", Locale: "ru-RU"},
	"UZ": {Country: "UZ", Currency: "UZS", CurrencySymbol: "сўм", Locale: "uz-UZ"},
	"AE": {Country: "AE", Currency: "AED", CurrencySymbol: "د.إ", Locale: "ar-AE"},
	"US": {Country: "US", Currency: "USD", CurrencySymbol: "$", Locale: "en-US"},
	"EU": {Country: "EU", Currency: "EUR", CurrencySymbol: "€", Locale: "en-EU"},
	"GB": {Country: "GB", Currency: "GBP", CurrencySymbol: "£", Locale: "en-GB"},
	"TR": {Country: "TR", Currency: "TRY", CurrencySymbol: "₺", Locale: "tr-TR"},
	"GE": {Country: "GE", Currency: "GEL", CurrencySymbol: "₾", Locale: "ka-GE"},
}

func (h *Handler) UpdateCompanySettings(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r.Context())
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only owners can change company settings")
		return
	}

	companyID := middleware.GetCompanyID(r.Context())

	var input struct {
		Country string `json:"country"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	preset, ok := currencyPresets[input.Country]
	if !ok {
		writeError(w, http.StatusBadRequest, "unsupported country code")
		return
	}

	_, err := h.db.Exec(r.Context(),
		`UPDATE companies SET country = $1, currency = $2, currency_symbol = $3, locale = $4 WHERE id = $5`,
		preset.Country, preset.Currency, preset.CurrencySymbol, preset.Locale, companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update settings")
		return
	}

	writeJSON(w, http.StatusOK, preset)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

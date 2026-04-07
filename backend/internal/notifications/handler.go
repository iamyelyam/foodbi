package notifications

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/foodbi/backend/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
	r.Get("/", h.List)
	r.Post("/{id}/read", h.MarkRead)
	r.Get("/unread-count", h.UnreadCount)
	return r
}

type Notification struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	IsRead    bool   `json:"is_read"`
	CreatedAt string `json:"created_at"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	companyID := middleware.GetCompanyID(r.Context())

	rows, err := h.db.Query(r.Context(),
		`SELECT id, type, title, message, is_read, created_at
		 FROM notifications WHERE (user_id = $1 OR user_id IS NULL) AND company_id = $2
		 ORDER BY created_at DESC LIMIT 50`, userID, companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch notifications")
		return
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		var t time.Time
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Message, &n.IsRead, &t); err != nil {
			continue
		}
		n.CreatedAt = t.Format(time.RFC3339)
		notifications = append(notifications, n)
	}
	if notifications == nil {
		notifications = []Notification{}
	}
	writeJSON(w, http.StatusOK, notifications)
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	h.db.Exec(r.Context(),
		`UPDATE notifications SET is_read = true WHERE id = $1 AND (user_id = $2 OR user_id IS NULL)`,
		id, userID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "read"})
}

func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	companyID := middleware.GetCompanyID(r.Context())

	var count int
	h.db.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM notifications WHERE (user_id = $1 OR user_id IS NULL) AND company_id = $2 AND is_read = false`,
		userID, companyID).Scan(&count)

	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

// CreateNotification is called internally by other services.
func CreateNotification(db *pgxpool.Pool, companyID uuid.UUID, userID *uuid.UUID, ntype, title, message string) error {
	_, err := db.Exec(nil,
		`INSERT INTO notifications (id, company_id, user_id, type, title, message, is_read, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, false, NOW())`,
		uuid.New(), companyID, userID, ntype, title, message)
	return err
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

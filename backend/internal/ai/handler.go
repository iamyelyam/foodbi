package ai

import (
	"encoding/json"
	"fmt"
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
	r.Get("/suggestions", h.GetSuggestions)
	r.Post("/tasks", h.CreateTask)
	r.Get("/tasks", h.ListTasks)
	return r
}

type Suggestion struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // menu_optimization, purchase_recommendation, price_adjustment
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // high, medium, low
	Data        any    `json:"data,omitempty"`
}

func (h *Handler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	locationID := r.URL.Query().Get("location_id")

	// Generate suggestions based on existing data patterns
	var suggestions []Suggestion

	// Top selling products recommendation
	var topProduct string
	var topRevenue float64
	args := []interface{}{companyID}
	query := `SELECT product_name, SUM(revenue) FROM product_sales_facts WHERE company_id = $1`
	if locationID != "" {
		query += ` AND location_id = $2`
		args = append(args, locationID)
	}
	query += ` GROUP BY product_name ORDER BY SUM(revenue) DESC LIMIT 1`
	h.db.QueryRow(r.Context(), query, args...).Scan(&topProduct, &topRevenue)

	if topProduct != "" {
		suggestions = append(suggestions, Suggestion{
			ID:          uuid.New().String(),
			Type:        "menu_optimization",
			Title:       "Promote top seller: " + topProduct,
			Description: "This product generates the highest revenue. Consider featuring it prominently on the menu or creating combo deals.",
			Impact:      "high",
		})
	}

	// Low margin product alert
	var lowMarginProduct string
	var margin float64
	mArgs := []interface{}{companyID}
	mQuery := `SELECT product_name,
		CASE WHEN SUM(revenue) > 0 THEN ((SUM(revenue) - SUM(cost_price)) / SUM(revenue)) * 100 ELSE 0 END as margin
		FROM product_sales_facts WHERE company_id = $1 AND cost_price > 0`
	if locationID != "" {
		mQuery += ` AND location_id = $2`
		mArgs = append(mArgs, locationID)
	}
	mQuery += ` GROUP BY product_name HAVING SUM(revenue) > 0 ORDER BY margin ASC LIMIT 1`
	h.db.QueryRow(r.Context(), mQuery, mArgs...).Scan(&lowMarginProduct, &margin)

	if lowMarginProduct != "" && margin < 30 {
		suggestions = append(suggestions, Suggestion{
			ID:          uuid.New().String(),
			Type:        "price_adjustment",
			Title:       "Low margin alert: " + lowMarginProduct,
			Description: "This product has only " + formatPercent(margin) + "% margin. Consider raising the price or finding a cheaper supplier.",
			Impact:      "medium",
		})
	}

	// Purchase pattern suggestion
	var supplierName string
	var purchaseTotal float64
	pArgs := []interface{}{companyID}
	pQuery := `SELECT supplier_name, SUM(total_sum) FROM purchase_facts WHERE company_id = $1 AND supplier_name IS NOT NULL`
	if locationID != "" {
		pQuery += ` AND location_id = $2`
		pArgs = append(pArgs, locationID)
	}
	pQuery += ` GROUP BY supplier_name ORDER BY SUM(total_sum) DESC LIMIT 1`
	h.db.QueryRow(r.Context(), pQuery, pArgs...).Scan(&supplierName, &purchaseTotal)

	if supplierName != "" {
		suggestions = append(suggestions, Suggestion{
			ID:          uuid.New().String(),
			Type:        "purchase_recommendation",
			Title:       "Review top supplier: " + supplierName,
			Description: "Your largest supplier by spend. Consider negotiating volume discounts or comparing prices with alternatives.",
			Impact:      "medium",
		})
	}

	if suggestions == nil {
		suggestions = []Suggestion{{
			ID:          uuid.New().String(),
			Type:        "menu_optimization",
			Title:       "Collect more data",
			Description: "AI suggestions improve with more data. Keep syncing with iiko for at least 7 days to get meaningful recommendations.",
			Impact:      "low",
		}}
	}

	writeJSON(w, http.StatusOK, suggestions)
}

type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type CreateTaskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())

	var input CreateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	id := uuid.New()
	_, err := h.db.Exec(r.Context(),
		`INSERT INTO ai_tasks (id, company_id, created_by, title, description, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'open', NOW())`,
		id, companyID, userID, input.Title, input.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": id, "status": "open"})
}

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())

	rows, err := h.db.Query(r.Context(),
		`SELECT id, title, description, status, created_at FROM ai_tasks WHERE company_id = $1 ORDER BY created_at DESC`,
		companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch tasks")
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var createdAt time.Time
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &createdAt); err != nil {
			continue
		}
		t.CreatedAt = createdAt.Format(time.RFC3339)
		tasks = append(tasks, t)
	}
	if tasks == nil {
		tasks = []Task{}
	}
	writeJSON(w, http.StatusOK, tasks)
}

func formatPercent(v float64) string {
	return fmt.Sprintf("%.1f", v)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

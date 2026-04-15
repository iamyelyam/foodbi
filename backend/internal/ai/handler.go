package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/foodbi/backend/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// formatProductName capitalizes first letter and lowercases the rest.
func formatProductName(name string) string {
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	runes := []rune(lower)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

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
			Title:       "Promote top seller: " + formatProductName(topProduct),
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
			Title:       "Low margin alert: " + formatProductName(lowMarginProduct),
			Description: "This product has only " + formatPercent(margin) + "% margin. Consider raising the price or finding a cheaper supplier.",
			Impact:      "medium",
		})
	}

	// Suspicious margin detection — likely misconfigured cost price in iiko
	// Margin > 90% (cost too low, or recipe not set up) or < 15% (cost too high, wrong recipe)
	suspArgs := []interface{}{companyID}
	suspQuery := `SELECT product_name,
		CASE WHEN SUM(revenue) > 0 THEN (1 - SUM(cost_price)/SUM(revenue)) * 100 ELSE 0 END as margin
		FROM product_sales_facts WHERE company_id = $1`
	if locationID != "" {
		suspQuery += ` AND location_id = $2`
		suspArgs = append(suspArgs, locationID)
	}
	suspQuery += ` GROUP BY product_name
		HAVING SUM(revenue) > 0
		AND ((1 - SUM(cost_price)/SUM(revenue)) * 100 > 90
		  OR (1 - SUM(cost_price)/SUM(revenue)) * 100 < 15)
		ORDER BY SUM(revenue) DESC LIMIT 10`
	rows, err := h.db.Query(r.Context(), suspQuery, suspArgs...)
	if err == nil {
		defer rows.Close()
		var suspicious []map[string]any
		for rows.Next() {
			var name string
			var m float64
			if err := rows.Scan(&name, &m); err != nil {
				continue
			}
			suspicious = append(suspicious, map[string]any{"product_name": name, "margin": m})
		}
		if len(suspicious) > 0 {
			names := ""
			for i, p := range suspicious {
				if i > 0 {
					names += ", "
				}
				names += fmt.Sprintf("%s (%.0f%%)", formatProductName(fmt.Sprintf("%v", p["product_name"])), p["margin"])
			}
			suggestions = append(suggestions, Suggestion{
				ID:          uuid.New().String(),
				Type:        "cost_configuration",
				Title:       fmt.Sprintf("Возможно неверно настроена себестоимость (%d product(s))", len(suspicious)),
				Description: "Следующие продукты имеют маржу >90% или <15%, что обычно указывает на неправильно настроенную себестоимость в iiko: " + names + ". Проверьте технологические карты блюд.",
				Impact:      "high",
				Data:        suspicious,
			})
		}
	}

	// Stock data issues — group negative amounts and unpriced items on the shelf
	stockArgs := []interface{}{companyID}
	stockLocFilter := ""
	if locationID != "" {
		stockLocFilter = " AND s.location_id = $2"
		stockArgs = append(stockArgs, locationID)
	}
	// Take the latest snapshot per product (stock is snapshotted over time)
	stockQuery := `
		WITH latest AS (
			SELECT DISTINCT ON (s.iiko_product_id)
				s.iiko_product_id, s.product_name, s.amount, s.cost_sum, s.snapshot_at
			FROM stock_snapshots s
			WHERE s.company_id = $1` + stockLocFilter + `
			ORDER BY s.iiko_product_id, s.snapshot_at DESC
		)
		SELECT product_name, amount, cost_sum,
			CASE
				WHEN amount < 0 THEN 'negative_amount'
				WHEN amount > 0 AND cost_sum = 0 THEN 'zero_cost'
			END AS issue
		FROM latest
		WHERE amount < 0 OR (amount > 0 AND cost_sum = 0)
		ORDER BY ABS(amount) DESC LIMIT 15`

	srows, serr := h.db.Query(r.Context(), stockQuery, stockArgs...)
	if serr == nil {
		defer srows.Close()
		var negative []map[string]any
		var zeroCost []map[string]any
		for srows.Next() {
			var name, issue string
			var amount, cost float64
			if err := srows.Scan(&name, &amount, &cost, &issue); err != nil {
				continue
			}
			row := map[string]any{"product_name": name, "amount": amount, "cost_sum": cost}
			if issue == "negative_amount" {
				negative = append(negative, row)
			} else if issue == "zero_cost" {
				zeroCost = append(zeroCost, row)
			}
		}

		if len(negative) > 0 {
			names := ""
			for i, p := range negative {
				if i > 0 {
					names += ", "
				}
				names += fmt.Sprintf("%s (%.1f)", formatProductName(fmt.Sprintf("%v", p["product_name"])), p["amount"])
			}
			suggestions = append(suggestions, Suggestion{
				ID:          uuid.New().String(),
				Type:        "stock_data_issue",
				Title:       fmt.Sprintf("Отрицательный остаток на складе (%d позиц.)", len(negative)),
				Description: "Эти позиции имеют отрицательное количество, что обычно означает ошибку ввода данных в iiko (продано больше, чем было на складе): " + names + ". Проверьте приходные накладные и списания.",
				Impact:      "high",
				Data:        negative,
			})
		}

		if len(zeroCost) > 0 {
			names := ""
			for i, p := range zeroCost {
				if i > 0 {
					names += ", "
				}
				names += formatProductName(fmt.Sprintf("%v", p["product_name"]))
			}
			suggestions = append(suggestions, Suggestion{
				ID:          uuid.New().String(),
				Type:        "stock_data_issue",
				Title:       fmt.Sprintf("Нулевая закупочная цена (%d позиц.)", len(zeroCost)),
				Description: "Эти товары есть в остатках, но без закупочной цены в iiko: " + names + ". Добавьте цену прихода, чтобы корректно считать себестоимость и маржу.",
				Impact:      "medium",
				Data:        zeroCost,
			})
		}
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

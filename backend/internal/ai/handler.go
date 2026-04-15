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

// Suggestion uses i18n KEYS + PARAMS rather than rendered text. This way the
// backend stays locale-agnostic and the frontend renders strings in whatever
// locale the user picked.
//
// Frontend resolves with `useT()`: e.g. `t(s.title_key, s.title_params)` where
// `t` interpolates `{name}` placeholders from the params map.
//
// All param values are pre-formatted strings/numbers — backend takes care of
// `formatProductName()` etc. so the frontend just substitutes verbatim.
type Suggestion struct {
	ID                string         `json:"id"`
	Type              string         `json:"type"` // menu_optimization, purchase_recommendation, price_adjustment, ...
	TitleKey          string         `json:"title_key"`
	TitleParams       map[string]any `json:"title_params,omitempty"`
	DescriptionKey    string         `json:"description_key"`
	DescriptionParams map[string]any `json:"description_params,omitempty"`
	// SolutionKey: omitted for suggestions where there's no clear single fix
	// (e.g. data-quality alerts that need investigation).
	SolutionKey    string         `json:"solution_key,omitempty"`
	SolutionParams map[string]any `json:"solution_params,omitempty"`
	Impact         string         `json:"impact"` // high, medium, low
	// Estimated monetary impact in restaurant currency (KZT). LossAmount is positive when there's
	// money slipping away today (data errors, low margins, perishables); GainAmount is positive
	// when acting on the suggestion would unlock additional revenue. Both are 0 / omitted when
	// the suggestion is qualitative and we can't put a credible number on it.
	LossAmount float64 `json:"loss_amount,omitempty"`
	GainAmount float64 `json:"gain_amount,omitempty"`
	Data       any     `json:"data,omitempty"`
}

// SuggestionsResponse — top-level shape consumed by the AI Suggestions page.
// `summary.total_loss` and `total_gain_with_ai` are the headline metrics shown
// at the top of the screen; they're sums over all returned suggestions so the
// client doesn't have to recompute.
type SuggestionsResponse struct {
	Summary struct {
		TotalLoss        float64 `json:"total_loss"`
		TotalGainWithAI  float64 `json:"total_gain_with_ai"`
		Date             string  `json:"date"` // YYYY-MM-DD when the snapshot was computed
	} `json:"summary"`
	Suggestions []Suggestion `json:"suggestions"`
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
		product := formatProductName(topProduct)
		suggestions = append(suggestions, Suggestion{
			ID:                uuid.New().String(),
			Type:              "menu_optimization",
			TitleKey:          "ai.s.topSeller.title",
			TitleParams:       map[string]any{"product": product},
			DescriptionKey:    "ai.s.topSeller.description",
			SolutionKey:       "ai.s.topSeller.solution",
			SolutionParams:    map[string]any{"product": product},
			Impact:            "high",
			// Rule of thumb: a successful promotion bumps top-seller volume by ~10%.
			GainAmount: topRevenue * 0.10,
		})
	}

	// Low margin product alert
	var lowMarginProduct string
	var margin float64
	var lowMarginRevenue float64
	mArgs := []interface{}{companyID}
	mQuery := `SELECT product_name, SUM(revenue) AS total_rev,
		CASE WHEN SUM(revenue) > 0 THEN ((SUM(revenue) - SUM(cost_price)) / SUM(revenue)) * 100 ELSE 0 END as margin
		FROM product_sales_facts WHERE company_id = $1 AND cost_price > 0`
	if locationID != "" {
		mQuery += ` AND location_id = $2`
		mArgs = append(mArgs, locationID)
	}
	mQuery += ` GROUP BY product_name HAVING SUM(revenue) > 0 ORDER BY margin ASC LIMIT 1`
	h.db.QueryRow(r.Context(), mQuery, mArgs...).Scan(&lowMarginProduct, &lowMarginRevenue, &margin)

	if lowMarginProduct != "" && margin < 30 {
		// Opportunity loss: how much extra profit if margin reached the 30% target.
		// = revenue × (target − actual)/100. Caps at 0 if margin >= 30 (defensive).
		opportunityLoss := lowMarginRevenue * (30.0 - margin) / 100.0
		if opportunityLoss < 0 {
			opportunityLoss = 0
		}
		product := formatProductName(lowMarginProduct)
		suggestions = append(suggestions, Suggestion{
			ID:                uuid.New().String(),
			Type:              "price_adjustment",
			TitleKey:          "ai.s.lowMargin.title",
			TitleParams:       map[string]any{"product": product},
			DescriptionKey:    "ai.s.lowMargin.description",
			DescriptionParams: map[string]any{"product": product, "margin": formatPercent(margin)},
			SolutionKey:       "ai.s.lowMargin.solution",
			SolutionParams:    map[string]any{"product": product},
			Impact:            "medium",
			LossAmount:        opportunityLoss,
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
				ID:                uuid.New().String(),
				Type:              "cost_configuration",
				TitleKey:          "ai.s.suspiciousMargin.title",
				TitleParams:       map[string]any{"count": len(suspicious)},
				DescriptionKey:    "ai.s.suspiciousMargin.description",
				DescriptionParams: map[string]any{"count": len(suspicious), "names": names},
				Impact:            "high",
				Data:              suspicious,
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
			var negativeLoss float64
			for i, p := range negative {
				if i > 0 {
					names += ", "
				}
				names += fmt.Sprintf("%s (%.1f)", formatProductName(fmt.Sprintf("%v", p["product_name"])), p["amount"])
				// cost_sum mirrors the sign of amount (negative for negative stock); take abs.
				if cs, ok := p["cost_sum"].(float64); ok {
					if cs < 0 {
						negativeLoss += -cs
					} else {
						negativeLoss += cs
					}
				}
			}
			suggestions = append(suggestions, Suggestion{
				ID:                uuid.New().String(),
				Type:              "stock_data_issue",
				TitleKey:          "ai.s.negativeStock.title",
				TitleParams:       map[string]any{"count": len(negative)},
				DescriptionKey:    "ai.s.negativeStock.description",
				DescriptionParams: map[string]any{"count": len(negative), "names": names},
				SolutionKey:       "ai.s.negativeStock.solution",
				Impact:            "high",
				LossAmount:        negativeLoss,
				Data:              negative,
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
				ID:                uuid.New().String(),
				Type:              "stock_data_issue",
				TitleKey:          "ai.s.zeroCost.title",
				TitleParams:       map[string]any{"count": len(zeroCost)},
				DescriptionKey:    "ai.s.zeroCost.description",
				DescriptionParams: map[string]any{"count": len(zeroCost), "names": names},
				Impact:            "medium",
				Data:              zeroCost,
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
			ID:             uuid.New().String(),
			Type:           "purchase_recommendation",
			TitleKey:       "ai.s.topSupplier.title",
			TitleParams:    map[string]any{"supplier": supplierName},
			DescriptionKey: "ai.s.topSupplier.description",
			SolutionKey:    "ai.s.topSupplier.solution",
			SolutionParams: map[string]any{"supplier": supplierName},
			Impact:         "medium",
			// Rule of thumb: a 5% volume discount on the top supplier compounds quickly.
			GainAmount: purchaseTotal * 0.05,
		})
	}

	if suggestions == nil {
		suggestions = []Suggestion{{
			ID:             uuid.New().String(),
			Type:           "menu_optimization",
			TitleKey:       "ai.s.collectMore.title",
			DescriptionKey: "ai.s.collectMore.description",
			Impact:         "low",
		}}
	}

	// Aggregate headline metrics for the page header.
	var resp SuggestionsResponse
	resp.Suggestions = suggestions
	for _, s := range suggestions {
		resp.Summary.TotalLoss += s.LossAmount
		resp.Summary.TotalGainWithAI += s.GainAmount
	}
	resp.Summary.Date = time.Now().Format("2006-01-02")

	writeJSON(w, http.StatusOK, resp)
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

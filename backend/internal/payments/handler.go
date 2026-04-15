package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/foodbi/backend/internal/telegram"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

const (
	maxFieldLen     = 255
	maxBodyBytes    = 64 * 1024 // 64 KB
	notifyQueueSize = 1024
	notifyWorkers   = 4
	telegramRateMin = 40 * time.Millisecond // ~25 msg/sec global
	webhookRPM      = 600                   // per-company per-minute
)

// notifyJob is queued for the telegram-sender worker pool.
type notifyJob struct {
	companyID uuid.UUID
	payload   WebhookPayload
	attemptAt time.Time
}

// Handler handles payment webhook requests.
type Handler struct {
	db        *pgxpool.Pool
	bot       *telegram.Bot
	jobs      chan notifyJob
	limiter   *companyRateLimiter
	stop      chan struct{}
	stopOnce  sync.Once
	wg        sync.WaitGroup
}

// NewHandler creates a payment webhook handler and starts the worker pool.
// Call Stop on shutdown to drain workers.
func NewHandler(db *pgxpool.Pool, bot *telegram.Bot) *Handler {
	h := &Handler{
		db:      db,
		bot:     bot,
		jobs:    make(chan notifyJob, notifyQueueSize),
		limiter: newCompanyRateLimiter(webhookRPM, time.Minute),
		stop:    make(chan struct{}),
	}
	for i := 0; i < notifyWorkers; i++ {
		h.wg.Add(1)
		go h.notifyWorker()
	}
	return h
}

// Stop drains workers gracefully.
func (h *Handler) Stop() {
	h.stopOnce.Do(func() {
		close(h.stop)
		close(h.jobs)
	})
	h.wg.Wait()
}

// WebhookPayload is the expected payload from the payment system.
type WebhookPayload struct {
	TerminalID  string `json:"terminal_id"`
	OrderID     string `json:"order_id"`
	TableNumber string `json:"table_number"`
	GuestName   string `json:"guest_name"`
	GuestPhone  string `json:"guest_phone"`
	Amount      int64  `json:"amount"` // KZT, no subunits
	Status      string `json:"status"` // "failed" or "success"
	Timestamp   string `json:"timestamp"`
}

// HandleWebhook processes incoming payment events.
// Route: POST /api/v1/webhooks/payment/{companyID}
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	companyIDStr := chi.URLParam(r, "companyID")
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid company_id")
		return
	}

	// Per-company rate limit (DDoS protection)
	if !h.limiter.allow(companyID) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	signature := r.Header.Get("X-Webhook-Signature")
	if err := h.verifySignature(r.Context(), companyID, body, signature); err != nil {
		log.Warn().Err(err).Str("company_id", companyID.String()).Msg("webhook signature verification failed")
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := validatePayload(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	attemptAt := time.Now()
	if payload.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, payload.Timestamp); err == nil {
			attemptAt = t
		}
	}

	// Idempotent insert: duplicate (company, order, status, attempt_at) is a no-op.
	tag, err := h.db.Exec(r.Context(),
		`INSERT INTO payment_attempts (company_id, terminal_id, order_id, table_number, guest_name, guest_phone, amount, status, attempt_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT ON CONSTRAINT payment_attempts_idempotency_uniq DO NOTHING`,
		companyID, payload.TerminalID, payload.OrderID, payload.TableNumber,
		payload.GuestName, payload.GuestPhone, payload.Amount, payload.Status, attemptAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			log.Error().Str("pg_code", pgErr.Code).Str("msg", pgErr.Message).Msg("payments: pg error")
		} else {
			log.Error().Err(err).Msg("payments: failed to store attempt")
		}
		writeError(w, http.StatusInternalServerError, "failed to store attempt")
		return
	}

	// Skip notification on idempotent duplicate
	if tag.RowsAffected() == 0 {
		log.Debug().Str("order_id", payload.OrderID).Msg("payments: duplicate webhook, skipping notify")
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
		return
	}

	// Enqueue (non-blocking) — drop if queue is full to keep webhook fast.
	select {
	case h.jobs <- notifyJob{companyID: companyID, payload: payload, attemptAt: attemptAt}:
	default:
		log.Warn().Msg("payments: notify queue full, dropping notification")
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// validatePayload enforces required fields, allowed status, and field length caps.
func validatePayload(p *WebhookPayload) error {
	if p.TerminalID == "" || p.OrderID == "" || p.TableNumber == "" {
		return fmt.Errorf("missing required fields: terminal_id, order_id, table_number")
	}
	if p.Status != "failed" && p.Status != "success" {
		return fmt.Errorf("status must be 'failed' or 'success'")
	}
	if p.Amount < 0 {
		return fmt.Errorf("amount must be >= 0")
	}
	for name, v := range map[string]string{
		"terminal_id":  p.TerminalID,
		"order_id":     p.OrderID,
		"table_number": p.TableNumber,
		"guest_name":   p.GuestName,
		"guest_phone":  p.GuestPhone,
	} {
		if len(v) > maxFieldLen {
			return fmt.Errorf("field %s exceeds %d chars", name, maxFieldLen)
		}
	}
	return nil
}

// notifyWorker drains the job queue, sending Telegram messages with global rate limit.
func (h *Handler) notifyWorker() {
	defer h.wg.Done()
	ticker := time.NewTicker(telegramRateMin)
	defer ticker.Stop()
	for job := range h.jobs {
		<-ticker.C // global throttle across all workers
		h.sendNotification(job)
	}
}

func (h *Handler) sendNotification(job notifyJob) {
	if h.bot == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("payments: panic in sendNotification")
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chats, err := h.bot.GetSubscribedChats(ctx, job.companyID, job.payload.TerminalID)
	if err != nil {
		log.Error().Err(err).Msg("payments: failed to get subscribed chats")
		return
	}
	if len(chats) == 0 {
		return
	}

	locationName := job.payload.TerminalID
	var name string
	err = h.db.QueryRow(ctx,
		`SELECT name FROM locations WHERE id::text = $1 AND company_id = $2`,
		job.payload.TerminalID, job.companyID).Scan(&name)
	if err == nil && name != "" {
		locationName = name
	}

	msg := h.formatMessage(ctx, job, locationName)
	if msg == "" {
		return // success without prior failures — no-op
	}

	for _, chatID := range chats {
		if err := h.bot.SendMessage(chatID, msg); err != nil {
			log.Error().Err(err).Int64("chat_id", chatID).Msg("payments: failed to send telegram notification")
		}
	}
}

func (h *Handler) formatMessage(ctx context.Context, job notifyJob, locationName string) string {
	p := job.payload
	timeStr := job.attemptAt.Format("15:04:05")

	switch p.Status {
	case "failed":
		return fmt.Sprintf(
			"⚠️ <b>Неуспешная оплата</b>\n\n"+
				"📍 Терминал: %s\n"+
				"🪑 Стол: %s\n"+
				"👤 %s\n"+
				"📱 %s\n"+
				"💰 %s ₸\n"+
				"🕐 %s",
			escapeHTML(locationName), escapeHTML(p.TableNumber),
			escapeHTML(p.GuestName), escapeHTML(p.GuestPhone),
			formatAmount(p.Amount), timeStr)

	case "success":
		var failCount int
		_ = h.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM payment_attempts
			 WHERE company_id = $1 AND order_id = $2 AND status = 'failed'`,
			job.companyID, p.OrderID).Scan(&failCount)
		if failCount == 0 {
			return ""
		}
		return fmt.Sprintf(
			"✅ <b>Оплата прошла успешно</b>\n\n"+
				"📍 Терминал: %s\n"+
				"🪑 Стол: %s\n"+
				"👤 %s\n"+
				"💰 %s ₸\n"+
				"🕐 %s\n"+
				"(после %d неуспешных попыток)",
			escapeHTML(locationName), escapeHTML(p.TableNumber),
			escapeHTML(p.GuestName),
			formatAmount(p.Amount), timeStr, failCount)
	}
	return ""
}

func (h *Handler) verifySignature(ctx context.Context, companyID uuid.UUID, body []byte, signature string) error {
	if signature == "" {
		return fmt.Errorf("missing signature")
	}

	var secret string
	err := h.db.QueryRow(ctx,
		`SELECT webhook_secret FROM companies WHERE id = $1 AND webhook_secret IS NOT NULL AND webhook_secret != ''`,
		companyID).Scan(&secret)
	if err != nil {
		return fmt.Errorf("company not found or no webhook secret")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

func formatAmount(amount int64) string {
	if amount < 0 {
		amount = -amount
	}
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, ' ')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// escapeHTML neutralizes Telegram HTML parse mode special chars.
func escapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return r.Replace(s)
}

// companyRateLimiter is a simple sliding-window per-company rate limiter.
type companyRateLimiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	buckets map[uuid.UUID][]time.Time
}

func newCompanyRateLimiter(max int, window time.Duration) *companyRateLimiter {
	return &companyRateLimiter{
		max:     max,
		window:  window,
		buckets: make(map[uuid.UUID][]time.Time),
	}
}

func (l *companyRateLimiter) allow(companyID uuid.UUID) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.window)

	bucket := l.buckets[companyID]
	// drop expired
	i := 0
	for ; i < len(bucket); i++ {
		if bucket[i].After(cutoff) {
			break
		}
	}
	bucket = bucket[i:]

	if len(bucket) >= l.max {
		l.buckets[companyID] = bucket
		return false
	}
	bucket = append(bucket, now)
	l.buckets[companyID] = bucket
	return true
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

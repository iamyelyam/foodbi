package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

const (
	maxTelegramResponseBytes = 1 << 20 // 1 MB cap on Telegram API responses
	sendMaxRetries           = 3
	sendRetryBaseDelay       = 1 * time.Second
)

// Bot handles Telegram Bot API long-polling and command dispatch.
type Bot struct {
	token  string
	db     *pgxpool.Pool
	client *http.Client
	offset int64
	mu     sync.Mutex
}

// NewBot creates a Telegram bot with the given token.
func NewBot(token string, db *pgxpool.Pool) *Bot {
	return &Bot{
		token: token,
		db:    db,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Start begins long-polling in a blocking loop. Cancel the context to stop.
func (b *Bot) Start(ctx context.Context) {
	log.Info().Msg("telegram bot started (long-polling)")
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("telegram bot stopped")
			return
		default:
			updates, err := b.getUpdates(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Error().Err(err).Msg("telegram: failed to get updates")
				time.Sleep(3 * time.Second)
				continue
			}
			for _, u := range updates {
				b.safeHandleUpdate(ctx, u)
				b.mu.Lock()
				if u.UpdateID >= b.offset {
					b.offset = u.UpdateID + 1
				}
				b.mu.Unlock()
			}
		}
	}
}

// safeHandleUpdate runs handleUpdate with panic recovery so a single bad message
// cannot kill the long-polling goroutine.
func (b *Bot) safeHandleUpdate(ctx context.Context, u update) {
	defer func() {
		if r := recover(); r != nil {
			chatID := int64(0)
			if u.Message != nil {
				chatID = u.Message.Chat.ID
			}
			log.Error().
				Interface("panic", r).
				Int64("chat_id", chatID).
				Int64("update_id", u.UpdateID).
				Bytes("stack", debug.Stack()).
				Msg("telegram: panic recovered in handleUpdate")
		}
	}()
	b.handleUpdate(ctx, u)
}

// SendMessage sends a text message to a chat with retry on transient failures.
func (b *Bot) SendMessage(chatID int64, text string) error {
	body, err := json.Marshal(map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	})
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.token)

	var lastErr error
	for attempt := 0; attempt < sendMaxRetries; attempt++ {
		if attempt > 0 {
			delay := sendRetryBaseDelay * time.Duration(1<<(attempt-1)) // 1s, 2s, 4s
			time.Sleep(delay)
		}

		err := b.doSend(url, body)
		if err == nil {
			return nil
		}
		lastErr = err

		// Don't retry on permanent errors (4xx that aren't rate-limit)
		var permErr *permanentError
		if errors.As(err, &permErr) {
			return err
		}
	}
	return fmt.Errorf("telegram send failed after %d attempts: %w", sendMaxRetries, lastErr)
}

// permanentError indicates a Telegram error that should not be retried (e.g. bad request).
type permanentError struct {
	code int
	desc string
}

func (e *permanentError) Error() string {
	return fmt.Sprintf("telegram permanent error %d: %s", e.code, e.desc)
}

func (b *Bot) doSend(url string, body []byte) error {
	resp, err := b.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram POST: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxTelegramResponseBytes))
	if err != nil {
		return fmt.Errorf("read telegram response: %w", err)
	}

	var result struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parse telegram response (status %d): %w", resp.StatusCode, err)
	}

	if !result.OK {
		// 4xx (other than 429 rate-limit) is permanent: bad token, blocked by user, invalid chat
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			return &permanentError{code: result.ErrorCode, desc: result.Description}
		}
		return fmt.Errorf("telegram error %d (status %d): %s", result.ErrorCode, resp.StatusCode, result.Description)
	}
	return nil
}

type update struct {
	UpdateID int64    `json:"update_id"`
	Message  *message `json:"message"`
}

type message struct {
	MessageID int64  `json:"message_id"`
	Chat      chat   `json:"chat"`
	Text      string `json:"text"`
}

type chat struct {
	ID int64 `json:"id"`
}

type getUpdatesResponse struct {
	OK          bool     `json:"ok"`
	Result      []update `json:"result"`
	Description string   `json:"description"`
}

func (b *Bot) getUpdates(ctx context.Context) ([]update, error) {
	b.mu.Lock()
	offset := b.offset
	b.mu.Unlock()

	body, err := json.Marshal(map[string]interface{}{
		"offset":  offset,
		"timeout": 30,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal getUpdates: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", b.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxTelegramResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read getUpdates response: %w", err)
	}

	var result getUpdatesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	if !result.OK {
		return nil, fmt.Errorf("telegram getUpdates not ok: %s", result.Description)
	}
	return result.Result, nil
}

func (b *Bot) handleUpdate(ctx context.Context, u update) {
	if u.Message == nil || u.Message.Text == "" {
		return
	}
	text := strings.TrimSpace(u.Message.Text)
	chatID := u.Message.Chat.ID

	parts := strings.SplitN(text, " ", 2)
	cmd := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	// Per-command timeout to prevent long-polling stall
	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch cmd {
	case "/start":
		b.cmdStart(cmdCtx, chatID, args)
	case "/locations":
		b.cmdLocations(cmdCtx, chatID)
	case "/subscribe":
		b.cmdSubscribe(cmdCtx, chatID, args)
	case "/unsubscribe":
		b.cmdUnsubscribe(cmdCtx, chatID, args)
	case "/status":
		b.cmdStatus(cmdCtx, chatID)
	case "/help":
		b.cmdHelp(chatID)
	default:
		b.SendMessage(chatID, "Неизвестная команда. Используйте /help для списка команд.")
	}
}

func (b *Bot) cmdStart(ctx context.Context, chatID int64, apiKey string) {
	if apiKey == "" {
		b.SendMessage(chatID, "Добро пожаловать в FoodBI Payment Bot!\n\nДля привязки отправьте:\n<code>/start ВАШ_API_КЛЮЧ</code>\n\nAPI-ключ можно найти в настройках FoodBI.")
		return
	}

	// Resolve company by bot_link_token (separate from webhook signing secret)
	var companyID uuid.UUID
	err := b.db.QueryRow(ctx,
		`SELECT id FROM companies WHERE bot_link_token = $1`, apiKey).Scan(&companyID)
	if err != nil {
		b.SendMessage(chatID, "Неверный API-ключ. Проверьте ключ в настройках FoodBI.")
		return
	}

	_, err = b.db.Exec(ctx,
		`INSERT INTO telegram_bot_links (company_id, telegram_chat_id)
		 VALUES ($1, $2)
		 ON CONFLICT (telegram_chat_id) DO UPDATE SET company_id = $1, linked_at = NOW()`,
		companyID, chatID)
	if err != nil {
		log.Error().Err(err).Int64("chat_id", chatID).Msg("telegram: failed to link bot")
		b.SendMessage(chatID, "Ошибка привязки. Попробуйте позже.")
		return
	}

	_, err = b.db.Exec(ctx,
		`INSERT INTO telegram_subscriptions (company_id, telegram_chat_id, terminal_ids, is_active)
		 VALUES ($1, $2, '{}', false)
		 ON CONFLICT (company_id, telegram_chat_id) DO NOTHING`,
		companyID, chatID)
	if err != nil {
		log.Error().Err(err).Msg("telegram: failed to create subscription")
	}

	b.SendMessage(chatID, "Бот привязан к вашей компании.\n\nИспользуйте /locations чтобы увидеть доступные терминалы, затем /subscribe для подписки.")
}

func (b *Bot) cmdLocations(ctx context.Context, chatID int64) {
	companyID, err := b.getCompanyForChat(ctx, chatID)
	if err != nil {
		b.SendMessage(chatID, "Бот не привязан. Используйте /start API_КЛЮЧ")
		return
	}

	rows, err := b.db.Query(ctx,
		`SELECT id, name, address FROM locations WHERE company_id = $1 ORDER BY name`,
		companyID)
	if err != nil {
		b.SendMessage(chatID, "Ошибка получения локаций.")
		return
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("<b>Доступные локации:</b>\n\n")
	count := 0
	for rows.Next() {
		var id uuid.UUID
		var name, address string
		if err := rows.Scan(&id, &name, &address); err != nil {
			continue
		}
		count++
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n   ID: <code>%s</code>\n   %s\n\n",
			count, escapeHTML(name), id.String(), escapeHTML(address)))
	}
	if count == 0 {
		b.SendMessage(chatID, "Нет доступных локаций.")
		return
	}
	sb.WriteString("Для подписки:\n<code>/subscribe ID1,ID2</code>")
	b.SendMessage(chatID, sb.String())
}

func (b *Bot) cmdSubscribe(ctx context.Context, chatID int64, args string) {
	if args == "" {
		b.SendMessage(chatID, "Укажите ID терминалов:\n<code>/subscribe ID1,ID2</code>\n\nСписок: /locations")
		return
	}

	companyID, err := b.getCompanyForChat(ctx, chatID)
	if err != nil {
		b.SendMessage(chatID, "Бот не привязан. Используйте /start API_КЛЮЧ")
		return
	}

	terminalIDs := parseTerminalIDs(args)
	if len(terminalIDs) == 0 {
		b.SendMessage(chatID, "Не удалось распознать ID терминалов.")
		return
	}

	_, err = b.db.Exec(ctx,
		`INSERT INTO telegram_subscriptions (company_id, telegram_chat_id, terminal_ids, is_active)
		 VALUES ($1, $2, $3, true)
		 ON CONFLICT (company_id, telegram_chat_id) DO UPDATE
		 SET terminal_ids = (
		     SELECT ARRAY(SELECT DISTINCT unnest(telegram_subscriptions.terminal_ids || $3::text[]))
		 ), is_active = true, updated_at = NOW()`,
		companyID, chatID, terminalIDs)
	if err != nil {
		log.Error().Err(err).Msg("telegram: failed to subscribe")
		b.SendMessage(chatID, "Ошибка подписки. Попробуйте позже.")
		return
	}

	b.SendMessage(chatID, fmt.Sprintf("Подписка обновлена. Добавлено терминалов: %d\nПроверить: /status", len(terminalIDs)))
}

func (b *Bot) cmdUnsubscribe(ctx context.Context, chatID int64, args string) {
	companyID, err := b.getCompanyForChat(ctx, chatID)
	if err != nil {
		b.SendMessage(chatID, "Бот не привязан. Используйте /start API_КЛЮЧ")
		return
	}

	if args == "all" || args == "" {
		_, err = b.db.Exec(ctx,
			`UPDATE telegram_subscriptions SET terminal_ids = '{}', is_active = false, updated_at = NOW()
			 WHERE company_id = $1 AND telegram_chat_id = $2`,
			companyID, chatID)
		if err != nil {
			b.SendMessage(chatID, "Ошибка отписки.")
			return
		}
		b.SendMessage(chatID, "Вы отписаны от всех терминалов.")
		return
	}

	terminalIDs := parseTerminalIDs(args)
	_, err = b.db.Exec(ctx,
		`UPDATE telegram_subscriptions
		 SET terminal_ids = (
		     SELECT ARRAY(SELECT unnest(terminal_ids) EXCEPT SELECT unnest($3::text[]))
		 ), updated_at = NOW()
		 WHERE company_id = $1 AND telegram_chat_id = $2`,
		companyID, chatID, terminalIDs)
	if err != nil {
		b.SendMessage(chatID, "Ошибка отписки.")
		return
	}
	b.SendMessage(chatID, fmt.Sprintf("Отписка от %d терминалов. Проверить: /status", len(terminalIDs)))
}

func (b *Bot) cmdStatus(ctx context.Context, chatID int64) {
	companyID, err := b.getCompanyForChat(ctx, chatID)
	if err != nil {
		b.SendMessage(chatID, "Бот не привязан. Используйте /start API_КЛЮЧ")
		return
	}

	var terminalIDs []string
	var isActive bool
	err = b.db.QueryRow(ctx,
		`SELECT terminal_ids, is_active FROM telegram_subscriptions
		 WHERE company_id = $1 AND telegram_chat_id = $2`,
		companyID, chatID).Scan(&terminalIDs, &isActive)
	if err != nil {
		b.SendMessage(chatID, "Нет активных подписок. Используйте /subscribe")
		return
	}

	status := "неактивна"
	if isActive {
		status = "активна"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>Статус подписки:</b> %s\n\n", status))
	if len(terminalIDs) == 0 {
		sb.WriteString("Терминалы: нет\n\nИспользуйте /subscribe для добавления.")
	} else {
		sb.WriteString(fmt.Sprintf("<b>Терминалы (%d):</b>\n", len(terminalIDs)))
		for _, tid := range terminalIDs {
			var name string
			err := b.db.QueryRow(ctx,
				`SELECT name FROM locations WHERE id::text = $1 AND company_id = $2`,
				tid, companyID).Scan(&name)
			if err != nil {
				sb.WriteString(fmt.Sprintf("• <code>%s</code>\n", tid))
			} else {
				sb.WriteString(fmt.Sprintf("• %s (<code>%s</code>)\n", escapeHTML(name), tid))
			}
		}
	}
	b.SendMessage(chatID, sb.String())
}

func (b *Bot) cmdHelp(chatID int64) {
	help := `<b>FoodBI Payment Bot</b>

Команды:
/start API_КЛЮЧ — привязать бот к компании
/locations — список терминалов/локаций
/subscribe ID1,ID2 — подписаться на уведомления
/unsubscribe ID1,ID2 — отписаться (или /unsubscribe all)
/status — текущие подписки
/help — эта справка

Бот отправляет уведомления о неуспешных оплатах за столом и сообщает, когда оплата проходит успешно.`
	b.SendMessage(chatID, help)
}

func (b *Bot) getCompanyForChat(ctx context.Context, chatID int64) (uuid.UUID, error) {
	var companyID uuid.UUID
	err := b.db.QueryRow(ctx,
		`SELECT company_id FROM telegram_bot_links WHERE telegram_chat_id = $1`,
		chatID).Scan(&companyID)
	return companyID, err
}

// GetSubscribedChats returns all active chat IDs subscribed to the given terminal.
func (b *Bot) GetSubscribedChats(ctx context.Context, companyID uuid.UUID, terminalID string) ([]int64, error) {
	rows, err := b.db.Query(ctx,
		`SELECT telegram_chat_id FROM telegram_subscriptions
		 WHERE company_id = $1 AND is_active = true AND $2 = ANY(terminal_ids)`,
		companyID, terminalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []int64
	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			continue
		}
		chats = append(chats, chatID)
	}
	return chats, nil
}

func parseTerminalIDs(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// escapeHTML escapes the 5 chars Telegram HTML mode treats specially.
func escapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return r.Replace(s)
}

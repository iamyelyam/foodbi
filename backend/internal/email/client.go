// Package email implements transactional email delivery via Resend, using the
// outbox pattern. Callers enqueue inside their existing DB transaction
// (see queue.go); a background processor (processor.go) drains the outbox and
// calls the Resend REST API directly (no SDK — raw net/http, mirroring the
// style of internal/ai/openai.go).
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultResendEndpoint = "https://api.resend.com/emails"
	userAgent             = "FoodBI-Backend/1.0 (+https://foodbi.local)"
)

// Client is a minimal Resend REST client. Construct with NewClient.
// A Client with apiKey == "" operates in dry-run mode: Send returns nil
// without making a network call. The processor checks DryRun() to decide
// whether to mark rows as 'sent' or 'dry_run_skipped'.
type Client struct {
	apiKey     string
	from       string
	fromName   string
	endpoint   string
	httpClient *http.Client
}

// NewClient builds a Client. apiKey may be empty in development; in that case
// DryRun() returns true and the processor skips actual sends.
func NewClient(apiKey, from, fromName string) *Client {
	if from == "" {
		from = "noreply@foodbi.local"
	}
	if fromName == "" {
		fromName = "FoodBI"
	}
	return &Client{
		apiKey:     apiKey,
		from:       from,
		fromName:   fromName,
		endpoint:   defaultResendEndpoint,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// DryRun reports whether Send is a no-op because no RESEND_API_KEY is configured.
func (c *Client) DryRun() bool { return c == nil || c.apiKey == "" }

// From returns the formatted From header value ("Name <email>").
func (c *Client) From() string {
	if c == nil {
		return ""
	}
	if c.fromName == "" {
		return c.from
	}
	return fmt.Sprintf("%s <%s>", c.fromName, c.from)
}

// SendError distinguishes retryable vs terminal failures. The processor uses
// IsRetryable to decide between status='retrying' and status='failed'.
type SendError struct {
	StatusCode int
	Body       string
	Retryable  bool
}

func (e *SendError) Error() string {
	return fmt.Sprintf("resend send failed: status=%d retryable=%t body=%s", e.StatusCode, e.Retryable, e.Body)
}

// IsRetryable extracts the retryable flag from err if it is a *SendError.
// Non-SendError errors (e.g. network timeouts) are treated as retryable.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	var se *SendError
	if errors.As(err, &se) {
		return se.Retryable
	}
	// Network / transport errors are retryable.
	return true
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// Send POSTs an email to Resend. In dry-run mode it returns nil immediately.
// 4xx (except 429) are terminal; 5xx and 429 are retryable; network errors are
// retryable.
func (c *Client) Send(ctx context.Context, to, subject, html string) error {
	if c.DryRun() {
		return nil
	}

	body, err := json.Marshal(resendRequest{
		From:    c.From(),
		To:      []string{to},
		Subject: subject,
		HTML:    html,
	})
	if err != nil {
		return fmt.Errorf("marshal resend request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("resend http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	retryable := resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests
	return &SendError{
		StatusCode: resp.StatusCode,
		Body:       string(respBody),
		Retryable:  retryable,
	}
}

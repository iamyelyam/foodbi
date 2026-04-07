package iiko

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	BaseURL         = "https://api-ru.iiko.services/api/1"
	TokenTTL        = 60 * time.Minute
	TokenRefreshAt  = 45 * time.Minute
	MaxRetries      = 3
	RetryBaseDelay  = 1 * time.Second
)

// Client manages authenticated requests to iiko Cloud API v2.
// Each client instance is scoped to a single API key (one per company).
type Client struct {
	apiKey     string
	httpClient *http.Client

	mu           sync.RWMutex
	token        string
	tokenExpiry  time.Time
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Authenticate obtains or refreshes the access token.
func (c *Client) Authenticate(ctx context.Context) error {
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.tokenExpiry.Add(-15*time.Minute)) {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	body, _ := json.Marshal(map[string]string{"apiLogin": c.apiKey})
	req, err := http.NewRequestWithContext(ctx, "POST", BaseURL+"/access_token", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		CorrelationID string `json:"correlationId"`
		Token         string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode auth response: %w", err)
	}

	c.mu.Lock()
	c.token = result.Token
	c.tokenExpiry = time.Now().Add(TokenTTL)
	c.mu.Unlock()

	log.Debug().Str("correlation_id", result.CorrelationID).Msg("iiko: authenticated")
	return nil
}

// doRequest executes an authenticated request with retry and backoff.
func (c *Client) doRequest(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
	if err := c.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			delay := RetryBaseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			log.Warn().Int("attempt", attempt+1).Str("path", path).Msg("iiko: retrying request")
		}

		req, err := http.NewRequestWithContext(ctx, method, BaseURL+path, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		c.mu.RLock()
		req.Header.Set("Authorization", "Bearer "+c.token)
		c.mu.RUnlock()
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusUnauthorized {
			// Token expired, force re-auth
			c.mu.Lock()
			c.token = ""
			c.mu.Unlock()
			if err := c.Authenticate(ctx); err != nil {
				return nil, fmt.Errorf("re-authenticate: %w", err)
			}
			lastErr = fmt.Errorf("token expired, re-authenticated")
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(respBody))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("iiko API error (status %d): %s", resp.StatusCode, string(respBody))
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// Post is a convenience method for POST requests.
func (c *Client) Post(ctx context.Context, path string, payload interface{}) ([]byte, error) {
	return c.doRequest(ctx, "POST", path, payload)
}

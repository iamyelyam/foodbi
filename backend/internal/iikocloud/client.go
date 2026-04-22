package iikocloud

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
	BaseURL        = "https://api-ru.iiko.services"
	MaxRetries     = 3
	RetryBaseDelay = 1 * time.Second
	// Token TTL is 1h; refresh 10 minutes before expiry to avoid races.
	tokenRefreshBefore = 10 * time.Minute
)

// Client manages authenticated requests to the iiko Cloud API.
// A single Client is created per company; its token is refreshed automatically.
type Client struct {
	apiLogin   string
	httpClient *http.Client

	mu          sync.RWMutex
	token       string
	tokenExpiry time.Time
}

// NewClient creates a new iiko Cloud client for the given apiLogin.
func NewClient(apiLogin string) *Client {
	return &Client{
		apiLogin: apiLogin,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Authenticate fetches a fresh token from iiko Cloud.
// Token TTL is 1 hour; we mark expiry at 50 min to refresh before the server expires it.
func (c *Client) Authenticate(ctx context.Context) error {
	body, err := json.Marshal(AuthRequest{APILogin: c.apiLogin})
	if err != nil {
		return fmt.Errorf("marshal auth request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", BaseURL+"/api/1/access_token", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read auth response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var authResp AuthResponse
	if err := json.Unmarshal(respBody, &authResp); err != nil {
		return fmt.Errorf("decode auth response: %w", err)
	}
	if authResp.Token == "" {
		return fmt.Errorf("iiko Cloud returned empty token")
	}

	c.mu.Lock()
	c.token = authResp.Token
	c.tokenExpiry = time.Now().Add(50 * time.Minute) // 1h TTL, refresh at 50min
	c.mu.Unlock()

	log.Debug().Str("apiLogin", c.apiLogin).Msg("iikocloud: authenticated")
	return nil
}

// ensureToken refreshes the token if it is absent or about to expire.
func (c *Client) ensureToken(ctx context.Context) error {
	c.mu.RLock()
	valid := c.token != "" && time.Now().Before(c.tokenExpiry.Add(-tokenRefreshBefore))
	c.mu.RUnlock()
	if valid {
		return nil
	}
	return c.Authenticate(ctx)
}

// doPost executes an authenticated POST request with JSON body.
// Retries up to MaxRetries times on 5xx errors.
func (c *Client) doPost(ctx context.Context, path string, payload interface{}) ([]byte, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, fmt.Errorf("token refresh: %w", err)
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
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
		}

		req, err := http.NewRequestWithContext(ctx, "POST", BaseURL+path, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		c.mu.RLock()
		token := c.token
		c.mu.RUnlock()

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

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
			// Token expired mid-flight — refresh and retry.
			if rerr := c.Authenticate(ctx); rerr != nil {
				return nil, fmt.Errorf("re-auth after 401: %w", rerr)
			}
			lastErr = fmt.Errorf("401 unauthorized, re-authenticated")
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(respBody))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("iiko Cloud error (status %d): %s", resp.StatusCode, string(respBody))
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

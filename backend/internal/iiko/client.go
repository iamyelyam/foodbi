package iiko

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	TokenTTL       = 15 * time.Minute
	MaxRetries     = 3
	RetryBaseDelay = 1 * time.Second
)

// Client manages authenticated requests to iiko Server API (resto).
type Client struct {
	baseURL    string // e.g. "https://palaushy-co.iiko.it"
	login      string
	passSHA1   string
	httpClient *http.Client

	mu          sync.RWMutex
	token       string
	tokenExpiry time.Time
}

func NewClient(baseURL, login, password string) *Client {
	// iiko Server API requires SHA1 hash of password
	h := sha1.Sum([]byte(password))
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		login:    login,
		passSHA1: fmt.Sprintf("%x", h),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Authenticate obtains or refreshes the access token.
func (c *Client) Authenticate(ctx context.Context) error {
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	authURL := fmt.Sprintf("%s/resto/api/auth?login=%s&pass=%s",
		c.baseURL, url.QueryEscape(c.login), url.QueryEscape(c.passSHA1))

	req, err := http.NewRequestWithContext(ctx, "GET", authURL, nil)
	if err != nil {
		return fmt.Errorf("create auth request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	token := strings.TrimSpace(string(body))

	if resp.StatusCode != http.StatusOK || token == "" {
		return fmt.Errorf("auth failed (status %d): %s", resp.StatusCode, token)
	}

	c.mu.Lock()
	c.token = token
	c.tokenExpiry = time.Now().Add(TokenTTL)
	c.mu.Unlock()

	log.Debug().Msg("iiko: authenticated to server API")
	return nil
}

// doGet executes an authenticated GET request with retry.
func (c *Client) doGet(ctx context.Context, path string, params url.Values) ([]byte, error) {
	if err := c.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()

	if params == nil {
		params = url.Values{}
	}
	params.Set("key", token)

	fullURL := fmt.Sprintf("%s%s?%s", c.baseURL, path, params.Encode())

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

		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

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
			c.mu.Lock()
			c.token = ""
			c.mu.Unlock()
			if err := c.Authenticate(ctx); err != nil {
				return nil, fmt.Errorf("re-authenticate: %w", err)
			}
			params.Set("key", c.token)
			fullURL = fmt.Sprintf("%s%s?%s", c.baseURL, path, params.Encode())
			lastErr = fmt.Errorf("token expired, re-authenticated")
			continue
		}

		if resp.StatusCode >= 500 {
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

// doPost executes an authenticated POST request with JSON body and retry.
func (c *Client) doPost(ctx context.Context, path string, payload interface{}) ([]byte, error) {
	if err := c.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	fullURL := fmt.Sprintf("%s%s?key=%s", c.baseURL, path, url.QueryEscape(token))

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

		req, err := http.NewRequestWithContext(ctx, "POST", fullURL, strings.NewReader(string(jsonBody)))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

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
			c.mu.Lock()
			c.token = ""
			c.mu.Unlock()
			if err := c.Authenticate(ctx); err != nil {
				return nil, fmt.Errorf("re-authenticate: %w", err)
			}
			fullURL = fmt.Sprintf("%s%s?key=%s", c.baseURL, path, url.QueryEscape(c.token))
			lastErr = fmt.Errorf("token expired")
			continue
		}

		if resp.StatusCode >= 500 {
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

// Package iikoweb is an HTTP client for iikoWeb-hosted iikoOffice tenants
// (e.g. https://<tenant>.iikoweb.ru). This is a THIRD API surface — distinct
// from iiko Server (/resto/api/*) and iiko Cloud (api-ru.iiko.services):
//
//   - Auth: POST /api/auth/login {login,password} → session cookie
//   - All subsequent calls reuse that cookie via net/http/cookiejar
//   - Re-login automatically if a 401 is seen mid-flight
//
// The data-side endpoints (sales / stock / invoices / OLAP) live in submodules
// of iikoOffice loaded after login; only the navigator/portal endpoints are
// publicly enumerable. Methods on this Client are intentionally minimal until
// we've explored a live session and confirmed the data endpoints — see
// project memory `project_iikoweb_third_api.md`.
package iikoweb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	MaxRetries     = 3
	RetryBaseDelay = 1 * time.Second
)

// Client manages an authenticated session against an iikoWeb tenant.
// One Client per (tenant URL + login) pair. Concurrency-safe via mu.
type Client struct {
	baseURL    string // e.g. "https://youcook-ala.iikoweb.ru"
	login      string
	password   string
	httpClient *http.Client

	mu              sync.RWMutex
	authenticatedAt time.Time
}

// NewClient creates an iikoWeb client for the given tenant URL + credentials.
// A cookie jar is attached so the session cookie returned by /api/auth/login
// is automatically applied to every subsequent request.
func NewClient(baseURL, login, password string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		login:    login,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
	}, nil
}

// BaseURL returns the tenant URL this client targets (no trailing slash).
func (c *Client) BaseURL() string { return c.baseURL }

// Authenticate performs POST /api/auth/login. Sets session cookie via the jar.
// Safe to call repeatedly — used both for initial auth and 401 re-auth.
func (c *Client) Authenticate(ctx context.Context) error {
	body, err := json.Marshal(AuthLoginRequest{Login: c.login, Password: c.password})
	if err != nil {
		return fmt.Errorf("marshal login: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read login response: %w", err)
	}

	// iikoWeb always returns HTTP 200 — success vs failure is signalled in JSON `error` field.
	var authResp AuthLoginResponse
	if jerr := json.Unmarshal(respBody, &authResp); jerr != nil {
		return fmt.Errorf("decode login response (status %d): %w; body=%s", resp.StatusCode, jerr, truncate(string(respBody), 200))
	}
	if authResp.Error {
		msg := authResp.Message
		if msg == "" {
			msg = authResp.ErrorMessage
		}
		return fmt.Errorf("iikoweb login rejected: %s", msg)
	}

	// Verify a session cookie was actually set — otherwise subsequent calls will 401.
	cookieURL, _ := url.Parse(c.baseURL)
	if len(c.httpClient.Jar.Cookies(cookieURL)) == 0 {
		return fmt.Errorf("iikoweb login returned ok but set no cookies")
	}

	c.mu.Lock()
	c.authenticatedAt = time.Now()
	c.mu.Unlock()

	log.Debug().Str("tenant", c.baseURL).Str("login", c.login).Msg("iikoweb: authenticated")
	return nil
}

// ensureAuth performs an initial login if we've never authenticated.
// We don't preemptively refresh — iikoWeb session TTLs vary and cookies are
// renewed by the server on each authenticated call. 401 mid-flight triggers
// re-auth in doRequest.
func (c *Client) ensureAuth(ctx context.Context) error {
	c.mu.RLock()
	authed := !c.authenticatedAt.IsZero()
	c.mu.RUnlock()
	if authed {
		return nil
	}
	return c.Authenticate(ctx)
}

// doRequest is the shared GET/POST helper. Retries on 5xx; auto re-auth on 401.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
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

		var reader io.Reader
		if bodyBytes != nil {
			reader = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
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
			// Session expired — re-login and retry.
			if rerr := c.Authenticate(ctx); rerr != nil {
				return nil, fmt.Errorf("re-auth after 401: %w", rerr)
			}
			lastErr = fmt.Errorf("401 unauthorized, re-authenticated")
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error (status %d): %s", resp.StatusCode, truncate(string(respBody), 300))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("iikoweb error (status %d): %s", resp.StatusCode, truncate(string(respBody), 300))
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doGet executes an authenticated GET request.
func (c *Client) doGet(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, "GET", path, nil)
}

// doPost executes an authenticated POST request with JSON body.
func (c *Client) doPost(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequest(ctx, "POST", path, body)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

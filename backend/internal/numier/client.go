package numier

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	BaseURL        = "https://www.numier.com/api/public/index.php/api"
	MaxRetries     = 3
	RetryBaseDelay = 1 * time.Second
)

// Client manages authenticated requests to NUMIER API.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: BaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest executes an authenticated GET request with apikey as header and filters as query params.
// NUMIER API: apikey is a header; start_date, end_date, pag are query parameters.
func (c *Client) doRequest(ctx context.Context, path string, params map[string]string) ([]byte, error) {
	fullURL := c.baseURL + path
	if len(params) > 0 {
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		fullURL = fullURL + "?" + q.Encode()
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

		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("apikey", c.apiKey)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("numier API not found (404): %s", path)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("numier API error (status %d): %s", resp.StatusCode, string(body))
		}

		return body, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// Validate checks if the API key works by fetching locales.
func (c *Client) Validate(ctx context.Context) error {
	locales, err := c.GetLocales(ctx)
	if err != nil {
		return fmt.Errorf("validate API key: %w", err)
	}
	if len(locales) == 0 {
		return fmt.Errorf("API key returned no establishments")
	}
	log.Debug().Int("establishments", len(locales)).Msg("numier: API key validated")
	return nil
}

// parseResponse unmarshals the standard NUMIER API response wrapper.
func parseResponse[T any](data []byte) (T, int, error) {
	var resp APIResponse[T]
	if err := json.Unmarshal(data, &resp); err != nil {
		var zero T
		return zero, 0, fmt.Errorf("decode response: %w", err)
	}
	if !resp.Response {
		var zero T
		return zero, 0, fmt.Errorf("numier API returned response=false: %s", resp.Message)
	}
	return resp.Result, resp.TotalPages, nil
}

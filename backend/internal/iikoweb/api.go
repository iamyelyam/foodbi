package iikoweb

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetAuthStatus calls GET /api/auth — returns tenant info + whether current
// session is authorized. Useful as a sanity probe and to read the iikoWeb
// version (AppVersion field in the response).
//
// This endpoint is unauthenticated; callable before login.
func (c *Client) GetAuthStatus(ctx context.Context) (*AuthStatusResponse, error) {
	// Bypass ensureAuth — this endpoint is the auth probe itself.
	req, err := c.httpClient.Get(c.baseURL + "/api/auth")
	if err != nil {
		return nil, fmt.Errorf("get auth status: %w", err)
	}
	defer req.Body.Close()

	var status AuthStatusResponse
	if err := json.NewDecoder(req.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode auth status: %w", err)
	}
	return &status, nil
}

// GetStores fetches the tenant's store list. Requires an authenticated session.
// Verified to exist in the navigator SPA chunks (path: /api/stores/list).
func (c *Client) GetStores(ctx context.Context) ([]Store, error) {
	data, err := c.doGet(ctx, "/api/stores/list")
	if err != nil {
		return nil, fmt.Errorf("get stores: %w", err)
	}
	// Response shape is best-effort — accept either {stores:[...]} or a bare array.
	var wrapped StoresListResponse
	if err := json.Unmarshal(data, &wrapped); err == nil && len(wrapped.Stores) > 0 {
		return wrapped.Stores, nil
	}
	var bare []Store
	if err := json.Unmarshal(data, &bare); err == nil {
		return bare, nil
	}
	return nil, fmt.Errorf("unexpected stores response shape: %s", truncate(string(data), 300))
}

// GetKpiMetricStores returns the raw KPI metric payload for stores. Schema TBD —
// stored as a generic map so callers can introspect during reverse-engineering.
func (c *Client) GetKpiMetricStores(ctx context.Context) (KpiMetricStoresResponse, error) {
	data, err := c.doGet(ctx, "/api/kpi-metric/stores")
	if err != nil {
		return nil, fmt.Errorf("get kpi metric stores: %w", err)
	}
	var resp KpiMetricStoresResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode kpi metric stores: %w", err)
	}
	return resp, nil
}

// ProbeEndpoint is a generic GET helper for endpoint discovery. Returns raw bytes
// + HTTP status. Used by `cmd/probe-iikoweb` to enumerate iikoOffice submodule
// endpoints (sales, stock, invoices) under a live session.
//
// Once an endpoint is confirmed, promote it to a typed method on Client.
func (c *Client) ProbeEndpoint(ctx context.Context, path string) ([]byte, error) {
	return c.doGet(ctx, path)
}

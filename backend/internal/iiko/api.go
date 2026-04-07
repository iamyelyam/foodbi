package iiko

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetOrganizations returns all organizations available to this API key.
func (c *Client) GetOrganizations(ctx context.Context) ([]Organization, error) {
	payload := map[string]interface{}{
		"organizationIds": nil,
		"returnAdditionalInfo": false,
		"includeDisabled": false,
	}

	data, err := c.Post(ctx, "/organizations", payload)
	if err != nil {
		return nil, fmt.Errorf("get organizations: %w", err)
	}

	var resp OrganizationsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode organizations: %w", err)
	}

	return resp.Organizations, nil
}

// GetOLAPReport fetches an OLAP report (revenue, orders, products).
func (c *Client) GetOLAPReport(ctx context.Context, req OLAPReportRequest) (*OLAPReportResponse, error) {
	data, err := c.Post(ctx, "/reports/olap", req)
	if err != nil {
		return nil, fmt.Errorf("get OLAP report: %w", err)
	}

	var resp OLAPReportResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode OLAP report: %w", err)
	}

	return &resp, nil
}

// GetPurchaseInvoices fetches purchase invoices for an organization.
func (c *Client) GetPurchaseInvoices(ctx context.Context, orgID, dateFrom, dateTo string) ([]PurchaseInvoice, error) {
	payload := map[string]interface{}{
		"organizationId": orgID,
		"dateFrom":       dateFrom,
		"dateTo":         dateTo,
	}

	data, err := c.Post(ctx, "/documents/purchase_invoice", payload)
	if err != nil {
		return nil, fmt.Errorf("get purchase invoices: %w", err)
	}

	var resp PurchaseInvoicesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode purchase invoices: %w", err)
	}

	return resp.Documents, nil
}

// GetStockBalance fetches current stock levels for an organization.
func (c *Client) GetStockBalance(ctx context.Context, orgID string) ([]StockItem, error) {
	payload := map[string]interface{}{
		"organizationId": orgID,
	}

	data, err := c.Post(ctx, "/resto/api/v2/entities/products/list", payload)
	if err != nil {
		return nil, fmt.Errorf("get stock balance: %w", err)
	}

	var resp StockBalanceResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode stock balance: %w", err)
	}

	return resp.Items, nil
}

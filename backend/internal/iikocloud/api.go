package iikocloud

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GetOrganizations fetches all organizations accessible with this API login.
func (c *Client) GetOrganizations(ctx context.Context) ([]Organization, error) {
	data, err := c.doPost(ctx, "/api/1/organizations", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("get organizations: %w", err)
	}
	var resp OrganizationsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode organizations: %w", err)
	}
	return resp.Organizations, nil
}

// GetTerminalGroups fetches terminal groups for the given organization IDs.
func (c *Client) GetTerminalGroups(ctx context.Context, orgIDs []string) ([]TerminalGroupEntry, error) {
	data, err := c.doPost(ctx, "/api/1/terminal_groups", TerminalGroupsRequest{OrganizationIDs: orgIDs})
	if err != nil {
		return nil, fmt.Errorf("get terminal groups: %w", err)
	}
	var resp TerminalGroupsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode terminal groups: %w", err)
	}
	return resp.TerminalGroups, nil
}

// GetNomenclature fetches the product catalog for an organization.
func (c *Client) GetNomenclature(ctx context.Context, orgID string) (*NomenclatureResponse, error) {
	data, err := c.doPost(ctx, "/api/1/nomenclature", NomenclatureRequest{OrganizationID: orgID})
	if err != nil {
		return nil, fmt.Errorf("get nomenclature: %w", err)
	}
	var resp NomenclatureResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode nomenclature: %w", err)
	}
	return &resp, nil
}

// GetOLAPReport fetches an OLAP sales report, paginating automatically (10 000 rows/page).
// RULES:
//   - ALWAYS include DishName in GroupByRowFields (iiko OLAP returns one arbitrary row per
//     order otherwise, not the order total).
//   - NEVER divide DishSumInt by 100 — values are already in KZT.
//   - SUM per UniqOrderId.Id in Go before upserting to the DB.
func (c *Client) GetOLAPReport(ctx context.Context, req OLAPReportRequest) ([]map[string]interface{}, error) {
	if req.PaginatorItemsOnPage == 0 {
		req.PaginatorItemsOnPage = 10000
	}
	req.PaginatorPage = 0

	var allRows []map[string]interface{}
	for {
		data, err := c.doPost(ctx, "/api/1/reports/olap", req)
		if err != nil {
			return allRows, fmt.Errorf("get OLAP page %d: %w", req.PaginatorPage, err)
		}
		var resp OLAPReportResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return allRows, fmt.Errorf("decode OLAP response: %w", err)
		}
		allRows = append(allRows, resp.Data...)
		if len(resp.Data) < req.PaginatorItemsOnPage {
			break
		}
		req.PaginatorPage++
	}
	return allRows, nil
}

// GetStoreBalance fetches current stock balance snapshot for the given organization IDs.
func (c *Client) GetStoreBalance(ctx context.Context, orgIDs []string) ([]StoreBalanceItem, error) {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000")
	data, err := c.doPost(ctx, "/api/1/reports/balance/stores", BalanceStoresRequest{
		OrganizationIDs: orgIDs,
		Timestamp:       timestamp,
	})
	if err != nil {
		return nil, fmt.Errorf("get store balance: %w", err)
	}
	var resp BalanceStoresResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode store balance: %w", err)
	}
	return resp.Balance, nil
}

package iiko

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"time"
)

// GetOLAPReport fetches an OLAP report from iiko Server API.
// Returns rows as []map[string]interface{} (iiko Server returns named fields, not arrays).
func (c *Client) GetOLAPReport(ctx context.Context, req OLAPReportRequest) (*OLAPReportResponse, error) {
	data, err := c.doPost(ctx, "/resto/api/v2/reports/olap", req)
	if err != nil {
		return nil, fmt.Errorf("get OLAP report: %w", err)
	}

	var resp OLAPReportResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode OLAP report: %w", err)
	}

	return &resp, nil
}

// GetPurchaseInvoices fetches incoming invoices from iiko Server API (XML endpoint).
func (c *Client) GetPurchaseInvoices(ctx context.Context, dateFrom, dateTo string) ([]PurchaseInvoice, error) {
	params := url.Values{
		"from": {dateFrom},
		"to":   {dateTo},
	}

	data, err := c.doGet(ctx, "/resto/api/documents/export/incomingInvoice", params)
	if err != nil {
		return nil, fmt.Errorf("get purchase invoices: %w", err)
	}

	// Parse XML response
	var xmlResp struct {
		XMLName   xml.Name `xml:"incomingInvoiceDtoes"`
		Documents []struct {
			ID             string `xml:"id"`
			IncomingDate   string `xml:"incomingDate"`
			DateIncoming   string `xml:"dateIncoming"`
			DocumentNumber string `xml:"documentNumber"`
			Supplier       string `xml:"supplier"`
			DefaultStore   string `xml:"defaultStore"`
			Status         string `xml:"status"`
			Items          []struct {
				Product string  `xml:"product"`
				Amount  float64 `xml:"amount"`
				Price   float64 `xml:"price"`
				Sum     float64 `xml:"sum"`
				Store   string  `xml:"store"`
				Code    string  `xml:"code"`
			} `xml:"items>item"`
		} `xml:"document"`
	}

	if err := xml.Unmarshal(data, &xmlResp); err != nil {
		return nil, fmt.Errorf("decode invoices XML: %w", err)
	}

	var invoices []PurchaseInvoice
	for _, doc := range xmlResp.Documents {
		inDate, _ := time.Parse("2006-01-02", doc.IncomingDate)
		var totalSum float64
		for _, item := range doc.Items {
			totalSum += item.Sum
		}
		invoices = append(invoices, PurchaseInvoice{
			ID:             doc.ID,
			IncomingDate:   inDate,
			DocumentNumber: doc.DocumentNumber,
			SupplierID:     doc.Supplier,
			SupplierName:   doc.Supplier, // Server API returns supplier ID, name resolved separately
			Status:         doc.Status,
			Sum:            totalSum,
		})
	}

	return invoices, nil
}

// GetStockBalance fetches current stock levels from iiko Server API.
func (c *Client) GetStockBalance(ctx context.Context) ([]StockItem, error) {
	timestamp := time.Now().Format("2006-01-02") + "T00:00:00.000"
	params := url.Values{
		"timestamp": {timestamp},
	}

	data, err := c.doGet(ctx, "/resto/api/v2/reports/balance/stores", params)
	if err != nil {
		return nil, fmt.Errorf("get stock balance: %w", err)
	}

	var rows []struct {
		Store   string  `json:"store"`
		Product string  `json:"product"`
		Amount  float64 `json:"amount"`
		Sum     float64 `json:"sum"`
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("decode stock balance: %w", err)
	}

	var items []StockItem
	for _, r := range rows {
		items = append(items, StockItem{
			ProductID:   r.Product,
			ProductName: r.Product, // Product ID, name resolved via nomenclature
			StoreID:     r.Store,
			Amount:      r.Amount,
			Sum:         r.Sum,
			Unit:        "",
		})
	}

	return items, nil
}

// GetNomenclature fetches product names and categories from iiko Server API.
func (c *Client) GetNomenclature(ctx context.Context) (map[string]ProductInfo, error) {
	data, err := c.doGet(ctx, "/resto/api/v2/entities/products/list", nil)
	if err != nil {
		return nil, fmt.Errorf("get nomenclature: %w", err)
	}

	var products []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Category string `json:"category"`
		Group    string `json:"group"`
		Unit     string `json:"mainUnit"`
		Type     string `json:"type"`
	}
	if err := json.Unmarshal(data, &products); err != nil {
		return nil, fmt.Errorf("decode nomenclature: %w", err)
	}

	result := make(map[string]ProductInfo, len(products))
	for _, p := range products {
		result[p.ID] = ProductInfo{
			ID:       p.ID,
			Name:     p.Name,
			Category: p.Category,
			Group:    p.Group,
			Unit:     p.Unit,
			Type:     p.Type,
		}
	}

	return result, nil
}

// GetSuppliers fetches supplier list from iiko Server API.
func (c *Client) GetSuppliers(ctx context.Context) (map[string]string, error) {
	data, err := c.doGet(ctx, "/resto/api/v2/entities/suppliers/list", nil)
	if err != nil {
		return nil, fmt.Errorf("get suppliers: %w", err)
	}

	var suppliers []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &suppliers); err != nil {
		return nil, fmt.Errorf("decode suppliers: %w", err)
	}

	result := make(map[string]string, len(suppliers))
	for _, s := range suppliers {
		result[s.ID] = s.Name
	}

	return result, nil
}

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
		items := make([]PurchaseInvoiceItem, 0, len(doc.Items))
		for _, item := range doc.Items {
			totalSum += item.Sum
			items = append(items, PurchaseInvoiceItem{
				ProductID:   item.Product,
				ProductName: item.Product, // GUID — resolved via nomenclature in sync
				Code:        item.Code,
				Amount:      item.Amount,
				Price:       item.Price,
				Sum:         item.Sum,
			})
		}
		invoices = append(invoices, PurchaseInvoice{
			ID:             doc.ID,
			IncomingDate:   inDate,
			DocumentNumber: doc.DocumentNumber,
			SupplierID:     doc.Supplier,
			SupplierName:   doc.Supplier, // Server API returns supplier ID, name resolved separately
			Status:         doc.Status,
			Sum:            totalSum,
			Items:          items,
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
// Tries XML v1 first (returns unit as readable string), falls back to JSON v2 (unit is GUID).
func (c *Client) GetNomenclature(ctx context.Context) (map[string]ProductInfo, error) {
	// Try XML v1 — returns mainUnit as text like "кг", "шт"
	if data, err := c.doGet(ctx, "/resto/api/products", nil); err == nil {
		var xmlResp struct {
			XMLName  xml.Name `xml:"products"`
			Products []struct {
				ID       string `xml:"id"`
				Name     string `xml:"name"`
				Category string `xml:"category"`
				Group    string `xml:"group"`
				Unit     string `xml:"mainUnit"`
				Type     string `xml:"type"`
			} `xml:"product"`
		}
		if err := xml.Unmarshal(data, &xmlResp); err == nil && len(xmlResp.Products) > 0 {
			result := make(map[string]ProductInfo, len(xmlResp.Products))
			for _, p := range xmlResp.Products {
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
	}

	// Fallback to JSON v2
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

// GetMeasureUnits fetches measure unit dictionary from iiko (GUID -> name like "кг", "шт", "л").
// Tries multiple known endpoints across iiko versions.
func (c *Client) GetMeasureUnits(ctx context.Context) (map[string]string, error) {
	endpoints := []string{
		"/resto/api/v2/entities/measureUnits/list",
		"/resto/api/v2/entities/measure-units/list",
		"/resto/api/v2/entities/measure/list",
	}

	for _, ep := range endpoints {
		data, err := c.doGet(ctx, ep, nil)
		if err != nil {
			continue
		}
		var units []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Code string `json:"code"`
		}
		if err := json.Unmarshal(data, &units); err != nil {
			continue
		}
		if len(units) == 0 {
			continue
		}
		result := make(map[string]string, len(units))
		for _, u := range units {
			if u.Name != "" {
				result[u.ID] = u.Name
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("no measure units endpoint matched")
}

// GetAssemblyChart fetches the technological card (recipe) for a single dish.
// Endpoint: /resto/api/v2/assemblyCharts/getPrepared (the only working variant on
// iiko Server v8+ for this restaurant; getRequired and others return 404).
// Date is required by iiko (@NotNull) and must be plain YYYY-MM-DD (no time part).
//
// Returns the items of the currently active prepared chart, or nil if the dish has
// no recipe defined (e.g. resold goods like beverages). Picks the chart whose
// dateFrom <= today AND (dateTo == nil OR dateTo > today). If none match, returns nil.
func (c *Client) GetAssemblyChart(ctx context.Context, dishID string) ([]RecipeComponent, error) {
	if dishID == "" {
		return nil, fmt.Errorf("dishID required")
	}
	today := time.Now().Format("2006-01-02")
	params := url.Values{
		"productId": {dishID},
		"date":      {today},
	}
	data, err := c.doGet(ctx, "/resto/api/v2/assemblyCharts/getPrepared", params)
	if err != nil {
		return nil, fmt.Errorf("get assembly chart: %w", err)
	}
	var resp preparedChartResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode assembly chart: %w", err)
	}
	if len(resp.PreparedCharts) == 0 {
		return nil, nil
	}

	// Pick the active chart (dateFrom <= today AND (dateTo nil OR > today)).
	// iiko returns multiple historical revisions; the active one is unambiguous.
	var active *struct {
		AssembledProductID string `json:"assembledProductId"`
		DateFrom           string `json:"dateFrom"`
		DateTo             *string `json:"dateTo"`
		Items              []struct {
			ProductID string  `json:"productId"`
			Amount    float64 `json:"amount"`
		} `json:"items"`
	}
	for i := range resp.PreparedCharts {
		ch := &resp.PreparedCharts[i]
		from, _ := time.Parse("2006-01-02", ch.DateFrom)
		if !from.IsZero() && from.After(time.Now()) {
			continue
		}
		if ch.DateTo != nil && *ch.DateTo != "" {
			to, _ := time.Parse("2006-01-02", *ch.DateTo)
			if !to.IsZero() && !to.After(time.Now()) {
				continue
			}
		}
		active = ch
		break
	}
	if active == nil {
		return nil, nil
	}

	out := make([]RecipeComponent, 0, len(active.Items))
	for _, it := range active.Items {
		if it.ProductID == "" || it.Amount <= 0 {
			continue
		}
		out = append(out, RecipeComponent{IngredientID: it.ProductID, Amount: it.Amount})
	}
	return out, nil
}

// GetSuppliers fetches supplier list from iiko Server API.
// Tries multiple endpoints since different iiko Server versions expose suppliers differently.
func (c *Client) GetSuppliers(ctx context.Context) (map[string]string, error) {
	// First try the XML employees endpoint — suppliers are often stored as employees with "Supplier" role.
	data, err := c.doGet(ctx, "/resto/api/employees", nil)
	if err == nil {
		var xmlResp struct {
			XMLName   xml.Name `xml:"employees"`
			Employees []struct {
				ID   string `xml:"id"`
				Name string `xml:"name"`
				Code string `xml:"code"`
				Type string `xml:"type"` // "SUPPLIER", "EMPLOYEE", etc.
			} `xml:"employee"`
		}
		if xmlErr := xml.Unmarshal(data, &xmlResp); xmlErr == nil {
			result := make(map[string]string, len(xmlResp.Employees))
			for _, e := range xmlResp.Employees {
				if e.Name != "" {
					result[e.ID] = e.Name
				}
			}
			if len(result) > 0 {
				return result, nil
			}
		}
	}

	// Fallback to JSON v2 endpoint
	data, err = c.doGet(ctx, "/resto/api/v2/entities/suppliers/list", nil)
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

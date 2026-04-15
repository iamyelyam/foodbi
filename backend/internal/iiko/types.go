package iiko

import "time"

// Organization represents an iiko organization (restaurant location).
type Organization struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Country           string `json:"country"`
	RestaurantAddress string `json:"restaurantAddress"`
}

// OLAP report types for iiko Server API.
type OLAPReportRequest struct {
	ReportType       string            `json:"reportType"`
	GroupByRowFields []string          `json:"groupByRowFields"`
	GroupByColFields []string          `json:"groupByColFields"`
	AggregateFields  []string          `json:"aggregateFields"`
	Filters          map[string]interface{} `json:"filters,omitempty"`

	// Cloud API fields (kept for compatibility)
	OrganizationID string `json:"organizationId,omitempty"`
	DateFrom       string `json:"dateFrom,omitempty"`
	DateTo         string `json:"dateTo,omitempty"`
}

// OLAPReportResponse — iiko Server returns rows as map[string]interface{}.
type OLAPReportResponse struct {
	Data []map[string]interface{} `json:"data"`
}

// Revenue data normalized for FoodBI.
type RevenueRecord struct {
	OrderID    string    `json:"order_id"`
	LocationID string    `json:"location_id"`
	OrderDate  time.Time `json:"order_date"`
	Revenue    float64   `json:"revenue"`
	Discount   float64   `json:"discount"`
	OrderType  string    `json:"order_type"`
	Status     string    `json:"status"`
	ItemCount  int       `json:"item_count"`
	WaiterName string    `json:"waiter_name"`
}

// Product sales data.
type ProductSalesRecord struct {
	ProductID   string    `json:"product_id"`
	ProductName string    `json:"product_name"`
	Category    string    `json:"category"`
	LocationID  string    `json:"location_id"`
	Date        time.Time `json:"date"`
	Quantity    float64   `json:"quantity"`
	Revenue     float64   `json:"revenue"`
	CostPrice   float64   `json:"cost_price"`
}

// Purchase invoice from iiko.
type PurchaseInvoice struct {
	ID             string              `json:"id"`
	IncomingDate   time.Time           `json:"incomingDate"`
	DocumentNumber string              `json:"documentNumber"`
	SupplierID     string              `json:"supplierId"`
	SupplierName   string              `json:"supplierName"`
	StoreID        string              `json:"storeId"`
	Status         string              `json:"status"`
	Sum            float64             `json:"sum"`
	Items          []PurchaseInvoiceItem `json:"items"`
}

// PurchaseInvoiceItem — single line from iiko invoice XML.
type PurchaseInvoiceItem struct {
	ProductID   string  `json:"productId"` // iiko product GUID
	ProductName string  `json:"productName"`
	Code        string  `json:"code"`
	Amount      float64 `json:"amount"`
	Price       float64 `json:"price"`
	Sum         float64 `json:"sum"`
}

// Stock balance from iiko.
type StockItem struct {
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	StoreID     string  `json:"storeId"`
	Amount      float64 `json:"amount"`
	Sum         float64 `json:"sum"`
	Unit        string  `json:"unit"`
}

// ProductInfo from iiko nomenclature.
type ProductInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Group    string `json:"group"`
	Unit     string `json:"unit"`
	Type     string `json:"type"`
}

// RecipeComponent — one ingredient line from an iiko assembly/prepared chart.
// amount = quantity of this ingredient per 1 unit of the parent dish.
type RecipeComponent struct {
	IngredientID string  `json:"ingredient_id"`
	Amount       float64 `json:"amount"`
}

// preparedChartResponse — JSON shape returned by /resto/api/v2/assemblyCharts/getPrepared.
// We only model the fields we actually consume.
type preparedChartResponse struct {
	PreparedCharts []struct {
		AssembledProductID string `json:"assembledProductId"`
		DateFrom           string `json:"dateFrom"`
		DateTo             *string `json:"dateTo"`
		Items              []struct {
			ProductID string  `json:"productId"`
			Amount    float64 `json:"amount"`
		} `json:"items"`
	} `json:"preparedCharts"`
}

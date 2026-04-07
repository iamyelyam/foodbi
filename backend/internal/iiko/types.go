package iiko

import "time"

// Organization represents an iiko organization (restaurant location).
type Organization struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Country             string `json:"country"`
	RestaurantAddress   string `json:"restaurantAddress"`
	UseUaeAccountingSystem bool `json:"useUaeAccountingSystem"`
}

type OrganizationsResponse struct {
	CorrelationID string         `json:"correlationId"`
	Organizations []Organization `json:"organizations"`
}

// OLAP report types for revenue/orders data.
type OLAPReportRequest struct {
	OrganizationID string   `json:"organizationId"`
	DateFrom       string   `json:"dateFrom"`      // "2024-01-01"
	DateTo         string   `json:"dateTo"`         // "2024-01-31"
	GroupByRowFields []string `json:"groupByRowFields"`
	GroupByColFields []string `json:"groupByColFields"`
	AggregateFields  []string `json:"aggregateFields"`
	Filters        interface{} `json:"filters,omitempty"`
}

type OLAPReportResponse struct {
	CorrelationID string          `json:"correlationId"`
	Data          [][]interface{} `json:"data"`
	Columns       []OLAPColumn    `json:"columns"`
}

type OLAPColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Revenue data normalized for FoodBI.
type RevenueRecord struct {
	OrderID      string    `json:"order_id"`
	LocationID   string    `json:"location_id"`
	OrderDate    time.Time `json:"order_date"`
	Revenue      float64   `json:"revenue"`
	Discount     float64   `json:"discount"`
	OrderType    string    `json:"order_type"`
	Status       string    `json:"status"`
	ItemCount    int       `json:"item_count"`
	WaiterName   string    `json:"waiter_name"`
}

// Product sales data.
type ProductSalesRecord struct {
	ProductID    string    `json:"product_id"`
	ProductName  string    `json:"product_name"`
	Category     string    `json:"category"`
	LocationID   string    `json:"location_id"`
	Date         time.Time `json:"date"`
	Quantity     float64   `json:"quantity"`
	Revenue      float64   `json:"revenue"`
	CostPrice    float64   `json:"cost_price"`
}

// Purchase invoice from iiko.
type PurchaseInvoice struct {
	ID             string    `json:"id"`
	IncomingDate   time.Time `json:"incomingDate"`
	DocumentNumber string    `json:"documentNumber"`
	SupplierID     string    `json:"supplierId"`
	SupplierName   string    `json:"supplierName"`
	StoreID        string    `json:"storeId"`
	Status         string    `json:"status"`
	Sum            float64   `json:"sum"`
}

type PurchaseInvoicesResponse struct {
	CorrelationID string            `json:"correlationId"`
	Documents     []PurchaseInvoice `json:"documents"`
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

type StockBalanceResponse struct {
	CorrelationID string      `json:"correlationId"`
	Items         []StockItem `json:"items"`
}

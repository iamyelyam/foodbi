package iikocloud

// AuthRequest is sent to POST /api/1/access_token.
type AuthRequest struct {
	APILogin string `json:"apiLogin"`
}

// AuthResponse is the token response from iiko Cloud.
type AuthResponse struct {
	CorrelationID string `json:"correlationId"`
	Token         string `json:"token"`
}

// Organization from iiko Cloud.
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OrganizationsResponse from POST /api/1/organizations.
type OrganizationsResponse struct {
	CorrelationID string         `json:"correlationId"`
	Organizations []Organization `json:"organizations"`
}

// TerminalGroupsRequest is sent to POST /api/1/terminal_groups.
type TerminalGroupsRequest struct {
	OrganizationIDs []string `json:"organizationIds"`
}

// TerminalGroupItem is a single terminal group.
type TerminalGroupItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"isActive"`
}

// TerminalGroupEntry groups terminal items under an organization.
type TerminalGroupEntry struct {
	OrganizationID string              `json:"organizationId"`
	Items          []TerminalGroupItem `json:"items"`
}

// TerminalGroupsResponse from POST /api/1/terminal_groups.
type TerminalGroupsResponse struct {
	CorrelationID  string               `json:"correlationId"`
	TerminalGroups []TerminalGroupEntry `json:"terminalGroups"`
}

// NomenclatureRequest is sent to POST /api/1/nomenclature.
type NomenclatureRequest struct {
	OrganizationID string `json:"organizationId"`
}

// NomenclatureGroup is a product group (category).
type NomenclatureGroup struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ParentID *string `json:"parentId,omitempty"`
}

// NomenclatureProduct from iiko Cloud nomenclature.
type NomenclatureProduct struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Code          string  `json:"code"`
	Type          string  `json:"type"` // "Dish", "Modifier", "Good"
	GroupID       string  `json:"groupId"`
	MeasureUnitID string  `json:"measureUnitId"`
}

// NomenclatureResponse from POST /api/1/nomenclature.
type NomenclatureResponse struct {
	CorrelationID string                `json:"correlationId"`
	Products      []NomenclatureProduct `json:"products"`
	Groups        []NomenclatureGroup   `json:"groups"`
}

// OLAPReportRequest is sent to POST /api/1/reports/olap.
// PaginatorItemsOnPage / PaginatorPage control pagination (10 000 rows per page).
//
// CLAUDE.md rules:
//   - ALWAYS include DishName in GroupByRowFields
//   - DishSumInt is already in KZT — NEVER divide by 100
//   - SUM per UniqOrderId.Id in Go after fetch
type OLAPReportRequest struct {
	OrganizationID       string                 `json:"organizationId"`
	ReportType           string                 `json:"reportType"`
	BuildSummary         bool                   `json:"buildSummary"`
	GroupByRowFields     []string               `json:"groupByRowFields"`
	AggregateFields      []string               `json:"aggregateFields"`
	Filters              map[string]interface{} `json:"filters,omitempty"`
	PaginatorPage        int                    `json:"paginatorPage"`
	PaginatorItemsOnPage int                    `json:"paginatorItemsOnPage"`
}

// OLAPReportResponse from POST /api/1/reports/olap.
type OLAPReportResponse struct {
	CorrelationID string                   `json:"correlationId"`
	Data          []map[string]interface{} `json:"data"`
}

// BalanceStoresRequest is sent to POST /api/1/reports/balance/stores.
type BalanceStoresRequest struct {
	OrganizationIDs []string `json:"organizationIds"`
	Timestamp       string   `json:"timestamp,omitempty"`
}

// StoreBalanceItem is a single product balance line from the stores balance report.
type StoreBalanceItem struct {
	StoreID     string  `json:"storeId"`
	StoreName   string  `json:"storeName"`
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	Amount      float64 `json:"amount"`
	Sum         float64 `json:"sum"`
	MeasureUnit string  `json:"measureUnit"`
}

// BalanceStoresResponse from POST /api/1/reports/balance/stores.
type BalanceStoresResponse struct {
	CorrelationID string             `json:"correlationId"`
	Balance       []StoreBalanceItem `json:"balance"`
}

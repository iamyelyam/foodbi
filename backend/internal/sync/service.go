package sync

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/foodbi/backend/internal/iiko"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// CompanySync holds the info needed to sync one company.
type CompanySync struct {
	CompanyID    uuid.UUID
	IikoURL      string
	IikoLogin    string
	IikoPassword string
	Locations    []LocationSync
}

type LocationSync struct {
	LocationID uuid.UUID
	IikoOrgID  string
	Name       string
}

// getDeptName returns the location name for use as an OLAP Department filter.
// Only returns a name when the location has a real iiko_org_id (different from its own id).
// Single-location setups or legacy locations (iiko_org_id = own id) get no filter.
func (s *Service) getDeptName(ctx context.Context, locationID uuid.UUID) string {
	var name string
	var iikoOrgID *string
	err := s.db.QueryRow(ctx, `SELECT name, iiko_org_id FROM locations WHERE id = $1`, locationID).Scan(&name, &iikoOrgID)
	if err != nil || iikoOrgID == nil || *iikoOrgID == "" {
		return ""
	}
	// Skip filter if iiko_org_id equals location's own id (legacy/fake value)
	if *iikoOrgID == locationID.String() {
		return ""
	}
	return name
}

// GetCompaniesToSync fetches all companies with iiko credentials configured.
func (s *Service) GetCompaniesToSync(ctx context.Context) ([]CompanySync, error) {
	rows, err := s.db.Query(ctx,
		`SELECT c.id, c.iiko_server_url, c.iiko_login, c.iiko_password, l.id, COALESCE(l.iiko_org_id, ''), l.name
		 FROM companies c
		 JOIN locations l ON l.company_id = c.id
		 WHERE c.iiko_server_url IS NOT NULL AND c.iiko_server_url != ''
		   AND c.iiko_login IS NOT NULL AND c.iiko_login != ''
		   AND COALESCE(l.pos_system, '') <> 'numier'
		 ORDER BY c.id, l.id`)
	if err != nil {
		return nil, fmt.Errorf("query companies: %w", err)
	}
	defer rows.Close()

	companyMap := make(map[uuid.UUID]*CompanySync)
	var order []uuid.UUID

	for rows.Next() {
		var cid, lid uuid.UUID
		var iikoURL, iikoLogin, iikoPass, orgID, locName string
		if err := rows.Scan(&cid, &iikoURL, &iikoLogin, &iikoPass, &lid, &orgID, &locName); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		cs, ok := companyMap[cid]
		if !ok {
			cs = &CompanySync{CompanyID: cid, IikoURL: iikoURL, IikoLogin: iikoLogin, IikoPassword: iikoPass}
			companyMap[cid] = cs
			order = append(order, cid)
		}
		cs.Locations = append(cs.Locations, LocationSync{
			LocationID: lid, IikoOrgID: orgID, Name: locName,
		})
	}

	result := make([]CompanySync, 0, len(order))
	for _, cid := range order {
		cs := companyMap[cid]
		// Prevent duplicate data: if this company has any multi-department locations
		// (with iiko_org_id set), skip the legacy locations (without iiko_org_id).
		// Otherwise one iiko server's orders would be written to both places.
		hasMultiDept := false
		for _, loc := range cs.Locations {
			if loc.IikoOrgID != "" {
				hasMultiDept = true
				break
			}
		}
		if hasMultiDept {
			filtered := make([]LocationSync, 0, len(cs.Locations))
			for _, loc := range cs.Locations {
				if loc.IikoOrgID != "" {
					filtered = append(filtered, loc)
				}
			}
			cs.Locations = filtered
		}
		result = append(result, *cs)
	}
	return result, nil
}

// GetString safely extracts a string from an OLAP row map.
func GetString(row map[string]interface{}, key string) string {
	v, ok := row[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// GetFloat safely extracts a float64 from an OLAP row map.
func GetFloat(row map[string]interface{}, key string) float64 {
	v, ok := row[key]
	if !ok || v == nil {
		return 0
	}
	f, ok := v.(float64)
	if ok {
		return f
	}
	// JSON numbers might be decoded as json.Number
	return 0
}

// OrderAgg accumulates per-dish OLAP rows into per-order totals.
type OrderAgg struct {
	OrderDate   string
	OrderNumber string
	OrderType   string // normalized: "delivery" / "takeaway" / "dine-in"
	WaiterName  string
	Revenue     float64
	Discount    float64
	ItemCount   int
}

// Known iiko system measure-unit GUIDs (derived empirically from this iiko tenant
// by matching products with self-evident units in their names).
var KnownUnitGUIDs = map[string]string{
	"7ba81c3a-8de5-8f9d-fb9f-e39efcbc57cc": "кг", // bulk: meat, grains, spices
	"6040d92d-e286-f4f9-a613-ed0e6fd241e1": "шт", // countable: toothpicks, nuggets, towels
	"cd19b5ea-1b32-a6e5-1df7-5d2784a0549a": "шт", // packaged goods: detergents, margarine
	"69859c74-db72-b006-cba5-326cf6f4fc6e": "л",  // liquids: oil, water
}

// guessUnitFromName looks for suffix hints in a product name (КГ/ГР/МЛ/Л/ШТ).
func guessUnitFromName(name string) string {
	upper := strings.ToUpper(name)
	if strings.Contains(upper, " КГ") || strings.HasSuffix(upper, "КГ") {
		return "кг"
	}
	if strings.Contains(upper, " ГР") || strings.HasSuffix(upper, "ГР") {
		return "гр"
	}
	if strings.Contains(upper, " МЛ") || strings.HasSuffix(upper, "МЛ") {
		return "мл"
	}
	if strings.HasSuffix(upper, " Л") || strings.Contains(upper, " Л ") {
		return "л"
	}
	return ""
}

// ResolveUnit turns an iiko unit GUID (or empty) into a human-readable unit string.
// Priority: iiko measure-unit map → known system GUIDs → guess from product name → "шт".
func ResolveUnit(rawUnit, productName string, measureUnits map[string]string) string {
	u := rawUnit
	// 1) iiko measure-unit map (most authoritative when available)
	if u != "" && measureUnits != nil {
		if n, ok := measureUnits[u]; ok && n != "" {
			return n
		}
	}
	// 2) Already a short human-readable unit (not a GUID)
	if u != "" && len(u) <= 30 {
		return u
	}
	// 3) Name heuristic (more reliable than guessed-GUID map — names contain "КГ"/"Л"/"ГР")
	if n := guessUnitFromName(productName); n != "" {
		return n
	}
	// 4) Known iiko system GUID fallback (best-effort guesses)
	if len(u) > 30 {
		if n, ok := KnownUnitGUIDs[u]; ok {
			return n
		}
	}
	// 5) Last resort
	return "шт"
}

// NormalizeOrderType maps iiko OrderServiceType to our 3 canonical values.
func NormalizeOrderType(raw string) string {
	switch raw {
	case "DeliveryByCourier", "DeliveryByClient", "Delivery":
		return "delivery"
	case "Common":
		return "takeaway"
	default:
		// DineIn, empty, or unknown → default to dine-in
		return "dine-in"
	}
}

// AggregateOrdersFromOLAP takes per-dish OLAP rows and aggregates them into per-order totals.
// iiko OLAP does NOT aggregate DishSumInt when grouping only by UniqOrderId.Id —
// it returns one arbitrary dish value. This function sums DishSumInt per order.
func AggregateOrdersFromOLAP(rows []map[string]interface{}) map[string]*OrderAgg {
	orders := make(map[string]*OrderAgg)
	for _, row := range rows {
		orderID := GetString(row, "UniqOrderId.Id")
		if orderID == "" {
			continue
		}
		agg, ok := orders[orderID]
		if !ok {
			orderNum := GetString(row, "OrderNum")
			if orderNum == "" {
				// iiko returns OrderNum as a number, not string
				if n := GetFloat(row, "OrderNum"); n > 0 {
					orderNum = fmt.Sprintf("%.0f", n)
				}
			}
			// Prefer OpenTime (has full timestamp) over OpenDate.Typed (date only)
			orderDate := GetString(row, "OpenTime")
			if orderDate == "" {
				orderDate = GetString(row, "OpenDate.Typed")
			}
			agg = &OrderAgg{
				OrderDate:   orderDate,
				OrderNumber: orderNum,
				OrderType:   NormalizeOrderType(GetString(row, "OrderServiceType")),
				WaiterName:  GetString(row, "WaiterName"),
			}
			orders[orderID] = agg
		}
		agg.Revenue += GetFloat(row, "DishSumInt")
		agg.Discount += GetFloat(row, "DishDiscountSumInt")
		agg.ItemCount += int(GetFloat(row, "DishAmountInt"))
	}
	return orders
}

// SyncRevenue pulls revenue/orders data from iiko Server API.
// iiko OLAP does NOT aggregate DishSumInt when grouping by UniqOrderId.Id alone —
// it returns one arbitrary dish value per order. So we fetch per-dish rows
// (including DishName in GroupByRowFields) and aggregate per-order in Go.
func (s *Service) SyncRevenue(ctx context.Context, client *iiko.Client, companyID, locationID uuid.UUID, iikoOrgID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "revenue")
	if err != nil {
		return err
	}
	start := time.Now()

	almatyTZ, _ := time.LoadLocation("Asia/Almaty")
	dateTo := time.Now().In(almatyTZ).Format("2006-01-02")
	dateFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, almatyTZ).Format("2006-01-02")

	// Include DishName in GroupByRowFields to get per-dish rows.
	// iiko returns DishSumInt per-dish; we SUM them per order in Go.
	filters := map[string]interface{}{
		"OpenDate.Typed": map[string]interface{}{
			"filterType": "DateRange",
			"periodType": "CUSTOM",
			"from":       dateFrom,
			"to":         dateTo,
			"includeLow": true, "includeHigh": true,
		},
	}
	// Filter by department name if location has one (multi-location setups)
	if deptName := s.getDeptName(ctx, locationID); deptName != "" {
		filters["Department"] = map[string]interface{}{
			"filterType": "IncludeValues",
			"values":     []string{deptName},
		}
	}
	report, err := client.GetOLAPReport(ctx, iiko.OLAPReportRequest{
		ReportType:       "SALES",
		GroupByRowFields: []string{"UniqOrderId.Id", "OrderNum", "OrderServiceType", "WaiterName", "OpenDate.Typed", "OpenTime", "DishName"},
		GroupByColFields: []string{},
		AggregateFields:  []string{"DishDiscountSumInt", "DishSumInt", "DishAmountInt"},
		Filters:          filters,
	})
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("fetch revenue report: %w", err)
	}

	// DEBUG: log first row to verify field names and values from iiko
	if len(report.Data) > 0 {
		first := report.Data[0]
		log.Info().
			Interface("first_row_keys", func() []string {
				keys := make([]string, 0, len(first))
				for k := range first {
					keys = append(keys, k)
				}
				return keys
			}()).
			Interface("first_row_data", first).
			Msg("sync: OLAP first row debug")
	}

	// Phase 1: aggregate per-dish rows into per-order totals
	orders := AggregateOrdersFromOLAP(report.Data)

	// Log aggregation stats for validation
	var totalRev float64
	for _, agg := range orders {
		totalRev += agg.Revenue
	}
	log.Info().Int("olap_rows", len(report.Data)).Int("unique_orders", len(orders)).
		Float64("total_revenue", totalRev).
		Str("location", locationID.String()).Msg("sync: revenue aggregated per order")

	// Phase 2: upsert aggregated order totals
	count := 0
	debugN := 0
	for orderID, agg := range orders {
		if debugN < 5 {
			log.Info().Str("order_id", orderID).Str("order_num", agg.OrderNumber).
				Float64("revenue", agg.Revenue).Float64("discount", agg.Discount).
				Int("items", agg.ItemCount).Str("date", agg.OrderDate).
				Msg("sync: DEBUG upsert sample")
			debugN++
		}
		almatyTZ4, _ := time.LoadLocation("Asia/Almaty")
		parsedDate, _ := time.ParseInLocation("2006-01-02T15:04:05", agg.OrderDate, almatyTZ4)
		if parsedDate.IsZero() {
			parsedDate, _ = time.ParseInLocation("2006-01-02T15:04:05.000", agg.OrderDate, almatyTZ4)
		}
		if parsedDate.IsZero() {
			parsedDate, _ = time.ParseInLocation("2006-01-02", agg.OrderDate, almatyTZ4)
		}
		if parsedDate.IsZero() {
			parsedDate = time.Now().In(almatyTZ4)
		}

		_, err := s.db.Exec(ctx,
			`INSERT INTO revenue_facts (company_id, location_id, iiko_order_id, order_number, order_date, revenue, discount, order_type, status, waiter_name, item_count, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
			 ON CONFLICT (company_id, location_id, iiko_order_id) DO UPDATE SET
			   order_date = EXCLUDED.order_date, revenue = EXCLUDED.revenue, discount = EXCLUDED.discount, status = EXCLUDED.status,
			   waiter_name = EXCLUDED.waiter_name, item_count = EXCLUDED.item_count, order_number = EXCLUDED.order_number, synced_at = NOW()`,
			companyID, locationID, orderID, agg.OrderNumber, parsedDate, agg.Revenue, agg.Discount, agg.OrderType, "closed", agg.WaiterName, agg.ItemCount)
		if err != nil {
			log.Warn().Err(err).Str("order_id", orderID).Msg("sync: failed to upsert revenue")
			continue
		}
		count++
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("records", count).Str("location", locationID.String()).Msg("sync: revenue complete")
	return nil
}

// SyncProductSales pulls product-level sales data from iiko Server API.
func (s *Service) SyncProductSales(ctx context.Context, client *iiko.Client, companyID, locationID uuid.UUID, iikoOrgID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "product_sales")
	if err != nil {
		return err
	}
	start := time.Now()

	almatyTZ2, _ := time.LoadLocation("Asia/Almaty")
	dateTo := time.Now().In(almatyTZ2).Format("2006-01-02")
	dateFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, almatyTZ2).Format("2006-01-02")

	psFilters := map[string]interface{}{
		"OpenDate.Typed": map[string]interface{}{
			"filterType": "DateRange",
			"periodType": "CUSTOM",
			"from":       dateFrom,
			"to":         dateTo,
			"includeLow": true, "includeHigh": true,
		},
	}
	if deptName := s.getDeptName(ctx, locationID); deptName != "" {
		psFilters["Department"] = map[string]interface{}{
			"filterType": "IncludeValues",
			"values":     []string{deptName},
		}
	}
	report, err := client.GetOLAPReport(ctx, iiko.OLAPReportRequest{
		ReportType:       "SALES",
		GroupByRowFields: []string{"DishName", "DishGroup", "DishCategory", "UniqOrderId.Id", "OpenDate.Typed"},
		GroupByColFields: []string{},
		AggregateFields:  []string{"DishAmountInt", "DishSumInt", "DishDiscountSumInt", "ProductCostBase.ProductCost"},
		Filters:          psFilters,
	})
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("fetch product sales report: %w", err)
	}

	count := 0
	for _, row := range report.Data {
		dishName := GetString(row, "DishName")
		if dishName == "" {
			continue
		}
		category := GetString(row, "DishGroup") // Use group as category
		if category == "" {
			category = GetString(row, "DishCategory")
		}
		orderID := GetString(row, "UniqOrderId.Id")
		saleDate := GetString(row, "OpenDate.Typed")
		quantity := GetFloat(row, "DishAmountInt")
		revenue := GetFloat(row, "DishSumInt")
		costPrice := GetFloat(row, "ProductCostBase.ProductCost")
		_ = GetFloat(row, "DishDiscountSumInt") // discount not stored in product_sales_facts

		// Generate stable product ID from name
		h := sha256.Sum256([]byte(dishName))
		productID := fmt.Sprintf("%x", h[:8])

		parsedDate, _ := time.Parse("2006-01-02", saleDate)
		if parsedDate.IsZero() {
			parsedDate = time.Now()
		}

		_, err := s.db.Exec(ctx,
			`INSERT INTO product_sales_facts (company_id, location_id, iiko_product_id, product_name, category, sale_date, quantity, revenue, cost_price, order_id, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
			 ON CONFLICT (company_id, iiko_product_id, sale_date, order_id) DO UPDATE SET
			   quantity = EXCLUDED.quantity, revenue = EXCLUDED.revenue, cost_price = EXCLUDED.cost_price, synced_at = NOW()`,
			companyID, locationID, productID, dishName, category, parsedDate, quantity, revenue, costPrice, orderID)
		if err != nil {
			log.Warn().Err(err).Str("product", dishName).Msg("sync: failed to upsert product sale")
			continue
		}
		count++
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("records", count).Str("location", locationID.String()).Msg("sync: product_sales complete")
	return nil
}

// SyncPurchases pulls purchase invoices from iiko Server API.
func (s *Service) SyncPurchases(ctx context.Context, client *iiko.Client, companyID, locationID uuid.UUID, iikoOrgID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "purchases")
	if err != nil {
		return err
	}
	start := time.Now()

	almatyTZ3, _ := time.LoadLocation("Asia/Almaty")
	dateFrom := time.Now().In(almatyTZ3).AddDate(0, 0, -30).Format("2006-01-02")
	dateTo := time.Now().In(almatyTZ3).Format("2006-01-02")

	invoices, err := client.GetPurchaseInvoices(ctx, dateFrom, dateTo)
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("fetch purchases: %w", err)
	}

	// Resolve supplier names
	supplierNames, supErr := client.GetSuppliers(ctx)
	if supErr != nil {
		log.Warn().Err(supErr).Msg("sync: GetSuppliers failed — invoices will keep supplier UUIDs")
	}

	// Resolve product names via nomenclature for line items
	nomenclature, nomErr := client.GetNomenclature(ctx)
	if nomErr != nil {
		log.Warn().Err(nomErr).Msg("sync: GetNomenclature failed — line items will show product UUIDs")
	}
	measureUnits, _ := client.GetMeasureUnits(ctx)

	count := 0
	for _, inv := range invoices {
		supplierName := inv.SupplierName
		if supplierNames != nil {
			if name, ok := supplierNames[inv.SupplierID]; ok && name != "" {
				supplierName = name
			}
		}

		// Upsert purchase_facts and fetch the row ID to link line items
		var purchaseRowID uuid.UUID
		err := s.db.QueryRow(ctx,
			`INSERT INTO purchase_facts (company_id, location_id, iiko_invoice_id, document_number, supplier_id, supplier_name, incoming_date, status, total_sum, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
			 ON CONFLICT (company_id, location_id, iiko_invoice_id) DO UPDATE SET
			   status = EXCLUDED.status, total_sum = EXCLUDED.total_sum, supplier_name = EXCLUDED.supplier_name, synced_at = NOW()
			 RETURNING id`,
			companyID, locationID, inv.ID, inv.DocumentNumber, inv.SupplierID, supplierName, inv.IncomingDate, inv.Status, inv.Sum).Scan(&purchaseRowID)
		if err != nil {
			log.Warn().Err(err).Str("invoice_id", inv.ID).Msg("sync: failed to upsert purchase")
			continue
		}

		// Replace line items for this purchase
		if _, err := s.db.Exec(ctx, `DELETE FROM purchase_line_items WHERE purchase_id = $1`, purchaseRowID); err == nil {
			for _, item := range inv.Items {
				// Resolve product name from nomenclature by GUID when possible
				productName := item.ProductName
				unit := ""
				if nomenclature != nil {
					if info, ok := nomenclature[item.ProductID]; ok && info.Name != "" {
						productName = info.Name
						unit = info.Unit
					}
				}
				unit = ResolveUnit(unit, productName, measureUnits)
				_, _ = s.db.Exec(ctx,
					`INSERT INTO purchase_line_items (purchase_id, product_code, product_name, unit, quantity, price, subtotal)
					 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
					purchaseRowID, item.Code, productName, unit, item.Amount, item.Price, item.Sum)
			}
		}

		count++
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("records", count).Str("location", locationID.String()).Msg("sync: purchases complete")
	return nil
}

// SyncStock pulls current stock levels from iiko Server API.
func (s *Service) SyncStock(ctx context.Context, client *iiko.Client, companyID, locationID uuid.UUID, iikoOrgID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "stock")
	if err != nil {
		return err
	}
	start := time.Now()

	items, err := client.GetStockBalance(ctx)
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("fetch stock: %w", err)
	}

	// Resolve product names from nomenclature
	nomenclature, _ := client.GetNomenclature(ctx)
	// Resolve unit GUIDs → readable names ("кг", "л", "шт", etc.)
	measureUnits, muErr := client.GetMeasureUnits(ctx)
	if muErr != nil {
		log.Warn().Err(muErr).Msg("sync: GetMeasureUnits failed — falling back to GUID + name heuristic")
	}


	count := 0
	for _, item := range items {
		productName := item.ProductName
		unit := item.Unit
		if nomenclature != nil {
			if info, ok := nomenclature[item.ProductID]; ok {
				productName = info.Name
				unit = info.Unit
			}
		}
		unit = ResolveUnit(unit, productName, measureUnits)

		_, err := s.db.Exec(ctx,
			`INSERT INTO stock_snapshots (company_id, location_id, iiko_product_id, product_name, amount, unit, cost_sum, snapshot_at, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
			companyID, locationID, item.ProductID, productName, item.Amount, unit, item.Sum)
		if err != nil {
			log.Warn().Err(err).Str("product", productName).Msg("sync: failed to insert stock")
			continue
		}
		count++
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("records", count).Str("location", locationID.String()).Msg("sync: stock complete")
	return nil
}

// SyncRecipes pulls technological cards (assembly charts) from iiko for every DISH
// and PREPARED type product. Stores the dish→ingredient breakdown in recipe_components
// so the stock UI can show "which dishes use this ingredient".
//
// One iiko round-trip per dish (~58 dishes for this restaurant), so this is bounded
// and safe to run hourly. Existing rows for a dish are deleted+reinserted on each
// sync to handle recipe changes (ingredient removed from card → row goes away).
func (s *Service) SyncRecipes(ctx context.Context, client *iiko.Client, companyID, locationID uuid.UUID, iikoOrgID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "recipes")
	if err != nil {
		return err
	}
	start := time.Now()

	nomen, err := client.GetNomenclature(ctx)
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("fetch nomenclature: %w", err)
	}

	// Filter to dishes/prepared items (the only types with assembly charts).
	// dishUnit captures whether the recipe is per-portion or per-kilogram so the UI
	// can render "0.24 л / порц." vs "1.84 л / кг". iiko sometimes returns the unit
	// as a GUID or empty string — fall back by type (DISH → порц., PREPARED → кг).
	type dishRef struct{ ID, Name, Unit string }
	var dishes []dishRef
	for _, p := range nomen {
		t := strings.ToUpper(p.Type)
		if t != "DISH" && t != "PREPARED" {
			continue
		}
		unit := p.Unit
		if unit == "" || len(unit) > 30 { // empty or GUID
			if t == "PREPARED" {
				unit = "кг"
			} else {
				unit = "порц."
			}
		}
		dishes = append(dishes, dishRef{ID: p.ID, Name: p.Name, Unit: unit})
	}
	log.Info().Int("dishes", len(dishes)).Msg("sync: recipes — fetching assembly charts")

	count := 0
	skipped := 0
	for _, d := range dishes {
		components, err := client.GetAssemblyChart(ctx, d.ID)
		if err != nil {
			log.Warn().Err(err).Str("dish", d.Name).Msg("sync: recipe fetch failed")
			continue
		}
		if len(components) == 0 {
			skipped++ // dish has no recipe (resold goods like sodas)
			continue
		}

		// Wipe & reinsert this dish's components atomically so removed ingredients drop out.
		tx, err := s.db.Begin(ctx)
		if err != nil {
			log.Warn().Err(err).Str("dish", d.Name).Msg("sync: recipe tx begin failed")
			continue
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM recipe_components WHERE company_id = $1 AND dish_iiko_id = $2`,
			companyID, d.ID); err != nil {
			_ = tx.Rollback(ctx)
			log.Warn().Err(err).Str("dish", d.Name).Msg("sync: recipe delete failed")
			continue
		}
		for _, c := range components {
			ingName := c.IngredientID
			ingUnit := ""
			if info, ok := nomen[c.IngredientID]; ok {
				if info.Name != "" {
					ingName = info.Name
				}
				ingUnit = info.Unit
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO recipe_components (company_id, dish_iiko_id, dish_name, ingredient_iiko_id, ingredient_name, amount, unit, dish_unit, synced_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
				companyID, d.ID, d.Name, c.IngredientID, ingName, c.Amount, ingUnit, d.Unit); err != nil {
				log.Warn().Err(err).Str("dish", d.Name).Str("ing", ingName).Msg("sync: recipe insert failed")
			}
		}
		if err := tx.Commit(ctx); err != nil {
			log.Warn().Err(err).Str("dish", d.Name).Msg("sync: recipe tx commit failed")
			continue
		}
		count += len(components)
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("dishes_with_recipe", len(dishes)-skipped).Int("dishes_skipped_no_recipe", skipped).
		Int("components", count).Str("location", locationID.String()).Msg("sync: recipes complete")
	return nil
}

// RefreshDashboardViews refreshes materialized views after sync.
func (s *Service) RefreshDashboardViews(ctx context.Context) error {
	_, err := s.db.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY dashboard_daily_revenue")
	if err != nil {
		log.Warn().Err(err).Msg("sync: failed to refresh dashboard view (may need initial data)")
	}
	return nil
}

// QueueSync inserts "queued" sync log entries for manual trigger.
func (s *Service) QueueSync(ctx context.Context, companyID, locationID uuid.UUID) error {
	for _, syncType := range []string{"revenue", "product_sales", "purchases", "stock"} {
		_, err := s.db.Exec(ctx,
			`INSERT INTO iiko_sync_log (id, company_id, location_id, sync_type, status, started_at)
			 VALUES ($1, $2, $3, $4, 'queued', NOW())`,
			uuid.New(), companyID, locationID, syncType)
		if err != nil {
			return fmt.Errorf("queue sync %s: %w", syncType, err)
		}
	}
	return nil
}

// ProcessQueue picks up "queued" sync log entries and runs them immediately.
// Called by the queue poller goroutine every 10 seconds.
func (s *Service) ProcessQueue(ctx context.Context) error {
	type queuedItem struct {
		ID         uuid.UUID
		CompanyID  uuid.UUID
		LocationID uuid.UUID
		SyncType   string
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, company_id, location_id, sync_type
		 FROM iiko_sync_log
		 WHERE status = 'queued'
		 ORDER BY started_at
		 LIMIT 50`)
	if err != nil {
		return fmt.Errorf("query queue: %w", err)
	}
	defer rows.Close()

	var items []queuedItem
	for rows.Next() {
		var item queuedItem
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.LocationID, &item.SyncType); err != nil {
			continue
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return nil
	}

	log.Info().Int("count", len(items)).Msg("sync: processing queued items")

	// Group by company to share iiko client
	type companyKey struct{ companyID uuid.UUID }
	clientCache := map[uuid.UUID]*iiko.Client{}

	for _, item := range items {
		// Mark as running
		s.db.Exec(ctx,
			`UPDATE iiko_sync_log SET status = 'running' WHERE id = $1`, item.ID)

		start := time.Now()

		// Get or create iiko client for this company
		client, ok := clientCache[item.CompanyID]
		if !ok {
			var iikoURL, iikoLogin, iikoPass string
			err := s.db.QueryRow(ctx,
				`SELECT iiko_server_url, iiko_login, iiko_password FROM companies
				 WHERE id = $1 AND iiko_server_url IS NOT NULL AND iiko_server_url != ''`,
				item.CompanyID).Scan(&iikoURL, &iikoLogin, &iikoPass)
			if err != nil {
				s.failSyncLog(ctx, item.ID, start, fmt.Errorf("no iiko config"))
				continue
			}
			client = iiko.NewClient(iikoURL, iikoLogin, iikoPass)
			if err := client.Authenticate(ctx); err != nil {
				s.failSyncLog(ctx, item.ID, start, fmt.Errorf("iiko auth: %w", err))
				continue
			}
			clientCache[item.CompanyID] = client
		}

		// Get iiko_org_id for this location
		var iikoOrgID string
		s.db.QueryRow(ctx,
			`SELECT COALESCE(iiko_org_id, '') FROM locations WHERE id = $1`,
			item.LocationID).Scan(&iikoOrgID)

		// Delete the queued log entry — the sync method creates its own running entry
		s.db.Exec(ctx, `DELETE FROM iiko_sync_log WHERE id = $1`, item.ID)

		// Run the appropriate sync
		var syncErr error
		switch item.SyncType {
		case "revenue":
			syncErr = s.SyncRevenue(ctx, client, item.CompanyID, item.LocationID, iikoOrgID)
		case "product_sales":
			syncErr = s.SyncProductSales(ctx, client, item.CompanyID, item.LocationID, iikoOrgID)
		case "purchases":
			syncErr = s.SyncPurchases(ctx, client, item.CompanyID, item.LocationID, iikoOrgID)
		case "stock":
			syncErr = s.SyncStock(ctx, client, item.CompanyID, item.LocationID, iikoOrgID)
		case "recipes":
			syncErr = s.SyncRecipes(ctx, client, item.CompanyID, item.LocationID, iikoOrgID)
		default:
			log.Warn().Str("type", item.SyncType).Msg("sync: unknown queued sync type")
			continue
		}

		if syncErr != nil {
			log.Error().Err(syncErr).Str("type", item.SyncType).Msg("sync: queued item failed")
		}
	}

	// Refresh dashboard views after processing queue
	if err := s.RefreshDashboardViews(ctx); err != nil {
		log.Warn().Err(err).Msg("sync: dashboard refresh after queue failed")
	}

	log.Info().Int("processed", len(items)).Msg("sync: queue processing complete")
	return nil
}

func (s *Service) startSyncLog(ctx context.Context, companyID uuid.UUID, locationID uuid.UUID, syncType string) (uuid.UUID, error) {
	id := uuid.New()
	_, err := s.db.Exec(ctx,
		`INSERT INTO iiko_sync_log (id, company_id, location_id, sync_type, status, started_at)
		 VALUES ($1, $2, $3, $4, 'running', NOW())`,
		id, companyID, locationID, syncType)
	return id, err
}

func (s *Service) completeSyncLog(ctx context.Context, logID uuid.UUID, start time.Time, count int) {
	duration := time.Since(start).Milliseconds()
	_, _ = s.db.Exec(ctx,
		`UPDATE iiko_sync_log SET status = 'success', records_synced = $1, completed_at = NOW(), duration_ms = $2 WHERE id = $3`,
		count, duration, logID)
}

func (s *Service) failSyncLog(ctx context.Context, logID uuid.UUID, start time.Time, syncErr error) {
	duration := time.Since(start).Milliseconds()
	_, _ = s.db.Exec(ctx,
		`UPDATE iiko_sync_log SET status = 'failed', error_message = $1, completed_at = NOW(), duration_ms = $2 WHERE id = $3`,
		syncErr.Error(), duration, logID)
}

package sync

import (
	"context"
	"crypto/sha256"
	"fmt"
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

// GetCompaniesToSync fetches all companies with iiko credentials configured.
func (s *Service) GetCompaniesToSync(ctx context.Context) ([]CompanySync, error) {
	rows, err := s.db.Query(ctx,
		`SELECT c.id, c.iiko_server_url, c.iiko_login, c.iiko_password, l.id, COALESCE(l.iiko_org_id, ''), l.name
		 FROM companies c
		 JOIN locations l ON l.company_id = c.id
		 WHERE c.iiko_server_url IS NOT NULL AND c.iiko_server_url != ''
		   AND c.iiko_login IS NOT NULL AND c.iiko_login != ''
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
		result = append(result, *companyMap[cid])
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
	Revenue     float64
	Discount    float64
	ItemCount   int
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
			agg = &OrderAgg{
				OrderDate:   GetString(row, "OpenDate.Typed"),
				OrderNumber: orderNum,
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

	dateTo := time.Now().Format("2006-01-02")
	dateFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")

	// Include DishName in GroupByRowFields to get per-dish rows.
	// iiko returns DishSumInt per-dish; we SUM them per order in Go.
	report, err := client.GetOLAPReport(ctx, iiko.OLAPReportRequest{
		ReportType:       "SALES",
		GroupByRowFields: []string{"UniqOrderId.Id", "OrderNum", "OpenDate.Typed", "DishName"},
		GroupByColFields: []string{},
		AggregateFields:  []string{"DishDiscountSumInt", "DishSumInt", "DishAmountInt"},
		Filters: map[string]interface{}{
			"OpenDate.Typed": map[string]interface{}{
				"filterType": "DateRange",
				"periodType": "CUSTOM",
				"from":       dateFrom,
				"to":         dateTo,
				"includeLow": true, "includeHigh": true,
			},
		},
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
	for orderID, agg := range orders {
		parsedDate, _ := time.Parse("2006-01-02", agg.OrderDate)
		if parsedDate.IsZero() {
			parsedDate = time.Now()
		}

		_, err := s.db.Exec(ctx,
			`INSERT INTO revenue_facts (company_id, location_id, iiko_order_id, order_number, order_date, revenue, discount, order_type, status, waiter_name, item_count, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
			 ON CONFLICT (company_id, iiko_order_id) DO UPDATE SET
			   revenue = EXCLUDED.revenue, discount = EXCLUDED.discount, status = EXCLUDED.status,
			   waiter_name = EXCLUDED.waiter_name, item_count = EXCLUDED.item_count, order_number = EXCLUDED.order_number, synced_at = NOW()`,
			companyID, locationID, orderID, agg.OrderNumber, parsedDate, agg.Revenue, agg.Discount, "dine-in", "closed", "", agg.ItemCount)
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

	dateTo := time.Now().Format("2006-01-02")
	dateFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")

	report, err := client.GetOLAPReport(ctx, iiko.OLAPReportRequest{
		ReportType:       "SALES",
		GroupByRowFields: []string{"DishName", "DishGroup", "DishCategory", "UniqOrderId.Id", "OpenDate.Typed"},
		GroupByColFields: []string{},
		AggregateFields:  []string{"DishAmountInt", "DishSumInt", "DishDiscountSumInt"},
		Filters: map[string]interface{}{
			"OpenDate.Typed": map[string]interface{}{
				"filterType": "DateRange",
				"periodType": "CUSTOM",
				"from":       dateFrom,
				"to":         dateTo,
				"includeLow": true, "includeHigh": true,
			},
		},
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
		discount := GetFloat(row, "DishDiscountSumInt")

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
			companyID, locationID, productID, dishName, category, parsedDate, quantity, revenue, discount, orderID)
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

	dateFrom := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	dateTo := time.Now().Format("2006-01-02")

	invoices, err := client.GetPurchaseInvoices(ctx, dateFrom, dateTo)
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("fetch purchases: %w", err)
	}

	// Resolve supplier names
	supplierNames, _ := client.GetSuppliers(ctx)

	count := 0
	for _, inv := range invoices {
		supplierName := inv.SupplierName
		if supplierNames != nil {
			if name, ok := supplierNames[inv.SupplierID]; ok {
				supplierName = name
			}
		}

		_, err := s.db.Exec(ctx,
			`INSERT INTO purchase_facts (company_id, location_id, iiko_invoice_id, document_number, supplier_id, supplier_name, incoming_date, status, total_sum, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
			 ON CONFLICT (company_id, iiko_invoice_id) DO UPDATE SET
			   status = EXCLUDED.status, total_sum = EXCLUDED.total_sum, supplier_name = EXCLUDED.supplier_name, synced_at = NOW()`,
			companyID, locationID, inv.ID, inv.DocumentNumber, inv.SupplierID, supplierName, inv.IncomingDate, inv.Status, inv.Sum)
		if err != nil {
			log.Warn().Err(err).Str("invoice_id", inv.ID).Msg("sync: failed to upsert purchase")
			continue
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

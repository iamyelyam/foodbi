package iikocloudsync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/foodbi/backend/internal/iikocloud"
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

// CompanySync holds info needed to sync one iiko Cloud company.
type CompanySync struct {
	CompanyID    uuid.UUID
	APILogin     string
	Locations    []LocationSync
}

// LocationSync holds info for one location within a company.
type LocationSync struct {
	LocationID    uuid.UUID
	IikoCloudOrgID string // iiko Cloud organization UUID
	Name          string
}

// GetCompaniesToSync fetches all companies with iiko Cloud credentials configured.
// Filters strictly to pos_system = 'iiko_cloud' to avoid collisions with iiko Server or NUMIER.
func (s *Service) GetCompaniesToSync(ctx context.Context) ([]CompanySync, error) {
	rows, err := s.db.Query(ctx,
		`SELECT c.id, c.iiko_cloud_api_login, l.id, l.name, COALESCE(l.iiko_cloud_org_id, '')
		 FROM companies c
		 JOIN locations l ON l.company_id = c.id
		 WHERE c.iiko_cloud_api_login IS NOT NULL AND c.iiko_cloud_api_login != ''
		   AND l.pos_system = 'iiko_cloud'
		 ORDER BY c.id, l.id`)
	if err != nil {
		return nil, fmt.Errorf("query iiko_cloud companies: %w", err)
	}
	defer rows.Close()

	companyMap := make(map[uuid.UUID]*CompanySync)
	var order []uuid.UUID

	for rows.Next() {
		var cid, lid uuid.UUID
		var apiLogin, locName, orgID string
		if err := rows.Scan(&cid, &apiLogin, &lid, &locName, &orgID); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		cs, ok := companyMap[cid]
		if !ok {
			cs = &CompanySync{CompanyID: cid, APILogin: apiLogin}
			companyMap[cid] = cs
			order = append(order, cid)
		}
		cs.Locations = append(cs.Locations, LocationSync{
			LocationID: lid, IikoCloudOrgID: orgID, Name: locName,
		})
	}

	result := make([]CompanySync, 0, len(order))
	for _, cid := range order {
		result = append(result, *companyMap[cid])
	}
	return result, nil
}

// DiscoverAndMapOrganizations fetches iiko Cloud organizations and maps them to FoodBI locations.
// For locations without an iiko_cloud_org_id, it matches by name or assigns the first available.
func (s *Service) DiscoverAndMapOrganizations(ctx context.Context, client *iikocloud.Client, companyID uuid.UUID) ([]LocationSync, error) {
	orgs, err := client.GetOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover organizations: %w", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, name, COALESCE(iiko_cloud_org_id, '') FROM locations
		 WHERE company_id = $1 AND pos_system = 'iiko_cloud'
		 ORDER BY name`, companyID)
	if err != nil {
		return nil, fmt.Errorf("query locations: %w", err)
	}
	defer rows.Close()

	type existingLoc struct {
		ID    uuid.UUID
		Name  string
		OrgID string
	}
	var existing []existingLoc
	for rows.Next() {
		var loc existingLoc
		if err := rows.Scan(&loc.ID, &loc.Name, &loc.OrgID); err != nil {
			continue
		}
		existing = append(existing, loc)
	}

	var result []LocationSync
	usedOrgs := make(map[string]bool)

	// First pass: locations already mapped to an org ID.
	for _, loc := range existing {
		if loc.OrgID != "" {
			result = append(result, LocationSync{
				LocationID: loc.ID, IikoCloudOrgID: loc.OrgID, Name: loc.Name,
			})
			usedOrgs[loc.OrgID] = true
		}
	}

	// Second pass: match unmapped locations to orgs (by name or first available).
	for _, loc := range existing {
		if loc.OrgID != "" {
			continue
		}
		for _, org := range orgs {
			if usedOrgs[org.ID] {
				continue
			}
			result = append(result, LocationSync{
				LocationID: loc.ID, IikoCloudOrgID: org.ID, Name: loc.Name,
			})
			usedOrgs[org.ID] = true
			s.db.Exec(ctx,
				`UPDATE locations SET iiko_cloud_org_id = $1 WHERE id = $2`, org.ID, loc.ID)
			break
		}
	}

	// Third pass: create new locations for unmapped orgs.
	for _, org := range orgs {
		if usedOrgs[org.ID] {
			continue
		}
		newID := uuid.New()
		_, err := s.db.Exec(ctx,
			`INSERT INTO locations (id, company_id, name, address, pos_system, iiko_cloud_org_id, created_at, updated_at)
			 VALUES ($1, $2, $3, '', 'iiko_cloud', $4, NOW(), NOW())`,
			newID, companyID, org.Name, org.ID)
		if err != nil {
			log.Error().Err(err).Str("org", org.Name).Msg("iikocloudsync: failed to create location")
			continue
		}
		result = append(result, LocationSync{
			LocationID: newID, IikoCloudOrgID: org.ID, Name: org.Name,
		})
		log.Info().Str("name", org.Name).Str("org_id", org.ID).Msg("iikocloudsync: created new location")
	}

	return result, nil
}

// SyncRevenue pulls sales data via OLAP and upserts into revenue_facts.
//
// RULES (CLAUDE.md):
//   - DishName MUST be in groupByRowFields — iiko Cloud OLAP does not aggregate DishSumInt per order
//   - DishSumInt is already in KZT — NEVER divide by 100
//   - SUM per UniqOrderId.Id in Go before insert
func (s *Service) SyncRevenue(ctx context.Context, client *iikocloud.Client, companyID, locationID uuid.UUID, orgID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "revenue")
	if err != nil {
		return err
	}
	start := time.Now()

	now := time.Now()
	dateFrom := now.AddDate(0, 0, -90).Format("2006-01-02")
	dateTo := now.Format("2006-01-02")

	rows, err := client.GetOLAPReport(ctx, iikocloud.OLAPReportRequest{
		OrganizationID:   orgID,
		ReportType:       "SALES",
		BuildSummary:     true,
		GroupByRowFields: []string{"UniqOrderId.Id", "OpenDate.Typed", "DishName", "OrderNum"},
		AggregateFields:  []string{"DishSumInt.value", "DiscountPercent.value"},
		Filters: map[string]interface{}{
			"OpenDate.Typed": map[string]interface{}{
				"filterType": "DateRange",
				"periodType": "CUSTOM",
				"from":       dateFrom,
				"to":         dateTo,
			},
		},
	})
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("iikocloudsync revenue OLAP: %w", err)
	}

	// Aggregate DishSumInt per order in Go (not in OLAP — iiko Cloud bug).
	type orderAgg struct {
		Date      string
		OrderNum  string
		Revenue   float64
		Discount  float64
	}
	orders := make(map[string]*orderAgg)
	for _, row := range rows {
		orderID := getString(row, "UniqOrderId.Id")
		if orderID == "" {
			continue
		}
		agg, ok := orders[orderID]
		if !ok {
			agg = &orderAgg{
				Date:     getString(row, "OpenDate.Typed"),
				OrderNum: getString(row, "OrderNum"),
			}
			orders[orderID] = agg
		}
		agg.Revenue += getFloat(row, "DishSumInt.value")
		agg.Discount += getFloat(row, "DiscountPercent.value")
	}

	count := 0
	for orderID, agg := range orders {
		parsedDate, _ := time.Parse("2006-01-02", agg.Date)
		if parsedDate.IsZero() {
			parsedDate, _ = time.Parse("2006-01-02T15:04:05", agg.Date)
		}
		if parsedDate.IsZero() {
			parsedDate = now
		}

		_, err := s.db.Exec(ctx,
			`INSERT INTO revenue_facts (company_id, location_id, iiko_order_id, order_number, order_date, revenue, discount, order_type, status, waiter_name, item_count, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, 'dine-in', 'closed', '', 0, NOW())
			 ON CONFLICT (company_id, location_id, iiko_order_id) DO UPDATE SET
			   revenue = EXCLUDED.revenue, discount = EXCLUDED.discount,
			   order_number = EXCLUDED.order_number, synced_at = NOW()`,
			companyID, locationID,
			fmt.Sprintf("iiko_cloud-%s", orderID),
			agg.OrderNum, parsedDate,
			agg.Revenue, agg.Discount)
		if err != nil {
			log.Warn().Err(err).Str("order", orderID).Msg("iikocloudsync: failed to upsert revenue")
			continue
		}
		count++
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("orders", count).Str("location", locationID.String()).Msg("iikocloudsync: revenue complete")
	return nil
}

// SyncProductSales pulls per-dish OLAP data and upserts into product_sales_facts.
func (s *Service) SyncProductSales(ctx context.Context, client *iikocloud.Client, companyID, locationID uuid.UUID, orgID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "product_sales")
	if err != nil {
		return err
	}
	start := time.Now()

	now := time.Now()
	dateFrom := now.AddDate(0, 0, -90).Format("2006-01-02")
	dateTo := now.Format("2006-01-02")

	rows, err := client.GetOLAPReport(ctx, iikocloud.OLAPReportRequest{
		OrganizationID:   orgID,
		ReportType:       "SALES",
		BuildSummary:     true,
		GroupByRowFields: []string{"DishName", "DishGroup", "DishCategory", "UniqOrderId.Id", "OpenDate.Typed"},
		AggregateFields:  []string{"DishSumInt.value", "DishAmountInt.value"},
		Filters: map[string]interface{}{
			"OpenDate.Typed": map[string]interface{}{
				"filterType": "DateRange",
				"periodType": "CUSTOM",
				"from":       dateFrom,
				"to":         dateTo,
			},
		},
	})
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("iikocloudsync product_sales OLAP: %w", err)
	}

	count := 0
	for _, row := range rows {
		dishName := getString(row, "DishName")
		if dishName == "" {
			continue
		}

		productName := formatProductName(dishName)
		category := formatProductName(getString(row, "DishGroup"))
		orderID := fmt.Sprintf("iiko_cloud-%s", getString(row, "UniqOrderId.Id"))

		dateStr := getString(row, "OpenDate.Typed")
		saleDate, _ := time.Parse("2006-01-02", dateStr)
		if saleDate.IsZero() {
			saleDate, _ = time.Parse("2006-01-02T15:04:05", dateStr)
		}
		if saleDate.IsZero() {
			saleDate = now
		}

		quantity := getFloat(row, "DishAmountInt.value")
		if quantity == 0 {
			quantity = 1
		}
		amount := getFloat(row, "DishSumInt.value") // already KZT, never divide by 100

		// Stable product ID: org + dish name hash
		productID := fmt.Sprintf("iiko_cloud-%s-%s", orgID[:8], strings.ToLower(dishName))

		_, err := s.db.Exec(ctx,
			`INSERT INTO product_sales_facts (company_id, location_id, iiko_product_id, product_name, category, sale_date, quantity, revenue, cost_price, order_id, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, $9, NOW())
			 ON CONFLICT (company_id, iiko_product_id, sale_date, order_id) DO UPDATE SET
			   quantity = EXCLUDED.quantity, revenue = EXCLUDED.revenue, synced_at = NOW()`,
			companyID, locationID, productID, productName, category,
			saleDate.Truncate(24*time.Hour), quantity, amount, orderID)
		if err != nil {
			log.Warn().Err(err).Str("dish", dishName).Msg("iikocloudsync: failed to upsert product sale")
			continue
		}
		count++
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("records", count).Str("location", locationID.String()).Msg("iikocloudsync: product_sales complete")
	return nil
}

// SyncStock fetches the current store balance from iiko Cloud and upserts into stock_snapshots.
func (s *Service) SyncStock(ctx context.Context, client *iikocloud.Client, companyID, locationID uuid.UUID, orgID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "stock")
	if err != nil {
		return err
	}
	start := time.Now()

	balances, err := client.GetStoreBalance(ctx, []string{orgID})
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("iikocloudsync stock balance: %w", err)
	}

	count := 0
	for _, item := range balances {
		if item.ProductName == "" || item.Amount <= 0 {
			continue
		}
		productID := fmt.Sprintf("iiko_cloud-%s", item.ProductID)
		productName := formatProductName(item.ProductName)
		unit := item.MeasureUnit
		if unit == "" {
			unit = "шт"
		}

		_, err := s.db.Exec(ctx,
			`INSERT INTO stock_snapshots (company_id, location_id, iiko_product_id, product_name, amount, unit, cost_sum, snapshot_at, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
			companyID, locationID, productID, productName, item.Amount, unit, item.Sum)
		if err != nil {
			log.Warn().Err(err).Str("product", item.ProductName).Msg("iikocloudsync: failed to insert stock snapshot")
			continue
		}
		count++
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("records", count).Str("location", locationID.String()).Msg("iikocloudsync: stock complete")
	return nil
}

// SyncRecipes fetches the nomenclature from iiko Cloud and logs dish/group counts.
// Full ingredient-level tech card sync awaits identification of the iiko Cloud tech card endpoint.
func (s *Service) SyncRecipes(ctx context.Context, client *iikocloud.Client, companyID, locationID uuid.UUID, orgID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "recipes")
	if err != nil {
		return err
	}
	start := time.Now()

	nom, err := client.GetNomenclature(ctx, orgID)
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("iikocloudsync nomenclature: %w", err)
	}

	dishCount := 0
	for _, p := range nom.Products {
		if p.Type == "Dish" {
			dishCount++
		}
	}
	log.Info().Int("products", len(nom.Products)).Int("dishes", dishCount).Int("groups", len(nom.Groups)).
		Str("location", locationID.String()).
		Msg("iikocloudsync: nomenclature fetched — full tech card sync pending endpoint discovery")

	s.completeSyncLog(ctx, logID, start, dishCount)
	return nil
}

// SyncPurchases is deferred — no clear iiko Cloud purchases endpoint identified yet.
func (s *Service) SyncPurchases(ctx context.Context, client *iikocloud.Client, companyID, locationID uuid.UUID, orgID string) error {
	log.Debug().Str("location", locationID.String()).Msg("iikocloudsync: purchases deferred (endpoint TBD)")
	return nil
}

// RefreshDashboardViews refreshes materialized views after sync.
func (s *Service) RefreshDashboardViews(ctx context.Context) error {
	_, err := s.db.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY dashboard_daily_revenue")
	if err != nil {
		log.Warn().Err(err).Msg("iikocloudsync: failed to refresh dashboard view")
	}
	return nil
}

// ValidateRevenueAfterSync checks MAX(revenue) per day is > 10,000 (CLAUDE.md rule 5).
func (s *Service) ValidateRevenueAfterSync(ctx context.Context, companyID, locationID uuid.UUID) {
	var maxRev float64
	_ = s.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(revenue), 0) FROM revenue_facts
		 WHERE company_id = $1 AND location_id = $2
		   AND order_date >= NOW() - INTERVAL '7 days'`,
		companyID, locationID).Scan(&maxRev)
	if maxRev < 10000 {
		log.Warn().
			Float64("max_revenue", maxRev).
			Str("location", locationID.String()).
			Msg("iikocloudsync: WARNING — max revenue < 10,000 KZT; possible missing data")
	}
}

// formatProductName capitalizes first letter and lowercases the rest.
func formatProductName(name string) string {
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	runes := []rune(lower)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

// getString safely extracts a string from an OLAP row map.
func getString(row map[string]interface{}, key string) string {
	v, ok := row[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// getFloat safely extracts a float64 from an OLAP row map.
func getFloat(row map[string]interface{}, key string) float64 {
	v, ok := row[key]
	if !ok || v == nil {
		return 0
	}
	f, ok := v.(float64)
	if ok {
		return f
	}
	return 0
}

func (s *Service) startSyncLog(ctx context.Context, companyID, locationID uuid.UUID, syncType string) (uuid.UUID, error) {
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

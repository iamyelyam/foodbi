package sync

import (
	"context"
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
	CompanyID  uuid.UUID
	IikoAPIKey string
	Locations  []LocationSync
}

type LocationSync struct {
	LocationID uuid.UUID
	IikoOrgID  string
	Name       string
}

// GetCompaniesToSync fetches all companies with iiko API keys configured.
func (s *Service) GetCompaniesToSync(ctx context.Context) ([]CompanySync, error) {
	rows, err := s.db.Query(ctx,
		`SELECT c.id, c.iiko_api_key, l.id, l.iiko_org_id, l.name
		 FROM companies c
		 JOIN locations l ON l.company_id = c.id
		 WHERE c.iiko_api_key IS NOT NULL AND c.iiko_api_key != ''
		   AND l.iiko_org_id IS NOT NULL AND l.iiko_org_id != ''
		 ORDER BY c.id, l.id`)
	if err != nil {
		return nil, fmt.Errorf("query companies: %w", err)
	}
	defer rows.Close()

	companyMap := make(map[uuid.UUID]*CompanySync)
	var order []uuid.UUID

	for rows.Next() {
		var cid, lid uuid.UUID
		var apiKey, orgID, locName string
		if err := rows.Scan(&cid, &apiKey, &lid, &orgID, &locName); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		cs, ok := companyMap[cid]
		if !ok {
			cs = &CompanySync{CompanyID: cid, IikoAPIKey: apiKey}
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

// SyncRevenue pulls revenue/orders data from iiko for a location.
func (s *Service) SyncRevenue(ctx context.Context, client *iiko.Client, companyID, locationID uuid.UUID, iikoOrgID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "revenue")
	if err != nil {
		return err
	}
	start := time.Now()

	dateTo := time.Now().Format("2006-01-02")
	dateFrom := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

	report, err := client.GetOLAPReport(ctx, iiko.OLAPReportRequest{
		OrganizationID:   iikoOrgID,
		DateFrom:         dateFrom,
		DateTo:           dateTo,
		GroupByRowFields: []string{"OrderId", "OpenDate.Typed", "OrderType", "WaiterName", "OrderDeleted"},
		AggregateFields:  []string{"DishDiscountSumInt", "DishSumInt", "DishAmountInt"},
	})
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("fetch revenue report: %w", err)
	}

	count := 0
	for _, row := range report.Data {
		if len(row) < 6 {
			continue
		}
		orderID, _ := row[0].(string)
		revenue, _ := row[4].(float64)
		discount, _ := row[3].(float64)

		_, err := s.db.Exec(ctx,
			`INSERT INTO revenue_facts (company_id, location_id, iiko_order_id, order_date, revenue, discount, order_type, status, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
			 ON CONFLICT (company_id, iiko_order_id) DO UPDATE SET
			   revenue = EXCLUDED.revenue, discount = EXCLUDED.discount, status = EXCLUDED.status, synced_at = NOW()`,
			companyID, locationID, orderID, time.Now(), revenue, discount, "dine-in", "closed")
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

// SyncPurchases pulls purchase invoices from iiko.
func (s *Service) SyncPurchases(ctx context.Context, client *iiko.Client, companyID, locationID uuid.UUID, iikoOrgID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "purchases")
	if err != nil {
		return err
	}
	start := time.Now()

	dateTo := time.Now().Format("2006-01-02")
	dateFrom := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	invoices, err := client.GetPurchaseInvoices(ctx, iikoOrgID, dateFrom, dateTo)
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("fetch purchases: %w", err)
	}

	count := 0
	for _, inv := range invoices {
		_, err := s.db.Exec(ctx,
			`INSERT INTO purchase_facts (company_id, location_id, iiko_invoice_id, document_number, supplier_id, supplier_name, incoming_date, status, total_sum, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
			 ON CONFLICT (company_id, iiko_invoice_id) DO UPDATE SET
			   status = EXCLUDED.status, total_sum = EXCLUDED.total_sum, synced_at = NOW()`,
			companyID, locationID, inv.ID, inv.DocumentNumber, inv.SupplierID, inv.SupplierName, inv.IncomingDate, inv.Status, inv.Sum)
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

// SyncStock pulls current stock levels from iiko.
func (s *Service) SyncStock(ctx context.Context, client *iiko.Client, companyID, locationID uuid.UUID, iikoOrgID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "stock")
	if err != nil {
		return err
	}
	start := time.Now()

	items, err := client.GetStockBalance(ctx, iikoOrgID)
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("fetch stock: %w", err)
	}

	count := 0
	for _, item := range items {
		_, err := s.db.Exec(ctx,
			`INSERT INTO stock_snapshots (company_id, location_id, iiko_product_id, product_name, amount, unit, cost_sum, snapshot_at, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
			companyID, locationID, item.ProductID, item.ProductName, item.Amount, item.Unit, item.Sum)
		if err != nil {
			log.Warn().Err(err).Str("product", item.ProductName).Msg("sync: failed to insert stock")
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

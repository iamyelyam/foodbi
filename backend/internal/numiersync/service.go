package numiersync

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/foodbi/backend/internal/numier"
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

// CompanySync holds info needed to sync one NUMIER company.
type CompanySync struct {
	CompanyID   uuid.UUID
	NumierKey   string
	Locations   []LocationSync
}

type LocationSync struct {
	LocationID  uuid.UUID
	NumierTpvID string // NUMIER establishment/TPV ID
	Name        string
}

// GetCompaniesToSync fetches all companies with NUMIER credentials configured.
func (s *Service) GetCompaniesToSync(ctx context.Context) ([]CompanySync, error) {
	rows, err := s.db.Query(ctx,
		`SELECT c.id, c.numier_api_key, l.id, l.name, COALESCE(l.numier_tpv_id, '')
		 FROM companies c
		 JOIN locations l ON l.company_id = c.id
		 WHERE c.numier_api_key IS NOT NULL AND c.numier_api_key != ''
		   AND l.pos_system = 'numier'
		 ORDER BY c.id, l.id`)
	if err != nil {
		return nil, fmt.Errorf("query numier companies: %w", err)
	}
	defer rows.Close()

	companyMap := make(map[uuid.UUID]*CompanySync)
	var order []uuid.UUID

	for rows.Next() {
		var cid, lid uuid.UUID
		var apiKey, locName, tpvID string
		if err := rows.Scan(&cid, &apiKey, &lid, &locName, &tpvID); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		cs, ok := companyMap[cid]
		if !ok {
			cs = &CompanySync{CompanyID: cid, NumierKey: apiKey}
			companyMap[cid] = cs
			order = append(order, cid)
		}
		cs.Locations = append(cs.Locations, LocationSync{
			LocationID: lid, NumierTpvID: tpvID, Name: locName,
		})
	}

	result := make([]CompanySync, 0, len(order))
	for _, cid := range order {
		result = append(result, *companyMap[cid])
	}
	return result, nil
}

// DiscoverAndMapLocales fetches NUMIER establishments and maps them to FoodBI locations.
// For locations without a numier_tpv_id, it tries to match by name or assigns the first available.
func (s *Service) DiscoverAndMapLocales(ctx context.Context, client *numier.Client, companyID uuid.UUID) ([]LocationSync, error) {
	locales, err := client.GetLocales(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover locales: %w", err)
	}

	// Fetch existing NUMIER locations for this company
	rows, err := s.db.Query(ctx,
		`SELECT id, name, numier_tpv_id FROM locations
		 WHERE company_id = $1 AND pos_system = 'numier'
		 ORDER BY name`, companyID)
	if err != nil {
		return nil, fmt.Errorf("query locations: %w", err)
	}
	defer rows.Close()

	type existingLoc struct {
		ID         uuid.UUID
		Name       string
		NumierTpvID *string
	}
	var existing []existingLoc
	for rows.Next() {
		var loc existingLoc
		if err := rows.Scan(&loc.ID, &loc.Name, &loc.NumierTpvID); err != nil {
			continue
		}
		existing = append(existing, loc)
	}

	// Build result: match existing locations to NUMIER locales
	var result []LocationSync
	usedLocales := make(map[string]bool)

	// First pass: locations that already have a TPV ID
	for _, loc := range existing {
		if loc.NumierTpvID != nil && *loc.NumierTpvID != "" {
			result = append(result, LocationSync{
				LocationID:  loc.ID,
				NumierTpvID: *loc.NumierTpvID,
				Name:        loc.Name,
			})
			usedLocales[*loc.NumierTpvID] = true
		}
	}

	// Second pass: match remaining locations to NUMIER locales by name or assign sequentially
	for _, loc := range existing {
		if loc.NumierTpvID != nil && *loc.NumierTpvID != "" {
			continue // already mapped
		}
		for _, locale := range locales {
			if usedLocales[locale.ID] {
				continue
			}
			// Map this location to the first available locale
			result = append(result, LocationSync{
				LocationID:  loc.ID,
				NumierTpvID: locale.ID,
				Name:        loc.Name,
			})
			usedLocales[locale.ID] = true
			// Persist the mapping
			s.db.Exec(ctx,
				`UPDATE locations SET numier_tpv_id = $1 WHERE id = $2`,
				locale.ID, loc.ID)
			break
		}
	}

	// Create new locations for unmapped NUMIER locales
	for _, locale := range locales {
		if usedLocales[locale.ID] {
			continue
		}
		newID := uuid.New()
		_, err := s.db.Exec(ctx,
			`INSERT INTO locations (id, company_id, name, address, pos_system, numier_tpv_id, created_at, updated_at)
			 VALUES ($1, $2, $3, '', 'numier', $4, NOW(), NOW())`,
			newID, companyID, locale.EstablishmentName, locale.ID)
		if err != nil {
			log.Error().Err(err).Str("locale", locale.EstablishmentName).Msg("numiersync: failed to create location")
			continue
		}
		result = append(result, LocationSync{
			LocationID:  newID,
			NumierTpvID: locale.ID,
			Name:        locale.EstablishmentName,
		})
		log.Info().Str("locale", locale.EstablishmentName).Str("tpv_id", locale.ID).Msg("numiersync: created new location")
	}

	return result, nil
}

// dateChunks splits a date range into chunks of maxDays (NUMIER limit: 34 days).
func dateChunks(from, to time.Time, maxDays int) [][2]string {
	var chunks [][2]string
	for from.Before(to) {
		end := from.AddDate(0, 0, maxDays-1)
		if end.After(to) {
			end = to
		}
		chunks = append(chunks, [2]string{
			from.Format("2006-01-02"),
			end.Format("2006-01-02"),
		})
		from = end.AddDate(0, 0, 1)
	}
	return chunks
}

// SyncRevenue pulls sales data from NUMIER and upserts into revenue_facts.
func (s *Service) SyncRevenue(ctx context.Context, client *numier.Client, companyID, locationID uuid.UUID, tpvID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "revenue")
	if err != nil {
		return err
	}
	start := time.Now()

	// Sync last 90 days in 34-day chunks (NUMIER max range)
	now := time.Now()
	dateFrom := now.AddDate(0, 0, -90)
	chunks := dateChunks(dateFrom, now, 34)

	count := 0
	for _, chunk := range chunks {
		sales, err := client.GetAllSales(ctx, tpvID, chunk[0], chunk[1])
		if err != nil {
			log.Warn().Err(err).Str("from", chunk[0]).Str("to", chunk[1]).Msg("numiersync: sales chunk failed")
			continue
		}

		for _, sale := range sales {
			// Generate stable order ID from ticket number (trim spaces — Serie comes as " 0001")
			orderID := fmt.Sprintf("numier-%s-%s", strings.TrimSpace(sale.Serie), strings.TrimSpace(sale.Number))

			parsedDate, _ := time.Parse("2006-01-02T15:04:05", sale.Date)
			if parsedDate.IsZero() {
				parsedDate, _ = time.Parse("2006-01-02", sale.BusinessDay)
			}
			if parsedDate.IsZero() {
				parsedDate = time.Now()
			}

			orderNum := sale.TaxDocumentNumber
			if orderNum == "" {
				orderNum = sale.Number
			}

			// Count items
			itemCount := 0
			for _, item := range sale.InvoiceItems {
				units, _ := strconv.ParseFloat(item.Units, 64)
				if units == 0 {
					units = 1
				}
				itemCount += int(units)
			}

			// Determine order type from channel
			orderType := normalizeChannel(sale.Channel)

			_, err := s.db.Exec(ctx,
				`INSERT INTO revenue_facts (company_id, location_id, iiko_order_id, order_number, order_date, revenue, discount, order_type, status, waiter_name, item_count, synced_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
				 ON CONFLICT (company_id, location_id, iiko_order_id) DO UPDATE SET
				   revenue = EXCLUDED.revenue, discount = EXCLUDED.discount, status = EXCLUDED.status,
				   item_count = EXCLUDED.item_count, order_number = EXCLUDED.order_number, synced_at = NOW()`,
				companyID, locationID, orderID, orderNum, parsedDate,
				sale.Totals.GrossAmount, // GrossAmount = с НДС
				0.0,                     // NUMIER has no discount field
				orderType, "closed", sale.User.UserCode, itemCount)
			if err != nil {
				log.Warn().Err(err).Str("order", orderID).Msg("numiersync: failed to upsert revenue")
				continue
			}
			count++
		}
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("records", count).Str("location", locationID.String()).Msg("numiersync: revenue complete")
	return nil
}

// SyncProductSales pulls per-product sales data from NUMIER sales tickets.
func (s *Service) SyncProductSales(ctx context.Context, client *numier.Client, companyID, locationID uuid.UUID, tpvID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "product_sales")
	if err != nil {
		return err
	}
	start := time.Now()

	// Fetch categories for name resolution
	categories, _ := client.GetCategories(ctx, tpvID)
	catMap := make(map[string]string)
	for _, cat := range categories {
		catMap[cat.ID] = cat.Name
	}

	now := time.Now()
	dateFrom := now.AddDate(0, 0, -90)
	chunks := dateChunks(dateFrom, now, 34)

	count := 0
	for _, chunk := range chunks {
		sales, err := client.GetAllSales(ctx, tpvID, chunk[0], chunk[1])
		if err != nil {
			log.Warn().Err(err).Str("from", chunk[0]).Str("to", chunk[1]).Msg("numiersync: product sales chunk failed")
			continue
		}

		for _, sale := range sales {
			orderID := fmt.Sprintf("numier-%s-%s", strings.TrimSpace(sale.Serie), strings.TrimSpace(sale.Number))
			saleDate, _ := time.Parse("2006-01-02T15:04:05", sale.Date)
			if saleDate.IsZero() {
				saleDate, _ = time.Parse("2006-01-02", sale.BusinessDay)
			}
			if saleDate.IsZero() {
				saleDate = time.Now()
			}

			for _, item := range sale.InvoiceItems {
				if item.Name == "" {
					continue
				}

				productName := formatProductName(item.Name)

				// Generate stable product ID from original name (case-insensitive)
				h := sha256.Sum256([]byte(strings.ToLower(item.Name)))
				productID := fmt.Sprintf("numier-%x", h[:8])

				category := formatProductName(catMap[item.IDCategory])
				quantity, _ := strconv.ParseFloat(item.Units, 64)
				if quantity == 0 {
					quantity = 1
				}
				amount, _ := strconv.ParseFloat(item.Amount, 64)

				_, err := s.db.Exec(ctx,
					`INSERT INTO product_sales_facts (company_id, location_id, iiko_product_id, product_name, category, sale_date, quantity, revenue, cost_price, order_id, synced_at)
					 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
					 ON CONFLICT (company_id, iiko_product_id, sale_date, order_id) DO UPDATE SET
					   quantity = EXCLUDED.quantity, revenue = EXCLUDED.revenue, synced_at = NOW()`,
					companyID, locationID, productID, productName, category,
					saleDate.Truncate(24*time.Hour), quantity, amount,
					0.0, // NUMIER doesn't provide cost_price in sales
					orderID)
				if err != nil {
					log.Warn().Err(err).Str("product", item.Name).Msg("numiersync: failed to upsert product sale")
					continue
				}
				count++
			}
		}
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("records", count).Str("location", locationID.String()).Msg("numiersync: product_sales complete")
	return nil
}

// SyncPurchases pulls expense/purchase data from NUMIER.
func (s *Service) SyncPurchases(ctx context.Context, client *numier.Client, companyID, locationID uuid.UUID, tpvID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "purchases")
	if err != nil {
		return err
	}
	start := time.Now()

	now := time.Now()
	dateFrom := now.AddDate(0, 0, -90)
	chunks := dateChunks(dateFrom, now, 34)

	count := 0
	for _, chunk := range chunks {
		expenses, err := client.GetAllExpenses(ctx, tpvID, chunk[0], chunk[1])
		if err != nil {
			log.Warn().Err(err).Str("from", chunk[0]).Str("to", chunk[1]).Msg("numiersync: purchases chunk failed")
			continue
		}

		for _, exp := range expenses {
			// Generate stable invoice ID from reference + date
			invoiceID := fmt.Sprintf("numier-%s-%s", exp.Reference, exp.Date)

			parsedDate, _ := time.Parse("2006-01-02T15:04:05", exp.Date)
			if parsedDate.IsZero() {
				parsedDate = time.Now()
			}

			// Upsert purchase_facts
			var purchaseRowID uuid.UUID
			err := s.db.QueryRow(ctx,
				`INSERT INTO purchase_facts (company_id, location_id, iiko_invoice_id, document_number, supplier_id, supplier_name, incoming_date, status, total_sum, synced_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
				 ON CONFLICT (company_id, location_id, iiko_invoice_id) DO UPDATE SET
				   status = EXCLUDED.status, total_sum = EXCLUDED.total_sum, supplier_name = EXCLUDED.supplier_name, synced_at = NOW()
				 RETURNING id`,
				companyID, locationID, invoiceID, exp.Reference,
				exp.Provider.ID, exp.Provider.Name,
				parsedDate, exp.Type, exp.Totals.GrossAmount).Scan(&purchaseRowID)
			if err != nil {
				log.Warn().Err(err).Str("ref", exp.Reference).Msg("numiersync: failed to upsert purchase")
				continue
			}

			// Replace line items
			if _, err := s.db.Exec(ctx, `DELETE FROM purchase_line_items WHERE purchase_id = $1`, purchaseRowID); err == nil {
				for _, item := range exp.ExpenseItems {
					_, _ = s.db.Exec(ctx,
						`INSERT INTO purchase_line_items (purchase_id, product_code, product_name, unit, quantity, price, subtotal)
						 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
						purchaseRowID, item.IDSubproduct, formatProductName(item.Name),
						item.Type, item.Units, item.GrossPrice, item.GrossAmount)
				}
			}

			count++
		}
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("records", count).Str("location", locationID.String()).Msg("numiersync: purchases complete")
	return nil
}

// SyncCalculatedStock computes stock levels from purchases minus sales.
// NUMIER has no stock endpoint, so we derive it from transaction data.
func (s *Service) SyncCalculatedStock(ctx context.Context, companyID, locationID uuid.UUID) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "stock")
	if err != nil {
		return err
	}
	start := time.Now()

	// Calculate: purchased quantities - sold quantities per product
	rows, err := s.db.Query(ctx,
		`WITH purchased AS (
			SELECT pli.product_name, pli.unit, SUM(pli.quantity) as qty, SUM(pli.subtotal) as cost
			FROM purchase_line_items pli
			JOIN purchase_facts pf ON pf.id = pli.purchase_id
			WHERE pf.company_id = $1 AND pf.location_id = $2
			GROUP BY pli.product_name, pli.unit
		),
		sold AS (
			SELECT product_name, SUM(quantity) as qty
			FROM product_sales_facts
			WHERE company_id = $1 AND location_id = $2
			GROUP BY product_name
		)
		SELECT p.product_name, p.unit,
			   p.qty - COALESCE(s.qty, 0) as stock_amount,
			   CASE WHEN p.qty > 0 THEN p.cost / p.qty * (p.qty - COALESCE(s.qty, 0)) ELSE 0 END as cost_sum
		FROM purchased p
		LEFT JOIN sold s ON LOWER(p.product_name) = LOWER(s.product_name)
		WHERE p.qty - COALESCE(s.qty, 0) > 0`,
		companyID, locationID)
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("calculate stock: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var productName, unit string
		var amount, costSum float64
		if err := rows.Scan(&productName, &unit, &amount, &costSum); err != nil {
			continue
		}

		h := sha256.Sum256([]byte(productName))
		productID := fmt.Sprintf("numier-%x", h[:8])

		_, err := s.db.Exec(ctx,
			`INSERT INTO stock_snapshots (company_id, location_id, iiko_product_id, product_name, amount, unit, cost_sum, snapshot_at, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
			companyID, locationID, productID, productName, amount, unit, costSum)
		if err != nil {
			log.Warn().Err(err).Str("product", productName).Msg("numiersync: failed to insert calculated stock")
			continue
		}
		count++
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("records", count).Str("location", locationID.String()).Msg("numiersync: calculated stock complete")
	return nil
}

// SyncRecipes pulls product recipes (escandallo) from NUMIER.
func (s *Service) SyncRecipes(ctx context.Context, client *numier.Client, companyID, locationID uuid.UUID, tpvID string) error {
	logID, err := s.startSyncLog(ctx, companyID, locationID, "recipes")
	if err != nil {
		return err
	}
	start := time.Now()

	products, err := client.GetAllProductsWithSubproducts(ctx, tpvID)
	if err != nil {
		s.failSyncLog(ctx, logID, start, err)
		return fmt.Errorf("fetch recipes: %w", err)
	}

	count := 0
	skipped := 0
	for _, p := range products {
		if len(p.Subproducts) == 0 {
			skipped++
			continue
		}

		dishID := fmt.Sprintf("numier-%s", p.IDProduct)

		// Wipe & reinsert atomically
		tx, err := s.db.Begin(ctx)
		if err != nil {
			log.Warn().Err(err).Str("dish", p.Name).Msg("numiersync: recipe tx begin failed")
			continue
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM recipe_components WHERE company_id = $1 AND dish_iiko_id = $2`,
			companyID, dishID); err != nil {
			_ = tx.Rollback(ctx)
			continue
		}

		for _, sub := range p.Subproducts {
			ingID := fmt.Sprintf("numier-%s", sub.IDSubproduct)
			unit := sub.Measurement
			if unit == "" {
				unit = "шт"
			}

			if _, err := tx.Exec(ctx,
				`INSERT INTO recipe_components (company_id, dish_iiko_id, dish_name, ingredient_iiko_id, ingredient_name, amount, unit, dish_unit, synced_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
				companyID, dishID, formatProductName(p.Name), ingID, formatProductName(sub.Name),
				sub.Escandallo, unit, "порц."); err != nil {
				log.Warn().Err(err).Str("dish", p.Name).Str("ing", sub.Name).Msg("numiersync: recipe insert failed")
			}
		}

		if err := tx.Commit(ctx); err != nil {
			log.Warn().Err(err).Str("dish", p.Name).Msg("numiersync: recipe tx commit failed")
			continue
		}
		count += len(p.Subproducts)
	}

	s.completeSyncLog(ctx, logID, start, count)
	log.Info().Int("products_with_recipe", len(products)-skipped).Int("skipped", skipped).
		Int("components", count).Str("location", locationID.String()).Msg("numiersync: recipes complete")
	return nil
}

// RefreshDashboardViews refreshes materialized views after sync.
func (s *Service) RefreshDashboardViews(ctx context.Context) error {
	_, err := s.db.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY dashboard_daily_revenue")
	if err != nil {
		log.Warn().Err(err).Msg("numiersync: failed to refresh dashboard view")
	}
	return nil
}

// formatProductName capitalizes first letter and lowercases the rest.
// Matches the same logic as frontend formatProductName() and backend ai/handler.go.
func formatProductName(name string) string {
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	runes := []rune(lower)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

// normalizeChannel maps NUMIER Channel to our canonical order types.
func normalizeChannel(channel string) string {
	lower := strings.ToLower(channel)
	switch {
	case strings.Contains(lower, "delivery") || strings.Contains(lower, "reparto"):
		return "delivery"
	case strings.Contains(lower, "takeaway") || strings.Contains(lower, "llevar") || strings.Contains(lower, "recoger"):
		return "takeaway"
	default:
		return "dine-in"
	}
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

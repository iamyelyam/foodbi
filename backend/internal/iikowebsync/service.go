// Package iikowebsync mirrors the NUMIER + iiko Cloud sync pattern for iikoWeb
// tenants. Data-side methods (SyncRevenue/Stock/Purchases/Recipes) are
// intentional stubs until iikoOffice submodule endpoints are reverse-engineered
// under a live session — see project memory `project_iikoweb_third_api.md`.
//
// What works today:
//   - GetCompaniesToSync — DB query scoped to pos_system='iikoweb'
//   - Auth handshake via iikoweb.Client.Authenticate
//   - GetStores probe (proves session is alive)
//
// What's stubbed (returns nil + log warning):
//   - SyncRevenue, SyncProductSales, SyncPurchases, SyncStock, SyncRecipes
package iikowebsync

import (
	"context"
	"fmt"

	"github.com/foodbi/backend/internal/iikoweb"
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

// CompanySync holds info needed to sync one iikoWeb company.
type CompanySync struct {
	CompanyID uuid.UUID
	URL       string
	Login     string
	Password  string
	Locations []LocationSync
}

type LocationSync struct {
	LocationID     uuid.UUID
	IikoWebStoreID string
	Name           string
}

// GetCompaniesToSync fetches all companies with iikoWeb credentials configured
// and at least one location whose pos_system='iikoweb'.
func (s *Service) GetCompaniesToSync(ctx context.Context) ([]CompanySync, error) {
	rows, err := s.db.Query(ctx,
		`SELECT c.id, c.iikoweb_url, c.iikoweb_login, c.iikoweb_password,
		        l.id, l.name, COALESCE(l.iikoweb_store_id, '')
		 FROM companies c
		 JOIN locations l ON l.company_id = c.id
		 WHERE c.iikoweb_url IS NOT NULL AND c.iikoweb_url != ''
		   AND c.iikoweb_login IS NOT NULL AND c.iikoweb_login != ''
		   AND c.iikoweb_password IS NOT NULL AND c.iikoweb_password != ''
		   AND l.pos_system = 'iikoweb'
		 ORDER BY c.id, l.id`)
	if err != nil {
		return nil, fmt.Errorf("query iikoweb companies: %w", err)
	}
	defer rows.Close()

	companyMap := make(map[uuid.UUID]*CompanySync)
	var order []uuid.UUID

	for rows.Next() {
		var cid, lid uuid.UUID
		var url, login, password, locName, storeID string
		if err := rows.Scan(&cid, &url, &login, &password, &lid, &locName, &storeID); err != nil {
			return nil, fmt.Errorf("scan iikoweb row: %w", err)
		}

		cs, ok := companyMap[cid]
		if !ok {
			cs = &CompanySync{CompanyID: cid, URL: url, Login: login, Password: password}
			companyMap[cid] = cs
			order = append(order, cid)
		}
		cs.Locations = append(cs.Locations, LocationSync{
			LocationID: lid, IikoWebStoreID: storeID, Name: locName,
		})
	}

	result := make([]CompanySync, 0, len(order))
	for _, cid := range order {
		result = append(result, *companyMap[cid])
	}
	return result, nil
}

// VerifySession authenticates and pings /api/stores/list to confirm the session
// is alive. This is what /omc-teams probe-iikoweb / TriggerSync currently rely
// on while data-sync methods are stubbed.
func (s *Service) VerifySession(ctx context.Context, client *iikoweb.Client) ([]iikoweb.Store, error) {
	if err := client.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("iikoweb auth: %w", err)
	}
	stores, err := client.GetStores(ctx)
	if err != nil {
		return nil, fmt.Errorf("iikoweb get stores: %w", err)
	}
	return stores, nil
}

// SyncRevenue is a stub. iikoWeb has no public OLAP-equivalent endpoint
// confirmed yet — pending live-session reverse engineering.
func (s *Service) SyncRevenue(ctx context.Context, client *iikoweb.Client, companyID, locationID uuid.UUID, storeID string) error {
	log.Warn().Str("company", companyID.String()).Str("location", locationID.String()).
		Msg("iikowebsync: SyncRevenue not implemented — pending endpoint discovery")
	return nil
}

// SyncProductSales is a stub. See SyncRevenue.
func (s *Service) SyncProductSales(ctx context.Context, client *iikoweb.Client, companyID, locationID uuid.UUID, storeID string) error {
	log.Warn().Str("company", companyID.String()).Str("location", locationID.String()).
		Msg("iikowebsync: SyncProductSales not implemented — pending endpoint discovery")
	return nil
}

// SyncPurchases is a stub.
func (s *Service) SyncPurchases(ctx context.Context, client *iikoweb.Client, companyID, locationID uuid.UUID, storeID string) error {
	log.Warn().Str("company", companyID.String()).Str("location", locationID.String()).
		Msg("iikowebsync: SyncPurchases not implemented — pending endpoint discovery")
	return nil
}

// SyncStock is a stub.
func (s *Service) SyncStock(ctx context.Context, client *iikoweb.Client, companyID, locationID uuid.UUID, storeID string) error {
	log.Warn().Str("company", companyID.String()).Str("location", locationID.String()).
		Msg("iikowebsync: SyncStock not implemented — pending endpoint discovery")
	return nil
}

// SyncRecipes is a stub.
func (s *Service) SyncRecipes(ctx context.Context, client *iikoweb.Client, companyID, locationID uuid.UUID, storeID string) error {
	log.Warn().Str("company", companyID.String()).Str("location", locationID.String()).
		Msg("iikowebsync: SyncRecipes not implemented — pending endpoint discovery")
	return nil
}

// RefreshDashboardViews is called after a sync cycle. No-op until data sync lands.
func (s *Service) RefreshDashboardViews(ctx context.Context) error {
	return nil
}

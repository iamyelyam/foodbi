package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/foodbi/backend/internal/database"
	"github.com/foodbi/backend/internal/iiko"
	"github.com/foodbi/backend/internal/numier"
	"github.com/foodbi/backend/internal/numiersync"
	gosync "github.com/foodbi/backend/internal/sync"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Global circuit breaker shared across all sync cycles.
// Initialized in main() so cooldown state persists across tick invocations.
var globalBreaker *gosync.CircuitBreaker

// workerPoolSize returns how many companies to sync in parallel.
// Set via SYNC_WORKER_POOL_SIZE env var (default 50).
func workerPoolSize() int {
	if v := os.Getenv("SYNC_WORKER_POOL_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 50
}

func main() {
	_ = godotenv.Load()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := database.NewPool(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	syncService := gosync.NewService(db)
	numierSyncService := numiersync.NewService(db)

	// Circuit breaker: skip companies that failed auth 3 times in the last hour.
	globalBreaker = gosync.NewCircuitBreaker(3, time.Hour)

	log.Info().Int("worker_pool", workerPoolSize()).Msg("sync worker started")

	// === iiko tickers ===

	go runTicker(ctx, 15*time.Minute, "revenue", func() {
		runSync(ctx, syncService, "revenue")
	})

	go runTicker(ctx, 15*time.Minute, "product_sales", func() {
		runSync(ctx, syncService, "product_sales")
	})

	go runTicker(ctx, 60*time.Minute, "purchases", func() {
		runSync(ctx, syncService, "purchases")
	})

	go runTicker(ctx, 30*time.Minute, "stock", func() {
		runSync(ctx, syncService, "stock")
	})

	go runTicker(ctx, 6*time.Hour, "recipes", func() {
		runSync(ctx, syncService, "recipes")
	})

	// Queue poller: picks up manual "Sync Now" requests every 10 seconds
	go runTicker(ctx, 10*time.Second, "queue", func() {
		if err := syncService.ProcessQueue(ctx); err != nil {
			log.Error().Err(err).Msg("sync: queue processing error")
		}
	})

	// === NUMIER tickers ===

	go runTicker(ctx, 15*time.Minute, "numier_revenue", func() {
		runNumierSync(ctx, numierSyncService, "revenue")
	})

	go runTicker(ctx, 15*time.Minute, "numier_product_sales", func() {
		runNumierSync(ctx, numierSyncService, "product_sales")
	})

	go runTicker(ctx, 60*time.Minute, "numier_purchases", func() {
		runNumierSync(ctx, numierSyncService, "purchases")
	})

	go runTicker(ctx, 30*time.Minute, "numier_stock", func() {
		runNumierSync(ctx, numierSyncService, "stock")
	})

	go runTicker(ctx, 6*time.Hour, "numier_recipes", func() {
		runNumierSync(ctx, numierSyncService, "recipes")
	})

	// Run initial syncs immediately
	go runSync(ctx, syncService, "all")
	go runNumierSync(ctx, numierSyncService, "all")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("sync worker shutting down")
	cancel()
	time.Sleep(2 * time.Second)
	log.Info().Msg("sync worker stopped")
}

func runTicker(ctx context.Context, interval time.Duration, name string, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Info().Str("sync_type", name).Msg("sync: tick triggered")
			fn()
		}
	}
}

// runSync processes all companies using a worker pool.
// Each worker handles one company (auth once, sync all its locations).
// Failed companies are recorded in iiko_sync_log but don't block others.
func runSync(ctx context.Context, svc *gosync.Service, syncType string) {
	companies, err := svc.GetCompaniesToSync(ctx)
	if err != nil {
		log.Error().Err(err).Msg("sync: failed to get companies")
		return
	}

	if len(companies) == 0 {
		log.Debug().Msg("sync: no companies with iiko configured")
		return
	}

	poolSize := workerPoolSize()
	if poolSize > len(companies) {
		poolSize = len(companies)
	}

	jobs := make(chan gosync.CompanySync, len(companies))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for company := range jobs {
				if ctx.Err() != nil {
					return
				}
				syncOneCompany(ctx, svc, company, syncType, workerID)
			}
		}(i)
	}

	// Enqueue all companies
	for _, company := range companies {
		jobs <- company
	}
	close(jobs)

	wg.Wait()

	if err := svc.RefreshDashboardViews(ctx); err != nil {
		log.Warn().Err(err).Msg("sync: dashboard view refresh failed")
	}

	log.Info().Str("type", syncType).Int("companies", len(companies)).Int("workers", poolSize).Msg("sync: cycle complete")
}

// syncOneCompany runs a single company's sync for all its locations.
// Authenticates once and reuses the client across all locations + sync types.
func syncOneCompany(ctx context.Context, svc *gosync.Service, company gosync.CompanySync, syncType string, workerID int) {
	// Circuit breaker: skip if company has been failing recently
	if globalBreaker != nil && !globalBreaker.Allow(company.CompanyID) {
		log.Debug().
			Str("company", company.CompanyID.String()).
			Int("worker", workerID).
			Msg("sync: circuit breaker open, skipping")
		return
	}

	client := iiko.NewClient(company.IikoURL, company.IikoLogin, company.IikoPassword)

	if err := client.Authenticate(ctx); err != nil {
		log.Error().Err(err).
			Str("company", company.CompanyID.String()).
			Int("worker", workerID).
			Msg("sync: iiko auth failed")
		if globalBreaker != nil {
			globalBreaker.RecordFailure(company.CompanyID)
		}
		return
	}
	if globalBreaker != nil {
		globalBreaker.RecordSuccess(company.CompanyID)
	}

	for _, loc := range company.Locations {
		if ctx.Err() != nil {
			return
		}
		logger := log.With().
			Str("company", company.CompanyID.String()).
			Str("location", loc.Name).
			Int("worker", workerID).
			Logger()

		if syncType == "all" || syncType == "revenue" {
			if err := svc.SyncRevenue(ctx, client, company.CompanyID, loc.LocationID, loc.IikoOrgID); err != nil {
				logger.Error().Err(err).Msg("sync: revenue failed")
			} else {
				svc.ValidateRevenueAfterSync(ctx, company.CompanyID, loc.LocationID)
			}
		}

		if syncType == "all" || syncType == "product_sales" {
			if err := svc.SyncProductSales(ctx, client, company.CompanyID, loc.LocationID, loc.IikoOrgID); err != nil {
				logger.Error().Err(err).Msg("sync: product_sales failed")
			}
		}

		if syncType == "all" || syncType == "purchases" {
			if err := svc.SyncPurchases(ctx, client, company.CompanyID, loc.LocationID, loc.IikoOrgID); err != nil {
				logger.Error().Err(err).Msg("sync: purchases failed")
			}
		}

		if syncType == "all" || syncType == "stock" {
			if err := svc.SyncStock(ctx, client, company.CompanyID, loc.LocationID, loc.IikoOrgID); err != nil {
				logger.Error().Err(err).Msg("sync: stock failed")
			}
		}

		if syncType == "all" || syncType == "recipes" {
			if err := svc.SyncRecipes(ctx, client, company.CompanyID, loc.LocationID, loc.IikoOrgID); err != nil {
				logger.Error().Err(err).Msg("sync: recipes failed")
			}
		}
	}
}

// runNumierSync processes all NUMIER companies using a worker pool.
func runNumierSync(ctx context.Context, svc *numiersync.Service, syncType string) {
	companies, err := svc.GetCompaniesToSync(ctx)
	if err != nil {
		log.Error().Err(err).Msg("numier-sync: failed to get companies")
		return
	}

	if len(companies) == 0 {
		log.Debug().Msg("numier-sync: no companies with numier configured")
		return
	}

	poolSize := workerPoolSize()
	if poolSize > len(companies) {
		poolSize = len(companies)
	}

	jobs := make(chan numiersync.CompanySync, len(companies))
	var wg sync.WaitGroup

	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for company := range jobs {
				if ctx.Err() != nil {
					return
				}
				syncOneNumierCompany(ctx, svc, company, syncType, workerID)
			}
		}(i)
	}

	for _, company := range companies {
		jobs <- company
	}
	close(jobs)

	wg.Wait()

	if err := svc.RefreshDashboardViews(ctx); err != nil {
		log.Warn().Err(err).Msg("numier-sync: dashboard view refresh failed")
	}

	log.Info().Str("type", syncType).Int("companies", len(companies)).Int("workers", poolSize).Msg("numier-sync: cycle complete")
}

func syncOneNumierCompany(ctx context.Context, svc *numiersync.Service, company numiersync.CompanySync, syncType string, workerID int) {
	client := numier.NewClient(company.NumierKey)

	locations, err := svc.DiscoverAndMapLocales(ctx, client, company.CompanyID)
	if err != nil {
		log.Error().Err(err).
			Str("company", company.CompanyID.String()).
			Int("worker", workerID).
			Msg("numier-sync: discover locales failed")
		locations = company.Locations
	}

	for _, loc := range locations {
		if ctx.Err() != nil {
			return
		}
		if loc.NumierTpvID == "" {
			log.Warn().Str("location", loc.Name).Msg("numier-sync: skipping location without TPV ID")
			continue
		}

		logger := log.With().
			Str("company", company.CompanyID.String()).
			Str("location", loc.Name).
			Str("tpv_id", loc.NumierTpvID).
			Int("worker", workerID).
			Logger()

		if syncType == "all" || syncType == "revenue" {
			if err := svc.SyncRevenue(ctx, client, company.CompanyID, loc.LocationID, loc.NumierTpvID); err != nil {
				logger.Error().Err(err).Msg("numier-sync: revenue failed")
			}
		}

		if syncType == "all" || syncType == "product_sales" {
			if err := svc.SyncProductSales(ctx, client, company.CompanyID, loc.LocationID, loc.NumierTpvID); err != nil {
				logger.Error().Err(err).Msg("numier-sync: product_sales failed")
			}
		}

		if syncType == "all" || syncType == "purchases" {
			if err := svc.SyncPurchases(ctx, client, company.CompanyID, loc.LocationID, loc.NumierTpvID); err != nil {
				logger.Error().Err(err).Msg("numier-sync: purchases failed")
			}
		}

		if syncType == "all" || syncType == "stock" {
			if err := svc.SyncCalculatedStock(ctx, company.CompanyID, loc.LocationID); err != nil {
				logger.Error().Err(err).Msg("numier-sync: calculated stock failed")
			}
		}

		if syncType == "all" || syncType == "recipes" {
			if err := svc.SyncRecipes(ctx, client, company.CompanyID, loc.LocationID, loc.NumierTpvID); err != nil {
				logger.Error().Err(err).Msg("numier-sync: recipes failed")
			}
		}
	}
}

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/foodbi/backend/internal/database"
	"github.com/foodbi/backend/internal/iiko"
	gosync "github.com/foodbi/backend/internal/sync"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

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

	log.Info().Msg("sync worker started")

	// Revenue + product sales sync: every 15 minutes
	go runTicker(ctx, 15*time.Minute, "revenue", func() {
		runSync(ctx, syncService, "revenue")
	})

	// Product sales sync: every 15 minutes
	go runTicker(ctx, 15*time.Minute, "product_sales", func() {
		runSync(ctx, syncService, "product_sales")
	})

	// Purchase sync: every hour
	go runTicker(ctx, 60*time.Minute, "purchases", func() {
		runSync(ctx, syncService, "purchases")
	})

	// Stock sync: every 30 minutes
	go runTicker(ctx, 30*time.Minute, "stock", func() {
		runSync(ctx, syncService, "stock")
	})

	// Run initial sync immediately
	go runSync(ctx, syncService, "all")

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

	for _, company := range companies {
		client := iiko.NewClient(company.IikoURL, company.IikoLogin, company.IikoPassword)

		if err := client.Authenticate(ctx); err != nil {
			log.Error().Err(err).Str("company", company.CompanyID.String()).Msg("sync: iiko auth failed")
			continue
		}

		for _, loc := range company.Locations {
			logger := log.With().
				Str("company", company.CompanyID.String()).
				Str("location", loc.Name).
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
		}
	}

	if err := svc.RefreshDashboardViews(ctx); err != nil {
		log.Warn().Err(err).Msg("sync: dashboard view refresh failed")
	}

	log.Info().Str("type", syncType).Int("companies", len(companies)).Msg("sync: cycle complete")
}

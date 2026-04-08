package sync

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ValidateRevenueAfterSync checks that synced revenue data looks sane.
// Logs warnings if values are suspiciously low (indicating aggregation bugs).
func (s *Service) ValidateRevenueAfterSync(ctx context.Context, companyID, locationID uuid.UUID) {
	var count int
	var avgRevenue, maxRevenue, sumRevenue float64

	err := s.db.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(AVG(revenue), 0), COALESCE(MAX(revenue), 0), COALESCE(SUM(revenue), 0)
		 FROM revenue_facts
		 WHERE company_id = $1 AND location_id = $2 AND order_date >= CURRENT_DATE`,
		companyID, locationID).Scan(&count, &avgRevenue, &maxRevenue, &sumRevenue)
	if err != nil {
		log.Warn().Err(err).Msg("sync: revenue validation query failed")
		return
	}

	logger := log.With().
		Str("company", companyID.String()).
		Str("location", locationID.String()).
		Int("order_count", count).
		Float64("avg_revenue", avgRevenue).
		Float64("max_revenue", maxRevenue).
		Float64("sum_revenue", sumRevenue).
		Logger()

	if count == 0 {
		logger.Info().Msg("sync: validation — no orders today (may be normal)")
		return
	}

	failed := false

	if maxRevenue < 10000 && count > 3 {
		logger.Error().Msg("sync: VALIDATION FAILED — MAX(revenue) < 10,000 KZT, likely per-dish values instead of order totals")
		failed = true
	}

	if avgRevenue < 5000 && count > 3 {
		logger.Error().Msg("sync: VALIDATION FAILED — AVG(revenue) < 5,000 KZT, likely per-dish values instead of order totals")
		failed = true
	}

	if sumRevenue < 100000 && count > 10 {
		logger.Error().Msg("sync: VALIDATION FAILED — daily SUM(revenue) < 100,000 KZT for 10+ orders")
		failed = true
	}

	if !failed {
		logger.Info().Msg("sync: revenue validation passed")
	}
}

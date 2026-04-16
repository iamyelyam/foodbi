package database

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// parseInt32Env returns the env var as int32, or the default if missing/invalid.
func parseInt32Env(key string, def int32) int32 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n > 0 {
			return int32(n)
		}
	}
	return def
}

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			getEnv("DB_USER", "foodbi"),
			getEnv("DB_PASSWORD", "foodbi"),
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_NAME", "foodbi"),
		)
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	// Sized for SaaS scale: 10K+ companies + API traffic.
	// DATABASE_MAX_CONNS / DATABASE_MIN_CONNS override defaults.
	config.MaxConns = parseInt32Env("DATABASE_MAX_CONNS", 100)
	config.MinConns = parseInt32Env("DATABASE_MIN_CONNS", 10)

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	log.Info().Int32("max_conns", config.MaxConns).Int32("min_conns", config.MinConns).Msg("database connected")
	return pool, nil
}

// SetTenantContext sets the RLS tenant for the current transaction.
// MUST be called inside a transaction (BEGIN...COMMIT).
func SetTenantContext(ctx context.Context, pool *pgxpool.Pool, companyID string) error {
	_, err := pool.Exec(ctx, "SET LOCAL app.current_tenant = $1", companyID)
	return err
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

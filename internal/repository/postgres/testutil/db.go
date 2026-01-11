package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func MustOpenDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatalf("DATABASE_URL is required for integration tests")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}

	// keep tests stable
	cfg.MaxConns = 4

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping db: %v", err)
	}

	return pool
}

// Optional: cleanup between tests (truncate)
func TruncateAll(t *testing.T, db *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Order matters because of FKs; RESTART IDENTITY for serial (not used) but fine.
	_, err := db.Exec(ctx, `
TRUNCATE
  payments,
  transaction_items,
  transactions,
  product_prices,
  products,
  customer_addresses,
  customers,
  customer_categories
RESTART IDENTITY CASCADE;
`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

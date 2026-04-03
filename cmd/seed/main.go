package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/riolentius/cahaya-gading-backend/internal/config"
	"github.com/riolentius/cahaya-gading-backend/internal/db"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	pool, err := db.NewPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	if err := seedCategories(ctx, pool); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	fmt.Println("✅ Seed complete")
}

func seedCategories(ctx context.Context, pool *pgxpool.Pool) error {
	categories := []struct {
		Code        string
		Name        string
		Description string
	}{
		{"REGULAR", "Regular", "Standard pricing tier"},
		{"SPECIAL", "Special", "Discounted pricing tier"},
		{"VIP", "VIP", "Best pricing tier for loyal customers"},
	}

	for _, c := range categories {
		_, err := pool.Exec(ctx, `
			INSERT INTO customer_categories (code, name, description)
			VALUES ($1, $2, $3)
			ON CONFLICT (code) DO NOTHING
		`, c.Code, c.Name, c.Description)
		if err != nil {
			return fmt.Errorf("insert category %s: %w", c.Code, err)
		}
		fmt.Printf("  → category: %s\n", c.Name)
	}

	return nil
}

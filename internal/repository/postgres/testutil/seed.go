package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func MustInsertCategory(t *testing.T, db *pgxpool.Pool, code, name string) string {
	t.Helper()

	var id string
	err := db.QueryRow(context.Background(), `
		INSERT INTO customer_categories (code, name)
		VALUES ($1, $2)
		RETURNING id::text
	`, code, name).Scan(&id)

	require.NoError(t, err)
	require.NotEmpty(t, id)
	return id
}

func MustInsertCustomer(t *testing.T, db *pgxpool.Pool, firstName, lastName, email string, categoryID *string) string {
	t.Helper()

	uniq := fmt.Sprintf("%d", time.Now().UnixNano())
	emailUniq := fmt.Sprintf("%s.%s", uniq, email)

	var id string
	err := db.QueryRow(context.Background(), `
		INSERT INTO customers (first_name, last_name, email, category_id)
		VALUES ($1, $2, $3, $4::uuid)
		RETURNING id::text
	`, firstName, lastName, emailUniq, categoryID).Scan(&id)

	require.NoError(t, err)
	require.NotEmpty(t, id)
	return id
}

func MustInsertProduct(t *testing.T, db *pgxpool.Pool, sku, name string, description *string, stockOnHand, stockReserved int) string {
	t.Helper()

	var id string
	err := db.QueryRow(context.Background(), `
		INSERT INTO products (sku, name, description, is_active, stock_on_hand, stock_reserved)
		VALUES ($1, $2, $3, true, $4, $5)
		RETURNING id::text
	`, sku, name, description, stockOnHand, stockReserved).Scan(&id)

	require.NoError(t, err)
	require.NotEmpty(t, id)
	return id
}

func MustInsertPrice(t *testing.T, db *pgxpool.Pool, productID string, categoryID *string, currency, amount string) string {
	t.Helper()

	var id string
	err := db.QueryRow(context.Background(), `
		INSERT INTO product_prices (product_id, category_id, currency, amount, valid_from, valid_to)
		VALUES ($1::uuid, $2::uuid, $3, $4::numeric, now(), NULL)
		RETURNING id::text
	`, productID, categoryID, currency, amount).Scan(&id)

	require.NoError(t, err)
	require.NotEmpty(t, id)
	return id
}

func seedAdmin(ctx context.Context, pool *pgxpool.Pool) error {
	admins := []struct {
		Email    string
		Password string
		Role     string
	}{
		{"admin@cahayagading.com", "admin123", "admin"},
		{"superadmin@cahayagading.com", "superadmin123", "superadmin"},
	}

	for _, a := range admins {
		hash, err := bcrypt.GenerateFromPassword([]byte(a.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = pool.Exec(ctx, `
            INSERT INTO admins (email, password_hash, role)
            VALUES ($1, $2, $3)
            ON CONFLICT (email) DO NOTHING
        `, a.Email, string(hash), a.Role)
		if err != nil {
			return fmt.Errorf("insert admin %s: %w", a.Email, err)
		}
		fmt.Printf("  → admin: %s (%s)\n", a.Email, a.Role)
	}
	return nil
}

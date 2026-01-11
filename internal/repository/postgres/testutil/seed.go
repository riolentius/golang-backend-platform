package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
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

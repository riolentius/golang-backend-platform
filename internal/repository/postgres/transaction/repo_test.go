package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	txuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/transaction"
)

// --- Helpers -------------------------------------------------------------

func mustTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	require.NotEmpty(t, dsn, "DATABASE_URL must be set for integration tests")

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)

	t.Cleanup(func() { pool.Close() })
	return pool
}

func mustExec(t *testing.T, pool *pgxpool.Pool, q string, args ...any) {
	t.Helper()
	_, err := pool.Exec(context.Background(), q, args...)
	require.NoError(t, err)
}

func mustQueryStr(t *testing.T, pool *pgxpool.Pool, q string, args ...any) string {
	t.Helper()
	var out string
	err := pool.QueryRow(context.Background(), q, args...).Scan(&out)
	require.NoError(t, err)
	return out
}

// truncateAll clears all test data between tests to avoid constraint violations
func truncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	mustExec(t, pool, `
		TRUNCATE TABLE
			payments,
			transaction_items,
			transactions,
			product_prices,
			products,
			customers,
			customer_categories
		RESTART IDENTITY CASCADE;
	`)
}

// seed minimal dataset with unique SKU and email per test run
func seedCustomerProductPrice(t *testing.T, pool *pgxpool.Pool) (customerID string, productID string) {
	t.Helper()

	suffix := time.Now().Format("150405.000")

	categoryID := mustQueryStr(t, pool, `
		INSERT INTO customer_categories (code, name)
		VALUES ($1, 'Regular')
		RETURNING id::text;
	`, "REGULAR-"+suffix)

	customerID = mustQueryStr(t, pool, `
		INSERT INTO customers (first_name, last_name, email, category_id)
		VALUES ('Rio', 'Test', $1, $2::uuid)
		RETURNING id::text;
	`, "rio.test."+suffix+"@example.com", categoryID)

	// unique SKU per test run — prevents duplicate key violation
	sku := "SKU-TX-" + suffix
	productID = mustQueryStr(t, pool, `
		INSERT INTO products (sku, name, description, is_active, stock_on_hand, stock_reserved)
		VALUES ($1, 'Teh Botol', 'Drink', true, 10, 0)
		RETURNING id::text;
	`, sku)

	mustExec(t, pool, `
		INSERT INTO product_prices (product_id, category_id, currency, amount, valid_from, valid_to)
		VALUES ($1::uuid, NULL, 'IDR', 5000, now(), NULL);
	`, productID)

	return customerID, productID
}

// --- Tests ---------------------------------------------------------------

func TestTransaction_Create_OK(t *testing.T) {
	pool := mustTestPool(t)
	truncateAll(t, pool)

	repo := NewTransactionRepo(pool)
	store := NewTransactionStoreAdapter(repo, pool)
	uc := txuc.New(store)

	customerID, productID := seedCustomerProductPrice(t, pool)

	out, err := uc.Create(context.Background(), txuc.CreateInput{
		CustomerID: customerID,
		Status:     txuc.StatusDraft,
		Items: []txuc.CreateItemIn{
			{ProductID: productID, Qty: 2},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotEmpty(t, out.ID)
	require.Equal(t, customerID, out.CustomerID)
	require.Equal(t, txuc.StatusDraft, out.Status)
	require.Equal(t, "10000.00", out.TotalAmount)
	require.Equal(t, "IDR", out.Currency)
	require.Len(t, out.Items, 1)
	require.Equal(t, productID, out.Items[0].ProductID)
	require.Equal(t, 2, out.Items[0].Qty)
}

func TestTransaction_Create_CustomerMissing(t *testing.T) {
	pool := mustTestPool(t)
	truncateAll(t, pool)

	repo := NewTransactionRepo(pool)
	store := NewTransactionStoreAdapter(repo, pool)
	uc := txuc.New(store)

	_, err := uc.Create(context.Background(), txuc.CreateInput{
		CustomerID: "00000000-0000-0000-0000-000000000000",
		Status:     txuc.StatusDraft,
		Items: []txuc.CreateItemIn{
			{ProductID: "00000000-0000-0000-0000-000000000000", Qty: 1},
		},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, txuc.ErrCustomerMissing)
}

func TestTransaction_StatusAndStockFlow(t *testing.T) {
	pool := mustTestPool(t)
	truncateAll(t, pool)

	repo := NewTransactionRepo(pool)
	store := NewTransactionStoreAdapter(repo, pool)
	uc := txuc.New(store)

	customerID, productID := seedCustomerProductPrice(t, pool)

	// Create draft — no stock change
	tx, err := uc.Create(context.Background(), txuc.CreateInput{
		CustomerID: customerID,
		Status:     txuc.StatusDraft,
		Items: []txuc.CreateItemIn{
			{ProductID: productID, Qty: 3},
		},
	})
	require.NoError(t, err)

	var onHand, reserved int
	err = pool.QueryRow(context.Background(), `
		SELECT stock_on_hand, stock_reserved FROM products WHERE id = $1::uuid
	`, productID).Scan(&onHand, &reserved)
	require.NoError(t, err)
	require.Equal(t, 10, onHand)
	require.Equal(t, 0, reserved)

	// Move to pending -> reserve stock
	tx, err = uc.UpdateStatus(context.Background(), tx.ID, txuc.UpdateStatusInput{Status: txuc.StatusPending})
	require.NoError(t, err)
	require.Equal(t, txuc.StatusPending, tx.Status)

	err = pool.QueryRow(context.Background(), `
		SELECT stock_on_hand, stock_reserved FROM products WHERE id = $1::uuid
	`, productID).Scan(&onHand, &reserved)
	require.NoError(t, err)
	require.Equal(t, 10, onHand)
	require.Equal(t, 3, reserved)

	// Fulfill -> commit stock
	tx, err = uc.Fulfill(context.Background(), tx.ID)
	require.NoError(t, err)
	require.Equal(t, txuc.StatusCompleted, tx.Status)

	err = pool.QueryRow(context.Background(), `
		SELECT stock_on_hand, stock_reserved FROM products WHERE id = $1::uuid
	`, productID).Scan(&onHand, &reserved)
	require.NoError(t, err)
	require.Equal(t, 7, onHand)
	require.Equal(t, 0, reserved)
}

func TestTransaction_InsufficientStock(t *testing.T) {
	pool := mustTestPool(t)
	truncateAll(t, pool)

	repo := NewTransactionRepo(pool)
	store := NewTransactionStoreAdapter(repo, pool)
	uc := txuc.New(store)

	customerID, productID := seedCustomerProductPrice(t, pool)

	// request 999 from stock 10
	_, err := uc.Create(context.Background(), txuc.CreateInput{
		CustomerID: customerID,
		Status:     txuc.StatusPending,
		Items: []txuc.CreateItemIn{
			{ProductID: productID, Qty: 999},
		},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, txuc.ErrInsufficientStock)
}

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

// seed minimal dataset:
// - customer_categories (optional)
// - customers
// - products (+ stock_on_hand)
// - product_prices (default price)
func seedCustomerProductPrice(t *testing.T, pool *pgxpool.Pool) (customerID string, productID string) {
	t.Helper()

	// customer category (optional) — if your schema allows null category_id, you can skip this
	categoryID := mustQueryStr(t, pool, `
		INSERT INTO customer_categories (code, name)
		VALUES ('REGULAR', 'Regular')
		ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name
		RETURNING id::text;
	`)

	customerID = mustQueryStr(t, pool, `
		INSERT INTO customers (first_name, last_name, email, category_id)
		VALUES ('Rio', 'Test', 'rio.tx.test.`+time.Now().Format("150405.000")+`@example.com', $1::uuid)
		RETURNING id::text;
	`, categoryID)

	productID = mustQueryStr(t, pool, `
		INSERT INTO products (sku, name, description, is_active, stock_on_hand, stock_reserved)
		VALUES ('SKU-TX-001', 'Teh Botol', 'Drink', true, 10, 0)
		RETURNING id::text;
	`)

	// default price (category_id NULL)
	mustExec(t, pool, `
		INSERT INTO product_prices (product_id, category_id, currency, amount, valid_from, valid_to)
		VALUES ($1::uuid, NULL, 'IDR', 5000, now(), NULL);
	`, productID)

	return customerID, productID
}

// --- Tests ---------------------------------------------------------------

// This test validates:
// - Create transaction works (inserts header + items)
// - Total amount is calculated
func TestTransaction_Create_OK(t *testing.T) {
	pool := mustTestPool(t)
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

	// Total should be 5000 * 2 = 10000.00
	require.Equal(t, "10000.00", out.TotalAmount)
	require.Equal(t, "IDR", out.Currency)
	require.Len(t, out.Items, 1)
	require.Equal(t, productID, out.Items[0].ProductID)
	require.Equal(t, 2, out.Items[0].Qty)
}

// This test validates:
// - Create fails when customer not found
func TestTransaction_Create_CustomerMissing(t *testing.T) {
	pool := mustTestPool(t)
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

// This validates reservation/commit workflow (best practice):
// - create as draft (no stock change)
// - update status to pending (reserve stock)
// - fulfill/commit -> deduct stock_on_hand
func TestTransaction_StatusAndStockFlow(t *testing.T) {
	pool := mustTestPool(t)
	repo := NewTransactionRepo(pool)
	store := NewTransactionStoreAdapter(repo, pool)
	uc := txuc.New(store)

	customerID, productID := seedCustomerProductPrice(t, pool)

	// Create draft (no reserve, no commit)
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

	// Move to pending -> reserve stock (reserved becomes +3)
	tx, err = uc.UpdateStatus(context.Background(), tx.ID, txuc.UpdateStatusInput{Status: txuc.StatusPending})
	require.NoError(t, err)
	require.Equal(t, txuc.StatusPending, tx.Status)

	err = pool.QueryRow(context.Background(), `
		SELECT stock_on_hand, stock_reserved FROM products WHERE id = $1::uuid
	`, productID).Scan(&onHand, &reserved)
	require.NoError(t, err)
	require.Equal(t, 10, onHand)
	require.Equal(t, 3, reserved)

	// Fulfill -> commit stock (on_hand should become 7, reserved should become 0)
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

// Validate insufficient stock when trying to reserve/commit more than available.
func TestTransaction_InsufficientStock(t *testing.T) {
	pool := mustTestPool(t)
	repo := NewTransactionRepo(pool)
	store := NewTransactionStoreAdapter(repo, pool)
	uc := txuc.New(store)

	customerID, productID := seedCustomerProductPrice(t, pool)

	// request 999 from stock 10
	_, err := uc.Create(context.Background(), txuc.CreateInput{
		CustomerID: customerID,
		Status:     txuc.StatusPending, // will validate stock
		Items: []txuc.CreateItemIn{
			{ProductID: productID, Qty: 999},
		},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, txuc.ErrInsufficientStock)
}

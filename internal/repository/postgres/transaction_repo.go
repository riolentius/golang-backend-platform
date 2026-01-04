package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRow struct {
	ID          string
	CustomerID  string
	Status      string
	Currency    string
	TotalAmount string
	Notes       *string
	CreatedAt   interface{} // we’ll scan into time.Time in adapter if you want; or set as time.Time directly
	UpdatedAt   interface{}
}

type TransactionItemRow struct {
	ID            string
	TransactionID string
	ProductID     string
	Qty           int
	UnitAmount    string
	LineTotal     string
	CreatedAt     interface{}
	UpdatedAt     interface{}
}

type TransactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepo(db *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{db: db}
}

func (r *TransactionRepo) Create(ctx context.Context, customerID string, notes *string) (*TransactionRow, error) {
	const q = `
INSERT INTO transactions (customer_id, notes)
VALUES ($1::uuid, $2)
RETURNING id::text, customer_id::text, status, currency, total_amount::text, notes, created_at, updated_at;
`
	row := r.db.QueryRow(ctx, q, customerID, notes)

	var out TransactionRow
	if err := row.Scan(&out.ID, &out.CustomerID, &out.Status, &out.Currency, &out.TotalAmount, &out.Notes, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

// Helpers for transaction-based workflow:
func (r *TransactionRepo) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.db.BeginTx(ctx, pgx.TxOptions{})
}

func ensureCustomerExists(ctx context.Context, tx pgx.Tx, customerID string) error {
	const q = `SELECT 1 FROM customers WHERE id = $1::uuid`
	var one int
	if err := tx.QueryRow(ctx, q, customerID).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return err
	}
	return nil
}

func ensureProductExists(ctx context.Context, tx pgx.Tx, productID string) error {
	const q = `SELECT 1 FROM products WHERE id = $1::uuid`
	var one int
	if err := tx.QueryRow(ctx, q, productID).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return err
	}
	return nil
}

func getLatestPriceAmount(ctx context.Context, tx pgx.Tx, productID string) (currency string, amount string, err error) {
	const q = `
SELECT currency, amount::text
FROM product_prices
WHERE product_id = $1::uuid
  AND (valid_to IS NULL OR valid_to >= now())
  AND valid_from <= now()
ORDER BY valid_from DESC, created_at DESC
LIMIT 1;
`
	if err := tx.QueryRow(ctx, q, productID).Scan(&currency, &amount); err != nil {
		return "", "", err
	}
	return currency, amount, nil
}

func insertTransaction(ctx context.Context, tx pgx.Tx, customerID string, notes *string) (*TransactionRow, error) {
	const q = `
INSERT INTO transactions (customer_id, notes)
VALUES ($1::uuid, $2)
RETURNING id::text, customer_id::text, status, currency, total_amount::text, notes, created_at, updated_at;
`
	row := tx.QueryRow(ctx, q, customerID, notes)

	var out TransactionRow
	if err := row.Scan(&out.ID, &out.CustomerID, &out.Status, &out.Currency, &out.TotalAmount, &out.Notes, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

func insertTransactionItem(ctx context.Context, tx pgx.Tx, transactionID string, productID string, qty int, unitAmount string, lineTotal string) (*TransactionItemRow, error) {
	const q = `
INSERT INTO transaction_items (transaction_id, product_id, qty, unit_amount, line_total)
VALUES ($1::uuid, $2::uuid, $3, $4::numeric, $5::numeric)
RETURNING id::text, transaction_id::text, product_id::text, qty, unit_amount::text, line_total::text, created_at, updated_at;
`
	row := tx.QueryRow(ctx, q, transactionID, productID, qty, unitAmount, lineTotal)

	var out TransactionItemRow
	if err := row.Scan(&out.ID, &out.TransactionID, &out.ProductID, &out.Qty, &out.UnitAmount, &out.LineTotal, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

func updateTransactionTotal(ctx context.Context, tx pgx.Tx, transactionID string, currency string, totalAmount string) (*TransactionRow, error) {
	const q = `
UPDATE transactions
SET currency = $2,
    total_amount = $3::numeric,
    updated_at = now()
WHERE id = $1::uuid
RETURNING id::text, customer_id::text, status, currency, total_amount::text, notes, created_at, updated_at;
`
	row := tx.QueryRow(ctx, q, transactionID, currency, totalAmount)

	var out TransactionRow
	if err := row.Scan(&out.ID, &out.CustomerID, &out.Status, &out.Currency, &out.TotalAmount, &out.Notes, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

// For list/detail/status we’ll keep it in separate methods (implement after adapter skeleton if you want).

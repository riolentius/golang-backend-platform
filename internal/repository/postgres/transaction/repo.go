package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	trxuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/transaction"
)

type TransactionRow struct {
	ID           string
	CustomerID   string
	CustomerName string
	Status       string
	Currency     string
	TotalAmount  string
	Notes        *string
	CreatedAt    interface{}
	UpdatedAt    interface{}
}

type TrxItemForFulfill struct {
	ProductID string
	Qty       int
}

type TrxStockMove struct {
	StockProductID string
	BaseQty        int
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

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
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

func (r *TransactionRepo) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.db.BeginTx(ctx, pgx.TxOptions{})
}

func ensureCustomerExists(ctx context.Context, q queryer, customerID string) error {
	const sql = `SELECT 1 FROM customers WHERE id = $1::uuid`
	var one int
	if err := q.QueryRow(ctx, sql, customerID).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return err
	}
	return nil
}

func ensureProductExists(ctx context.Context, q queryer, productID string) error {
	const sql = `SELECT 1 FROM products WHERE id = $1::uuid`
	var one int
	if err := q.QueryRow(ctx, sql, productID).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return err
	}
	return nil
}

func getEffectivePriceAmount(
	ctx context.Context,
	tx pgx.Tx,
	productID string,
	categoryID *string, // can be nil
) (currency string, amount string, err error) {
	const q = `
SELECT currency, amount::text
FROM product_prices
WHERE product_id = $1::uuid
  AND (
    ($2::uuid IS NOT NULL AND category_id = $2::uuid)
    OR category_id IS NULL
  )
  AND valid_from <= now()
  AND (valid_to IS NULL OR now() < valid_to)
ORDER BY (category_id IS NULL) ASC, valid_from DESC, created_at DESC
LIMIT 1;
`
	// note: if categoryID is nil, $2::uuid becomes NULL, query falls back to category_id IS NULL.
	if err := tx.QueryRow(ctx, q, productID, categoryID).Scan(&currency, &amount); err != nil {
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

func getCustomerCategoryID(ctx context.Context, tx pgx.Tx, customerID string) (*string, error) {
	const q = `
SELECT category_id::text
FROM customers
WHERE id = $1::uuid
`
	var cat *string
	if err := tx.QueryRow(ctx, q, customerID).Scan(&cat); err != nil {
		return nil, err
	}
	return cat, nil // can be nil (customer without category)
}

// getCustomerName resolves a display name ("First Last") for a customer.
// Used to populate Transaction.CustomerName on write paths (Create, UpdateItems)
// that don't already JOIN against customers.
func getCustomerName(ctx context.Context, tx pgx.Tx, customerID string) (string, error) {
	const q = `
SELECT COALESCE(first_name, '') ||
  CASE WHEN last_name IS NULL OR last_name = '' THEN '' ELSE ' ' || last_name END
FROM customers
WHERE id = $1::uuid
`
	var name string
	if err := tx.QueryRow(ctx, q, customerID).Scan(&name); err != nil {
		return "", err
	}
	return name, nil
}

func lockTransactionStatus(ctx context.Context, tx pgx.Tx, transactionID string) (string, error) {
	const q = `
SELECT status
FROM transactions
WHERE id = $1::uuid
FOR UPDATE;
`
	var status string
	if err := tx.QueryRow(ctx, q, transactionID).Scan(&status); err != nil {
		return "", err
	}
	return status, nil
}

// lockTransactionRow locks the transaction row and also returns the customer_id,
// needed when rewriting items (to resolve the customer's price category).
func lockTransactionRow(ctx context.Context, tx pgx.Tx, transactionID string) (status string, customerID string, err error) {
	const q = `
SELECT status, customer_id::text
FROM transactions
WHERE id = $1::uuid
FOR UPDATE;
`
	if err := tx.QueryRow(ctx, q, transactionID).Scan(&status, &customerID); err != nil {
		return "", "", err
	}
	return status, customerID, nil
}

func deleteTransactionItems(ctx context.Context, tx pgx.Tx, transactionID string) error {
	const q = `DELETE FROM transaction_items WHERE transaction_id = $1::uuid;`
	_, err := tx.Exec(ctx, q, transactionID)
	return err
}

func listTransactionStockMoves(ctx context.Context, tx pgx.Tx, transactionID string) ([]TrxStockMove, error) {
	const q = `
SELECT
  COALESCE(p.base_product_id, p.id)::text AS stock_product_id,
  (ti.qty * p.pack_size)::numeric::text AS base_qty
FROM transaction_items ti
JOIN products p ON p.id = ti.product_id
WHERE ti.transaction_id = $1::uuid;
`
	rows, err := tx.Query(ctx, q, transactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TrxStockMove, 0, 10)
	for rows.Next() {
		var m TrxStockMove
		var baseQtyStr string

		if err := rows.Scan(&m.StockProductID, &baseQtyStr); err != nil {
			return nil, err
		}

		// v1: must be integer (packaging)
		f, err := strconv.ParseFloat(baseQtyStr, 64)
		if err != nil {
			return nil, err
		}
		if f != math.Trunc(f) {
			return nil, errors.New("non-integer base_qty not supported in v1")
		}
		m.BaseQty = int(f)

		out = append(out, m)
	}
	return out, rows.Err()
}

func lockProductStock(ctx context.Context, tx pgx.Tx, productID string) (onHand int, reserved int, err error) {
	const q = `
SELECT stock_on_hand, stock_reserved
FROM products
WHERE id = $1::uuid
FOR UPDATE;
`
	if err := tx.QueryRow(ctx, q, productID).Scan(&onHand, &reserved); err != nil {
		return 0, 0, err
	}
	return onHand, reserved, nil
}

func deductStockOnHand(ctx context.Context, tx pgx.Tx, productID string, qty int) error {
	const q = `
UPDATE products
SET stock_on_hand = stock_on_hand - $2,
    updated_at = now()
WHERE id = $1::uuid;
`
	_, err := tx.Exec(ctx, q, productID, qty)
	return err
}

func updateTransactionStatus(ctx context.Context, tx pgx.Tx, transactionID string, status string) (*TransactionRow, error) {
	const q = `
UPDATE transactions
SET status = $2,
    updated_at = now()
WHERE id = $1::uuid
RETURNING id::text, customer_id::text, status, currency, total_amount::text, notes, created_at, updated_at;
`
	row := tx.QueryRow(ctx, q, transactionID, status)

	var out TransactionRow
	if err := row.Scan(&out.ID, &out.CustomerID, &out.Status, &out.Currency, &out.TotalAmount, &out.Notes, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

func getStockRule(ctx context.Context, q queryer, productID string) (stockProductID string, packSize float64, err error) {
	const sql = `
SELECT
  COALESCE(base_product_id, id)::text AS stock_product_id,
  pack_size::text
FROM products
WHERE id = $1::uuid;
`
	var packStr string
	if err := q.QueryRow(ctx, sql, productID).Scan(&stockProductID, &packStr); err != nil {
		return "", 0, err
	}

	ps, err := strconv.ParseFloat(packStr, 64)
	if err != nil {
		return "", 0, err
	}
	if ps <= 0 {
		return "", 0, errors.New("invalid pack_size")
	}
	return stockProductID, ps, nil
}

func reserveStock(ctx context.Context, tx pgx.Tx, productID string, qty int) error {
	const q = `
UPDATE products
SET stock_reserved = stock_reserved + $2,
    updated_at = now()
WHERE id = $1::uuid;
`
	_, err := tx.Exec(ctx, q, productID, qty)
	return err
}

func releaseReservedStock(ctx context.Context, tx pgx.Tx, productID string, qty int) error {
	const q = `
UPDATE products
SET stock_reserved = stock_reserved - $2,
    updated_at = now()
WHERE id = $1::uuid
  AND stock_reserved >= $2;
`
	ct, err := tx.Exec(ctx, q, productID, qty)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("reserved stock insufficient: stock_product=%s required=%d", productID, qty)
	}
	return nil
}

func commitStockForTx(ctx context.Context, tx pgx.Tx, transactionID string) error {
	moves, err := listTransactionStockMoves(ctx, tx, transactionID)
	if err != nil {
		return err
	}
	if len(moves) == 0 {
		return pgx.ErrNoRows
	}

	need := map[string]int{}
	for _, m := range moves {
		need[m.StockProductID] += m.BaseQty
	}

	for stockID, qty := range need {
		onHand, reserved, err := lockProductStock(ctx, tx, stockID)
		if err != nil {
			return err
		}

		// since we are committing reserved stock, ensure reservation exists
		if reserved < qty {
			return fmt.Errorf("reserved stock insufficient: stock_product=%s reserved=%d required=%d", stockID, reserved, qty)
		}
		if onHand < qty {
			return fmt.Errorf("on_hand insufficient: stock_product=%s on_hand=%d required=%d", stockID, onHand, qty)
		}

		// commit: on_hand -= qty, reserved -= qty
		const q = `
UPDATE products
SET stock_on_hand = stock_on_hand - $2,
    stock_reserved = stock_reserved - $2,
    updated_at = now()
WHERE id = $1::uuid;
`
		if _, err := tx.Exec(ctx, q, stockID, qty); err != nil {
			return err
		}
	}

	return nil
}

func reserveStockForTx(ctx context.Context, tx pgx.Tx, transactionID string) error {
	moves, err := listTransactionStockMoves(ctx, tx, transactionID)
	if err != nil {
		return err
	}
	if len(moves) == 0 {
		return pgx.ErrNoRows
	}

	need := map[string]int{}
	for _, m := range moves {
		need[m.StockProductID] += m.BaseQty
	}

	for stockID, qty := range need {
		onHand, reserved, err := lockProductStock(ctx, tx, stockID)
		if err != nil {
			return err
		}

		available := onHand - reserved
		if available < qty {
			return fmt.Errorf("%w: stock_product=%s available=%d required=%d", trxuc.ErrInsufficientStock, stockID, available, qty)
		}

		if err := reserveStock(ctx, tx, stockID, qty); err != nil {
			return err
		}
	}

	return nil
}

func releaseStockForTx(ctx context.Context, tx pgx.Tx, transactionID string) error {
	moves, err := listTransactionStockMoves(ctx, tx, transactionID)
	if err != nil {
		return err
	}
	if len(moves) == 0 {
		return pgx.ErrNoRows
	}

	need := map[string]int{}
	for _, m := range moves {
		need[m.StockProductID] += m.BaseQty
	}

	for stockID, qty := range need {
		// lock to serialize concurrent operations
		_, _, err := lockProductStock(ctx, tx, stockID)
		if err != nil {
			return err
		}

		if err := releaseReservedStock(ctx, tx, stockID, qty); err != nil {
			return err
		}
	}

	return nil
}

type CustomerTxSummaryRow struct {
	ID            string
	Status        string
	TotalAmount   string
	PaidAmount    string
	BalanceDue    string
	PaymentStatus string
	CreatedAt     time.Time
}

// ListByCustomer returns a condensed transaction history for one customer —
// every transaction regardless of status, newest first.
func (r *TransactionRepo) ListByCustomer(ctx context.Context, customerID string) ([]CustomerTxSummaryRow, error) {
	const q = `
SELECT
  id::text,
  status,
  total_amount::text,
  paid_amount::text,
  (total_amount - paid_amount)::text AS balance_due,
  payment_status,
  created_at
FROM transactions
WHERE customer_id = $1::uuid
ORDER BY created_at DESC;
`
	rows, err := r.db.Query(ctx, q, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]CustomerTxSummaryRow, 0, 10)
	for rows.Next() {
		var row CustomerTxSummaryRow
		if err := rows.Scan(
			&row.ID, &row.Status, &row.TotalAmount, &row.PaidAmount,
			&row.BalanceDue, &row.PaymentStatus, &row.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *TransactionRepo) GetOutstandingTotalForCustomer(ctx context.Context, customerID string) (string, error) {
	const q = `
SELECT COALESCE(SUM(total_amount - paid_amount), 0)::text
FROM transactions
WHERE customer_id = $1::uuid
  AND status != 'cancelled';
`
	var total string
	if err := r.db.QueryRow(ctx, q, customerID).Scan(&total); err != nil {
		return "", err
	}
	return total, nil
}

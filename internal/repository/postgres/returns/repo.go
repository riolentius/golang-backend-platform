package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTransactionMissing   = errors.New("transaction not found")
	ErrNotFulfilled         = errors.New("transaction not completed")
	ErrItemNotInTransaction = errors.New("item not in transaction")
	ErrQtyExceedsReturnable = errors.New("qty exceeds returnable")
	ErrInvalidPackSize      = errors.New("non-integer base quantity for restock")
)

type ReturnRow struct {
	ID             string
	TransactionID  string
	TotalAmount    string
	Currency       string
	Note           *string
	CreatedByEmail *string
	CreatedAt      time.Time
}

type ReturnItemRow struct {
	ID                string
	ReturnID          string
	TransactionItemID string
	ProductID         string
	ProductName       string
	SKU               *string
	Qty               int
	UnitAmount        string
	LineTotal         string
	Restock           bool
}

type TransactionStateRow struct {
	ID            string
	TotalAmount   string
	PaidAmount    string
	PaymentStatus string
	Currency      string
}

type ReturnableItemRow struct {
	TransactionItemID string
	ProductID         string
	ProductName       string
	SKU               *string
	UnitAmount        string
	QtySold           int
	QtyReturned       int
	QtyReturnable     int
}

// CreateItemArg mirrors the usecase input without importing it (repo is
// downstream of usecase).
type CreateItemArg struct {
	TransactionItemID string
	Qty               int
	Restock           bool
}

type ReturnRepo struct {
	db *pgxpool.Pool
}

func NewReturnRepo(db *pgxpool.Pool) *ReturnRepo {
	return &ReturnRepo{db: db}
}

func (r *ReturnRepo) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.db.BeginTx(ctx, pgx.TxOptions{})
}

// Create records a return and applies every consequence atomically:
// insert header + lines, restock where asked (stock + ledger), reduce the
// transaction total, and re-derive payment_status.
func (r *ReturnRepo) Create(
	ctx context.Context,
	transactionID string,
	note *string,
	items []CreateItemArg,
	createdByEmail *string,
) (*ReturnRow, []ReturnItemRow, *TransactionStateRow, error) {
	tx, err := r.Begin(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1) lock the transaction; returns are only valid after fulfillment
	var status, currency string
	const lockQ = `SELECT status, currency FROM transactions WHERE id = $1::uuid FOR UPDATE;`
	if err := tx.QueryRow(ctx, lockQ, transactionID).Scan(&status, &currency); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, ErrTransactionMissing
		}
		return nil, nil, nil, fmt.Errorf("lock transaction: %w", err)
	}
	if status != "completed" {
		return nil, nil, nil, ErrNotFulfilled
	}

	// 2) header first (total_amount rolled up from the lines afterwards)
	const insHeaderQ = `
INSERT INTO returns (transaction_id, currency, note, created_by_email)
VALUES ($1::uuid, $2, $3, $4)
RETURNING id::text, transaction_id::text, total_amount::text, currency, note, created_by_email, created_at;
`
	var hdr ReturnRow
	if err := tx.QueryRow(ctx, insHeaderQ, transactionID, currency, note, createdByEmail).Scan(
		&hdr.ID, &hdr.TransactionID, &hdr.TotalAmount, &hdr.Currency,
		&hdr.Note, &hdr.CreatedByEmail, &hdr.CreatedAt,
	); err != nil {
		return nil, nil, nil, fmt.Errorf("insert returns header: %w", err)
	}

	outItems := make([]ReturnItemRow, 0, len(items))

	for _, it := range items {
		// 3a) the sold line must belong to this transaction
		var soldQty int
		const soldQ = `
SELECT qty
FROM transaction_items
WHERE id = $1::uuid AND transaction_id = $2::uuid;
`
		if err := tx.QueryRow(ctx, soldQ, it.TransactionItemID, transactionID).Scan(&soldQty); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil, nil, ErrItemNotInTransaction
			}
			return nil, nil, nil, fmt.Errorf("lookup sold item %s: %w", it.TransactionItemID, err)
		}

		// 3b) cumulative guard across all prior returns on this line
		var alreadyReturned int
		const priorQ = `
SELECT COALESCE(SUM(qty), 0)::int
FROM return_items
WHERE transaction_item_id = $1::uuid AND return_id <> $2::uuid;
`
		if err := tx.QueryRow(ctx, priorQ, it.TransactionItemID, hdr.ID).Scan(&alreadyReturned); err != nil {
			return nil, nil, nil, fmt.Errorf("sum prior returns: %w", err)
		}
		if alreadyReturned+it.Qty > soldQty {
			return nil, nil, nil, ErrQtyExceedsReturnable
		}

		// 3c) insert the line. unit_amount and line_total are derived from the
		// sold line inside SQL, so the value follows the ORIGINAL sale price and
		// the multiply happens in numeric (no float rounding).
		// $2 is cast explicitly: it feeds both an integer column (qty) and a
		// numeric expression (line_total), and without the cast Postgres can
		// deduce a single conflicting type for it.
		const insItemQ = `
INSERT INTO return_items
  (return_id, transaction_item_id, product_id, qty, unit_amount, line_total, restock)
SELECT $1::uuid, ti.id, ti.product_id, $2::int, ti.unit_amount, (ti.unit_amount * $2::int), $3::boolean
FROM transaction_items ti
WHERE ti.id = $4::uuid
RETURNING id::text, return_id::text, transaction_item_id::text, product_id::text,
          qty, unit_amount::text, line_total::text, restock;
`
		var ri ReturnItemRow
		if err := tx.QueryRow(ctx, insItemQ, hdr.ID, it.Qty, it.Restock, it.TransactionItemID).Scan(
			&ri.ID, &ri.ReturnID, &ri.TransactionItemID, &ri.ProductID,
			&ri.Qty, &ri.UnitAmount, &ri.LineTotal, &ri.Restock,
		); err != nil {
			return nil, nil, nil, fmt.Errorf("insert return_item (txItem=%s qty=%d): %w", it.TransactionItemID, it.Qty, err)
		}

		// display fields (no deleted_at filter — a since-deleted product must
		// still render on historical returns)
		const prodQ = `SELECT name, sku FROM products WHERE id = $1::uuid;`
		if err := tx.QueryRow(ctx, prodQ, ri.ProductID).Scan(&ri.ProductName, &ri.SKU); err != nil {
			return nil, nil, nil, fmt.Errorf("load product %s: %w", ri.ProductID, err)
		}

		// 3d) the decision: back to sellable stock, or written off
		if it.Restock {
			if err := restockReturnedItem(ctx, tx, ri.ProductID, it.Qty, hdr.ID, createdByEmail); err != nil {
				return nil, nil, nil, fmt.Errorf("restock product %s: %w", ri.ProductID, err)
			}
		}

		outItems = append(outItems, ri)
	}

	// 4) roll the header total up from its lines
	const sumQ = `
UPDATE returns
SET total_amount = (SELECT COALESCE(SUM(line_total), 0) FROM return_items WHERE return_id = $1::uuid)
WHERE id = $1::uuid
RETURNING total_amount::text;
`
	if err := tx.QueryRow(ctx, sumQ, hdr.ID).Scan(&hdr.TotalAmount); err != nil {
		return nil, nil, nil, fmt.Errorf("roll up return total: %w", err)
	}

	// 5) reduce the transaction total by the returned value
	const updTotalQ = `
UPDATE transactions
SET total_amount = GREATEST(total_amount - $2::numeric, 0),
    updated_at = now()
WHERE id = $1::uuid;
`
	if _, err := tx.Exec(ctx, updTotalQ, transactionID, hdr.TotalAmount); err != nil {
		return nil, nil, nil, fmt.Errorf("reduce transaction total (by %s): %w", hdr.TotalAmount, err)
	}

	// 6) re-derive payment_status against the new total. paid_amount is left
	// untouched on purpose: a return moves no money, it just means the customer
	// has now overpaid relative to the reduced total.
	const recomputeQ = `
UPDATE transactions
SET payment_status = CASE
      WHEN paid_amount = 0 THEN 'unpaid'
      WHEN paid_amount < total_amount THEN 'partial'
      WHEN paid_amount = total_amount THEN 'paid'
      ELSE 'overpaid'
    END,
    updated_at = now()
WHERE id = $1::uuid
RETURNING id::text, total_amount::text, paid_amount::text, payment_status, currency;
`
	var st TransactionStateRow
	if err := tx.QueryRow(ctx, recomputeQ, transactionID).Scan(
		&st.ID, &st.TotalAmount, &st.PaidAmount, &st.PaymentStatus, &st.Currency,
	); err != nil {
		return nil, nil, nil, fmt.Errorf("recompute payment_status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("commit: %w", err)
	}
	return &hdr, outItems, &st, nil
}

// restockReturnedItem puts returned goods back into sellable stock.
//
// Stock lives on the base product (a packaging variant draws from its base,
// scaled by pack_size), while the ledger row is recorded in the sold product's
// own unit — the same convention transaction_items and recordStockOutForTx use.
func restockReturnedItem(
	ctx context.Context,
	tx pgx.Tx,
	productID string,
	qty int,
	returnID string,
	createdByEmail *string,
) error {
	var stockProductID string
	var packSize float64
	const resolveQ = `
SELECT COALESCE(base_product_id, id)::text, pack_size::float8
FROM products
WHERE id = $1::uuid;
`
	if err := tx.QueryRow(ctx, resolveQ, productID).Scan(&stockProductID, &packSize); err != nil {
		return err
	}

	baseQtyF := float64(qty) * packSize
	baseQty := int(baseQtyF)
	if baseQtyF != float64(baseQty) {
		return ErrInvalidPackSize
	}

	// Lock the row we're about to change — that's the base product, which may
	// differ from the variant that was sold.
	const lockStockQ = `SELECT 1 FROM products WHERE id = $1::uuid FOR UPDATE;`
	if _, err := tx.Exec(ctx, lockStockQ, stockProductID); err != nil {
		return err
	}

	const updStockQ = `
UPDATE products
SET stock_on_hand = stock_on_hand + $2,
    updated_at = now()
WHERE id = $1::uuid;
`
	if _, err := tx.Exec(ctx, updStockQ, stockProductID, baseQty); err != nil {
		return err
	}

	const movQ = `
INSERT INTO stock_movements
  (product_id, direction, quantity, source, reference_id, note, created_by_email)
VALUES ($1::uuid, 'in', $2, 'return', $3::uuid, 'Returned to stock', $4);
`
	_, err := tx.Exec(ctx, movQ, productID, qty, returnID, createdByEmail)
	return err
}

func (r *ReturnRepo) ListByTransaction(ctx context.Context, transactionID string) ([]ReturnRow, error) {
	const q = `
SELECT id::text, transaction_id::text, total_amount::text, currency, note, created_by_email, created_at
FROM returns
WHERE transaction_id = $1::uuid
ORDER BY created_at DESC;
`
	rows, err := r.db.Query(ctx, q, transactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ReturnRow, 0, 8)
	for rows.Next() {
		var rr ReturnRow
		if err := rows.Scan(
			&rr.ID, &rr.TransactionID, &rr.TotalAmount, &rr.Currency,
			&rr.Note, &rr.CreatedByEmail, &rr.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, rr)
	}
	return out, rows.Err()
}

// ListItemsByReturnIDs fetches all lines for a set of returns, grouped by return id.
func (r *ReturnRepo) ListItemsByReturnIDs(ctx context.Context, returnIDs []string) (map[string][]ReturnItemRow, error) {
	grouped := make(map[string][]ReturnItemRow, len(returnIDs))
	if len(returnIDs) == 0 {
		return grouped, nil
	}

	// compare as text[] — avoids relying on pgx encoding []string into a uuid[]
	const q = `
SELECT ri.id::text, ri.return_id::text, ri.transaction_item_id::text, ri.product_id::text,
       p.name, p.sku, ri.qty, ri.unit_amount::text, ri.line_total::text, ri.restock
FROM return_items ri
JOIN products p ON p.id = ri.product_id
WHERE ri.return_id::text = ANY($1::text[])
ORDER BY p.name ASC;
`
	rows, err := r.db.Query(ctx, q, returnIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ri ReturnItemRow
		if err := rows.Scan(
			&ri.ID, &ri.ReturnID, &ri.TransactionItemID, &ri.ProductID,
			&ri.ProductName, &ri.SKU, &ri.Qty, &ri.UnitAmount, &ri.LineTotal, &ri.Restock,
		); err != nil {
			return nil, err
		}
		grouped[ri.ReturnID] = append(grouped[ri.ReturnID], ri)
	}
	return grouped, rows.Err()
}

// ListReturnableItems reports, per sold line, how much can still be returned.
func (r *ReturnRepo) ListReturnableItems(ctx context.Context, transactionID string) ([]ReturnableItemRow, error) {
	const q = `
SELECT
  ti.id::text,
  ti.product_id::text,
  p.name,
  p.sku,
  ti.unit_amount::text,
  ti.qty,
  COALESCE(SUM(ri.qty), 0)::int              AS qty_returned,
  (ti.qty - COALESCE(SUM(ri.qty), 0))::int   AS qty_returnable
FROM transaction_items ti
JOIN products p ON p.id = ti.product_id
LEFT JOIN return_items ri ON ri.transaction_item_id = ti.id
WHERE ti.transaction_id = $1::uuid
GROUP BY ti.id, ti.product_id, p.name, p.sku, ti.unit_amount, ti.qty
ORDER BY p.name ASC;
`
	rows, err := r.db.Query(ctx, q, transactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ReturnableItemRow, 0, 10)
	for rows.Next() {
		var it ReturnableItemRow
		if err := rows.Scan(
			&it.TransactionItemID, &it.ProductID, &it.ProductName, &it.SKU,
			&it.UnitAmount, &it.QtySold, &it.QtyReturned, &it.QtyReturnable,
		); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

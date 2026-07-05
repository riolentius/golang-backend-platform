package postgres

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProductMissing  = errors.New("product not found")
	ErrInvalidPackSize = errors.New("quantity does not convert to a whole base unit")
)

type MovementRow struct {
	ID             string
	ProductID      string
	Direction      string
	Quantity       int
	Source         string
	ReferenceID    *string
	Note           *string
	CreatedByEmail *string
	CreatedAt      time.Time
}

type StockRepo struct {
	db *pgxpool.Pool
}

func NewStockRepo(db *pgxpool.Pool) *StockRepo {
	return &StockRepo{db: db}
}

func (r *StockRepo) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.db.BeginTx(ctx, pgx.TxOptions{})
}

func resolveStockProduct(ctx context.Context, tx pgx.Tx, productID string) (stockProductID string, packSize float64, err error) {
	const q = `
SELECT
  COALESCE(base_product_id, id)::text AS stock_product_id,
  pack_size::text
FROM products
WHERE id = $1::uuid;
`
	var packStr string
	if err := tx.QueryRow(ctx, q, productID).Scan(&stockProductID, &packStr); err != nil {
		return "", 0, err
	}
	ps, err := strconv.ParseFloat(packStr, 64)
	if err != nil {
		return "", 0, err
	}
	return stockProductID, ps, nil
}

func (r *StockRepo) StockIn(
	ctx context.Context,
	productID string,
	quantity int,
	note *string,
	createdByEmail *string,
) (*MovementRow, string, int, error) {
	tx, err := r.Begin(ctx)
	if err != nil {
		return nil, "", 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	stockProductID, packSize, err := resolveStockProduct(ctx, tx, productID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", 0, ErrProductMissing
		}
		return nil, "", 0, err
	}

	baseQtyFloat := float64(quantity) * packSize
	baseQty := int(baseQtyFloat)
	if baseQtyFloat != float64(baseQty) {
		return nil, "", 0, ErrInvalidPackSize
	}

	const lockQ = `SELECT 1 FROM products WHERE id = $1::uuid FOR UPDATE;`
	if _, err := tx.Exec(ctx, lockQ, stockProductID); err != nil {
		return nil, "", 0, err
	}

	const updateQ = `
UPDATE products
SET stock_on_hand = stock_on_hand + $2,
    updated_at = now()
WHERE id = $1::uuid
RETURNING stock_on_hand;
`
	var newStockOnHand int
	if err := tx.QueryRow(ctx, updateQ, stockProductID, baseQty).Scan(&newStockOnHand); err != nil {
		return nil, "", 0, err
	}

	const insertQ = `
INSERT INTO stock_movements (product_id, direction, quantity, source, note, created_by_email)
VALUES ($1::uuid, 'in', $2, 'manual', $3, $4)
RETURNING id::text, product_id::text, direction, quantity, source, reference_id::text, note, created_by_email, created_at;
`
	var row MovementRow
	if err := tx.QueryRow(ctx, insertQ, productID, quantity, note, createdByEmail).Scan(
		&row.ID, &row.ProductID, &row.Direction, &row.Quantity, &row.Source,
		&row.ReferenceID, &row.Note, &row.CreatedByEmail, &row.CreatedAt,
	); err != nil {
		return nil, "", 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, "", 0, err
	}

	return &row, stockProductID, newStockOnHand, nil
}

func (r *StockRepo) ListByProduct(ctx context.Context, productID string, filter stockucFilter) ([]MovementRow, error) {
	const q = `
SELECT
  id::text, product_id::text, direction, quantity, source,
  reference_id::text, note, created_by_email, created_at
FROM stock_movements
WHERE product_id = $1::uuid
  AND ($2::text        IS NULL OR direction   =  $2)
  AND ($3::timestamptz IS NULL OR created_at  >= $3)
  AND ($4::timestamptz IS NULL OR created_at  <  $4 + interval '1 day')
ORDER BY created_at DESC
LIMIT $5 OFFSET $6;
`
	rows, err := r.db.Query(ctx, q,
		productID,
		filter.Direction,
		filter.From,
		filter.To,
		filter.Limit,
		filter.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]MovementRow, 0, filter.Limit)
	for rows.Next() {
		var m MovementRow
		if err := rows.Scan(
			&m.ID, &m.ProductID, &m.Direction, &m.Quantity, &m.Source,
			&m.ReferenceID, &m.Note, &m.CreatedByEmail, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type stockucFilter struct {
	Direction *string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

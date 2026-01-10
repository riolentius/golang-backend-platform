package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductRow struct {
	ID            string
	SKU           *string
	Name          string
	Description   *string
	IsActive      bool
	StockOnHand   int
	StockReserved int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ProductRepo struct {
	db *pgxpool.Pool
}

func NewProductRepo(db *pgxpool.Pool) *ProductRepo {
	return &ProductRepo{db: db}
}

func (r *ProductRepo) Create(
	ctx context.Context,
	sku *string,
	name string,
	description *string,
	stockOnHand int,
) (*ProductRow, error) {
	const q = `
INSERT INTO products (sku, name, description, stock_on_hand)
VALUES ($1, $2, $3, $4)
RETURNING
  id::text, sku, name, description, is_active,
  stock_on_hand, stock_reserved,
  created_at, updated_at;
`
	row := r.db.QueryRow(ctx, q, sku, name, description, stockOnHand)

	var out ProductRow
	if err := row.Scan(
		&out.ID,
		&out.SKU,
		&out.Name,
		&out.Description,
		&out.IsActive,
		&out.StockOnHand,
		&out.StockReserved,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ProductRepo) List(ctx context.Context, limit int, offset int) ([]ProductRow, error) {
	const q = `
SELECT
  id::text, sku, name, description, is_active,
  stock_on_hand, stock_reserved,
  created_at, updated_at
FROM products
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
`
	rows, err := r.db.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ProductRow, 0, limit)
	for rows.Next() {
		var p ProductRow
		if err := rows.Scan(
			&p.ID,
			&p.SKU,
			&p.Name,
			&p.Description,
			&p.IsActive,
			&p.StockOnHand,
			&p.StockReserved,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ProductRepo) Update(
	ctx context.Context,
	id string,
	sku *string,
	name *string,
	description *string,
	isActive *bool,
	stockOnHand *int,
) (*ProductRow, error) {
	const q = `
UPDATE products
SET
  sku = COALESCE($2, sku),
  name = COALESCE($3, name),
  description = COALESCE($4, description),
  is_active = COALESCE($5, is_active),
  stock_on_hand = COALESCE($6, stock_on_hand),
  updated_at = now()
WHERE id = $1::uuid
RETURNING
  id::text, sku, name, description, is_active,
  stock_on_hand, stock_reserved,
  created_at, updated_at;
`
	row := r.db.QueryRow(ctx, q, id, sku, name, description, isActive, stockOnHand)

	var out ProductRow
	if err := row.Scan(
		&out.ID,
		&out.SKU,
		&out.Name,
		&out.Description,
		&out.IsActive,
		&out.StockOnHand,
		&out.StockReserved,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

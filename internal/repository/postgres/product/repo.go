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
	Cost          string
	IsActive      bool
	StockOnHand   int
	StockReserved int
	CategoryID    *string
	CategoryName  *string
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
	cost string,
	stockOnHand int,
	categoryID *string,
) (*ProductRow, error) {
	const q = `
WITH inserted AS (
  INSERT INTO products (sku, name, description, cost, stock_on_hand, category_id)
  VALUES ($1, $2, $3, $4, $5, $6::uuid)
  RETURNING id, sku, name, description, cost, is_active, stock_on_hand, stock_reserved, category_id, created_at, updated_at
)
SELECT
  i.id::text, i.sku, i.name, i.description, i.cost::text,
  i.is_active, i.stock_on_hand, i.stock_reserved,
  i.category_id::text, pc.name AS category_name,
  i.created_at, i.updated_at
FROM inserted i
LEFT JOIN product_categories pc ON pc.id = i.category_id;
`
	var out ProductRow
	if err := r.db.QueryRow(ctx, q, sku, name, description, cost, stockOnHand, categoryID).Scan(
		&out.ID, &out.SKU, &out.Name, &out.Description, &out.Cost,
		&out.IsActive, &out.StockOnHand, &out.StockReserved,
		&out.CategoryID, &out.CategoryName,
		&out.CreatedAt, &out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ProductRepo) List(ctx context.Context, limit int, offset int) ([]ProductRow, error) {
	const q = `
SELECT
  p.id::text, p.sku, p.name, p.description, p.cost::text,
  p.is_active, p.stock_on_hand, p.stock_reserved,
  p.category_id::text, pc.name AS category_name,
  p.created_at, p.updated_at
FROM products p
LEFT JOIN product_categories pc ON pc.id = p.category_id
ORDER BY p.created_at DESC
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
			&p.ID, &p.SKU, &p.Name, &p.Description, &p.Cost,
			&p.IsActive, &p.StockOnHand, &p.StockReserved,
			&p.CategoryID, &p.CategoryName,
			&p.CreatedAt, &p.UpdatedAt,
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
	cost *string,
	isActive *bool,
	stockOnHand *int,
	categoryID *string,
) (*ProductRow, error) {
	const q = `
WITH updated AS (
  UPDATE products
  SET
    sku           = COALESCE($2, sku),
    name          = COALESCE($3, name),
    description   = COALESCE($4, description),
    cost          = COALESCE($5::numeric, cost),
    is_active     = COALESCE($6, is_active),
    stock_on_hand = COALESCE($7, stock_on_hand),
    category_id   = COALESCE($8::uuid, category_id),
    updated_at    = now()
  WHERE id = $1::uuid
  RETURNING id, sku, name, description, cost, is_active, stock_on_hand, stock_reserved, category_id, created_at, updated_at
)
SELECT
  u.id::text, u.sku, u.name, u.description, u.cost::text,
  u.is_active, u.stock_on_hand, u.stock_reserved,
  u.category_id::text, pc.name AS category_name,
  u.created_at, u.updated_at
FROM updated u
LEFT JOIN product_categories pc ON pc.id = u.category_id;
`
	var out ProductRow
	if err := r.db.QueryRow(ctx, q,
		id, sku, name, description, cost, isActive, stockOnHand, categoryID,
	).Scan(
		&out.ID, &out.SKU, &out.Name, &out.Description, &out.Cost,
		&out.IsActive, &out.StockOnHand, &out.StockReserved,
		&out.CategoryID, &out.CategoryName,
		&out.CreatedAt, &out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductCategoryRow struct {
	ID          string
	Name        string
	Description *string
	CreatedAt   time.Time
}

type ProductCategoryRepo struct {
	db *pgxpool.Pool
}

func NewProductCategoryRepo(db *pgxpool.Pool) *ProductCategoryRepo {
	return &ProductCategoryRepo{db: db}
}

func (r *ProductCategoryRepo) List(ctx context.Context) ([]ProductCategoryRow, error) {
	const q = `
SELECT id::text, name, description, created_at
FROM product_categories
ORDER BY name ASC;
`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ProductCategoryRow, 0)
	for rows.Next() {
		var row ProductCategoryRow
		if err := rows.Scan(&row.ID, &row.Name, &row.Description, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *ProductCategoryRepo) Create(ctx context.Context, name string, description *string) (*ProductCategoryRow, error) {
	const q = `
INSERT INTO product_categories (name, description)
VALUES ($1, $2)
RETURNING id::text, name, description, created_at;
`
	var out ProductCategoryRow
	if err := r.db.QueryRow(ctx, q, name, description).Scan(
		&out.ID, &out.Name, &out.Description, &out.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ProductCategoryRepo) Update(ctx context.Context, id string, name *string, description *string) (*ProductCategoryRow, error) {
	const q = `
UPDATE product_categories
SET
  name        = COALESCE($2, name),
  description = COALESCE($3, description),
  updated_at  = now()
WHERE id = $1::uuid
RETURNING id::text, name, description, created_at;
`
	var out ProductCategoryRow
	if err := r.db.QueryRow(ctx, q, id, name, description).Scan(
		&out.ID, &out.Name, &out.Description, &out.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

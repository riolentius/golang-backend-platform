package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductPriceRow struct {
	ID         string
	ProductID  string
	CategoryID *string
	Currency   string
	Amount     string // keep as string for safety; convert later if you want decimal type
	ValidFrom  time.Time
	ValidTo    *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ProductPriceRepo struct {
	db *pgxpool.Pool
}

func NewProductPriceRepo(db *pgxpool.Pool) *ProductPriceRepo {
	return &ProductPriceRepo{db: db}
}

func (r *ProductPriceRepo) Create(ctx context.Context, productID string, categoryID *string, currency string, amount string, validFrom *time.Time, validTo *time.Time) (*ProductPriceRow, error) {
	const q = `
INSERT INTO product_prices (product_id, category_id, currency, amount, valid_from, valid_to)
VALUES ($1::uuid, $2::uuid, $3, $4::numeric, COALESCE($5, now()), $6)
RETURNING id::text, product_id::text, category_id::text, currency, amount::text, valid_from, valid_to, created_at, updated_at;
`
	row := r.db.QueryRow(ctx, q, productID, categoryID, currency, amount, validFrom, validTo)

	var out ProductPriceRow
	if err := row.Scan(&out.ID, &out.ProductID, &out.CategoryID, &out.Currency, &out.Amount, &out.ValidFrom, &out.ValidTo, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ProductPriceRepo) ListByProduct(ctx context.Context, productID string) ([]ProductPriceRow, error) {
	const q = `
SELECT id::text, product_id::text, category_id::text, currency, amount::text, valid_from, valid_to, created_at, updated_at
FROM product_prices
WHERE product_id = $1::uuid
ORDER BY created_at DESC;
`
	rows, err := r.db.Query(ctx, q, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProductPriceRow
	for rows.Next() {
		var p ProductPriceRow
		if err := rows.Scan(&p.ID, &p.ProductID, &p.CategoryID, &p.Currency, &p.Amount, &p.ValidFrom, &p.ValidTo, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ProductPriceRepo) Update(ctx context.Context, id string, currency *string, amount *string, validFrom *time.Time, validTo *time.Time, categoryID *string) (*ProductPriceRow, error) {
	const q = `
UPDATE product_prices
SET
  currency = COALESCE($2, currency),
  amount = COALESCE($3::numeric, amount),
  valid_from = COALESCE($4, valid_from),
  valid_to = COALESCE($5, valid_to),
  category_id = COALESCE($6::uuid, category_id),
  updated_at = now()
WHERE id = $1::uuid
RETURNING id::text, product_id::text, category_id::text, currency, amount::text, valid_from, valid_to, created_at, updated_at;
`
	row := r.db.QueryRow(ctx, q, id, currency, amount, validFrom, validTo, categoryID)

	var out ProductPriceRow
	if err := row.Scan(&out.ID, &out.ProductID, &out.CategoryID, &out.Currency, &out.Amount, &out.ValidFrom, &out.ValidTo, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomerCategoryRow struct {
	ID          string
	Code        string
	Name        string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CustomerCategoryRepo struct {
	db *pgxpool.Pool
}

func NewCustomerCategoryRepo(db *pgxpool.Pool) *CustomerCategoryRepo {
	return &CustomerCategoryRepo{db: db}
}

func (r *CustomerCategoryRepo) List(ctx context.Context) ([]CustomerCategoryRow, error) {
	const q = `
SELECT
  id::text, code, name, description, created_at, updated_at
FROM customer_categories
ORDER BY name ASC;
`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]CustomerCategoryRow, 0)
	for rows.Next() {
		var c CustomerCategoryRow
		if err := rows.Scan(
			&c.ID,
			&c.Code,
			&c.Name,
			&c.Description,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

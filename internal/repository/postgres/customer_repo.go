package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomerRow struct {
	ID                   string
	FirstName            string
	LastName             *string
	Email                string
	Phone                *string
	IdentificationNumber *string
	CategoryID           *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type CustomerRepo struct {
	db *pgxpool.Pool
}

func NewCustomerRepo(db *pgxpool.Pool) *CustomerRepo {
	return &CustomerRepo{db: db}
}

func (r *CustomerRepo) Create(ctx context.Context, in CustomerRow) (*CustomerRow, error) {
	const q = `
INSERT INTO customers (
  id, first_name, last_name, email, phone, identification_number, category_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING
  id::text, first_name, last_name, email, phone, identification_number, category_id, created_at, updated_at;
`
	id := uuid.New().String()

	var out CustomerRow
	err := r.db.QueryRow(ctx, q,
		id,
		in.FirstName,
		in.LastName,
		in.Email,
		in.Phone,
		in.IdentificationNumber,
		in.CategoryID,
	).Scan(
		&out.ID,
		&out.FirstName,
		&out.LastName,
		&out.Email,
		&out.Phone,
		&out.IdentificationNumber,
		&out.CategoryID,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *CustomerRepo) GetByID(ctx context.Context, id string) (*CustomerRow, error) {
	const q = `
SELECT
  id::text, first_name, last_name, email, phone, identification_number, category_id, created_at, updated_at
FROM customers
WHERE id = $1
LIMIT 1;
`
	var out CustomerRow
	if err := r.db.QueryRow(ctx, q, id).Scan(
		&out.ID,
		&out.FirstName,
		&out.LastName,
		&out.Email,
		&out.Phone,
		&out.IdentificationNumber,
		&out.CategoryID,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	return &out, nil
}

func (r *CustomerRepo) List(ctx context.Context, limit, offset int) ([]CustomerRow, error) {
	const q = `
SELECT
  id::text, first_name, last_name, email, phone, identification_number, category_id, created_at, updated_at
FROM customers
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
`
	rows, err := r.db.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]CustomerRow, 0, limit)
	for rows.Next() {
		var c CustomerRow
		if err := rows.Scan(
			&c.ID,
			&c.FirstName,
			&c.LastName,
			&c.Email,
			&c.Phone,
			&c.IdentificationNumber,
			&c.CategoryID,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CustomerRepo) Update(ctx context.Context, id string, in CustomerRow) (*CustomerRow, error) {
	const q = `
UPDATE customers
SET
  first_name = COALESCE($2, first_name),
  last_name  = COALESCE($3, last_name),
  email      = COALESCE($4, email),
  phone      = COALESCE($5, phone),
  identification_number = COALESCE($6, identification_number),
  category_id = COALESCE($7, category_id),
  updated_at = now()
WHERE id = $1
RETURNING
  id::text, first_name, last_name, email, phone, identification_number, category_id, created_at, updated_at;
`

	var out CustomerRow
	err := r.db.QueryRow(ctx, q,
		id,
		nullIfEmptyStrPtr(in.FirstName),
		in.LastName,
		in.Email,
		in.Phone,
		in.IdentificationNumber,
		in.CategoryID,
	).Scan(
		&out.ID,
		&out.FirstName,
		&out.LastName,
		&out.Email,
		&out.Phone,
		&out.IdentificationNumber,
		&out.CategoryID,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	return &out, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// used only for update where first_name is string not *string in CustomerRow
func nullIfEmptyStrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

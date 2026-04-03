package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomerAddressRow struct {
	ID           string
	CustomerID   string
	Label        *string
	AddressLine1 string
	AddressLine2 *string
	City         *string
	Province     *string
	PostalCode   *string
	Country      string
	IsDefault    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Separate struct for Update where required fields become optional
type CustomerAddressUpdateRow struct {
	Label        *string
	AddressLine1 *string
	AddressLine2 *string
	City         *string
	Province     *string
	PostalCode   *string
	Country      *string
	IsDefault    *bool
}

type CustomerAddressRepo struct {
	db *pgxpool.Pool
}

func NewCustomerAddressRepo(db *pgxpool.Pool) *CustomerAddressRepo {
	return &CustomerAddressRepo{db: db}
}

func (r *CustomerAddressRepo) Create(ctx context.Context, customerID string, in CustomerAddressRow) (*CustomerAddressRow, error) {
	const q = `
INSERT INTO customer_addresses (
  customer_id, label, address_line1, address_line2,
  city, province, postal_code, country, is_default
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING
  id::text, customer_id::text, label, address_line1, address_line2,
  city, province, postal_code, country, is_default, created_at, updated_at;
`
	var out CustomerAddressRow
	err := r.db.QueryRow(ctx, q,
		customerID,
		in.Label, in.AddressLine1, in.AddressLine2,
		in.City, in.Province, in.PostalCode,
		in.Country, in.IsDefault,
	).Scan(
		&out.ID, &out.CustomerID, &out.Label,
		&out.AddressLine1, &out.AddressLine2,
		&out.City, &out.Province, &out.PostalCode,
		&out.Country, &out.IsDefault,
		&out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *CustomerAddressRepo) ListByCustomer(ctx context.Context, customerID string) ([]CustomerAddressRow, error) {
	const q = `
SELECT
  id::text, customer_id::text, label, address_line1, address_line2,
  city, province, postal_code, country, is_default, created_at, updated_at
FROM customer_addresses
WHERE customer_id = $1
ORDER BY is_default DESC, created_at ASC;
`
	rows, err := r.db.Query(ctx, q, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]CustomerAddressRow, 0)
	for rows.Next() {
		var a CustomerAddressRow
		if err := rows.Scan(
			&a.ID, &a.CustomerID, &a.Label,
			&a.AddressLine1, &a.AddressLine2,
			&a.City, &a.Province, &a.PostalCode,
			&a.Country, &a.IsDefault,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *CustomerAddressRepo) GetByID(ctx context.Context, id string) (*CustomerAddressRow, error) {
	const q = `
SELECT
  id::text, customer_id::text, label, address_line1, address_line2,
  city, province, postal_code, country, is_default, created_at, updated_at
FROM customer_addresses WHERE id = $1 LIMIT 1;
`
	var out CustomerAddressRow
	err := r.db.QueryRow(ctx, q, id).Scan(
		&out.ID, &out.CustomerID, &out.Label,
		&out.AddressLine1, &out.AddressLine2,
		&out.City, &out.Province, &out.PostalCode,
		&out.Country, &out.IsDefault,
		&out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	return &out, nil
}

// Update uses *string for all fields so COALESCE works correctly
func (r *CustomerAddressRepo) Update(ctx context.Context, id string, in CustomerAddressUpdateRow) (*CustomerAddressRow, error) {
	const q = `
UPDATE customer_addresses SET
  label         = COALESCE($2, label),
  address_line1 = COALESCE($3, address_line1),
  address_line2 = COALESCE($4, address_line2),
  city          = COALESCE($5, city),
  province      = COALESCE($6, province),
  postal_code   = COALESCE($7, postal_code),
  country       = COALESCE($8, country),
  is_default    = COALESCE($9, is_default),
  updated_at    = now()
WHERE id = $1
RETURNING
  id::text, customer_id::text, label, address_line1, address_line2,
  city, province, postal_code, country, is_default, created_at, updated_at;
`
	var out CustomerAddressRow
	err := r.db.QueryRow(ctx, q,
		id,
		in.Label, in.AddressLine1, in.AddressLine2,
		in.City, in.Province, in.PostalCode,
		in.Country, in.IsDefault,
	).Scan(
		&out.ID, &out.CustomerID, &out.Label,
		&out.AddressLine1, &out.AddressLine2,
		&out.City, &out.Province, &out.PostalCode,
		&out.Country, &out.IsDefault,
		&out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	return &out, nil
}

func (r *CustomerAddressRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM customer_addresses WHERE id = $1`, id)
	return err
}

func (r *CustomerAddressRepo) ClearDefault(ctx context.Context, customerID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE customer_addresses SET is_default = false WHERE customer_id = $1`,
		customerID,
	)
	return err
}

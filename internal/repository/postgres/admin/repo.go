package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminRow struct {
	ID           string
	Email        string
	PasswordHash string
	IsActive     bool
}

type AdminRepo struct {
	db *pgxpool.Pool
}

func NewAdminRepo(db *pgxpool.Pool) *AdminRepo {
	return &AdminRepo{db: db}
}

func (r *AdminRepo) FindByEmail(ctx context.Context, email string) (*AdminRow, error) {
	const q = `
SELECT id::text, email, password_hash, is_active
FROM admins
WHERE email = $1
LIMIT 1;
`
	row := r.db.QueryRow(ctx, q, email)

	var out AdminRow
	if err := row.Scan(&out.ID, &out.Email, &out.PasswordHash, &out.IsActive); err != nil {
		return nil, err
	}
	return &out, nil
}

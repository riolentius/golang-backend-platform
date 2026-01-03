package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Admin struct {
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

func (r *AdminRepo) FindByEmail(ctx context.Context, email string) (*Admin, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id::text, email, password_hash, is_active
		FROM admins
		WHERE email = $1
	`, email)

	var a Admin
	if err := row.Scan(&a.ID, &a.Email, &a.PasswordHash, &a.IsActive); err != nil {
		return nil, err
	}
	return &a, nil
}

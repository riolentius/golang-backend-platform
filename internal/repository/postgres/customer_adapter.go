package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	customeruc "github.com/riolentius/cahaya-gading-backend/internal/usecase/customer"
)

type CustomerStoreAdapter struct {
	repo *CustomerRepo
}

func NewCustomerStoreAdapter(repo *CustomerRepo) *CustomerStoreAdapter {
	return &CustomerStoreAdapter{repo: repo}
}

func (a *CustomerStoreAdapter) Create(ctx context.Context, in customeruc.CreateInput) (*customeruc.Customer, error) {
	row, err := a.repo.Create(ctx, CustomerRow{
		FirstName:            in.FirstName,
		LastName:             in.LastName,
		Email:                in.Email,
		Phone:                in.Phone,
		IdentificationNumber: in.IdentificationNumber,
		CategoryID:           in.CategoryID,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, customeruc.ErrEmailConflict
		}
		return nil, err
	}
	return mapCustomer(row), nil
}

func (a *CustomerStoreAdapter) GetByID(ctx context.Context, id string) (*customeruc.Customer, error) {
	row, err := a.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, customeruc.ErrNotFound
		}
		return nil, err
	}
	return mapCustomer(row), nil
}

func (a *CustomerStoreAdapter) List(ctx context.Context, q customeruc.ListQuery) ([]customeruc.Customer, error) {
	rows, err := a.repo.List(ctx, q.Limit, q.Offset)
	if err != nil {
		return nil, err
	}
	out := make([]customeruc.Customer, 0, len(rows))
	for i := range rows {
		out = append(out, *mapCustomer(&rows[i]))
	}
	return out, nil
}

func (a *CustomerStoreAdapter) Update(ctx context.Context, id string, in customeruc.UpdateInput) (*customeruc.Customer, error) {
	rowIn := CustomerRow{}
	if in.FirstName != nil {
		rowIn.FirstName = *in.FirstName
	}
	rowIn.LastName = in.LastName
	rowIn.Email = in.Email
	rowIn.Phone = in.Phone
	rowIn.IdentificationNumber = in.IdentificationNumber
	rowIn.CategoryID = in.CategoryID

	row, err := a.repo.Update(ctx, id, rowIn)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, customeruc.ErrEmailConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, customeruc.ErrNotFound
		}
		return nil, err
	}
	return mapCustomer(row), nil
}

func mapCustomer(r *CustomerRow) *customeruc.Customer {
	return &customeruc.Customer{
		ID:                   r.ID,
		FirstName:            r.FirstName,
		LastName:             r.LastName,
		Email:                r.Email,
		Phone:                r.Phone,
		IdentificationNumber: r.IdentificationNumber,
		CategoryID:           r.CategoryID,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

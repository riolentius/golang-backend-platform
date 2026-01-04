package transaction

import (
	"context"
	"errors"
)

var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrCustomerMissing = errors.New("customer not found")
	ErrProductMissing  = errors.New("product not found")
	ErrPriceMissing    = errors.New("product price not found")
)

type Store interface {
	Create(ctx context.Context, in CreateInput) (*Transaction, error)
	List(ctx context.Context, in ListInput) ([]Transaction, error)
	GetByID(ctx context.Context, id string) (*Transaction, error)
	UpdateStatus(ctx context.Context, id string, status string) (*Transaction, error)
}

type Usecase struct {
	store Store
}

func New(store Store) *Usecase {
	return &Usecase{store: store}
}

func (u *Usecase) Create(ctx context.Context, in CreateInput) (*Transaction, error) {
	if in.CustomerID == "" || len(in.Items) == 0 {
		return nil, ErrInvalidInput
	}
	for _, it := range in.Items {
		if it.ProductID == "" || it.Qty <= 0 {
			return nil, ErrInvalidInput
		}
	}
	return u.store.Create(ctx, in)
}

func (u *Usecase) List(ctx context.Context, in ListInput) ([]Transaction, error) {
	if in.Limit <= 0 || in.Limit > 100 {
		in.Limit = 20
	}
	if in.Offset < 0 {
		in.Offset = 0
	}
	return u.store.List(ctx, in)
}

func (u *Usecase) GetByID(ctx context.Context, id string) (*Transaction, error) {
	if id == "" {
		return nil, ErrInvalidInput
	}
	return u.store.GetByID(ctx, id)
}

func (u *Usecase) UpdateStatus(ctx context.Context, id string, in UpdateStatusInput) (*Transaction, error) {
	if id == "" || in.Status == "" {
		return nil, ErrInvalidInput
	}
	return u.store.UpdateStatus(ctx, id, in.Status)
}

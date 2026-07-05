package stock

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrProductMissing  = errors.New("product not found")
	ErrInvalidPackSize = errors.New("quantity does not convert to a whole base unit for this product")
)

type Store interface {
	StockIn(ctx context.Context, productID string, quantity int, note *string, createdByEmail *string) (*StockInResult, error)
	ListByProduct(ctx context.Context, productID string, filter ListFilter) ([]Movement, error)
}

type Usecase struct {
	store Store
}

func New(store Store) *Usecase {
	return &Usecase{store: store}
}

func (u *Usecase) StockIn(ctx context.Context, productID string, in StockInInput, createdByEmail *string) (*StockInResult, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" || in.Quantity <= 0 {
		return nil, ErrInvalidInput
	}
	return u.store.StockIn(ctx, productID, in.Quantity, in.Note, createdByEmail)
}

func (u *Usecase) ListByProduct(ctx context.Context, productID string, filter ListFilter) ([]Movement, error) {
	if strings.TrimSpace(productID) == "" {
		return nil, ErrInvalidInput
	}
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return u.store.ListByProduct(ctx, productID, filter)
}

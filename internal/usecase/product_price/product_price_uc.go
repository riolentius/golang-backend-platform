package product_price

import (
	"context"
	"errors"
	"time"
)

type Store interface {
	CreateForProduct(ctx context.Context, productID string, in CreateInput) (*ProductPrice, error)
	ListForProduct(ctx context.Context, productID string) ([]ProductPrice, error)
	Update(ctx context.Context, priceID string, in UpdateInput) (*ProductPrice, error)
}

type Usecase struct {
	store Store
}

func New(store Store) *Usecase {
	return &Usecase{store: store}
}

func (u *Usecase) CreateForProduct(ctx context.Context, productID string, in CreateInput) (*ProductPrice, error) {
	if productID == "" {
		return nil, errors.New("product id is required")
	}
	if in.Currency == "" {
		in.Currency = "IDR"
	}
	if in.ValidFrom == nil {
		now := time.Now()
		in.ValidFrom = &now
	}
	if in.Amount == "" {
		return nil, errors.New("amount is required")
	}
	return u.store.CreateForProduct(ctx, productID, in)
}

func (u *Usecase) ListForProduct(ctx context.Context, productID string) ([]ProductPrice, error) {
	if productID == "" {
		return nil, errors.New("product id is required")
	}
	return u.store.ListForProduct(ctx, productID)
}

func (u *Usecase) Update(ctx context.Context, priceID string, in UpdateInput) (*ProductPrice, error) {
	if priceID == "" {
		return nil, errors.New("price id is required")
	}
	return u.store.Update(ctx, priceID, in)
}

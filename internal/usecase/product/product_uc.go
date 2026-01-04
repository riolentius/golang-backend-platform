package product

import (
	"context"
	"errors"
)

var ErrInvalidInput = errors.New("invalid input")

type Product struct {
	ID          string  `json:"id"`
	SKU         *string `json:"sku,omitempty"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	IsActive    bool    `json:"isActive"`
}

type ProductStore interface {
	Create(ctx context.Context, sku *string, name string, description *string) (*Product, error)
	List(ctx context.Context, limit int, offset int) ([]Product, error)
	Update(ctx context.Context, id string, sku *string, name *string, description *string, isActive *bool) (*Product, error)
}

type Usecase struct {
	store ProductStore
}

func New(store ProductStore) *Usecase {
	return &Usecase{store: store}
}

type CreateInput struct {
	SKU         *string `json:"sku"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (u *Usecase) Create(ctx context.Context, in CreateInput) (*Product, error) {
	if in.Name == "" {
		return nil, ErrInvalidInput
	}
	return u.store.Create(ctx, in.SKU, in.Name, in.Description)
}

func (u *Usecase) List(ctx context.Context, limit, offset int) ([]Product, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return u.store.List(ctx, limit, offset)
}

type UpdateInput struct {
	SKU         *string `json:"sku"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"isActive"`
}

func (u *Usecase) Update(ctx context.Context, id string, in UpdateInput) (*Product, error) {
	if id == "" {
		return nil, ErrInvalidInput
	}
	return u.store.Update(ctx, id, in.SKU, in.Name, in.Description, in.IsActive)
}

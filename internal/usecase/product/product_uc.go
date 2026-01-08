package product

import (
	"context"
	"errors"
	"strings"
)

var ErrInvalidInput = errors.New("invalid input")

type Product struct {
	ID            string  `json:"id"`
	SKU           *string `json:"sku,omitempty"`
	Name          string  `json:"name"`
	Description   *string `json:"description,omitempty"`
	IsActive      bool    `json:"isActive"`
	StockOnHand   int     `json:"stockOnHand"`
	StockReserved int     `json:"stockReserved"`
}

type ProductStore interface {
	Create(ctx context.Context, sku *string, name string, description *string, stockOnHand int) (*Product, error)
	List(ctx context.Context, limit int, offset int) ([]Product, error)
	Update(ctx context.Context, id string, sku *string, name *string, description *string, isActive *bool, stockOnHand *int) (*Product, error)
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
	StockOnHand *int    `json:"stockOnHand"` // optional; default 0
}

func (u *Usecase) Create(ctx context.Context, in CreateInput) (*Product, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}

	stock := 0
	if in.StockOnHand != nil {
		if *in.StockOnHand < 0 {
			return nil, ErrInvalidInput
		}
		stock = *in.StockOnHand
	}

	return u.store.Create(ctx, in.SKU, name, in.Description, stock)
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
	StockOnHand *int    `json:"stockOnHand"` // admin-adjustable
}

func (u *Usecase) Update(ctx context.Context, id string, in UpdateInput) (*Product, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidInput
	}

	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return nil, ErrInvalidInput
		}
		in.Name = &n
	}

	if in.StockOnHand != nil && *in.StockOnHand < 0 {
		return nil, ErrInvalidInput
	}

	return u.store.Update(ctx, id, in.SKU, in.Name, in.Description, in.IsActive, in.StockOnHand)
}

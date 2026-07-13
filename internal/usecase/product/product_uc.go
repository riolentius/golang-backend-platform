package product

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("product not found")
)

// ProductCategory is embedded in Product responses.
type ProductCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Product struct {
	ID            string           `json:"id"`
	SKU           *string          `json:"sku,omitempty"`
	Name          string           `json:"name"`
	Description   *string          `json:"description,omitempty"`
	Cost          string           `json:"cost"`
	IsActive      bool             `json:"isActive"`
	StockOnHand   int              `json:"stockOnHand"`
	StockReserved int              `json:"stockReserved"`
	Category      *ProductCategory `json:"category,omitempty"`
}

// ListParams holds server-side filtering + pagination for the product list.
type ListParams struct {
	Search string // matches name or sku (case-insensitive substring); "" = no filter
	Status string // "active" | "inactive" | "" (all)
	Limit  int
	Offset int
}

// ListResult carries the page of items plus the total count of all rows that
// match the filter (before limit/offset) — so the frontend can render pagination.
type ListResult struct {
	Items []Product `json:"items"`
	Total int       `json:"total"`
}

type ProductStore interface {
	Create(ctx context.Context, sku *string, name string, description *string, cost string, stockOnHand int, categoryID *string, createdByEmail *string) (*Product, error)
	List(ctx context.Context, p ListParams) ([]Product, int, error)
	GetByID(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, id string, sku *string, name *string, description *string, cost *string, isActive *bool, stockOnHand *int, categoryID *string, createdByEmail *string) (*Product, error)
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
	Cost        string  `json:"cost"`
	StockOnHand *int    `json:"stockOnHand"`
	CategoryID  *string `json:"categoryId"`
}

func (u *Usecase) Create(ctx context.Context, in CreateInput, createdByEmail *string) (*Product, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}

	cost := strings.TrimSpace(in.Cost)
	if cost == "" {
		cost = "0"
	}

	stock := 0
	if in.StockOnHand != nil {
		if *in.StockOnHand < 0 {
			return nil, ErrInvalidInput
		}
		stock = *in.StockOnHand
	}

	return u.store.Create(ctx, in.SKU, name, in.Description, cost, stock, in.CategoryID, createdByEmail)
}

func (u *Usecase) List(ctx context.Context, p ListParams) (*ListResult, error) {
	// Default page size 20; allow up to 200 so the frontend can request larger pages.
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Limit > 200 {
		p.Limit = 200
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	p.Search = strings.TrimSpace(p.Search)

	// normalize status to the two values the repo understands, else treat as "all"
	switch p.Status {
	case "active", "inactive":
		// keep
	default:
		p.Status = ""
	}

	items, total, err := u.store.List(ctx, p)
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: total}, nil
}

type UpdateInput struct {
	SKU         *string `json:"sku"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Cost        *string `json:"cost"`
	IsActive    *bool   `json:"isActive"`
	StockOnHand *int    `json:"stockOnHand"`
	CategoryID  *string `json:"categoryId"`
}

func (u *Usecase) GetByID(ctx context.Context, id string) (*Product, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidInput
	}
	return u.store.GetByID(ctx, id)
}

func (u *Usecase) Update(ctx context.Context, id string, in UpdateInput, createdByEmail *string) (*Product, error) {
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
	return u.store.Update(ctx, id, in.SKU, in.Name, in.Description, in.Cost, in.IsActive, in.StockOnHand, in.CategoryID, createdByEmail)
}

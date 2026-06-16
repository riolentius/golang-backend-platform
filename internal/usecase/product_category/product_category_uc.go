package productcategory

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("product category not found")
)

type ProductCategory struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Store interface {
	List(ctx context.Context) ([]ProductCategory, error)
	Create(ctx context.Context, name string, description *string) (*ProductCategory, error)
	Update(ctx context.Context, id string, name *string, description *string) (*ProductCategory, error)
}

type Usecase struct {
	store Store
}

func New(store Store) *Usecase {
	return &Usecase{store: store}
}

type CreateInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type UpdateInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (u *Usecase) List(ctx context.Context) ([]ProductCategory, error) {
	return u.store.List(ctx)
}

func (u *Usecase) Create(ctx context.Context, in CreateInput) (*ProductCategory, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	return u.store.Create(ctx, name, in.Description)
}

func (u *Usecase) Update(ctx context.Context, id string, in UpdateInput) (*ProductCategory, error) {
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
	return u.store.Update(ctx, id, in.Name, in.Description)
}

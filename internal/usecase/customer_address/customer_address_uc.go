package customer_address

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("address not found")
)

type CustomerAddress struct {
	ID           string    `json:"id"`
	CustomerID   string    `json:"customerId"`
	Label        *string   `json:"label,omitempty"`
	AddressLine1 string    `json:"addressLine1"`
	AddressLine2 *string   `json:"addressLine2,omitempty"`
	City         *string   `json:"city,omitempty"`
	Province     *string   `json:"province,omitempty"`
	PostalCode   *string   `json:"postalCode,omitempty"`
	Country      string    `json:"country"`
	IsDefault    bool      `json:"isDefault"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Label        *string `json:"label"`
	AddressLine1 string  `json:"addressLine1"`
	AddressLine2 *string `json:"addressLine2"`
	City         *string `json:"city"`
	Province     *string `json:"province"`
	PostalCode   *string `json:"postalCode"`
	Country      string  `json:"country"`
	IsDefault    bool    `json:"isDefault"`
}

type UpdateInput struct {
	Label        *string `json:"label"`
	AddressLine1 *string `json:"addressLine1"`
	AddressLine2 *string `json:"addressLine2"`
	City         *string `json:"city"`
	Province     *string `json:"province"`
	PostalCode   *string `json:"postalCode"`
	Country      *string `json:"country"`
	IsDefault    *bool   `json:"isDefault"`
}

type Store interface {
	Create(ctx context.Context, customerID string, in CreateInput) (*CustomerAddress, error)
	ListByCustomer(ctx context.Context, customerID string) ([]CustomerAddress, error)
	GetByID(ctx context.Context, id string) (*CustomerAddress, error)
	Update(ctx context.Context, id string, in UpdateInput) (*CustomerAddress, error)
	Delete(ctx context.Context, id string) error
	ClearDefault(ctx context.Context, customerID string) error
}

type Usecase struct {
	store Store
}

func New(store Store) *Usecase {
	return &Usecase{store: store}
}

func (u *Usecase) Create(ctx context.Context, customerID string, in CreateInput) (*CustomerAddress, error) {
	if _, err := uuid.Parse(customerID); err != nil {
		return nil, ErrInvalidInput
	}
	if strings.TrimSpace(in.AddressLine1) == "" {
		return nil, ErrInvalidInput
	}
	if in.Country == "" {
		in.Country = "ID"
	}

	// If this is marked as default, clear existing defaults first
	if in.IsDefault {
		if err := u.store.ClearDefault(ctx, customerID); err != nil {
			return nil, err
		}
	}

	return u.store.Create(ctx, customerID, in)
}

func (u *Usecase) ListByCustomer(ctx context.Context, customerID string) ([]CustomerAddress, error) {
	if _, err := uuid.Parse(customerID); err != nil {
		return nil, ErrInvalidInput
	}
	return u.store.ListByCustomer(ctx, customerID)
}

func (u *Usecase) Update(ctx context.Context, id string, in UpdateInput) (*CustomerAddress, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidInput
	}

	// If setting as default, clear existing defaults first
	if in.IsDefault != nil && *in.IsDefault {
		addr, err := u.store.GetByID(ctx, id)
		if err != nil {
			return nil, ErrNotFound
		}
		if err := u.store.ClearDefault(ctx, addr.CustomerID); err != nil {
			return nil, err
		}
	}

	return u.store.Update(ctx, id, in)
}

func (u *Usecase) Delete(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalidInput
	}
	return u.store.Delete(ctx, id)
}

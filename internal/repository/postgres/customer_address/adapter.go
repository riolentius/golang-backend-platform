package postgres

import (
	"context"

	addruc "github.com/riolentius/cahaya-gading-backend/internal/usecase/customer_address"
)

type CustomerAddressStoreAdapter struct {
	repo *CustomerAddressRepo
}

func NewCustomerAddressStoreAdapter(repo *CustomerAddressRepo) *CustomerAddressStoreAdapter {
	return &CustomerAddressStoreAdapter{repo: repo}
}

func toUC(r CustomerAddressRow) addruc.CustomerAddress {
	return addruc.CustomerAddress{
		ID:           r.ID,
		CustomerID:   r.CustomerID,
		Label:        r.Label,
		AddressLine1: r.AddressLine1,
		AddressLine2: r.AddressLine2,
		City:         r.City,
		Province:     r.Province,
		PostalCode:   r.PostalCode,
		Country:      r.Country,
		IsDefault:    r.IsDefault,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func (a *CustomerAddressStoreAdapter) Create(ctx context.Context, customerID string, in addruc.CreateInput) (*addruc.CustomerAddress, error) {
	row, err := a.repo.Create(ctx, customerID, CustomerAddressRow{
		Label:        in.Label,
		AddressLine1: in.AddressLine1,
		AddressLine2: in.AddressLine2,
		City:         in.City,
		Province:     in.Province,
		PostalCode:   in.PostalCode,
		Country:      in.Country,
		IsDefault:    in.IsDefault,
	})
	if err != nil {
		return nil, err
	}
	result := toUC(*row)
	return &result, nil
}

func (a *CustomerAddressStoreAdapter) ListByCustomer(ctx context.Context, customerID string) ([]addruc.CustomerAddress, error) {
	rows, err := a.repo.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	out := make([]addruc.CustomerAddress, 0, len(rows))
	for _, r := range rows {
		out = append(out, toUC(r))
	}
	return out, nil
}

func (a *CustomerAddressStoreAdapter) GetByID(ctx context.Context, id string) (*addruc.CustomerAddress, error) {
	row, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	result := toUC(*row)
	return &result, nil
}

func (a *CustomerAddressStoreAdapter) Update(ctx context.Context, id string, in addruc.UpdateInput) (*addruc.CustomerAddress, error) {
	// All fields are *string/*bool — maps directly to CustomerAddressUpdateRow
	row, err := a.repo.Update(ctx, id, CustomerAddressUpdateRow{
		Label:        in.Label,
		AddressLine1: in.AddressLine1,
		AddressLine2: in.AddressLine2,
		City:         in.City,
		Province:     in.Province,
		PostalCode:   in.PostalCode,
		Country:      in.Country,
		IsDefault:    in.IsDefault,
	})
	if err != nil {
		return nil, err
	}
	result := toUC(*row)
	return &result, nil
}

func (a *CustomerAddressStoreAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

func (a *CustomerAddressStoreAdapter) ClearDefault(ctx context.Context, customerID string) error {
	return a.repo.ClearDefault(ctx, customerID)
}

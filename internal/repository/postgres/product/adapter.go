package postgres

import (
	"context"

	productuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product"
)

type ProductStoreAdapter struct {
	repo *ProductRepo
}

func NewProductStoreAdapter(repo *ProductRepo) *ProductStoreAdapter {
	return &ProductStoreAdapter{repo: repo}
}

func (a *ProductStoreAdapter) Create(
	ctx context.Context,
	sku *string,
	name string,
	description *string,
	stockOnHand int,
) (*productuc.Product, error) {
	row, err := a.repo.Create(ctx, sku, name, description, stockOnHand)
	if err != nil {
		return nil, err
	}
	return mapProductRowToUC(row), nil
}

func (a *ProductStoreAdapter) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]productuc.Product, error) {
	rows, err := a.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	out := make([]productuc.Product, 0, len(rows))
	for i := range rows {
		out = append(out, *mapProductRowToUC(&rows[i]))
	}
	return out, nil
}

func (a *ProductStoreAdapter) Update(
	ctx context.Context,
	id string,
	sku *string,
	name *string,
	description *string,
	isActive *bool,
	stockOnHand *int,
) (*productuc.Product, error) {
	row, err := a.repo.Update(ctx, id, sku, name, description, isActive, stockOnHand)
	if err != nil {
		return nil, err
	}
	return mapProductRowToUC(row), nil
}

func mapProductRowToUC(r *ProductRow) *productuc.Product {
	return &productuc.Product{
		ID:            r.ID,
		SKU:           r.SKU,
		Name:          r.Name,
		Description:   r.Description,
		IsActive:      r.IsActive,
		StockOnHand:   r.StockOnHand,
		StockReserved: r.StockReserved,
	}
}

var _ productuc.ProductStore = (*ProductStoreAdapter)(nil)

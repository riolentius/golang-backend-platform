package postgres

import (
	"context"

	priceuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product_price"
)

type ProductPriceStoreAdapter struct {
	repo *ProductPriceRepo
}

func NewProductPriceStoreAdapter(repo *ProductPriceRepo) *ProductPriceStoreAdapter {
	return &ProductPriceStoreAdapter{repo: repo}
}

func (a *ProductPriceStoreAdapter) CreateForProduct(
	ctx context.Context,
	productID string,
	in priceuc.CreateInput,
) (*priceuc.ProductPrice, error) {
	row, err := a.repo.Create(
		ctx,
		productID,
		in.CategoryID,
		in.Currency,
		in.Amount,
		in.ValidFrom,
		in.ValidTo,
	)
	if err != nil {
		return nil, err
	}

	return mapProductPriceRowToUC(row), nil
}

func (a *ProductPriceStoreAdapter) ListForProduct(
	ctx context.Context,
	productID string,
) ([]priceuc.ProductPrice, error) {
	rows, err := a.repo.ListByProduct(ctx, productID)
	if err != nil {
		return nil, err
	}

	out := make([]priceuc.ProductPrice, 0, len(rows))
	for i := range rows {
		ucItem := mapProductPriceRowToUC(&rows[i])
		out = append(out, *ucItem)
	}
	return out, nil
}

func (a *ProductPriceStoreAdapter) Update(
	ctx context.Context,
	priceID string,
	in priceuc.UpdateInput,
) (*priceuc.ProductPrice, error) {
	row, err := a.repo.Update(
		ctx,
		priceID,
		in.Currency,
		in.Amount,
		in.ValidFrom,
		in.ValidTo,
		in.CategoryID,
	)
	if err != nil {
		return nil, err
	}

	return mapProductPriceRowToUC(row), nil
}

func mapProductPriceRowToUC(r *ProductPriceRow) *priceuc.ProductPrice {
	return &priceuc.ProductPrice{
		ID:         r.ID,
		ProductID:  r.ProductID,
		CategoryID: r.CategoryID,
		Currency:   r.Currency,
		Amount:     r.Amount,
		ValidFrom:  r.ValidFrom,
		ValidTo:    r.ValidTo,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

var _ priceuc.Store = (*ProductPriceStoreAdapter)(nil)

package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	prodcatuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product_category"
)

type ProductCategoryStoreAdapter struct {
	repo *ProductCategoryRepo
}

func NewProductCategoryStoreAdapter(repo *ProductCategoryRepo) *ProductCategoryStoreAdapter {
	return &ProductCategoryStoreAdapter{repo: repo}
}

func (a *ProductCategoryStoreAdapter) List(ctx context.Context) ([]prodcatuc.ProductCategory, error) {
	rows, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]prodcatuc.ProductCategory, 0, len(rows))
	for _, r := range rows {
		out = append(out, toUC(r))
	}
	return out, nil
}

func (a *ProductCategoryStoreAdapter) Create(ctx context.Context, name string, description *string) (*prodcatuc.ProductCategory, error) {
	row, err := a.repo.Create(ctx, name, description)
	if err != nil {
		return nil, err
	}
	c := toUC(*row)
	return &c, nil
}

func (a *ProductCategoryStoreAdapter) Update(ctx context.Context, id string, name *string, description *string) (*prodcatuc.ProductCategory, error) {
	row, err := a.repo.Update(ctx, id, name, description)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, prodcatuc.ErrNotFound
		}
		return nil, err
	}
	c := toUC(*row)
	return &c, nil
}

func toUC(r ProductCategoryRow) prodcatuc.ProductCategory {
	return prodcatuc.ProductCategory{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
	}
}

// compile-time interface check
var _ prodcatuc.Store = (*ProductCategoryStoreAdapter)(nil)

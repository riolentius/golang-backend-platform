package postgres

import (
	"context"

	customercategory "github.com/riolentius/cahaya-gading-backend/internal/usecase/customer_category"
)

type CustomerCategoryStoreAdapter struct {
	repo *CustomerCategoryRepo
}

func NewCustomerCategoryStoreAdapter(repo *CustomerCategoryRepo) *CustomerCategoryStoreAdapter {
	return &CustomerCategoryStoreAdapter{repo: repo}
}

func (a *CustomerCategoryStoreAdapter) List(ctx context.Context) ([]customercategory.CustomerCategory, error) {
	rows, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]customercategory.CustomerCategory, 0, len(rows))
	for _, r := range rows {
		out = append(out, customercategory.CustomerCategory{
			ID:          r.ID,
			Code:        r.Code,
			Name:        r.Name,
			Description: r.Description,
		})
	}
	return out, nil
}

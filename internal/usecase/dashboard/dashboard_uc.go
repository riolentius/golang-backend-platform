package dashboard

import "context"

type Store interface {
	GetRecentTransactions(ctx context.Context, limit int) ([]RecentTransaction, error)
	GetLowStockItems(ctx context.Context, threshold int) ([]LowStockItem, error)
	GetTopProducts(ctx context.Context, limit int) ([]TopProduct, error)
}

type Usecase struct {
	store Store
}

func New(store Store) *Usecase {
	return &Usecase{store: store}
}

func (u *Usecase) GetSummary(ctx context.Context) (*Summary, error) {
	txns, err := u.store.GetRecentTransactions(ctx, 5)
	if err != nil {
		return nil, err
	}

	items, err := u.store.GetLowStockItems(ctx, LowStockThreshold)
	if err != nil {
		return nil, err
	}

	return &Summary{
		RecentTransactions: txns,
		LowStock: LowStockSummary{
			HasLowStock: len(items) > 0,
			Items:       items,
		},
	}, nil
}

func (u *Usecase) GetTopProducts(ctx context.Context, limit int) ([]TopProduct, error) {
	return u.store.GetTopProducts(ctx, limit)
}

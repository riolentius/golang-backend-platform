package postgres

import (
	"context"

	dashboarduc "github.com/riolentius/cahaya-gading-backend/internal/usecase/dashboard"
)

type DashboardStoreAdapter struct {
	repo *DashboardRepo
}

func NewDashboardStoreAdapter(repo *DashboardRepo) *DashboardStoreAdapter {
	return &DashboardStoreAdapter{repo: repo}
}

func (a *DashboardStoreAdapter) GetRecentTransactions(ctx context.Context, limit int) ([]dashboarduc.RecentTransaction, error) {
	rows, err := a.repo.GetRecentTransactions(ctx, limit)
	if err != nil {
		return nil, err
	}

	out := make([]dashboarduc.RecentTransaction, 0, len(rows))
	for _, r := range rows {
		out = append(out, dashboarduc.RecentTransaction{
			ID:            r.ID,
			CustomerName:  r.CustomerName,
			TotalAmount:   r.TotalAmount,
			Currency:      r.Currency,
			Status:        r.Status,
			PaymentStatus: r.PaymentStatus,
			CreatedAt:     r.CreatedAt,
		})
	}
	return out, nil
}

func (a *DashboardStoreAdapter) GetLowStockItems(ctx context.Context, threshold int) ([]dashboarduc.LowStockItem, error) {
	rows, err := a.repo.GetLowStockItems(ctx, threshold)
	if err != nil {
		return nil, err
	}

	out := make([]dashboarduc.LowStockItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, dashboarduc.LowStockItem{
			ID:             r.ID,
			Name:           r.Name,
			SKU:            r.SKU,
			StockOnHand:    r.StockOnHand,
			StockReserved:  r.StockReserved,
			AvailableStock: r.AvailableStock,
		})
	}
	return out, nil
}

var _ dashboarduc.Store = (*DashboardStoreAdapter)(nil)

func (a *DashboardStoreAdapter) GetTopProducts(ctx context.Context, limit int) ([]dashboarduc.TopProduct, error) {
	rows, err := a.repo.GetTopProducts(ctx, limit)
	if err != nil {
		return nil, err
	}

	out := make([]dashboarduc.TopProduct, 0, len(rows))
	for _, r := range rows {
		out = append(out, dashboarduc.TopProduct{
			ID:           r.ID,
			Name:         r.Name,
			SKU:          r.SKU,
			TotalQtySold: r.TotalQtySold,
			TotalRevenue: r.TotalRevenue,
		})
	}
	return out, nil
}

package postgres

import (
	"context"
	"errors"

	stockuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/stock"
)

type StockStoreAdapter struct {
	repo *StockRepo
}

func NewStockStoreAdapter(repo *StockRepo) *StockStoreAdapter {
	return &StockStoreAdapter{repo: repo}
}

func (a *StockStoreAdapter) StockIn(
	ctx context.Context,
	productID string,
	quantity int,
	note *string,
	createdByEmail *string,
) (*stockuc.StockInResult, error) {
	row, stockProductID, newStockOnHand, err := a.repo.StockIn(ctx, productID, quantity, note, createdByEmail)
	if err != nil {
		if errors.Is(err, ErrProductMissing) {
			return nil, stockuc.ErrProductMissing
		}
		if errors.Is(err, ErrInvalidPackSize) {
			return nil, stockuc.ErrInvalidPackSize
		}
		return nil, err
	}

	return &stockuc.StockInResult{
		Movement:       toUC(row),
		StockProductID: stockProductID,
		NewStockOnHand: newStockOnHand,
	}, nil
}

func (a *StockStoreAdapter) ListByProduct(ctx context.Context, productID string, filter stockuc.ListFilter) ([]stockuc.Movement, error) {
	rows, err := a.repo.ListByProduct(ctx, productID, stockucFilter{
		Direction: filter.Direction,
		From:      filter.From,
		To:        filter.To,
		Limit:     filter.Limit,
		Offset:    filter.Offset,
	})
	if err != nil {
		return nil, err
	}

	out := make([]stockuc.Movement, 0, len(rows))
	for i := range rows {
		out = append(out, toUC(&rows[i]))
	}
	return out, nil
}

func toUC(r *MovementRow) stockuc.Movement {
	return stockuc.Movement{
		ID:             r.ID,
		ProductID:      r.ProductID,
		Direction:      r.Direction,
		Quantity:       r.Quantity,
		Source:         r.Source,
		ReferenceID:    r.ReferenceID,
		Note:           r.Note,
		CreatedByEmail: r.CreatedByEmail,
		CreatedAt:      r.CreatedAt,
	}
}

var _ stockuc.Store = (*StockStoreAdapter)(nil)

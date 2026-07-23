package postgres

import (
	"context"
	"errors"

	returnsuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/returns"
)

type ReturnStoreAdapter struct {
	repo *ReturnRepo
}

func NewReturnStoreAdapter(repo *ReturnRepo) *ReturnStoreAdapter {
	return &ReturnStoreAdapter{repo: repo}
}

func (a *ReturnStoreAdapter) Create(
	ctx context.Context,
	in returnsuc.CreateInput,
	createdByEmail *string,
) (*returnsuc.CreateResult, error) {
	args := make([]CreateItemArg, 0, len(in.Items))
	for _, it := range in.Items {
		args = append(args, CreateItemArg{
			TransactionItemID: it.TransactionItemID,
			Qty:               it.Qty,
			Restock:           it.Restock,
		})
	}

	hdr, items, state, err := a.repo.Create(ctx, in.TransactionID, in.Note, args, createdByEmail)
	if err != nil {
		return nil, mapRepoErr(err)
	}

	ret := toUCReturn(hdr)
	ret.Items = make([]returnsuc.ReturnItem, 0, len(items))
	for i := range items {
		ret.Items = append(ret.Items, toUCReturnItem(&items[i]))
	}

	return &returnsuc.CreateResult{
		Return: ret,
		Transaction: returnsuc.TransactionState{
			ID:            state.ID,
			TotalAmount:   state.TotalAmount,
			PaidAmount:    state.PaidAmount,
			PaymentStatus: state.PaymentStatus,
			Currency:      state.Currency,
		},
	}, nil
}

func (a *ReturnStoreAdapter) ListByTransaction(ctx context.Context, transactionID string) ([]returnsuc.Return, error) {
	headers, err := a.repo.ListByTransaction(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(headers))
	for i := range headers {
		ids = append(ids, headers[i].ID)
	}

	grouped, err := a.repo.ListItemsByReturnIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := make([]returnsuc.Return, 0, len(headers))
	for i := range headers {
		ret := toUCReturn(&headers[i])
		rows := grouped[ret.ID]
		ret.Items = make([]returnsuc.ReturnItem, 0, len(rows))
		for j := range rows {
			ret.Items = append(ret.Items, toUCReturnItem(&rows[j]))
		}
		out = append(out, ret)
	}
	return out, nil
}

func (a *ReturnStoreAdapter) ListReturnableItems(ctx context.Context, transactionID string) ([]returnsuc.ReturnableItem, error) {
	rows, err := a.repo.ListReturnableItems(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	out := make([]returnsuc.ReturnableItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, returnsuc.ReturnableItem{
			TransactionItemID: r.TransactionItemID,
			ProductID:         r.ProductID,
			ProductName:       r.ProductName,
			SKU:               r.SKU,
			UnitAmount:        r.UnitAmount,
			QtySold:           r.QtySold,
			QtyReturned:       r.QtyReturned,
			QtyReturnable:     r.QtyReturnable,
		})
	}
	return out, nil
}

// mapRepoErr translates repo-level sentinels into domain errors so the handler
// can return clean status codes instead of falling through to a 500.
func mapRepoErr(err error) error {
	switch {
	case errors.Is(err, ErrTransactionMissing):
		return returnsuc.ErrTransactionMissing
	case errors.Is(err, ErrNotFulfilled):
		return returnsuc.ErrNotFulfilled
	case errors.Is(err, ErrItemNotInTransaction):
		return returnsuc.ErrItemNotInTransaction
	case errors.Is(err, ErrQtyExceedsReturnable):
		return returnsuc.ErrQtyExceedsReturnable
	case errors.Is(err, ErrInvalidPackSize):
		return returnsuc.ErrInvalidInput
	default:
		return err
	}
}

func toUCReturn(r *ReturnRow) returnsuc.Return {
	return returnsuc.Return{
		ID:             r.ID,
		TransactionID:  r.TransactionID,
		TotalAmount:    r.TotalAmount,
		Currency:       r.Currency,
		Note:           r.Note,
		CreatedByEmail: r.CreatedByEmail,
		CreatedAt:      r.CreatedAt,
	}
}

func toUCReturnItem(r *ReturnItemRow) returnsuc.ReturnItem {
	return returnsuc.ReturnItem{
		ID:                r.ID,
		TransactionItemID: r.TransactionItemID,
		ProductID:         r.ProductID,
		ProductName:       r.ProductName,
		SKU:               r.SKU,
		Qty:               r.Qty,
		UnitAmount:        r.UnitAmount,
		LineTotal:         r.LineTotal,
		Restock:           r.Restock,
	}
}

var _ returnsuc.Store = (*ReturnStoreAdapter)(nil)

package postgres

import (
	"context"
	"math/big"

	"github.com/jackc/pgx/v5"
	trxuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/transaction"
)

func (a *TransactionStoreAdapter) GetViewByID(ctx context.Context, id string) (*trxuc.TransactionView, error) {
	h, err := a.repo.GetViewHeader(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, trxuc.ErrTransactionMissing
		}
		return nil, err
	}

	items, err := a.repo.GetViewItems(ctx, id)
	if err != nil {
		return nil, err
	}

	pays, err := a.repo.GetViewPayments(ctx, id)
	if err != nil {
		return nil, err
	}

	// balanceDue = total - paid (string numeric)
	balance := computeBalance(h.TotalAmount, h.PaidAmount)

	out := &trxuc.TransactionView{
		ID:            h.ID,
		CustomerID:    h.CustomerID,
		CustomerName:  h.CustomerName,
		CategoryID:    h.CategoryID,
		Status:        h.Status,
		Currency:      h.Currency,
		TotalAmount:   h.TotalAmount,
		PaidAmount:    h.PaidAmount,
		PaymentStatus: h.PaymentStatus,
		BalanceDue:    balance,
		Notes:         h.Notes,
		CreatedAt:     h.CreatedAt,
		UpdatedAt:     h.UpdatedAt,
		Items:         make([]trxuc.ViewItem, 0, len(items)),
		Payments:      make([]trxuc.ViewPay, 0, len(pays)),
	}

	for _, it := range items {
		out.Items = append(out.Items, trxuc.ViewItem{
			ProductID:   it.ProductID,
			SKU:         it.SKU,
			ProductName: it.ProductName,
			Qty:         it.Qty,
			UnitAmount:  it.UnitAmount,
			LineTotal:   it.LineTotal,
		})
	}

	for _, p := range pays {
		out.Payments = append(out.Payments, trxuc.ViewPay{
			ID:         p.ID,
			Method:     p.Method,
			Amount:     p.Amount,
			Currency:   p.Currency,
			PaidAt:     p.PaidAt,
			SenderName: p.SenderName,
			Reference:  p.Reference,
			Note:       p.Note,
			Status:     p.Status,
		})
	}

	return out, nil
}

func computeBalance(totalStr, paidStr string) string {
	// uses big.Rat to avoid float errors
	total := new(big.Rat)
	paid := new(big.Rat)

	if _, ok := total.SetString(totalStr); !ok {
		return "0"
	}
	if _, ok := paid.SetString(paidStr); !ok {
		paid.SetInt64(0)
	}

	total.Sub(total, paid)
	// format to 2 decimals
	return total.FloatString(2)
}

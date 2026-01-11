package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	payuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/payment"
)

type PaymentStoreAdapter struct {
	repo *PaymentRepo
}

func NewPaymentStoreAdapter(repo *PaymentRepo) *PaymentStoreAdapter {
	return &PaymentStoreAdapter{repo: repo}
}

func (a *PaymentStoreAdapter) Create(ctx context.Context, in payuc.CreateInput) (*payuc.Payment, *payuc.TransactionPaymentState, error) {
	tx, err := a.repo.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1) lock transaction row (ensures transaction exists + prevents race)
	total, currency, err := lockTransactionForPayment(ctx, tx, in.TransactionID)
	if err != nil {
		if pgx.ErrNoRows == err || isNoRows(err) {
			return nil, nil, payuc.ErrTransactionMissing
		}
		return nil, nil, err
	}

	paidAt := time.Now()
	if in.PaidAt != nil {
		paidAt = *in.PaidAt
	}

	// 2) insert payment
	row, err := insertPayment(ctx, tx, PaymentRow{
		TransactionID: in.TransactionID,
		Method:        in.Method,
		Amount:        in.Amount,
		Currency:      currency,
		PaidAt:        paidAt,
		SenderName:    in.SenderName,
		Reference:     in.Reference,
		Note:          in.Note,
		Status:        "posted",
	})
	if err != nil {
		return nil, nil, err
	}

	// 3) recompute + update paid_amount + payment_status
	stateRow, err := recomputeAndUpdateTransactionPaymentState(ctx, tx, in.TransactionID)
	if err != nil {
		return nil, nil, err
	}

	_ = total

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	return mapPaymentRowToUC(row), mapStateRowToUC(stateRow), nil
}

func (a *PaymentStoreAdapter) ListByTransaction(ctx context.Context, transactionID string) ([]payuc.Payment, error) {
	rows, err := a.repo.ListByTransaction(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	out := make([]payuc.Payment, 0, len(rows))
	for i := range rows {
		out = append(out, *mapPaymentRowToUC(&rows[i]))
	}
	return out, nil
}

func mapPaymentRowToUC(r *PaymentRow) *payuc.Payment {
	return &payuc.Payment{
		ID:            r.ID,
		TransactionID: r.TransactionID,
		Method:        r.Method,
		Amount:        r.Amount,
		Currency:      r.Currency,
		PaidAt:        r.PaidAt,
		SenderName:    r.SenderName,
		Reference:     r.Reference,
		Note:          r.Note,
		Status:        r.Status,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func mapStateRowToUC(r *TransactionPaymentStateRow) *payuc.TransactionPaymentState {
	return &payuc.TransactionPaymentState{
		TransactionID: r.TransactionID,
		PaidAmount:    r.PaidAmount,
		PaymentStatus: r.PaymentStatus,
		TotalAmount:   r.TotalAmount,
		Currency:      r.Currency,
	}
}

// Compile-time check
var _ payuc.Store = (*PaymentStoreAdapter)(nil)

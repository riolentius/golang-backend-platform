package postgres

import (
	"context"
	"errors"
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

	if err := ensureTransactionExists(ctx, tx, in.TransactionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, payuc.ErrTransactionMissing
		}
		return nil, nil, err
	}

	// transaction currency is source of truth
	_, trxCurrency, err := getTransactionTotals(ctx, tx, in.TransactionID)
	if err != nil {
		return nil, nil, err
	}

	paidAt := time.Now()
	if in.PaidAt != nil {
		paidAt = *in.PaidAt
	}

	pRow, err := insertPayment(
		ctx, tx,
		in.TransactionID,
		in.Method,
		in.Amount,
		trxCurrency,
		paidAt,
		in.SenderName,
		in.Reference,
		in.Note,
	)
	if err != nil {
		return nil, nil, err
	}

	paidSum, err := sumPostedPayments(ctx, tx, in.TransactionID)
	if err != nil {
		return nil, nil, err
	}

	stateRow, err := updateTransactionPaymentState(ctx, tx, in.TransactionID, paidSum)
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}

	outPayment := mapPaymentRowToUC(pRow)
	outState := mapTxPaymentStateRowToUC(stateRow)
	return outPayment, outState, nil
}

func (a *PaymentStoreAdapter) ListByTransaction(ctx context.Context, transactionID string) ([]payuc.Payment, error) {
	rows, err := listPaymentsByTransaction(ctx, a.repo.db, transactionID)
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

func mapTxPaymentStateRowToUC(r *TxPaymentStateRow) *payuc.TransactionPaymentState {
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

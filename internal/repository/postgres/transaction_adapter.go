package postgres

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	trxuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/transaction"
)

type TransactionStoreAdapter struct {
	repo *TransactionRepo
}

func NewTransactionStoreAdapter(repo *TransactionRepo) *TransactionStoreAdapter {
	return &TransactionStoreAdapter{repo: repo}
}

func (a *TransactionStoreAdapter) Create(ctx context.Context, in trxuc.CreateInput) (*trxuc.Transaction, error) {
	tx, err := a.repo.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// customer exists
	if err := ensureCustomerExists(ctx, tx, in.CustomerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, trxuc.ErrCustomerMissing
		}
		return nil, err
	}

	// create transaction
	trxRow, err := insertTransaction(ctx, tx, in.CustomerID, in.Notes)
	if err != nil {
		return nil, err
	}

	var (
		items      []trxuc.Item
		totalCents float64 // we’ll do float64 only for v1; later switch to decimal library
		currency   string
	)

	for _, it := range in.Items {
		if err := ensureProductExists(ctx, tx, it.ProductID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, trxuc.ErrProductMissing
			}
			return nil, err
		}

		cur, unitStr, err := getLatestPriceAmount(ctx, tx, it.ProductID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, trxuc.ErrPriceMissing
			}
			return nil, err
		}

		// enforce single-currency for v1
		if currency == "" {
			currency = cur
		} else if currency != cur {
			return nil, errors.New("multi-currency not supported")
		}

		unit, err := strconv.ParseFloat(unitStr, 64)
		if err != nil {
			return nil, err
		}

		line := unit * float64(it.Qty)
		totalCents += line

		itemRow, err := insertTransactionItem(ctx, tx, trxRow.ID, it.ProductID, it.Qty, unitStr, formatMoney(line))
		if err != nil {
			return nil, err
		}

		items = append(items, mapTrxItemRow(itemRow))
	}

	// update total
	finalRow, err := updateTransactionTotal(ctx, tx, trxRow.ID, currency, formatMoney(totalCents))
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	out := mapTrxRow(finalRow)
	out.Items = items
	return out, nil
}

func (a *TransactionStoreAdapter) List(ctx context.Context, in trxuc.ListInput) ([]trxuc.Transaction, error) {
	// implement next (simple select)
	return nil, errors.New("not implemented")
}

func (a *TransactionStoreAdapter) GetByID(ctx context.Context, id string) (*trxuc.Transaction, error) {
	// implement next (select header + items)
	return nil, errors.New("not implemented")
}

func (a *TransactionStoreAdapter) UpdateStatus(ctx context.Context, id string, status string) (*trxuc.Transaction, error) {
	// implement next (update status)
	return nil, errors.New("not implemented")
}

func mapTrxRow(r *TransactionRow) *trxuc.Transaction {
	return &trxuc.Transaction{
		ID:          r.ID,
		CustomerID:  r.CustomerID,
		Status:      r.Status,
		Currency:    r.Currency,
		TotalAmount: r.TotalAmount,
		Notes:       r.Notes,
		CreatedAt:   mustTime(r.CreatedAt),
		UpdatedAt:   mustTime(r.UpdatedAt),
	}
}

func mapTrxItemRow(r *TransactionItemRow) trxuc.Item {
	return trxuc.Item{
		ID:            r.ID,
		TransactionID: r.TransactionID,
		ProductID:     r.ProductID,
		Qty:           r.Qty,
		UnitAmount:    r.UnitAmount,
		LineTotal:     r.LineTotal,
		CreatedAt:     mustTime(r.CreatedAt),
		UpdatedAt:     mustTime(r.UpdatedAt),
	}
}

func mustTime(v any) time.Time {
	t, ok := v.(time.Time)
	if ok {
		return t
	}
	return time.Time{}
}

func formatMoney(v float64) string {
	// numeric(18,2) formatting
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// Compile-time check
var _ trxuc.Store = (*TransactionStoreAdapter)(nil)

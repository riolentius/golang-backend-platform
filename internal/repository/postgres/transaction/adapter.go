package postgres

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	trxuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/transaction"
)

type TransactionStoreAdapter struct {
	repo *TransactionRepo
	db   *pgxpool.Pool
}

func NewTransactionStoreAdapter(repo *TransactionRepo, db *pgxpool.Pool) *TransactionStoreAdapter {
	return &TransactionStoreAdapter{
		repo: repo,
		db:   db,
	}
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
		totalCents float64
		currency   string
	)

	customerCategoryID, err := getCustomerCategoryID(ctx, tx, in.CustomerID)
	if err != nil {
		return nil, err
	}

	for _, it := range in.Items {
		if err := ensureProductExists(ctx, tx, it.ProductID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, trxuc.ErrProductMissing
			}
			return nil, err
		}

		cur, unitStr, err := getEffectivePriceAmount(ctx, tx, it.ProductID, customerCategoryID)
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

func (a *TransactionStoreAdapter) ProductExists(ctx context.Context, productID string) (bool, error) {
	err := ensureProductExists(ctx, a.db, productID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (a *TransactionStoreAdapter) GetAvailableStock(ctx context.Context, productID string) (int, error) {
	const q = `
SELECT stock_on_hand - stock_reserved
FROM products
WHERE id = $1::uuid;
`
	var avail int
	if err := a.db.QueryRow(ctx, q, productID).Scan(&avail); err != nil {
		return 0, err
	}
	return avail, nil
}

func (a *TransactionStoreAdapter) GetStockRule(ctx context.Context, productID string) (string, float64, error) {
	return getStockRule(ctx, a.db, productID)
}

func (a *TransactionStoreAdapter) GetReservedStockForTx(ctx context.Context, transactionID string) (map[string]int, error) {
	// implement next (select reserved stock for transaction)
	return nil, errors.New("not implemented")
}

func (a *TransactionStoreAdapter) GetCommittedStockForTx(ctx context.Context, transactionID string) (map[string]int, error) {
	// implement next (select committed stock for transaction)
	return nil, errors.New("not implemented")
}

func (a *TransactionStoreAdapter) CustomerExists(ctx context.Context, customerID string) (bool, error) {
	err := ensureCustomerExists(ctx, a.db, customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (a *TransactionStoreAdapter) Fulfill(ctx context.Context, transactionID string) (*trxuc.Transaction, error) {
	tx, err := a.repo.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	status, err := lockTransactionStatus(ctx, tx, transactionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, trxuc.ErrTransactionMissing
		}
		return nil, err
	}

	if status == trxuc.StatusCompleted {
		return nil, trxuc.ErrAlreadyFulfilled
	}
	if status == trxuc.StatusCancelled {
		return nil, trxuc.ErrTransactionCanceled
	}
	if status != trxuc.StatusPending {
		return nil, trxuc.ErrInvalidTransition
	}

	if err := commitStockForTx(ctx, tx, transactionID); err != nil {
		return nil, err
	}

	row, err := updateTransactionStatus(ctx, tx, transactionID, trxuc.StatusCompleted)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return mapTrxRow(row), nil
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

func (a *TransactionStoreAdapter) ReserveStockForTx(ctx context.Context, transactionID string) error {
	tx, err := a.repo.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	status, err := lockTransactionStatus(ctx, tx, transactionID)
	if err != nil {
		return err
	}
	// reserve happens only for draft -> pending transition
	if status != trxuc.StatusDraft {
		return trxuc.ErrInvalidTransition
	}

	if err := reserveStockForTx(ctx, tx, transactionID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (a *TransactionStoreAdapter) ReleaseStockForTx(ctx context.Context, transactionID string) error {
	tx, err := a.repo.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	status, err := lockTransactionStatus(ctx, tx, transactionID)
	if err != nil {
		return err
	}
	// release happens only for pending -> cancelled transition
	if status != trxuc.StatusPending {
		return trxuc.ErrInvalidTransition
	}

	if err := releaseStockForTx(ctx, tx, transactionID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (a *TransactionStoreAdapter) CommitStockForTx(ctx context.Context, transactionID string) error {
	tx, err := a.repo.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	status, err := lockTransactionStatus(ctx, tx, transactionID)
	if err != nil {
		return err
	}
	// commit happens only for pending -> completed transition
	if status != trxuc.StatusPending {
		return trxuc.ErrInvalidTransition
	}

	if err := commitStockForTx(ctx, tx, transactionID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Compile-time check
var _ trxuc.Store = (*TransactionStoreAdapter)(nil)

package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"
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

	customerName, err := getCustomerName(ctx, tx, in.CustomerID)
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

		disc, err := parseDiscount(it.Discount, unit)
		if err != nil {
			return nil, err
		}

		line := (unit - disc) * float64(it.Qty)
		totalCents += line

		itemRow, err := insertTransactionItem(ctx, tx, trxRow.ID, it.ProductID, it.Qty, unitStr, formatMoney(disc), formatMoney(line))
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
	out.CustomerName = customerName
	out.Items = items
	return out, nil
}

func (a *TransactionStoreAdapter) List(ctx context.Context, in trxuc.ListInput) (*trxuc.ListResult, error) {
	var status *string
	if in.Status != nil {
		if s := strings.TrimSpace(*in.Status); s != "" {
			status = &s
		}
	}
	var search *string
	if s := strings.TrimSpace(in.Search); s != "" {
		pat := "%" + s + "%"
		search = &pat
	}

	const where = `
FROM transactions t
JOIN customers c ON c.id = t.customer_id
WHERE ($1::text IS NULL OR t.status = $1)
  AND ($2::text IS NULL
    OR t.id::text ILIKE $2
    OR (COALESCE(c.first_name,'') || CASE WHEN c.last_name IS NULL OR c.last_name='' THEN '' ELSE ' '||c.last_name END) ILIKE $2)
`

	var total int
	if err := a.db.QueryRow(ctx, `SELECT count(*) `+where, status, search).Scan(&total); err != nil {
		return nil, err
	}

	const cols = `
SELECT
  t.id::text, t.customer_id::text,
  COALESCE(c.first_name,'') || CASE WHEN c.last_name IS NULL OR c.last_name='' THEN '' ELSE ' '||c.last_name END AS customer_name,
  t.status, t.currency, t.total_amount::text, t.notes, t.created_at, t.updated_at
`

	orderBy := `
ORDER BY
  COALESCE(c.first_name,'') || CASE WHEN c.last_name IS NULL OR c.last_name='' THEN '' ELSE ' '||c.last_name END ASC
`

	switch in.Sort {
	case "newest":
		orderBy = `ORDER BY t.created_at DESC`
	case "oldest":
		orderBy = `ORDER BY t.created_at ASC`
	}

	q := cols + where + `
` + orderBy + `
LIMIT $3 OFFSET $4;
`

	rows, err := a.db.Query(ctx, q, status, search, in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]trxuc.Transaction, 0)
	for rows.Next() {
		var row TransactionRow
		if err := rows.Scan(
			&row.ID, &row.CustomerID, &row.CustomerName, &row.Status, &row.Currency,
			&row.TotalAmount, &row.Notes, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, *mapTrxRow(&row))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &trxuc.ListResult{Items: items, Total: total}, nil
}

func (a *TransactionStoreAdapter) GetByID(ctx context.Context, id string) (*trxuc.Transaction, error) {
	const q = `
SELECT
  t.id::text, t.customer_id::text,
  COALESCE(c.first_name,'') || CASE WHEN c.last_name IS NULL OR c.last_name='' THEN '' ELSE ' '||c.last_name END AS customer_name,
  t.status, t.currency, t.total_amount::text, t.notes, t.created_at, t.updated_at
FROM transactions t
JOIN customers c ON c.id = t.customer_id
WHERE t.id = $1::uuid;
`
	var row TransactionRow
	err := a.db.QueryRow(ctx, q, id).Scan(
		&row.ID, &row.CustomerID, &row.CustomerName, &row.Status, &row.Currency,
		&row.TotalAmount, &row.Notes, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, trxuc.ErrTransactionMissing
		}
		return nil, err
	}

	// fetch items
	const itemQ = `
SELECT id::text, transaction_id::text, product_id::text, qty, unit_amount::text, discount_amount::text, line_total::text, created_at, updated_at
FROM transaction_items
WHERE transaction_id = $1::uuid
ORDER BY created_at ASC;
`
	rows, err := a.db.Query(ctx, itemQ, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []trxuc.Item
	for rows.Next() {
		var it TransactionItemRow
		if err := rows.Scan(
			&it.ID, &it.TransactionID, &it.ProductID,
			&it.Qty, &it.UnitAmount, &it.DiscountAmount, &it.LineTotal,
			&it.CreatedAt, &it.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, mapTrxItemRow(&it))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := mapTrxRow(&row)
	out.Items = items
	return out, nil
}

func (a *TransactionStoreAdapter) UpdateStatus(ctx context.Context, id string, status string) (*trxuc.Transaction, error) {
	tx, err := a.repo.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row, err := updateTransactionStatus(ctx, tx, id, status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, trxuc.ErrTransactionMissing
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return mapTrxRow(row), nil
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

func (a *TransactionStoreAdapter) UpdateItems(ctx context.Context, id string, items []trxuc.UpdateItemIn) (*trxuc.Transaction, error) {
	tx, err := a.repo.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	status, customerID, err := lockTransactionRow(ctx, tx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, trxuc.ErrTransactionMissing
		}
		return nil, err
	}

	if status != trxuc.StatusDraft && status != trxuc.StatusPending {
		return nil, trxuc.ErrTransactionNotEditable
	}

	// Stock was already reserved for this transaction if it's pending.
	// Release the old reservation before rewriting items, then re-reserve
	// against the new item list further down.
	wasPending := status == trxuc.StatusPending
	if wasPending {
		if err := releaseStockForTx(ctx, tx, id); err != nil {
			return nil, err
		}
	}

	if err := deleteTransactionItems(ctx, tx, id); err != nil {
		return nil, err
	}

	customerCategoryID, err := getCustomerCategoryID(ctx, tx, customerID)
	if err != nil {
		return nil, err
	}

	customerName, err := getCustomerName(ctx, tx, customerID)
	if err != nil {
		return nil, err
	}

	var (
		totalCents float64
		currency   string
	)

	for _, it := range items {
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

		if currency == "" {
			currency = cur
		} else if currency != cur {
			return nil, errors.New("multi-currency not supported")
		}

		unit, err := strconv.ParseFloat(unitStr, 64)
		if err != nil {
			return nil, err
		}

		disc, err := parseDiscount(it.Discount, unit)
		if err != nil {
			return nil, err
		}

		line := (unit - disc) * float64(it.Qty)
		totalCents += line

		if _, err := insertTransactionItem(ctx, tx, id, it.ProductID, it.Qty, unitStr, formatMoney(disc), formatMoney(line)); err != nil {
			return nil, err
		}
	}

	finalRow, err := updateTransactionTotal(ctx, tx, id, currency, formatMoney(totalCents))
	if err != nil {
		return nil, err
	}

	if wasPending {
		if err := reserveStockForTx(ctx, tx, id); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	out := mapTrxRow(finalRow)
	out.CustomerName = customerName
	return out, nil
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
		ID:           r.ID,
		CustomerID:   r.CustomerID,
		CustomerName: r.CustomerName,
		Status:       r.Status,
		Currency:     r.Currency,
		TotalAmount:  r.TotalAmount,
		Notes:        r.Notes,
		CreatedAt:    mustTime(r.CreatedAt),
		UpdatedAt:    mustTime(r.UpdatedAt),
	}
}

func mapTrxItemRow(r *TransactionItemRow) trxuc.Item {
	return trxuc.Item{
		ID:             r.ID,
		TransactionID:  r.TransactionID,
		ProductID:      r.ProductID,
		Qty:            r.Qty,
		UnitAmount:     r.UnitAmount,
		DiscountAmount: r.DiscountAmount,
		LineTotal:      r.LineTotal,
		CreatedAt:      mustTime(r.CreatedAt),
		UpdatedAt:      mustTime(r.UpdatedAt),
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

func parseDiscount(raw string, unit float64) (float64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, nil
	}
	d, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, errors.New("discount is not a valid number")
	}
	if d < 0 {
		return 0, errors.New("discount cannot be negative")
	}
	if d > unit {
		return 0, errors.New("discount cannot exceed the unit price")
	}
	return d, nil
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

func (a *TransactionStoreAdapter) ListByCustomer(ctx context.Context, customerID string) (*trxuc.CustomerTransactionsResult, error) {
	rows, err := a.repo.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	outstanding, err := a.repo.GetOutstandingTotalForCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	items := make([]trxuc.CustomerTransactionSummary, 0, len(rows))
	for _, r := range rows {
		items = append(items, trxuc.CustomerTransactionSummary{
			ID:            r.ID,
			Status:        r.Status,
			TotalAmount:   r.TotalAmount,
			PaidAmount:    r.PaidAmount,
			BalanceDue:    r.BalanceDue,
			PaymentStatus: r.PaymentStatus,
			CreatedAt:     r.CreatedAt,
		})
	}

	return &trxuc.CustomerTransactionsResult{
		Items:            items,
		TotalOutstanding: outstanding,
	}, nil
}

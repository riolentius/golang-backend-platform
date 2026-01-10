package postgres

import (
	"context"
	"time"
)

type TransactionViewHeaderRow struct {
	ID            string
	CustomerID    string
	CustomerName  string
	CategoryID    *string
	Status        string
	Currency      string
	TotalAmount   string
	PaidAmount    string
	PaymentStatus string
	Notes         *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type TransactionViewItemRow struct {
	ProductID   string
	SKU         *string
	ProductName string
	Qty         int
	UnitAmount  string
	LineTotal   string
}

type TransactionViewPaymentRow struct {
	ID         string
	Method     string
	Amount     string
	Currency   string
	PaidAt     time.Time
	SenderName *string
	Reference  *string
	Note       *string
	Status     string
}

func (r *TransactionRepo) GetViewHeader(ctx context.Context, id string) (*TransactionViewHeaderRow, error) {
	const q = `
SELECT
  t.id::text,
  t.customer_id::text,
  COALESCE(c.first_name,'') || CASE WHEN c.last_name IS NULL OR c.last_name='' THEN '' ELSE ' '||c.last_name END AS customer_name,
  c.category_id::text,
  t.status,
  t.currency,
  t.total_amount::text,
  t.paid_amount::text,
  t.payment_status,
  t.notes,
  t.created_at,
  t.updated_at
FROM transactions t
JOIN customers c ON c.id = t.customer_id
WHERE t.id = $1::uuid;
`
	row := r.db.QueryRow(ctx, q, id)
	var out TransactionViewHeaderRow
	if err := row.Scan(
		&out.ID,
		&out.CustomerID,
		&out.CustomerName,
		&out.CategoryID,
		&out.Status,
		&out.Currency,
		&out.TotalAmount,
		&out.PaidAmount,
		&out.PaymentStatus,
		&out.Notes,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *TransactionRepo) GetViewItems(ctx context.Context, id string) ([]TransactionViewItemRow, error) {
	const q = `
SELECT
  ti.product_id::text,
  p.sku,
  p.name,
  ti.qty,
  ti.unit_amount::text,
  ti.line_total::text
FROM transaction_items ti
JOIN products p ON p.id = ti.product_id
WHERE ti.transaction_id = $1::uuid
ORDER BY ti.created_at ASC;
`
	rows, err := r.db.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TransactionViewItemRow, 0, 10)
	for rows.Next() {
		var it TransactionViewItemRow
		if err := rows.Scan(&it.ProductID, &it.SKU, &it.ProductName, &it.Qty, &it.UnitAmount, &it.LineTotal); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *TransactionRepo) GetViewPayments(ctx context.Context, id string) ([]TransactionViewPaymentRow, error) {
	const q = `
SELECT
  p.id::text,
  p.method,
  p.amount::text,
  p.currency,
  p.paid_at,
  p.sender_name,
  p.reference,
  p.note,
  p.status
FROM payments p
WHERE p.transaction_id = $1::uuid
ORDER BY p.paid_at DESC, p.created_at DESC;
`
	rows, err := r.db.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TransactionViewPaymentRow, 0, 10)
	for rows.Next() {
		var p TransactionViewPaymentRow
		if err := rows.Scan(&p.ID, &p.Method, &p.Amount, &p.Currency, &p.PaidAt, &p.SenderName, &p.Reference, &p.Note, &p.Status); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

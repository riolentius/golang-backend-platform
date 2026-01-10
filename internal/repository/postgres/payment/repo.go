package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRow struct {
	ID            string
	TransactionID string
	Method        string
	Amount        string
	Currency      string
	PaidAt        time.Time
	SenderName    *string
	Reference     *string
	Note          *string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type TransactionPaymentStateRow struct {
	TransactionID string
	PaidAmount    string
	PaymentStatus string
	TotalAmount   string
	Currency      string
}

type PaymentRepo struct {
	db *pgxpool.Pool
}

func NewPaymentRepo(db *pgxpool.Pool) *PaymentRepo {
	return &PaymentRepo{db: db}
}

func (r *PaymentRepo) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.db.BeginTx(ctx, pgx.TxOptions{})
}

func lockTransactionForPayment(ctx context.Context, tx pgx.Tx, transactionID string) (totalAmount string, currency string, err error) {
	// Lock the transaction row to avoid race conditions on paid_amount/payment_status
	const q = `
SELECT total_amount::text, currency
FROM transactions
WHERE id = $1::uuid
FOR UPDATE;
`
	if err := tx.QueryRow(ctx, q, transactionID).Scan(&totalAmount, &currency); err != nil {
		return "", "", err
	}
	return totalAmount, currency, nil
}

func insertPayment(ctx context.Context, tx pgx.Tx, in PaymentRow) (*PaymentRow, error) {
	const q = `
INSERT INTO payments (
  transaction_id, method, amount, currency, paid_at,
  sender_name, reference, note, status
)
VALUES (
  $1::uuid, $2, $3::numeric, $4, COALESCE($5, now()),
  $6, $7, $8, COALESCE($9, 'posted')
)
RETURNING
  id::text,
  transaction_id::text,
  method,
  amount::text,
  currency,
  paid_at,
  sender_name,
  reference,
  note,
  status,
  created_at,
  updated_at;
`
	row := tx.QueryRow(
		ctx, q,
		in.TransactionID,
		in.Method,
		in.Amount,
		in.Currency,
		in.PaidAt,
		in.SenderName,
		in.Reference,
		in.Note,
		in.Status,
	)

	var out PaymentRow
	if err := row.Scan(
		&out.ID,
		&out.TransactionID,
		&out.Method,
		&out.Amount,
		&out.Currency,
		&out.PaidAt,
		&out.SenderName,
		&out.Reference,
		&out.Note,
		&out.Status,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

func recomputeAndUpdateTransactionPaymentState(ctx context.Context, tx pgx.Tx, transactionID string) (*TransactionPaymentStateRow, error) {
	// paid_amount = sum(posted payments)
	// payment_status based on paid_amount vs total_amount
	const q = `
WITH paid AS (
  SELECT
    COALESCE(SUM(amount), 0)::numeric AS paid_amount
  FROM payments
  WHERE transaction_id = $1::uuid
    AND status = 'posted'
),
upd AS (
  UPDATE transactions t
  SET
    paid_amount = paid.paid_amount,
    payment_status = CASE
      WHEN paid.paid_amount = 0 THEN 'unpaid'
      WHEN paid.paid_amount < t.total_amount THEN 'partial'
      WHEN paid.paid_amount = t.total_amount THEN 'paid'
      ELSE 'overpaid'
    END,
    updated_at = now()
  FROM paid
  WHERE t.id = $1::uuid
  RETURNING
    t.id::text,
    t.paid_amount::text,
    t.payment_status,
    t.total_amount::text,
    t.currency
)
SELECT * FROM upd;
`
	var out TransactionPaymentStateRow
	if err := tx.QueryRow(ctx, q, transactionID).Scan(
		&out.TransactionID,
		&out.PaidAmount,
		&out.PaymentStatus,
		&out.TotalAmount,
		&out.Currency,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *PaymentRepo) ListByTransaction(ctx context.Context, transactionID string) ([]PaymentRow, error) {
	const q = `
SELECT
  id::text,
  transaction_id::text,
  method,
  amount::text,
  currency,
  paid_at,
  sender_name,
  reference,
  note,
  status,
  created_at,
  updated_at
FROM payments
WHERE transaction_id = $1::uuid
ORDER BY paid_at DESC, created_at DESC;
`
	rows, err := r.db.Query(ctx, q, transactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PaymentRow, 0, 10)
	for rows.Next() {
		var p PaymentRow
		if err := rows.Scan(
			&p.ID,
			&p.TransactionID,
			&p.Method,
			&p.Amount,
			&p.Currency,
			&p.PaidAt,
			&p.SenderName,
			&p.Reference,
			&p.Note,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Helper you’ll use in adapter to classify “missing transaction”
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

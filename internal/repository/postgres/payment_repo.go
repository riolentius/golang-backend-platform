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

type TxPaymentStateRow struct {
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

func ensureTransactionExists(ctx context.Context, tx pgx.Tx, transactionID string) error {
	const q = `SELECT 1 FROM transactions WHERE id = $1::uuid`
	var one int
	if err := tx.QueryRow(ctx, q, transactionID).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return err
	}
	return nil
}

func insertPayment(
	ctx context.Context,
	tx pgx.Tx,
	transactionID string,
	method string,
	amount string,
	currency string,
	paidAt time.Time,
	senderName *string,
	reference *string,
	note *string,
) (*PaymentRow, error) {
	const q = `
INSERT INTO payments (transaction_id, method, amount, currency, paid_at, sender_name, reference, note)
VALUES ($1::uuid, $2, $3::numeric, $4, $5, $6, $7, $8)
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
	row := tx.QueryRow(ctx, q, transactionID, method, amount, currency, paidAt, senderName, reference, note)

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

func sumPostedPayments(ctx context.Context, tx pgx.Tx, transactionID string) (string, error) {
	const q = `
SELECT COALESCE(SUM(amount), 0)::text
FROM payments
WHERE transaction_id = $1::uuid
  AND status = 'posted';
`
	var paid string
	if err := tx.QueryRow(ctx, q, transactionID).Scan(&paid); err != nil {
		return "", err
	}
	return paid, nil
}

func getTransactionTotals(ctx context.Context, tx pgx.Tx, transactionID string) (totalAmount string, currency string, err error) {
	const q = `
SELECT total_amount::text, currency
FROM transactions
WHERE id = $1::uuid;
`
	if err := tx.QueryRow(ctx, q, transactionID).Scan(&totalAmount, &currency); err != nil {
		return "", "", err
	}
	return totalAmount, currency, nil
}

func updateTransactionPaymentState(
	ctx context.Context,
	tx pgx.Tx,
	transactionID string,
	paidAmount string,
) (*TxPaymentStateRow, error) {
	// Compute payment_status in SQL (numeric compare)
	const q = `
UPDATE transactions
SET
  paid_amount = $2::numeric,
  payment_status = CASE
    WHEN $2::numeric = 0 THEN 'unpaid'
    WHEN $2::numeric < total_amount THEN 'partial'
    WHEN $2::numeric = total_amount THEN 'paid'
    ELSE 'overpaid'
  END,
  updated_at = now()
WHERE id = $1::uuid
RETURNING
  id::text,
  paid_amount::text,
  payment_status,
  total_amount::text,
  currency;
`
	row := tx.QueryRow(ctx, q, transactionID, paidAmount)

	var out TxPaymentStateRow
	if err := row.Scan(&out.TransactionID, &out.PaidAmount, &out.PaymentStatus, &out.TotalAmount, &out.Currency); err != nil {
		return nil, err
	}
	return &out, nil
}

func listPaymentsByTransaction(ctx context.Context, db *pgxpool.Pool, transactionID string) ([]PaymentRow, error) {
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
ORDER BY paid_at ASC, created_at ASC;
`
	rows, err := db.Query(ctx, q, transactionID)
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

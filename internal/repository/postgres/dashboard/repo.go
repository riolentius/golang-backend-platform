package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardRepo struct {
	db *pgxpool.Pool
}

func NewDashboardRepo(db *pgxpool.Pool) *DashboardRepo {
	return &DashboardRepo{db: db}
}

type RecentTransactionRow struct {
	ID            string
	CustomerName  string
	TotalAmount   string
	Currency      string
	Status        string
	PaymentStatus string
	CreatedAt     time.Time
}

type LowStockItemRow struct {
	ID             string
	Name           string
	SKU            *string
	StockOnHand    int
	StockReserved  int
	AvailableStock int
}

type TopProductRow struct {
	ID           string
	Name         string
	SKU          *string
	TotalQtySold int
	TotalRevenue string
}

func (r *DashboardRepo) GetRecentTransactions(ctx context.Context, limit int) ([]RecentTransactionRow, error) {
	const q = `
SELECT
  t.id::text,
  COALESCE(c.first_name, '') ||
    CASE WHEN c.last_name IS NULL OR c.last_name = '' THEN '' ELSE ' ' || c.last_name END
    AS customer_name,
  t.total_amount::text,
  t.currency,
  t.status,
  t.payment_status,
  t.created_at
FROM transactions t
JOIN customers c ON c.id = t.customer_id
ORDER BY t.created_at DESC
LIMIT $1;
`
	rows, err := r.db.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RecentTransactionRow, 0, limit)
	for rows.Next() {
		var row RecentTransactionRow
		if err := rows.Scan(
			&row.ID,
			&row.CustomerName,
			&row.TotalAmount,
			&row.Currency,
			&row.Status,
			&row.PaymentStatus,
			&row.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *DashboardRepo) GetLowStockItems(ctx context.Context, threshold int) ([]LowStockItemRow, error) {
	const q = `
SELECT
  id::text,
  name,
  sku,
  stock_on_hand,
  stock_reserved,
  (stock_on_hand - stock_reserved) AS available_stock
FROM products
WHERE is_active = true
  AND (stock_on_hand - stock_reserved) <= $1
ORDER BY (stock_on_hand - stock_reserved) ASC, name ASC;
`
	rows, err := r.db.Query(ctx, q, threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]LowStockItemRow, 0, 16)
	for rows.Next() {
		var row LowStockItemRow
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.SKU,
			&row.StockOnHand,
			&row.StockReserved,
			&row.AvailableStock,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *DashboardRepo) GetTopProducts(ctx context.Context, limit int) ([]TopProductRow, error) {
	const q = `
SELECT
  p.id::text,
  p.name,
  p.sku,
  SUM(ti.qty)::int         AS total_qty_sold,
  SUM(ti.line_total)::text AS total_revenue
FROM transaction_items ti
JOIN products     p ON p.id = ti.product_id
JOIN transactions t ON t.id = ti.transaction_id
WHERE t.status IN ('pending', 'completed')
GROUP BY p.id, p.name, p.sku
ORDER BY SUM(ti.line_total) DESC
LIMIT $1;
`
	rows, err := r.db.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TopProductRow, 0, limit)
	for rows.Next() {
		var row TopProductRow
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.SKU,
			&row.TotalQtySold,
			&row.TotalRevenue,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductRow struct {
	ID            string
	SKU           *string
	Name          string
	Description   *string
	Cost          string
	IsActive      bool
	StockOnHand   int
	StockReserved int
	CategoryID    *string
	CategoryName  *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ProductRepo struct {
	db *pgxpool.Pool
}

func NewProductRepo(db *pgxpool.Pool) *ProductRepo {
	return &ProductRepo{db: db}
}

func (r *ProductRepo) Create(
	ctx context.Context,
	sku *string,
	name string,
	description *string,
	cost string,
	stockOnHand int,
	categoryID *string,
	createdByEmail *string,
) (*ProductRow, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertQ = `
WITH inserted AS (
  INSERT INTO products (sku, name, description, cost, stock_on_hand, category_id)
  VALUES ($1, $2, $3, $4, $5, $6::uuid)
  RETURNING id, sku, name, description, cost, is_active, stock_on_hand, stock_reserved, category_id, created_at, updated_at
)
SELECT
  i.id::text, i.sku, i.name, i.description, i.cost::text,
  i.is_active, i.stock_on_hand, i.stock_reserved,
  i.category_id::text, pc.name AS category_name,
  i.created_at, i.updated_at
FROM inserted i
LEFT JOIN product_categories pc ON pc.id = i.category_id;
`
	var out ProductRow
	if err := tx.QueryRow(ctx, insertQ, sku, name, description, cost, stockOnHand, categoryID).Scan(
		&out.ID, &out.SKU, &out.Name, &out.Description, &out.Cost,
		&out.IsActive, &out.StockOnHand, &out.StockReserved,
		&out.CategoryID, &out.CategoryName,
		&out.CreatedAt, &out.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if stockOnHand > 0 {
		const movQ = `
INSERT INTO stock_movements (product_id, direction, quantity, source, created_by_email)
VALUES ($1::uuid, 'in', $2, 'initial', $3);
`
		if _, err := tx.Exec(ctx, movQ, out.ID, stockOnHand, createdByEmail); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &out, nil
}

// ProductListFilter mirrors product.ListParams without importing the usecase
// package (avoids an import cycle — repo is downstream of usecase).
type ProductListFilter struct {
	Search string
	Status string // "active" | "inactive" | ""
	Limit  int
	Offset int
	Sort   string // "alphabet" | "newest" | "oldest"
}

// List returns a filtered, paginated page of products plus the total count of
// rows matching the filter (before limit/offset). Uses nullable params so the
// query stays a single static string — search "" and status "" both no-op.
func (r *ProductRepo) List(ctx context.Context, f ProductListFilter) ([]ProductRow, int, error) {
	// search: NULL when empty → the ILIKE condition short-circuits to TRUE.
	var search *string
	if s := strings.TrimSpace(f.Search); s != "" {
		pat := "%" + s + "%"
		search = &pat
	}

	// status → nullable bool: "active"→true, "inactive"→false, ""→NULL (no filter)
	var activeFilter *bool
	switch f.Status {
	case "active":
		t := true
		activeFilter = &t
	case "inactive":
		fl := false
		activeFilter = &fl
	}

	const where = `
WHERE ($1::text IS NULL OR p.name ILIKE $1 OR p.sku ILIKE $1)
  AND ($2::bool IS NULL OR p.is_active = $2)
`

	var total int
	if err := r.db.QueryRow(
		ctx,
		`SELECT count(*) FROM products p`+where,
		search,
		activeFilter,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Default: alphabetical by product name.
	orderBy := "p.name ASC"

	switch f.Sort {
	case "newest":
		orderBy = "p.created_at DESC"
	case "oldest":
		orderBy = "p.created_at ASC"
	case "alphabet", "":

	}

	q := `
SELECT
  p.id::text, p.sku, p.name, p.description, p.cost::text,
  p.is_active, p.stock_on_hand, p.stock_reserved,
  p.category_id::text, pc.name AS category_name,
  p.created_at, p.updated_at
FROM products p
LEFT JOIN product_categories pc ON pc.id = p.category_id
` + where + `
ORDER BY ` + orderBy + `
LIMIT $3 OFFSET $4;
`

	rows, err := r.db.Query(
		ctx,
		q,
		search,
		activeFilter,
		f.Limit,
		f.Offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]ProductRow, 0, f.Limit)
	for rows.Next() {
		var p ProductRow
		if err := rows.Scan(
			&p.ID, &p.SKU, &p.Name, &p.Description, &p.Cost,
			&p.IsActive, &p.StockOnHand, &p.StockReserved,
			&p.CategoryID, &p.CategoryName,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return out, total, nil
}

func (r *ProductRepo) GetByID(ctx context.Context, id string) (*ProductRow, error) {
	const q = `
SELECT
  p.id::text, p.sku, p.name, p.description, p.cost::text,
  p.is_active, p.stock_on_hand, p.stock_reserved,
  p.category_id::text, pc.name AS category_name,
  p.created_at, p.updated_at
FROM products p
LEFT JOIN product_categories pc ON pc.id = p.category_id
WHERE p.id = $1::uuid;
`
	var p ProductRow
	if err := r.db.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.Cost,
		&p.IsActive, &p.StockOnHand, &p.StockReserved,
		&p.CategoryID, &p.CategoryName,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepo) Update(
	ctx context.Context,
	id string,
	sku *string,
	name *string,
	description *string,
	cost *string,
	isActive *bool,
	stockOnHand *int,
	categoryID *string,
	createdByEmail *string,
) (*ProductRow, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var oldStock int
	const lockQ = `SELECT stock_on_hand FROM products WHERE id = $1::uuid FOR UPDATE;`
	if err := tx.QueryRow(ctx, lockQ, id).Scan(&oldStock); err != nil {
		return nil, err
	}

	const updateQ = `
WITH updated AS (
  UPDATE products
  SET
    sku           = COALESCE($2, sku),
    name          = COALESCE($3, name),
    description   = COALESCE($4, description),
    cost          = COALESCE($5::numeric, cost),
    is_active     = COALESCE($6, is_active),
    stock_on_hand = COALESCE($7, stock_on_hand),
    category_id   = COALESCE($8::uuid, category_id),
    updated_at    = now()
  WHERE id = $1::uuid
  RETURNING id, sku, name, description, cost, is_active, stock_on_hand, stock_reserved, category_id, created_at, updated_at
)
SELECT
  u.id::text, u.sku, u.name, u.description, u.cost::text,
  u.is_active, u.stock_on_hand, u.stock_reserved,
  u.category_id::text, pc.name AS category_name,
  u.created_at, u.updated_at
FROM updated u
LEFT JOIN product_categories pc ON pc.id = u.category_id;
`
	var out ProductRow
	if err := tx.QueryRow(ctx, updateQ,
		id, sku, name, description, cost, isActive, stockOnHand, categoryID,
	).Scan(
		&out.ID, &out.SKU, &out.Name, &out.Description, &out.Cost,
		&out.IsActive, &out.StockOnHand, &out.StockReserved,
		&out.CategoryID, &out.CategoryName,
		&out.CreatedAt, &out.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if stockOnHand != nil {
		delta := out.StockOnHand - oldStock
		if delta != 0 {
			direction := "in"
			qty := delta
			if delta < 0 {
				direction = "out"
				qty = -delta
			}
			const movQ = `
INSERT INTO stock_movements (product_id, direction, quantity, source, created_by_email)
VALUES ($1::uuid, $2, $3, 'adjustment', $4);
`
			if _, err := tx.Exec(ctx, movQ, out.ID, direction, qty, createdByEmail); err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &out, nil
}

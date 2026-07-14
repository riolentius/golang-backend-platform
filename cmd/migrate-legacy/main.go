// Command migrate-legacy is a ONE-OFF, re-runnable importer that pulls data from
// the old SQL Server database ("CahayaGadingP") into the Cahaya Gading Postgres schema.
//
// It intentionally writes straight to Postgres tables (NOT through the usecase
// layer) because legacy data can't satisfy the new validation rules — e.g. the
// customer phone-required rule. Migrations are a legitimate place to bypass those.
//
// Every phase is idempotent: it wipes only the rows it previously imported (keyed
// on the legacy id stashed in sku / a legacy_id note) and re-inserts. Safe to run
// repeatedly during UAT while mappings are tuned.
//
// Run order is fixed by FK dependencies:
//
//	categories → products → customers → transactions → items → initial stock
//
// Add -dry to print counts without writing:
//
//	go run ./cmd/migrate-legacy -dry
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/microsoft/go-mssqldb"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─────────────────────────────────────────────────────────────────────────────
// Mapping tables — the business decisions, kept in one visible place.
// ─────────────────────────────────────────────────────────────────────────────

// legacy customer kat (1-4) → new customer_categories.code
var custCategoryByKat = map[string]string{
	"1": "REGULAR",
	"2": "SPECIAL",
	"3": "VIP",
	"4": "WAREHOUSE", // hidden internal transfer type
}

// legacy m_product.Status_Item → products.is_active
func productActive(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "A")
}

// legacy htx.Status → transactions.status
func mapTxStatus(legacy string) string {
	switch strings.ToLower(strings.TrimSpace(legacy)) {
	case "active":
		return "pending"
	case "lunas":
		return "completed"
	case "cancel":
		return "cancelled"
	default:
		return "pending"
	}
}

// legacy htx.Status_Pembayaran → transactions.payment_status
func mapPaymentStatus(legacy string) string {
	if strings.EqualFold(strings.TrimSpace(legacy), "Lunas") {
		return "paid"
	}
	return "unpaid"
}

// isPhone reports whether pemilik holds a real phone number vs a placeholder like "-".
func normalizePhone(pemilik string) string {
	p := strings.TrimSpace(pemilik)
	if p == "" || p == "-" {
		return "0"
	}
	return p
}

// ─────────────────────────────────────────────────────────────────────────────

var dryRun = flag.Bool("dry", false, "read and count only; write nothing")

func main() {
	_ = godotenv.Load()
	flag.Parse()

	mssqlDSN := os.Getenv("MSSQL_DSN")
	pgURL := os.Getenv("DATABASE_URL")
	if mssqlDSN == "" || pgURL == "" {
		log.Fatal("both MSSQL_DSN and DATABASE_URL env vars are required")
	}

	ctx := context.Background()

	// source: SQL Server
	src, err := sql.Open("sqlserver", mssqlDSN)
	if err != nil {
		log.Fatalf("mssql open: %v", err)
	}
	defer src.Close()
	if err := src.PingContext(ctx); err != nil {
		log.Fatalf("mssql ping: %v", err)
	}

	// destination: Postgres
	dst, err := pgxpool.New(ctx, pgURL)
	if err != nil {
		log.Fatalf("postgres connect: %v", err)
	}
	defer dst.Close()
	if err := dst.Ping(ctx); err != nil {
		log.Fatalf("postgres ping: %v", err)
	}

	log.Printf("connected. dry-run=%v", *dryRun)

	m := &migrator{src: src, dst: dst, ctx: ctx}

	// category code → uuid, resolved once and reused everywhere
	if err := m.loadCategoryIDs(); err != nil {
		log.Fatalf("load category ids: %v", err)
	}

	if err := m.migrateProducts(); err != nil {
		log.Fatalf("products: %v", err)
	}
	if err := m.migrateCustomers(); err != nil {
		log.Fatalf("customers: %v", err)
	}
	if err := m.migrateTransactions(); err != nil {
		log.Fatalf("transactions: %v", err)
	}

	log.Println("✅ legacy migration complete")
}

type migrator struct {
	src *sql.DB
	dst *pgxpool.Pool
	ctx context.Context

	custCategoryID map[string]string // code → uuid (customer_categories)
	prodCategoryID map[string]string // UPPER(name) → uuid (product_categories)

	// legacy id → new uuid, built during each phase for FK linking
	productByLegacyID map[string]productRef
	customerByName    map[string]string // lower(nama) → customer uuid
	warehouseCustID   string
}

type productRef struct {
	uuid string
	// the effective unit prices per customer-category code, for building items
	priceRegular float64
	priceSpecial float64
	priceVIP     float64
}

// ─────────────────────────────────────────────────────────────────────────────
// Category id resolution. customer_categories seeded by cmd/seed; WAREHOUSE may
// not exist yet, so create it on demand (hidden 4th tier).
// ─────────────────────────────────────────────────────────────────────────────

func (m *migrator) loadCategoryIDs() error {
	m.custCategoryID = map[string]string{}

	rows, err := m.dst.Query(m.ctx, `SELECT code, id::text FROM customer_categories`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var code, id string
		if err := rows.Scan(&code, &id); err != nil {
			rows.Close()
			return err
		}
		m.custCategoryID[strings.ToUpper(code)] = id
	}
	rows.Close()

	// ensure WAREHOUSE exists (hidden internal transfer tier)
	if _, ok := m.custCategoryID["WAREHOUSE"]; !ok {
		if *dryRun {
			m.custCategoryID["WAREHOUSE"] = "00000000-0000-0000-0000-000000000000"
		} else {
			var id string
			err := m.dst.QueryRow(m.ctx, `
				INSERT INTO customer_categories (code, name, description)
				VALUES ('WAREHOUSE', 'Warehouse', 'Internal stock transfer — hidden, always zero-priced')
				ON CONFLICT (code) DO UPDATE SET code = EXCLUDED.code
				RETURNING id::text`).Scan(&id)
			if err != nil {
				return fmt.Errorf("ensure WAREHOUSE category: %w", err)
			}
			m.custCategoryID["WAREHOUSE"] = id
		}
	}

	log.Printf("customer categories resolved: %d", len(m.custCategoryID))
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Products.  m_product → products + product_categories + product_prices + initial stock movement
//
// Legacy id is stored in products.sku so the phase is idempotent and so
// transaction items can resolve product_id from the old dtx description.
// Because legacy dtx links items by *description text* (not product id), we build
// a description→product map as we go.
// ─────────────────────────────────────────────────────────────────────────────

func (m *migrator) migrateProducts() error {
	m.prodCategoryID = map[string]string{}
	m.productByLegacyID = map[string]productRef{}

	const q = `
SELECT id, [desc], kategori, cost, harga_umum, harga_dibawah, harga_diatas, stock, uom, Status_Item
FROM m_product`
	rows, err := m.src.QueryContext(m.ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()

	var count, skipped int
	for rows.Next() {
		var (
			id, desc, kategori, uom, statusItem   sql.NullString
			cost, hUmum, hDibawah, hDiatas, stock sql.NullFloat64
		)
		if err := rows.Scan(&id, &desc, &kategori, &cost, &hUmum, &hDibawah, &hDiatas, &stock, &uom, &statusItem); err != nil {
			return err
		}
		if !id.Valid || strings.TrimSpace(id.String) == "" || !desc.Valid {
			skipped++
			continue
		}

		if *dryRun {
			count++
			continue
		}

		catID, err := m.ensureProductCategory(kategori.String)
		if err != nil {
			return err
		}

		newID, err := m.upsertProduct(id.String, desc.String, uom.String, catID, cost.Float64, stock.Float64, productActive(statusItem.String))
		if err != nil {
			return fmt.Errorf("product %s: %w", id.String, err)
		}

		if err := m.upsertProductPrices(newID, hUmum.Float64, hDibawah.Float64, hDiatas.Float64); err != nil {
			return fmt.Errorf("prices %s: %w", id.String, err)
		}

		// index by legacy id AND by description (dtx links by description text)
		ref := productRef{
			uuid:         newID,
			priceRegular: hUmum.Float64,
			priceSpecial: hDiatas.Float64,  // harga_diatas → Special
			priceVIP:     hDibawah.Float64, // harga_dibawah → VIP
		}
		m.productByLegacyID[strings.TrimSpace(id.String)] = ref
		m.productByLegacyID["desc::"+normalizeDesc(desc.String)] = ref

		count++
	}
	log.Printf("products: %d imported, %d skipped", count, skipped)
	return rows.Err()
}

func (m *migrator) ensureProductCategory(name string) (string, error) {
	key := strings.ToUpper(strings.TrimSpace(name))
	if key == "" {
		key = "OTHERS"
	}
	if id, ok := m.prodCategoryID[key]; ok {
		return id, nil
	}
	var id string
	err := m.dst.QueryRow(m.ctx, `
		INSERT INTO product_categories (name)
		VALUES ($1)
		ON CONFLICT DO NOTHING
		RETURNING id::text`, key).Scan(&id)
	if err == pgx.ErrNoRows {
		// already existed → fetch it
		err = m.dst.QueryRow(m.ctx, `SELECT id::text FROM product_categories WHERE name = $1`, key).Scan(&id)
	}
	if err != nil {
		return "", err
	}
	m.prodCategoryID[key] = id
	return id, nil
}

func (m *migrator) upsertProduct(legacyID, name, uom, categoryID string, cost, stock float64, active bool) (string, error) {
	// Idempotency keys on a dedicated legacy marker stored in `description`
	// ("legacy:<id>"), NOT on name or sku. This keeps the lookup stable and unique
	// regardless of what name/sku hold, so re-runs update the right row instead of
	// inserting duplicates.
	//   name = legacy desc (the product's real name)
	//   sku  = legacy uom  (PC, PACK, LS ...) — intentionally non-unique
	legacy := strings.TrimSpace(legacyID)
	marker := "legacy:" + legacy
	name = strings.TrimSpace(name)
	if stock < 0 {
		stock = 0
	}

	// sku = UOM. NULL when the legacy uom is blank rather than an empty string.
	var skuVal *string
	if u := strings.TrimSpace(uom); u != "" {
		skuVal = &u
	}

	var id string
	err := m.dst.QueryRow(m.ctx, `SELECT id::text FROM products WHERE description = $1 LIMIT 1`, marker).Scan(&id)
	switch {
	case err == nil:
		// exists → update everything (keeps name/sku correct even if a prior run swapped them)
		if _, uErr := m.dst.Exec(m.ctx, `
			UPDATE products SET
				name = $2, sku = $3, cost = $4, stock_on_hand = $5,
				category_id = $6::uuid, is_active = $7, updated_at = now()
			WHERE id = $1::uuid`,
			id, name, skuVal, cost, int(stock), categoryID, active); uErr != nil {
			return "", uErr
		}
	case err == pgx.ErrNoRows:
		// new → insert
		if iErr := m.dst.QueryRow(m.ctx, `
			INSERT INTO products (sku, name, description, cost, stock_on_hand, category_id, is_active)
			VALUES ($1, $2, $3, $4, $5, $6::uuid, $7)
			RETURNING id::text`,
			skuVal, name, marker, cost, int(stock), categoryID, active).Scan(&id); iErr != nil {
			return "", iErr
		}
	default:
		return "", err
	}

	// one 'initial' stock movement per product (fresh ledger). Idempotent: delete prior initial rows first.
	if _, err := m.dst.Exec(m.ctx, `DELETE FROM stock_movements WHERE product_id = $1::uuid AND source = 'initial'`, id); err != nil {
		return "", err
	}
	if int(stock) > 0 {
		if _, err := m.dst.Exec(m.ctx, `
			INSERT INTO stock_movements (product_id, direction, quantity, source, note, created_by_email)
			VALUES ($1::uuid, 'in', $2, 'initial', 'Legacy import', 'legacy-migration')`,
			id, int(stock)); err != nil {
			return "", err
		}
	}
	return id, nil
}

// upsertProductPrices writes the three category prices. harga_umum→Regular,
// harga_dibawah→VIP, harga_diatas→Special. Idempotent: clears this product's
// prices first. WAREHOUSE tier is intentionally omitted (always zero at item time).
func (m *migrator) upsertProductPrices(productID string, hUmum, hDibawah, hDiatas float64) error {
	if _, err := m.dst.Exec(m.ctx, `DELETE FROM product_prices WHERE product_id = $1::uuid`, productID); err != nil {
		return err
	}
	priceByCat := []struct {
		code   string
		amount float64
	}{
		{"REGULAR", hUmum},
		{"VIP", hDibawah},
		{"SPECIAL", hDiatas},
	}
	for _, p := range priceByCat {
		catID := m.custCategoryID[p.code]
		if catID == "" {
			continue
		}
		if _, err := m.dst.Exec(m.ctx, `
			INSERT INTO product_prices (product_id, category_id, amount)
			VALUES ($1::uuid, $2::uuid, $3)`,
			productID, catID, p.amount); err != nil {
			return err
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Customers.  m_customer → customers + customer_addresses
//   nama    → first_name (business name, last_name null)
//   pemilik → phone (or "0" when "-"/empty)
//   alamat  → customer_addresses row when non-empty
//   kat     → category
// Legacy id stored in identification_number for idempotency + tx linking.
// ─────────────────────────────────────────────────────────────────────────────

func (m *migrator) migrateCustomers() error {
	m.customerByName = map[string]string{}

	const q = `SELECT id, nama, alamat, pemilik, kat FROM m_customer`
	rows, err := m.src.QueryContext(m.ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()

	var count, skipped int
	for rows.Next() {
		var id, nama, alamat, pemilik, kat sql.NullString
		if err := rows.Scan(&id, &nama, &alamat, &pemilik, &kat); err != nil {
			return err
		}
		if !nama.Valid || strings.TrimSpace(nama.String) == "" {
			skipped++
			continue
		}
		if *dryRun {
			count++
			continue
		}

		code := custCategoryByKat[strings.TrimSpace(kat.String)]
		if code == "" {
			code = "REGULAR"
		}
		catID := m.custCategoryID[code]
		phone := normalizePhone(pemilik.String)

		newID, err := m.upsertCustomer(id.String, nama.String, phone, catID)
		if err != nil {
			return fmt.Errorf("customer %s: %w", nama.String, err)
		}

		if a := strings.TrimSpace(alamat.String); a != "" {
			if err := m.upsertCustomerAddress(newID, a); err != nil {
				return fmt.Errorf("address %s: %w", nama.String, err)
			}
		}

		m.customerByName[strings.ToLower(strings.TrimSpace(nama.String))] = newID
		if code == "WAREHOUSE" {
			m.warehouseCustID = newID
		}
		count++
	}
	log.Printf("customers: %d imported, %d skipped", count, skipped)
	return rows.Err()
}

func (m *migrator) upsertCustomer(legacyID, name, phone, categoryID string) (string, error) {
	// email is nullable now (migration 0011). We leave it NULL for legacy.
	// idempotency key: identification_number = legacy id (unique per legacy row).
	legacy := strings.TrimSpace(legacyID)
	var id string

	// try find existing by legacy id first
	err := m.dst.QueryRow(m.ctx,
		`SELECT id::text FROM customers WHERE identification_number = $1`, legacy).Scan(&id)
	if err == nil {
		_, uErr := m.dst.Exec(m.ctx, `
			UPDATE customers SET first_name=$2, phone=$3, category_id=$4::uuid, updated_at=now()
			WHERE id=$1::uuid`, id, strings.TrimSpace(name), phone, categoryID)
		return id, uErr
	}
	if err != pgx.ErrNoRows {
		return "", err
	}

	err = m.dst.QueryRow(m.ctx, `
		INSERT INTO customers (first_name, phone, category_id, identification_number)
		VALUES ($1, $2, $3::uuid, $4)
		RETURNING id::text`,
		strings.TrimSpace(name), phone, categoryID, legacy).Scan(&id)
	return id, err
}

func (m *migrator) upsertCustomerAddress(customerID, addr string) error {
	// wipe + reinsert this customer's addresses for idempotency
	if _, err := m.dst.Exec(m.ctx, `DELETE FROM customer_addresses WHERE customer_id = $1::uuid`, customerID); err != nil {
		return err
	}
	_, err := m.dst.Exec(m.ctx, `
		INSERT INTO customer_addresses (customer_id, address_line1, is_default)
		VALUES ($1::uuid, $2, true)`, customerID, addr)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// Transactions.  htx → transactions,  dtx → transaction_items
//   htx.ID == dtx.id  (invoice number is the join key)
//   customer linked by name match; "Legacy Customer" fallback
//   dtx line items link to products by DESCRIPTION text
// Idempotent: transactions keyed on notes = 'legacy:'||ID.
// ─────────────────────────────────────────────────────────────────────────────

func (m *migrator) migrateTransactions() error {
	fallbackCustID, err := m.ensureLegacyCustomer()
	if err != nil {
		return err
	}

	const q = `
SELECT ID, total_amount, customer_name, Created_Date, Outstanding, Status, Status_Pembayaran
FROM htx`
	rows, err := m.src.QueryContext(m.ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()

	type htxRow struct {
		id            string
		total         int64
		customerName  string
		createdDate   time.Time
		outstanding   int64
		status        string
		paymentStatus string
	}
	var headers []htxRow
	for rows.Next() {
		var (
			id, custName, status, payStatus sql.NullString
			total, outstanding              sql.NullInt64
			created                         sql.NullTime
		)
		if err := rows.Scan(&id, &total, &custName, &created, &outstanding, &status, &payStatus); err != nil {
			return err
		}
		if !id.Valid || strings.TrimSpace(id.String) == "" {
			continue
		}
		headers = append(headers, htxRow{
			id:            strings.TrimSpace(id.String),
			total:         total.Int64,
			customerName:  strings.TrimSpace(custName.String),
			createdDate:   created.Time,
			outstanding:   outstanding.Int64,
			status:        status.String,
			paymentStatus: payStatus.String,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	if *dryRun {
		log.Printf("transactions: %d headers found (dry-run, not written)", len(headers))
		return nil
	}

	var count, itemCount, unmatched int
	for _, h := range headers {
		custID := fallbackCustID
		if h.customerName != "" {
			if cid, ok := m.customerByName[strings.ToLower(h.customerName)]; ok {
				custID = cid
			} else {
				unmatched++
			}
		}

		paid := float64(h.total - h.outstanding)
		if paid < 0 {
			paid = 0
		}

		txID, err := m.upsertTransaction(h.id, custID, mapTxStatus(h.status), mapPaymentStatus(h.paymentStatus),
			float64(h.total), paid, h.createdDate)
		if err != nil {
			return fmt.Errorf("tx %s: %w", h.id, err)
		}

		n, err := m.importItems(h.id, txID)
		if err != nil {
			return fmt.Errorf("items for %s: %w", h.id, err)
		}
		itemCount += n
		count++
	}
	log.Printf("transactions: %d imported, %d items, %d customer-name unmatched (fell back to Legacy Customer)",
		count, itemCount, unmatched)
	return nil
}

func (m *migrator) ensureLegacyCustomer() (string, error) {
	if *dryRun {
		return "00000000-0000-0000-0000-000000000000", nil
	}
	regID := m.custCategoryID["REGULAR"]
	var id string
	err := m.dst.QueryRow(m.ctx,
		`SELECT id::text FROM customers WHERE identification_number = 'LEGACY-FALLBACK'`).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return "", err
	}
	err = m.dst.QueryRow(m.ctx, `
		INSERT INTO customers (first_name, phone, category_id, identification_number)
		VALUES ('Legacy Customer', '0', $1::uuid, 'LEGACY-FALLBACK')
		RETURNING id::text`, regID).Scan(&id)
	return id, err
}

func (m *migrator) upsertTransaction(legacyID, custID, status, payStatus string, total, paid float64, created time.Time) (string, error) {
	note := "legacy:" + legacyID
	var id string
	err := m.dst.QueryRow(m.ctx, `SELECT id::text FROM transactions WHERE notes = $1`, note).Scan(&id)
	if err == nil {
		// re-run: clear existing items, update header
		if _, e := m.dst.Exec(m.ctx, `DELETE FROM transaction_items WHERE transaction_id = $1::uuid`, id); e != nil {
			return "", e
		}
		_, uErr := m.dst.Exec(m.ctx, `
			UPDATE transactions SET customer_id=$2::uuid, status=$3, payment_status=$4,
			  total_amount=$5, paid_amount=$6, created_at=$7, updated_at=now()
			WHERE id=$1::uuid`, id, custID, status, payStatus, total, paid, created)
		return id, uErr
	}
	if err != pgx.ErrNoRows {
		return "", err
	}
	err = m.dst.QueryRow(m.ctx, `
		INSERT INTO transactions (customer_id, status, payment_status, total_amount, paid_amount, notes, created_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
		RETURNING id::text`,
		custID, status, payStatus, total, paid, note, created).Scan(&id)
	return id, err
}

// importItems reads dtx rows for a legacy invoice and inserts transaction_items.
// Items link to products by description text. Unmatched descriptions are skipped
// with a warning (rather than failing the whole invoice) — expected for legacy junk.
func (m *migrator) importItems(legacyInvoiceID, txID string) (int, error) {
	const q = `
SELECT description_item, harga_item, qty_item, total
FROM dtx WHERE id = @p1`
	rows, err := m.src.QueryContext(m.ctx, q, sql.Named("p1", legacyInvoiceID))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var n int
	for rows.Next() {
		var desc sql.NullString
		var harga, qty, total sql.NullInt64
		if err := rows.Scan(&desc, &harga, &qty, &total); err != nil {
			return n, err
		}
		q := int(qty.Int64)
		if q <= 0 {
			q = 1 // schema requires qty > 0; legacy sometimes has 0
		}

		ref, ok := m.productByLegacyID["desc::"+normalizeDesc(desc.String)]
		if !ok {
			log.Printf("  ⚠ tx %s: no product match for %q", legacyInvoiceID, strings.TrimSpace(desc.String))
			continue
		}

		unit := float64(harga.Int64)
		line := float64(total.Int64)
		if line == 0 {
			line = unit * float64(q)
		}

		if _, err := m.dst.Exec(m.ctx, `
			INSERT INTO transaction_items (transaction_id, product_id, qty, unit_amount, line_total)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5)`,
			txID, ref.uuid, q, unit, line); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

// normalizeDesc lowercases and collapses whitespace so dtx description text
// matches m_product.desc despite minor spacing differences.
func normalizeDesc(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

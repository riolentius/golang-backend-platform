package postgres

import (
	"context"
	"testing"
	"time"

	testutil "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/testutil"
	trxrepo "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/transaction"
	payuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/payment"
	trxuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/transaction"
)

func TestPayment_CreateAndList_UpdatesTransactionState(t *testing.T) {
	db := testutil.MustOpenDB(t)
	defer db.Close()

	testutil.TruncateAll(t, db)

	ctx := context.Background()

	// --- seed minimal data ---
	// category
	catID := testutil.MustInsertCategory(t, db, "REGULAR", "Regular")

	// customer
	custID := testutil.MustInsertCustomer(t, db, "Rio", "Test", "rio@test.local", &catID)

	// product
	prodID := testutil.MustInsertProduct(t, db, "SKU-1", "Knee Volley", nil, 10, 0)

	// default price (category NULL)
	testutil.MustInsertPrice(t, db, prodID, nil, "IDR", "5000.00")

	// create transaction (1 item qty 2 => 10,000)
	trxRepo := trxrepo.NewTransactionRepo(db)
	trxStore := trxrepo.NewTransactionStoreAdapter(trxRepo, db)

	trx, err := trxStore.Create(ctx, trxuc.CreateInput{
		CustomerID: custID,
		Items: []trxuc.CreateItemIn{
			{ProductID: prodID, Qty: 2},
		},
	})

	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	_ = trx

	// --- payment under test ---
	pRepo := NewPaymentRepo(db)
	pStore := NewPaymentStoreAdapter(pRepo)

	pUC := payuc.New(pStore)

	now := time.Now().UTC()

	// pay partial 5,000
	p1, state1, err := pUC.Create(ctx, payuc.CreateInput{
		TransactionID: trx.ID,
		Method:        "cash",
		Amount:        "5000.00",
		PaidAt:        &now,
	})
	if err != nil {
		t.Fatalf("create payment 1: %v", err)
	}
	if p1.ID == "" {
		t.Fatalf("payment id empty")
	}
	if state1.PaymentStatus != "partial" {
		t.Fatalf("expected payment_status=partial got=%s", state1.PaymentStatus)
	}

	// pay remaining 5,000
	p2, state2, err := pUC.Create(ctx, payuc.CreateInput{
		TransactionID: trx.ID,
		Method:        "transfer",
		Amount:        "5000.00",
	})
	if err != nil {
		t.Fatalf("create payment 2: %v", err)
	}
	if p2.ID == "" {
		t.Fatalf("payment id empty")
	}
	if state2.PaymentStatus != "paid" {
		t.Fatalf("expected payment_status=paid got=%s", state2.PaymentStatus)
	}

	items, err := pUC.ListByTransaction(ctx, trx.ID)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(items))
	}
}

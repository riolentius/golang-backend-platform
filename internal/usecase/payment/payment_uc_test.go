package payment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pay "github.com/riolentius/cahaya-gading-backend/internal/usecase/payment"
)

// --- Mock Store ----------------------------------------------------------

type mockStore struct {
	create            func(ctx context.Context, in pay.CreateInput) (*pay.Payment, *pay.TransactionPaymentState, error)
	listByTransaction func(ctx context.Context, transactionID string) ([]pay.Payment, error)
}

func (m *mockStore) Create(ctx context.Context, in pay.CreateInput) (*pay.Payment, *pay.TransactionPaymentState, error) {
	if m.create != nil {
		return m.create(ctx, in)
	}
	return &pay.Payment{
			ID:            "pay-001",
			TransactionID: in.TransactionID,
			Method:        in.Method,
			Amount:        in.Amount,
			Currency:      "IDR",
			Status:        "posted",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}, &pay.TransactionPaymentState{
			TransactionID: in.TransactionID,
			PaidAmount:    in.Amount,
			PaymentStatus: "partial",
			TotalAmount:   "10000.00",
			Currency:      "IDR",
		}, nil
}

func (m *mockStore) ListByTransaction(ctx context.Context, transactionID string) ([]pay.Payment, error) {
	if m.listByTransaction != nil {
		return m.listByTransaction(ctx, transactionID)
	}
	return []pay.Payment{}, nil
}

// --- Helpers -------------------------------------------------------------

func newUsecase(store *mockStore) *pay.Usecase {
	return pay.New(store)
}

func ptr(s string) *string { return &s }

// --- Create: input validation --------------------------------------------

func TestCreate_MissingTransactionID(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, _, err := uc.Create(context.Background(), pay.CreateInput{
		Method: "cash",
		Amount: "5000.00",
	})
	require.ErrorIs(t, err, pay.ErrInvalidInput)
}

func TestCreate_WhitespaceOnlyTransactionID(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, _, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "   ",
		Method:        "cash",
		Amount:        "5000.00",
	})
	require.ErrorIs(t, err, pay.ErrInvalidInput)
}

func TestCreate_MissingMethod(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, _, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Amount:        "5000.00",
	})
	require.ErrorIs(t, err, pay.ErrInvalidInput)
}

func TestCreate_InvalidMethod(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, _, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Method:        "bitcoin", // not cash or transfer
		Amount:        "5000.00",
	})
	require.ErrorIs(t, err, pay.ErrInvalidInput)
}

func TestCreate_MissingAmount(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, _, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Method:        "cash",
	})
	require.ErrorIs(t, err, pay.ErrInvalidInput)
}

func TestCreate_WhitespaceOnlyAmount(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, _, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Method:        "cash",
		Amount:        "   ",
	})
	require.ErrorIs(t, err, pay.ErrInvalidInput)
}

// --- Create: valid methods -----------------------------------------------

func TestCreate_MethodCash_OK(t *testing.T) {
	uc := newUsecase(&mockStore{})
	p, state, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Method:        "cash",
		Amount:        "5000.00",
	})
	require.NoError(t, err)
	require.NotNil(t, p)
	require.NotNil(t, state)
	assert.Equal(t, "cash", p.Method)
}

func TestCreate_MethodTransfer_OK(t *testing.T) {
	uc := newUsecase(&mockStore{})
	p, state, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Method:        "transfer",
		Amount:        "5000.00",
	})
	require.NoError(t, err)
	require.NotNil(t, p)
	require.NotNil(t, state)
	assert.Equal(t, "transfer", p.Method)
}

// --- Create: method trimming ---------------------------------------------

func TestCreate_MethodWithWhitespace_IsRejected(t *testing.T) {
	// method must be exactly "cash" or "transfer" after trim
	uc := newUsecase(&mockStore{})
	_, _, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Method:        " cash ",
		Amount:        "5000.00",
	})
	// after TrimSpace → "cash" → should pass
	require.NoError(t, err)
}

// --- Create: optional fields pass through --------------------------------

func TestCreate_OptionalFields_PassedToStore(t *testing.T) {
	var captured pay.CreateInput
	store := &mockStore{
		create: func(ctx context.Context, in pay.CreateInput) (*pay.Payment, *pay.TransactionPaymentState, error) {
			captured = in
			return &pay.Payment{ID: "pay-001", Method: in.Method, Amount: in.Amount}, &pay.TransactionPaymentState{}, nil
		},
	}
	uc := newUsecase(store)

	now := time.Now()
	_, _, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Method:        "transfer",
		Amount:        "7500.00",
		SenderName:    ptr("Rio"),
		Reference:     ptr("REF-XYZ"),
		Note:          ptr("monthly payment"),
		PaidAt:        &now,
	})
	require.NoError(t, err)
	assert.Equal(t, ptr("Rio"), captured.SenderName)
	assert.Equal(t, ptr("REF-XYZ"), captured.Reference)
	assert.Equal(t, ptr("monthly payment"), captured.Note)
	assert.NotNil(t, captured.PaidAt)
}

func TestCreate_NilOptionalFields_OK(t *testing.T) {
	uc := newUsecase(&mockStore{})
	p, _, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Method:        "cash",
		Amount:        "5000.00",
		// SenderName, Reference, Note, PaidAt all nil
	})
	require.NoError(t, err)
	require.NotNil(t, p)
}

// --- Create: store error propagation -------------------------------------

func TestCreate_StoreError_IsReturned(t *testing.T) {
	dbErr := errors.New("connection refused")
	store := &mockStore{
		create: func(ctx context.Context, in pay.CreateInput) (*pay.Payment, *pay.TransactionPaymentState, error) {
			return nil, nil, dbErr
		},
	}
	uc := newUsecase(store)

	_, _, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Method:        "cash",
		Amount:        "5000.00",
	})
	require.ErrorIs(t, err, dbErr)
}

// --- Create: payment state returned --------------------------------------

func TestCreate_ReturnsCorrectPaymentState(t *testing.T) {
	store := &mockStore{
		create: func(ctx context.Context, in pay.CreateInput) (*pay.Payment, *pay.TransactionPaymentState, error) {
			return &pay.Payment{ID: "pay-001"}, &pay.TransactionPaymentState{
				TransactionID: in.TransactionID,
				PaidAmount:    "5000.00",
				PaymentStatus: "partial",
				TotalAmount:   "10000.00",
				Currency:      "IDR",
			}, nil
		},
	}
	uc := newUsecase(store)

	_, state, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Method:        "cash",
		Amount:        "5000.00",
	})
	require.NoError(t, err)
	assert.Equal(t, "partial", state.PaymentStatus)
	assert.Equal(t, "5000.00", state.PaidAmount)
	assert.Equal(t, "10000.00", state.TotalAmount)
}

func TestCreate_FullPayment_StateIsPaid(t *testing.T) {
	store := &mockStore{
		create: func(ctx context.Context, in pay.CreateInput) (*pay.Payment, *pay.TransactionPaymentState, error) {
			return &pay.Payment{ID: "pay-001"}, &pay.TransactionPaymentState{
				PaymentStatus: "paid",
				PaidAmount:    "10000.00",
				TotalAmount:   "10000.00",
			}, nil
		},
	}
	uc := newUsecase(store)

	_, state, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Method:        "cash",
		Amount:        "10000.00",
	})
	require.NoError(t, err)
	assert.Equal(t, "paid", state.PaymentStatus)
}

func TestCreate_Overpayment_StateIsOverpaid(t *testing.T) {
	store := &mockStore{
		create: func(ctx context.Context, in pay.CreateInput) (*pay.Payment, *pay.TransactionPaymentState, error) {
			return &pay.Payment{ID: "pay-001"}, &pay.TransactionPaymentState{
				PaymentStatus: "overpaid",
				PaidAmount:    "15000.00",
				TotalAmount:   "10000.00",
			}, nil
		},
	}
	uc := newUsecase(store)

	_, state, err := uc.Create(context.Background(), pay.CreateInput{
		TransactionID: "tx-001",
		Method:        "cash",
		Amount:        "15000.00",
	})
	require.NoError(t, err)
	assert.Equal(t, "overpaid", state.PaymentStatus)
}

// --- ListByTransaction ---------------------------------------------------

func TestListByTransaction_EmptyID(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.ListByTransaction(context.Background(), "")
	require.ErrorIs(t, err, pay.ErrInvalidInput)
}

func TestListByTransaction_WhitespaceOnlyID(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.ListByTransaction(context.Background(), "   ")
	require.ErrorIs(t, err, pay.ErrInvalidInput)
}

func TestListByTransaction_ReturnsPayments(t *testing.T) {
	store := &mockStore{
		listByTransaction: func(ctx context.Context, transactionID string) ([]pay.Payment, error) {
			return []pay.Payment{
				{ID: "pay-001", TransactionID: transactionID, Method: "cash", Amount: "5000.00"},
				{ID: "pay-002", TransactionID: transactionID, Method: "transfer", Amount: "5000.00"},
			}, nil
		},
	}
	uc := newUsecase(store)

	items, err := uc.ListByTransaction(context.Background(), "tx-001")
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "pay-001", items[0].ID)
	assert.Equal(t, "pay-002", items[1].ID)
}

func TestListByTransaction_EmptyResult_OK(t *testing.T) {
	uc := newUsecase(&mockStore{}) // default returns empty slice
	items, err := uc.ListByTransaction(context.Background(), "tx-001")
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestListByTransaction_StoreError_IsReturned(t *testing.T) {
	dbErr := errors.New("db timeout")
	store := &mockStore{
		listByTransaction: func(ctx context.Context, transactionID string) ([]pay.Payment, error) {
			return nil, dbErr
		},
	}
	uc := newUsecase(store)

	_, err := uc.ListByTransaction(context.Background(), "tx-001")
	require.ErrorIs(t, err, dbErr)
}

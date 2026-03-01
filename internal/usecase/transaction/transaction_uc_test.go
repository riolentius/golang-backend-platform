package transaction_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tx "github.com/riolentius/cahaya-gading-backend/internal/usecase/transaction"
)

// --- Mock Store ----------------------------------------------------------

type mockStore struct {
	// control what each method returns
	customerExists    func(ctx context.Context, id string) (bool, error)
	productExists     func(ctx context.Context, id string) (bool, error)
	getStockRule      func(ctx context.Context, productID string) (string, float64, error)
	getAvailableStock func(ctx context.Context, stockProductID string) (int, error)
	create            func(ctx context.Context, in tx.CreateInput) (*tx.Transaction, error)
	list              func(ctx context.Context, in tx.ListInput) ([]tx.Transaction, error)
	getByID           func(ctx context.Context, id string) (*tx.Transaction, error)
	reserveStock      func(ctx context.Context, txID string) error
	releaseStock      func(ctx context.Context, txID string) error
	commitStock       func(ctx context.Context, txID string) error
	updateStatus      func(ctx context.Context, id string, status string) (*tx.Transaction, error)
	getViewByID       func(ctx context.Context, id string) (*tx.TransactionView, error)
	fulfill           func(ctx context.Context, id string) (*tx.Transaction, error)
}

func (m *mockStore) CustomerExists(ctx context.Context, id string) (bool, error) {
	if m.customerExists != nil {
		return m.customerExists(ctx, id)
	}
	return true, nil
}

func (m *mockStore) ProductExists(ctx context.Context, id string) (bool, error) {
	if m.productExists != nil {
		return m.productExists(ctx, id)
	}
	return true, nil
}

func (m *mockStore) GetStockRule(ctx context.Context, productID string) (string, float64, error) {
	if m.getStockRule != nil {
		return m.getStockRule(ctx, productID)
	}
	return productID, 1.0, nil // default: product is its own stock, pack=1
}

func (m *mockStore) GetAvailableStock(ctx context.Context, stockProductID string) (int, error) {
	if m.getAvailableStock != nil {
		return m.getAvailableStock(ctx, stockProductID)
	}
	return 100, nil // plenty of stock by default
}

func (m *mockStore) Create(ctx context.Context, in tx.CreateInput) (*tx.Transaction, error) {
	if m.create != nil {
		return m.create(ctx, in)
	}
	return &tx.Transaction{
		ID:         "tx-001",
		CustomerID: in.CustomerID,
		Status:     in.Status,
	}, nil
}

func (m *mockStore) List(ctx context.Context, in tx.ListInput) ([]tx.Transaction, error) {
	if m.list != nil {
		return m.list(ctx, in)
	}
	return []tx.Transaction{}, nil
}

func (m *mockStore) GetByID(ctx context.Context, id string) (*tx.Transaction, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return &tx.Transaction{ID: id, Status: tx.StatusDraft}, nil
}

func (m *mockStore) ReserveStockForTx(ctx context.Context, txID string) error {
	if m.reserveStock != nil {
		return m.reserveStock(ctx, txID)
	}
	return nil
}

func (m *mockStore) ReleaseStockForTx(ctx context.Context, txID string) error {
	if m.releaseStock != nil {
		return m.releaseStock(ctx, txID)
	}
	return nil
}

func (m *mockStore) CommitStockForTx(ctx context.Context, txID string) error {
	if m.commitStock != nil {
		return m.commitStock(ctx, txID)
	}
	return nil
}

func (m *mockStore) UpdateStatus(ctx context.Context, id string, status string) (*tx.Transaction, error) {
	if m.updateStatus != nil {
		return m.updateStatus(ctx, id, status)
	}
	return &tx.Transaction{ID: id, Status: status}, nil
}

func (m *mockStore) GetViewByID(ctx context.Context, id string) (*tx.TransactionView, error) {
	if m.getViewByID != nil {
		return m.getViewByID(ctx, id)
	}
	return &tx.TransactionView{ID: id}, nil
}

func (m *mockStore) Fulfill(ctx context.Context, id string) (*tx.Transaction, error) {
	if m.fulfill != nil {
		return m.fulfill(ctx, id)
	}
	return &tx.Transaction{ID: id, Status: tx.StatusCompleted}, nil
}

// --- Helpers -------------------------------------------------------------

func newUsecase(store *mockStore) *tx.Usecase {
	return tx.New(store)
}

func validItem() tx.CreateItemIn {
	return tx.CreateItemIn{ProductID: "prod-001", Qty: 1}
}

// --- Create Tests --------------------------------------------------------

func TestCreate_MissingCustomerID(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.Create(context.Background(), tx.CreateInput{
		Items: []tx.CreateItemIn{validItem()},
	})
	require.ErrorIs(t, err, tx.ErrInvalidInput)
}

func TestCreate_NoItems(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "cust-001",
		Items:      []tx.CreateItemIn{},
	})
	require.ErrorIs(t, err, tx.ErrInvalidInput)
}

func TestCreate_ItemMissingProductID(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "cust-001",
		Items:      []tx.CreateItemIn{{ProductID: "", Qty: 1}},
	})
	require.ErrorIs(t, err, tx.ErrInvalidInput)
}

func TestCreate_ItemZeroQty(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "cust-001",
		Items:      []tx.CreateItemIn{{ProductID: "prod-001", Qty: 0}},
	})
	require.ErrorIs(t, err, tx.ErrInvalidInput)
}

func TestCreate_InvalidStatus(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "cust-001",
		Status:     "bogus",
		Items:      []tx.CreateItemIn{validItem()},
	})
	require.ErrorIs(t, err, tx.ErrInvalidStatus)
}

func TestCreate_DefaultStatusIsDraft(t *testing.T) {
	var capturedStatus string
	store := &mockStore{
		create: func(ctx context.Context, in tx.CreateInput) (*tx.Transaction, error) {
			capturedStatus = in.Status
			return &tx.Transaction{ID: "tx-001", CustomerID: in.CustomerID, Status: in.Status}, nil
		},
	}
	uc := newUsecase(store)

	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "cust-001",
		// Status intentionally left empty
		Items: []tx.CreateItemIn{validItem()},
	})
	require.NoError(t, err)
	assert.Equal(t, tx.StatusDraft, capturedStatus)
}

func TestCreate_CustomerNotFound(t *testing.T) {
	store := &mockStore{
		customerExists: func(ctx context.Context, id string) (bool, error) {
			return false, nil
		},
	}
	uc := newUsecase(store)

	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "ghost-cust",
		Items:      []tx.CreateItemIn{validItem()},
	})
	require.ErrorIs(t, err, tx.ErrCustomerMissing)
}

func TestCreate_CustomerStoreError(t *testing.T) {
	dbErr := errors.New("db connection lost")
	store := &mockStore{
		customerExists: func(ctx context.Context, id string) (bool, error) {
			return false, dbErr
		},
	}
	uc := newUsecase(store)

	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "cust-001",
		Items:      []tx.CreateItemIn{validItem()},
	})
	require.ErrorIs(t, err, dbErr)
}

func TestCreate_DraftDoesNotReserveStock(t *testing.T) {
	reserveCalled := false
	store := &mockStore{
		reserveStock: func(ctx context.Context, txID string) error {
			reserveCalled = true
			return nil
		},
	}
	uc := newUsecase(store)

	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "cust-001",
		Status:     tx.StatusDraft,
		Items:      []tx.CreateItemIn{validItem()},
	})
	require.NoError(t, err)
	assert.False(t, reserveCalled, "draft should NOT reserve stock")
}

func TestCreate_PendingReservesStock(t *testing.T) {
	reserveCalled := false
	store := &mockStore{
		reserveStock: func(ctx context.Context, txID string) error {
			reserveCalled = true
			return nil
		},
	}
	uc := newUsecase(store)

	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "cust-001",
		Status:     tx.StatusPending,
		Items:      []tx.CreateItemIn{validItem()},
	})
	require.NoError(t, err)
	assert.True(t, reserveCalled, "pending should reserve stock")
}

func TestCreate_CompletedReservesAndCommitsStock(t *testing.T) {
	reserveCalled := false
	commitCalled := false
	store := &mockStore{
		reserveStock: func(ctx context.Context, txID string) error {
			reserveCalled = true
			return nil
		},
		commitStock: func(ctx context.Context, txID string) error {
			commitCalled = true
			return nil
		},
	}
	uc := newUsecase(store)

	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "cust-001",
		Status:     tx.StatusCompleted,
		Items:      []tx.CreateItemIn{validItem()},
	})
	require.NoError(t, err)
	assert.True(t, reserveCalled, "completed should reserve stock")
	assert.True(t, commitCalled, "completed should also commit stock")
}

func TestCreate_InsufficientStock(t *testing.T) {
	store := &mockStore{
		getAvailableStock: func(ctx context.Context, stockProductID string) (int, error) {
			return 0, nil // no stock
		},
	}
	uc := newUsecase(store)

	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "cust-001",
		Status:     tx.StatusPending, // triggers stock check
		Items:      []tx.CreateItemIn{{ProductID: "prod-001", Qty: 5}},
	})
	require.ErrorIs(t, err, tx.ErrInsufficientStock)
}

func TestCreate_InvalidPackSize_NonInteger(t *testing.T) {
	store := &mockStore{
		getStockRule: func(ctx context.Context, productID string) (string, float64, error) {
			return productID, 1.5, nil // fractional pack — invalid in v1
		},
	}
	uc := newUsecase(store)

	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "cust-001",
		Status:     tx.StatusPending,
		Items:      []tx.CreateItemIn{validItem()},
	})
	require.ErrorIs(t, err, tx.ErrInvalidPackSize)
}

func TestCreate_InvalidPackSize_EmptyStockProduct(t *testing.T) {
	store := &mockStore{
		getStockRule: func(ctx context.Context, productID string) (string, float64, error) {
			return "", 0, nil // empty stockProductID
		},
	}
	uc := newUsecase(store)

	_, err := uc.Create(context.Background(), tx.CreateInput{
		CustomerID: "cust-001",
		Status:     tx.StatusPending,
		Items:      []tx.CreateItemIn{validItem()},
	})
	require.ErrorIs(t, err, tx.ErrInvalidPackSize)
}

// --- UpdateStatus Tests --------------------------------------------------

func TestUpdateStatus_EmptyID(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.UpdateStatus(context.Background(), "", tx.UpdateStatusInput{Status: tx.StatusPending})
	require.ErrorIs(t, err, tx.ErrInvalidInput)
}

func TestUpdateStatus_EmptyStatus(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.UpdateStatus(context.Background(), "tx-001", tx.UpdateStatusInput{})
	require.ErrorIs(t, err, tx.ErrInvalidInput)
}

func TestUpdateStatus_InvalidStatus(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.UpdateStatus(context.Background(), "tx-001", tx.UpdateStatusInput{Status: "flying"})
	require.ErrorIs(t, err, tx.ErrInvalidStatus)
}

func TestUpdateStatus_InvalidTransition_CompletedToPending(t *testing.T) {
	store := &mockStore{
		getByID: func(ctx context.Context, id string) (*tx.Transaction, error) {
			return &tx.Transaction{ID: id, Status: tx.StatusCompleted}, nil
		},
	}
	uc := newUsecase(store)
	_, err := uc.UpdateStatus(context.Background(), "tx-001", tx.UpdateStatusInput{Status: tx.StatusPending})
	require.ErrorIs(t, err, tx.ErrInvalidTransition)
}

func TestUpdateStatus_InvalidTransition_CancelledToDraft(t *testing.T) {
	store := &mockStore{
		getByID: func(ctx context.Context, id string) (*tx.Transaction, error) {
			return &tx.Transaction{ID: id, Status: tx.StatusCancelled}, nil
		},
	}
	uc := newUsecase(store)
	_, err := uc.UpdateStatus(context.Background(), "tx-001", tx.UpdateStatusInput{Status: tx.StatusDraft})
	require.ErrorIs(t, err, tx.ErrInvalidTransition)
}

func TestUpdateStatus_DraftToPending_ReservesStock(t *testing.T) {
	reserveCalled := false
	store := &mockStore{
		getByID: func(ctx context.Context, id string) (*tx.Transaction, error) {
			return &tx.Transaction{ID: id, Status: tx.StatusDraft}, nil
		},
		reserveStock: func(ctx context.Context, txID string) error {
			reserveCalled = true
			return nil
		},
	}
	uc := newUsecase(store)

	_, err := uc.UpdateStatus(context.Background(), "tx-001", tx.UpdateStatusInput{Status: tx.StatusPending})
	require.NoError(t, err)
	assert.True(t, reserveCalled)
}

func TestUpdateStatus_PendingToCancelled_ReleasesStock(t *testing.T) {
	releaseCalled := false
	store := &mockStore{
		getByID: func(ctx context.Context, id string) (*tx.Transaction, error) {
			return &tx.Transaction{ID: id, Status: tx.StatusPending}, nil
		},
		releaseStock: func(ctx context.Context, txID string) error {
			releaseCalled = true
			return nil
		},
	}
	uc := newUsecase(store)

	_, err := uc.UpdateStatus(context.Background(), "tx-001", tx.UpdateStatusInput{Status: tx.StatusCancelled})
	require.NoError(t, err)
	assert.True(t, releaseCalled)
}

func TestUpdateStatus_DraftToCancelled_NoStockOps(t *testing.T) {
	reserveCalled, releaseCalled, commitCalled := false, false, false
	store := &mockStore{
		getByID: func(ctx context.Context, id string) (*tx.Transaction, error) {
			return &tx.Transaction{ID: id, Status: tx.StatusDraft}, nil
		},
		reserveStock: func(ctx context.Context, txID string) error { reserveCalled = true; return nil },
		releaseStock: func(ctx context.Context, txID string) error { releaseCalled = true; return nil },
		commitStock:  func(ctx context.Context, txID string) error { commitCalled = true; return nil },
	}
	uc := newUsecase(store)

	_, err := uc.UpdateStatus(context.Background(), "tx-001", tx.UpdateStatusInput{Status: tx.StatusCancelled})
	require.NoError(t, err)
	assert.False(t, reserveCalled, "draft→cancelled should not reserve")
	assert.False(t, releaseCalled, "draft→cancelled should not release")
	assert.False(t, commitCalled, "draft→cancelled should not commit")
}

func TestUpdateStatus_PendingToCompleted_CommitsStock(t *testing.T) {
	commitCalled := false
	store := &mockStore{
		getByID: func(ctx context.Context, id string) (*tx.Transaction, error) {
			return &tx.Transaction{ID: id, Status: tx.StatusPending}, nil
		},
		commitStock: func(ctx context.Context, txID string) error {
			commitCalled = true
			return nil
		},
	}
	uc := newUsecase(store)

	_, err := uc.UpdateStatus(context.Background(), "tx-001", tx.UpdateStatusInput{Status: tx.StatusCompleted})
	require.NoError(t, err)
	assert.True(t, commitCalled)
}

// --- List Tests ----------------------------------------------------------

func TestList_ClampsLimitAndOffset(t *testing.T) {
	var capturedInput tx.ListInput
	store := &mockStore{
		list: func(ctx context.Context, in tx.ListInput) ([]tx.Transaction, error) {
			capturedInput = in
			return []tx.Transaction{}, nil
		},
	}
	uc := newUsecase(store)

	// bad values: limit=0, offset=-5
	_, err := uc.List(context.Background(), tx.ListInput{Limit: 0, Offset: -5})
	require.NoError(t, err)
	assert.Equal(t, 20, capturedInput.Limit, "limit=0 should clamp to default 20")
	assert.Equal(t, 0, capturedInput.Offset, "negative offset should clamp to 0")
}

func TestList_ClampsExcessiveLimit(t *testing.T) {
	var capturedInput tx.ListInput
	store := &mockStore{
		list: func(ctx context.Context, in tx.ListInput) ([]tx.Transaction, error) {
			capturedInput = in
			return []tx.Transaction{}, nil
		},
	}
	uc := newUsecase(store)

	_, err := uc.List(context.Background(), tx.ListInput{Limit: 9999, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 20, capturedInput.Limit, "limit > 100 should clamp to default 20")
}

// --- GetByID / GetViewByID Tests -----------------------------------------

func TestGetByID_EmptyID(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.GetByID(context.Background(), "")
	require.ErrorIs(t, err, tx.ErrInvalidInput)
}

func TestGetViewByID_EmptyID(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.GetViewByID(context.Background(), "")
	require.ErrorIs(t, err, tx.ErrInvalidInput)
}

// --- Fulfill Tests -------------------------------------------------------

func TestFulfill_EmptyID(t *testing.T) {
	uc := newUsecase(&mockStore{})
	_, err := uc.Fulfill(context.Background(), "")
	require.ErrorIs(t, err, tx.ErrInvalidInput)
}

func TestFulfill_Success(t *testing.T) {
	fulfillCalled := false
	store := &mockStore{
		fulfill: func(ctx context.Context, id string) (*tx.Transaction, error) {
			fulfillCalled = true
			return &tx.Transaction{ID: id, Status: tx.StatusCompleted}, nil
		},
	}
	uc := newUsecase(store)

	out, err := uc.Fulfill(context.Background(), "tx-001")
	require.NoError(t, err)
	require.True(t, fulfillCalled)
	assert.Equal(t, tx.StatusCompleted, out.Status)
}

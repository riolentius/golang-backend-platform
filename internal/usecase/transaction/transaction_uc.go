package transaction

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrCustomerMissing   = errors.New("customer not found")
	ErrProductMissing    = errors.New("product not found")
	ErrPriceMissing      = errors.New("product price not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidStatus     = errors.New("invalid status")
	ErrInvalidTransition = errors.New("invalid status transition")
)

const (
	StatusDraft     = "draft"     // created but not submitted
	StatusPending   = "pending"   // submitted (reserve stock)
	StatusCompleted = "completed" // paid/finished (commit stock)
	StatusCancelled = "cancelled" // cancelled (release reservation)
)

type Store interface {
	// Validation helpers
	CustomerExists(ctx context.Context, customerID string) (bool, error)
	ProductExists(ctx context.Context, productID string) (bool, error)

	// Inventory
	// Available = on_hand - reserved (or computed any way you store)
	GetAvailableStock(ctx context.Context, productID string) (int, error)

	// Transaction persistence
	Create(ctx context.Context, in CreateInput) (*Transaction, error)
	List(ctx context.Context, in ListInput) ([]Transaction, error)
	GetByID(ctx context.Context, id string) (*Transaction, error)

	// Status + inventory operations (should be atomic inside DB tx in adapter)
	ReserveStockForTx(ctx context.Context, txID string) error
	ReleaseStockForTx(ctx context.Context, txID string) error
	CommitStockForTx(ctx context.Context, txID string) error
	UpdateStatus(ctx context.Context, id string, status string) (*Transaction, error)
}

type Usecase struct {
	store Store
}

func New(store Store) *Usecase {
	return &Usecase{store: store}
}

func (u *Usecase) Create(ctx context.Context, in CreateInput) (*Transaction, error) {
	// 1) Validate input shape
	if in.CustomerID == "" || len(in.Items) == 0 {
		return nil, ErrInvalidInput
	}
	for _, it := range in.Items {
		if it.ProductID == "" || it.Qty <= 0 {
			return nil, ErrInvalidInput
		}
	}
	if in.Status == "" {
		in.Status = StatusDraft // default
	}
	if !isValidStatus(in.Status) {
		return nil, ErrInvalidStatus
	}

	// 2) Validate customer exists
	ok, err := u.store.CustomerExists(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCustomerMissing
	}

	// 3) Validate products exist + stock availability (only if reserving now)
	// We check stock only when status is pending (reserve) or completed (commit).
	if in.Status == StatusPending || in.Status == StatusCompleted {
		for _, it := range in.Items {
			ok, err := u.store.ProductExists(ctx, it.ProductID)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("%w: %s", ErrProductMissing, it.ProductID)
			}

			avail, err := u.store.GetAvailableStock(ctx, it.ProductID)
			if err != nil {
				return nil, err
			}
			if avail < it.Qty {
				return nil, fmt.Errorf("%w: product=%s available=%d requested=%d",
					ErrInsufficientStock, it.ProductID, avail, it.Qty)
			}
		}
	}

	// 4) Create the transaction row + items
	tx, err := u.store.Create(ctx, in)
	if err != nil {
		return nil, err
	}

	// 5) If created as pending -> reserve stock now
	// If created as completed -> reserve + commit (or directly commit depending on your implementation)
	switch tx.Status {
	case StatusPending:
		if err := u.store.ReserveStockForTx(ctx, tx.ID); err != nil {
			return nil, err
		}
	case StatusCompleted:
		// safest: reserve then commit (adapter can implement in one DB tx)
		if err := u.store.ReserveStockForTx(ctx, tx.ID); err != nil {
			return nil, err
		}
		if err := u.store.CommitStockForTx(ctx, tx.ID); err != nil {
			return nil, err
		}
	}

	return tx, nil
}

func (u *Usecase) List(ctx context.Context, in ListInput) ([]Transaction, error) {
	if in.Limit <= 0 || in.Limit > 100 {
		in.Limit = 20
	}
	if in.Offset < 0 {
		in.Offset = 0
	}
	return u.store.List(ctx, in)
}

func (u *Usecase) GetByID(ctx context.Context, id string) (*Transaction, error) {
	if id == "" {
		return nil, ErrInvalidInput
	}
	return u.store.GetByID(ctx, id)
}

func (u *Usecase) UpdateStatus(ctx context.Context, id string, in UpdateStatusInput) (*Transaction, error) {
	if id == "" || in.Status == "" {
		return nil, ErrInvalidInput
	}
	if !isValidStatus(in.Status) {
		return nil, ErrInvalidStatus
	}

	// Load current state so we can enforce transitions
	cur, err := u.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !isValidTransition(cur.Status, in.Status) {
		return nil, ErrInvalidTransition
	}

	// Transition rules + inventory side effects
	// IMPORTANT: in a real implementation, your adapter should do these atomically in 1 DB transaction.
	switch {
	case cur.Status == StatusDraft && in.Status == StatusPending:
		// reserve
		if err := u.store.ReserveStockForTx(ctx, id); err != nil {
			return nil, err
		}
	case cur.Status == StatusPending && in.Status == StatusCancelled:
		// release reservation
		if err := u.store.ReleaseStockForTx(ctx, id); err != nil {
			return nil, err
		}
	case cur.Status == StatusPending && in.Status == StatusCompleted:
		// commit: reduce on_hand and reserved
		if err := u.store.CommitStockForTx(ctx, id); err != nil {
			return nil, err
		}
		// draft -> cancelled: no stock action
	}

	return u.store.UpdateStatus(ctx, id, in.Status)
}

func isValidStatus(s string) bool {
	switch s {
	case StatusDraft, StatusPending, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}

func isValidTransition(from, to string) bool {
	switch from {
	case StatusDraft:
		return to == StatusPending || to == StatusCancelled
	case StatusPending:
		return to == StatusCompleted || to == StatusCancelled
	case StatusCompleted:
		return false
	case StatusCancelled:
		return false
	default:
		return false
	}
}

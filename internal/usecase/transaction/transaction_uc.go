package transaction

import (
	"context"
	"errors"
	"fmt"
	"math"
)

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrCustomerMissing     = errors.New("customer not found")
	ErrProductMissing      = errors.New("product not found")
	ErrPriceMissing        = errors.New("product price not found")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrInvalidStatus       = errors.New("invalid status")
	ErrInvalidTransition   = errors.New("invalid status transition")
	ErrAlreadyFulfilled    = errors.New("transaction already fulfilled")
	ErrTransactionMissing  = errors.New("transaction not found")
	ErrTransactionCanceled = errors.New("transaction cancelled")
	ErrInvalidPackSize     = errors.New("invalid pack size")
)

const (
	StatusDraft     = "draft"
	StatusPending   = "pending"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
)

type Store interface {
	CustomerExists(ctx context.Context, customerID string) (bool, error)
	ProductExists(ctx context.Context, productID string) (bool, error)

	// NEW: resolve inventory product + conversion
	// stockProductID: where stock is stored (base_product_id if exists, else the product itself)
	// packSize: how many base units consumed by qty=1 of productID (default 1)
	GetStockRule(ctx context.Context, productID string) (stockProductID string, packSize float64, err error)

	// IMPORTANT: available stock is in BASE UNITS of the stockProductID
	GetAvailableStock(ctx context.Context, stockProductID string) (int, error)

	Create(ctx context.Context, in CreateInput) (*Transaction, error)
	List(ctx context.Context, in ListInput) ([]Transaction, error)
	GetByID(ctx context.Context, id string) (*Transaction, error)

	ReserveStockForTx(ctx context.Context, txID string) error
	ReleaseStockForTx(ctx context.Context, txID string) error
	CommitStockForTx(ctx context.Context, txID string) error

	UpdateStatus(ctx context.Context, id string, status string) (*Transaction, error)
	GetViewByID(ctx context.Context, id string) (*TransactionView, error)

	Fulfill(ctx context.Context, id string) (*Transaction, error)
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
		in.Status = StatusDraft
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
	if in.Status == StatusPending || in.Status == StatusCompleted {
		for _, it := range in.Items {
			ok, err := u.store.ProductExists(ctx, it.ProductID)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("%w: %s", ErrProductMissing, it.ProductID)
			}

			stockProductID, packSize, err := u.store.GetStockRule(ctx, it.ProductID)
			if err != nil {
				return nil, err
			}
			if stockProductID == "" || packSize <= 0 {
				return nil, ErrInvalidPackSize
			}

			// v1 rule: packSize must be a whole number (boxes/packs)
			if math.Trunc(packSize) != packSize {
				return nil, fmt.Errorf("%w: product=%s pack_size=%v", ErrInvalidPackSize, it.ProductID, packSize)
			}

			requiredBaseUnits := int(packSize) * it.Qty

			avail, err := u.store.GetAvailableStock(ctx, stockProductID)
			if err != nil {
				return nil, err
			}
			if avail < requiredBaseUnits {
				return nil, fmt.Errorf(
					"%w: product=%s stock_product=%s available=%d required=%d",
					ErrInsufficientStock, it.ProductID, stockProductID, avail, requiredBaseUnits,
				)
			}
		}
	}

	// 4) Create the transaction row + items
	tx, err := u.store.Create(ctx, in)
	if err != nil {
		return nil, err
	}

	// 5) If created as pending -> reserve stock now
	// If created as completed -> reserve + commit
	switch tx.Status {
	case StatusPending:
		if err := u.store.ReserveStockForTx(ctx, tx.ID); err != nil {
			return nil, err
		}
	case StatusCompleted:
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

	cur, err := u.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !isValidTransition(cur.Status, in.Status) {
		return nil, ErrInvalidTransition
	}

	switch {
	case cur.Status == StatusDraft && in.Status == StatusPending:
		if err := u.store.ReserveStockForTx(ctx, id); err != nil {
			return nil, err
		}
	case cur.Status == StatusPending && in.Status == StatusCancelled:
		if err := u.store.ReleaseStockForTx(ctx, id); err != nil {
			return nil, err
		}
	case cur.Status == StatusPending && in.Status == StatusCompleted:
		if err := u.store.CommitStockForTx(ctx, id); err != nil {
			return nil, err
		}
	}

	return u.store.UpdateStatus(ctx, id, in.Status)
}

func (u *Usecase) Fulfill(ctx context.Context, id string) (*Transaction, error) {
	if id == "" {
		return nil, ErrInvalidInput
	}
	return u.store.Fulfill(ctx, id)
}

func (u *Usecase) GetViewByID(ctx context.Context, id string) (*TransactionView, error) {
	if id == "" {
		return nil, ErrInvalidInput
	}
	return u.store.GetViewByID(ctx, id)
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
	case StatusCompleted, StatusCancelled:
		return false
	default:
		return false
	}
}

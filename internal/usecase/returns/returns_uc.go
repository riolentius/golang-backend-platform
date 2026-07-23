package returns

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidInput         = errors.New("invalid input")
	ErrTransactionMissing   = errors.New("transaction not found")
	ErrNotFulfilled         = errors.New("only completed transactions can accept returns")
	ErrItemNotInTransaction = errors.New("item does not belong to this transaction")
	ErrQtyExceedsReturnable = errors.New("return quantity exceeds the remaining returnable quantity")
	ErrDuplicateItem        = errors.New("the same transaction item appears twice; combine them into one line")
)

type Store interface {
	Create(ctx context.Context, in CreateInput, createdByEmail *string) (*CreateResult, error)
	ListByTransaction(ctx context.Context, transactionID string) ([]Return, error)
	ListReturnableItems(ctx context.Context, transactionID string) ([]ReturnableItem, error)
}

type Usecase struct {
	store Store
}

func New(store Store) *Usecase {
	return &Usecase{store: store}
}

func (u *Usecase) Create(ctx context.Context, in CreateInput, createdByEmail *string) (*CreateResult, error) {
	in.TransactionID = strings.TrimSpace(in.TransactionID)
	if in.TransactionID == "" || len(in.Items) == 0 {
		return nil, ErrInvalidInput
	}

	// A return form naturally posts every sold line, most with qty 0. Drop those
	// rather than rejecting the request, and reject only if nothing is left.
	filtered := make([]CreateItemIn, 0, len(in.Items))
	seen := make(map[string]struct{}, len(in.Items))
	for i := range in.Items {
		id := strings.TrimSpace(in.Items[i].TransactionItemID)
		if in.Items[i].Qty == 0 {
			continue // not being returned
		}
		if id == "" || in.Items[i].Qty < 0 {
			return nil, ErrInvalidInput
		}
		if _, dup := seen[id]; dup {
			return nil, ErrDuplicateItem
		}
		seen[id] = struct{}{}
		in.Items[i].TransactionItemID = id
		filtered = append(filtered, in.Items[i])
	}
	if len(filtered) == 0 {
		return nil, ErrInvalidInput
	}
	in.Items = filtered

	return u.store.Create(ctx, in, createdByEmail)
}

func (u *Usecase) ListByTransaction(ctx context.Context, transactionID string) ([]Return, error) {
	if strings.TrimSpace(transactionID) == "" {
		return nil, ErrInvalidInput
	}
	return u.store.ListByTransaction(ctx, transactionID)
}

func (u *Usecase) ListReturnableItems(ctx context.Context, transactionID string) ([]ReturnableItem, error) {
	if strings.TrimSpace(transactionID) == "" {
		return nil, ErrInvalidInput
	}
	return u.store.ListReturnableItems(ctx, transactionID)
}

package payment

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrTransactionMissing = errors.New("transaction not found")
)

type Payment struct {
	ID            string    `json:"id"`
	TransactionID string    `json:"transactionId"`
	Method        string    `json:"method"` // cash | transfer
	Amount        string    `json:"amount"` // keep as string (numeric) for now
	Currency      string    `json:"currency"`
	PaidAt        time.Time `json:"paidAt"`
	SenderName    *string   `json:"senderName,omitempty"`
	Reference     *string   `json:"reference,omitempty"`
	Note          *string   `json:"note,omitempty"`
	Status        string    `json:"status"` // posted | voided
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type TransactionPaymentState struct {
	TransactionID string `json:"transactionId"`
	PaidAmount    string `json:"paidAmount"`
	PaymentStatus string `json:"paymentStatus"` // unpaid|partial|paid|overpaid
	TotalAmount   string `json:"totalAmount"`
	Currency      string `json:"currency"`
}

type Store interface {
	Create(ctx context.Context, in CreateInput) (*Payment, *TransactionPaymentState, error)
	ListByTransaction(ctx context.Context, transactionID string) ([]Payment, error)
}

type Usecase struct {
	store Store
}

func New(store Store) *Usecase {
	return &Usecase{store: store}
}

type CreateInput struct {
	TransactionID string     `json:"-"`
	Method        string     `json:"method"`
	Amount        string     `json:"amount"` // accept string to avoid float issues
	SenderName    *string    `json:"senderName"`
	Reference     *string    `json:"reference"`
	Note          *string    `json:"note"`
	PaidAt        *time.Time `json:"paidAt"` // optional (default now)
}

func (u *Usecase) Create(ctx context.Context, in CreateInput) (*Payment, *TransactionPaymentState, error) {
	if strings.TrimSpace(in.TransactionID) == "" {
		return nil, nil, ErrInvalidInput
	}
	m := strings.TrimSpace(in.Method)
	if m != "cash" && m != "transfer" {
		return nil, nil, ErrInvalidInput
	}
	in.Method = m

	if strings.TrimSpace(in.Amount) == "" {
		return nil, nil, ErrInvalidInput
	}
	// simple numeric validation can be added later; DB will enforce numeric cast

	// optional: if transfer, you probably want senderName/reference
	// keep flexible for v1.

	return u.store.Create(ctx, in)
}

func (u *Usecase) ListByTransaction(ctx context.Context, transactionID string) ([]Payment, error) {
	if strings.TrimSpace(transactionID) == "" {
		return nil, ErrInvalidInput
	}
	return u.store.ListByTransaction(ctx, transactionID)
}

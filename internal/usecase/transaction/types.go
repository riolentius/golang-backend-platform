package transaction

import "time"

type Transaction struct {
	ID           string    `json:"id"`
	CustomerID   string    `json:"customerId"`
	CustomerName string    `json:"customerName,omitempty"`
	Status       string    `json:"status"`
	Currency     string    `json:"currency"`
	TotalAmount  string    `json:"totalAmount"`
	Notes        *string   `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Items        []Item    `json:"items,omitempty"`
}

type Item struct {
	ID            string    `json:"id"`
	TransactionID string    `json:"transactionId"`
	ProductID     string    `json:"productId"`
	Qty           int       `json:"qty"`
	UnitAmount    string    `json:"unitAmount"`
	LineTotal     string    `json:"lineTotal"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type CreateInput struct {
	CustomerID string         `json:"customerId"`
	Notes      *string        `json:"notes"`
	Items      []CreateItemIn `json:"items"`
	Status     string         `json:"status"`
}

type CreateItemIn struct {
	ProductID string `json:"productId"`
	Qty       int    `json:"qty"`
}

type ListInput struct {
	Limit  int
	Offset int
	Status *string
}

type UpdateStatusInput struct {
	Status string `json:"status"`
}

type UpdateItemsInput struct {
	Items []UpdateItemIn `json:"items"`
}

type UpdateItemIn struct {
	ProductID string `json:"productId"`
	Qty       int    `json:"qty"`
}

type CustomerTransactionSummary struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	TotalAmount   string    `json:"totalAmount"`
	PaidAmount    string    `json:"paidAmount"`
	BalanceDue    string    `json:"balanceDue"`
	PaymentStatus string    `json:"paymentStatus"`
	CreatedAt     time.Time `json:"createdAt"`
}

type CustomerTransactionsResult struct {
	Items            []CustomerTransactionSummary `json:"items"`
	TotalOutstanding string                       `json:"totalOutstanding"`
}

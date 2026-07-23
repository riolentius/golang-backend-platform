package returns

import "time"

type Return struct {
	ID             string       `json:"id"`
	TransactionID  string       `json:"transactionId"`
	TotalAmount    string       `json:"totalAmount"`
	Currency       string       `json:"currency"`
	Note           *string      `json:"note,omitempty"`
	CreatedByEmail *string      `json:"createdByEmail,omitempty"`
	CreatedAt      time.Time    `json:"createdAt"`
	Items          []ReturnItem `json:"items"`
}

type ReturnItem struct {
	ID                string  `json:"id"`
	TransactionItemID string  `json:"transactionItemId"`
	ProductID         string  `json:"productId"`
	ProductName       string  `json:"productName"`
	SKU               *string `json:"sku,omitempty"`
	Qty               int     `json:"qty"`
	UnitAmount        string  `json:"unitAmount"`
	LineTotal         string  `json:"lineTotal"`
	Restock           bool    `json:"restock"`
}

type CreateItemIn struct {
	TransactionItemID string `json:"transactionItemId"`
	Qty               int    `json:"qty"`
	Restock           bool   `json:"restock"`
}

type CreateInput struct {
	TransactionID string         `json:"-"`
	Note          *string        `json:"note"`
	Items         []CreateItemIn `json:"items"`
}

type TransactionState struct {
	ID            string `json:"id"`
	TotalAmount   string `json:"totalAmount"`
	PaidAmount    string `json:"paidAmount"`
	PaymentStatus string `json:"paymentStatus"`
	Currency      string `json:"currency"`
}

type CreateResult struct {
	Return      Return           `json:"return"`
	Transaction TransactionState `json:"transaction"`
}

type ReturnableItem struct {
	TransactionItemID string  `json:"transactionItemId"`
	ProductID         string  `json:"productId"`
	ProductName       string  `json:"productName"`
	SKU               *string `json:"sku,omitempty"`
	UnitAmount        string  `json:"unitAmount"`
	QtySold           int     `json:"qtySold"`
	QtyReturned       int     `json:"qtyReturned"`
	QtyReturnable     int     `json:"qtyReturnable"`
}

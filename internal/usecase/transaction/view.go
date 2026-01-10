package transaction

import "time"

type TransactionView struct {
	ID            string     `json:"id"`
	CustomerID    string     `json:"customerId"`
	CustomerName  string     `json:"customerName"`
	CategoryID    *string    `json:"categoryId,omitempty"`
	Status        string     `json:"status"`
	Currency      string     `json:"currency"`
	TotalAmount   string     `json:"totalAmount"`
	PaidAmount    string     `json:"paidAmount"`
	PaymentStatus string     `json:"paymentStatus"`
	BalanceDue    string     `json:"balanceDue"`
	Notes         *string    `json:"notes,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	Items         []ViewItem `json:"items"`
	Payments      []ViewPay  `json:"payments"`
}

type ViewItem struct {
	ProductID   string  `json:"productId"`
	SKU         *string `json:"sku,omitempty"`
	ProductName string  `json:"productName"`
	Qty         int     `json:"qty"`
	UnitAmount  string  `json:"unitAmount"`
	LineTotal   string  `json:"lineTotal"`
}

type ViewPay struct {
	ID         string    `json:"id"`
	Method     string    `json:"method"`
	Amount     string    `json:"amount"`
	Currency   string    `json:"currency"`
	PaidAt     time.Time `json:"paidAt"`
	SenderName *string   `json:"senderName,omitempty"`
	Reference  *string   `json:"reference,omitempty"`
	Note       *string   `json:"note,omitempty"`
	Status     string    `json:"status"`
}

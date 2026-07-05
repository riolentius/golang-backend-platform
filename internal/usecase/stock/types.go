package stock

import "time"

type Movement struct {
	ID             string    `json:"id"`
	ProductID      string    `json:"productId"`
	Direction      string    `json:"direction"`
	Quantity       int       `json:"quantity"`
	Source         string    `json:"source"`
	ReferenceID    *string   `json:"referenceId,omitempty"`
	Note           *string   `json:"note,omitempty"`
	CreatedByEmail *string   `json:"createdByEmail,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type StockInInput struct {
	Quantity int     `json:"quantity"`
	Note     *string `json:"note"`
}

type StockInResult struct {
	Movement       Movement `json:"movement"`
	StockProductID string   `json:"stockProductId"`
	NewStockOnHand int      `json:"newStockOnHand"`
}

type ListFilter struct {
	Direction *string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

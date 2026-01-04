package product_price

import "time"

type ProductPrice struct {
	ID         string     `json:"id"`
	ProductID  string     `json:"productId"`
	CategoryID *string    `json:"categoryId,omitempty"`
	Currency   string     `json:"currency"`
	Amount     string     `json:"amount"` // keep as string to avoid float issues in JSON
	ValidFrom  time.Time  `json:"validFrom"`
	ValidTo    *time.Time `json:"validTo,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type CreateInput struct {
	CategoryID *string    `json:"categoryId"`
	Currency   string     `json:"currency"`
	Amount     string     `json:"amount"`
	ValidFrom  *time.Time `json:"validFrom"`
	ValidTo    *time.Time `json:"validTo"`
}

type UpdateInput struct {
	CategoryID *string    `json:"categoryId"`
	Currency   *string    `json:"currency"`
	Amount     *string    `json:"amount"`
	ValidFrom  *time.Time `json:"validFrom"`
	ValidTo    *time.Time `json:"validTo"`
}

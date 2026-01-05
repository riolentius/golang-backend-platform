package customer

import "time"

type Customer struct {
	ID                   string    `json:"id"`
	FirstName            string    `json:"firstName"`
	LastName             *string   `json:"lastName,omitempty"`
	Email                string    `json:"email"`
	Phone                *string   `json:"phone,omitempty"`
	IdentificationNumber *string   `json:"identificationNumber,omitempty"`
	CategoryID           *string   `json:"categoryId,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type CreateInput struct {
	FirstName            string  `json:"firstName"`
	LastName             *string `json:"lastName"`
	Email                string  `json:"email"`
	Phone                *string `json:"phone"`
	IdentificationNumber *string `json:"identificationNumber"`
	CategoryID           *string `json:"categoryId"`
}

type UpdateInput struct {
	FirstName            *string `json:"firstName"`
	LastName             *string `json:"lastName"`
	Email                *string `json:"email"`
	Phone                *string `json:"phone"`
	IdentificationNumber *string `json:"identificationNumber"`
	CategoryID           *string `json:"categoryId"`
}

type ListQuery struct {
	Limit  int
	Offset int
}

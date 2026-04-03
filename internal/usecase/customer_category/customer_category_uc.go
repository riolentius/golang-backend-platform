package customercategory

import "context"

type CustomerCategory struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type Store interface {
	List(ctx context.Context) ([]CustomerCategory, error)
}

type Usecase struct {
	store Store
}

func New(store Store) *Usecase {
	return &Usecase{store: store}
}

func (u *Usecase) List(ctx context.Context) ([]CustomerCategory, error) {
	return u.store.List(ctx)
}

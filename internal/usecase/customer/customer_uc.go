package customer

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput  = errors.New("invalid input")
	ErrNotFound      = errors.New("not found")
	ErrEmailConflict = errors.New("email already exists")
)

type Store interface {
	Create(ctx context.Context, in CreateInput) (*Customer, error)
	GetByID(ctx context.Context, id string) (*Customer, error)
	List(ctx context.Context, q ListQuery) ([]Customer, error)
	Update(ctx context.Context, id string, in UpdateInput) (*Customer, error)
}

type Usecase struct {
	store Store
}

func New(store Store) *Usecase {
	return &Usecase{store: store}
}

func (u *Usecase) Create(ctx context.Context, in CreateInput) (*Customer, error) {
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.Phone = strings.TrimSpace(in.Phone)
	in.CategoryID = strings.TrimSpace(in.CategoryID)

	// Required in both flows: walk-in (transaction-inline) and B2B (Add Customer form).
	if in.FirstName == "" || in.Phone == "" || in.CategoryID == "" {
		return nil, ErrInvalidInput
	}

	if _, err := uuid.Parse(in.CategoryID); err != nil {
		return nil, ErrInvalidInput
	}

	// Email is optional — walk-in customers usually won't have one.
	// Normalize an explicit empty string to "no email" rather than rejecting it.
	if in.Email != nil {
		e := strings.TrimSpace(strings.ToLower(*in.Email))
		switch {
		case e == "":
			in.Email = nil
		case !strings.Contains(e, "@"):
			return nil, ErrInvalidInput
		default:
			in.Email = &e
		}
	}

	return u.store.Create(ctx, in)
}

func (u *Usecase) GetByID(ctx context.Context, id string) (*Customer, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidInput
	}
	return u.store.GetByID(ctx, id)
}

func (u *Usecase) List(ctx context.Context, q ListQuery) ([]Customer, error) {
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 50
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	return u.store.List(ctx, q)
}

func (u *Usecase) Update(ctx context.Context, id string, in UpdateInput) (*Customer, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidInput
	}

	if in.Email != nil {
		e := strings.TrimSpace(strings.ToLower(*in.Email))
		if e == "" || !strings.Contains(e, "@") {
			return nil, ErrInvalidInput
		}
		in.Email = &e
	}

	if in.FirstName != nil {
		f := strings.TrimSpace(*in.FirstName)
		if f == "" {
			return nil, ErrInvalidInput
		}
		in.FirstName = &f
	}

	// Phone and CategoryID are required fields — if the request explicitly
	// includes them, they must not be blanked out.
	if in.Phone != nil {
		p := strings.TrimSpace(*in.Phone)
		if p == "" {
			return nil, ErrInvalidInput
		}
		in.Phone = &p
	}

	if in.CategoryID != nil {
		c := strings.TrimSpace(*in.CategoryID)
		if c == "" {
			return nil, ErrInvalidInput
		}
		if _, err := uuid.Parse(c); err != nil {
			return nil, ErrInvalidInput
		}
		in.CategoryID = &c
	}

	return u.store.Update(ctx, id, in)
}

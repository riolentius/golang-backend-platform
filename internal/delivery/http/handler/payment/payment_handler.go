package payment

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	payuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/payment"
	"github.com/riolentius/cahaya-gading-backend/pkg/apierr"
)

type Handler struct {
	uc *payuc.Usecase
}

func New(uc *payuc.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) CreateForTransaction(c *fiber.Ctx) error {
	trxID := c.Params("id")

	var req payuc.CreateInput
	if err := c.BodyParser(&req); err != nil {
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}
	req.TransactionID = trxID

	p, state, err := h.uc.Create(c.Context(), req)
	if err != nil {
		return mapErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"payment":     p,
		"transaction": state,
	})
}

func (h *Handler) ListForTransaction(c *fiber.Ctx) error {
	trxID := c.Params("id")

	items, err := h.uc.ListByTransaction(c.Context(), trxID)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

// mapErr translates payment usecase errors to consistent HTTP responses.
func mapErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, payuc.ErrInvalidInput):
		return apierr.BadRequest(c, apierr.CodeInvalidInput, err.Error())
	case errors.Is(err, payuc.ErrTransactionMissing):
		return apierr.NotFound(c, err.Error())
	default:
		return apierr.Internal(c)
	}
}

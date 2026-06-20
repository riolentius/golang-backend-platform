package transaction

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	txuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/transaction"
	"github.com/riolentius/cahaya-gading-backend/pkg/apierr"
)

type Handler struct {
	uc *txuc.Usecase
}

func New(uc *txuc.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var in txuc.CreateInput
	if err := c.BodyParser(&in); err != nil {
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}

	out, err := h.uc.Create(c.Context(), in)
	if err != nil {
		return mapErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *Handler) List(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	out, err := h.uc.List(c.Context(), txuc.ListInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(fiber.Map{"items": out})
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	out, err := h.uc.GetByID(c.Context(), id)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(out)
}

func (h *Handler) GetViewByID(c *fiber.Ctx) error {
	id := c.Params("id")
	out, err := h.uc.GetViewByID(c.Context(), id)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(out)
}

func (h *Handler) UpdateStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	var in txuc.UpdateStatusInput
	if err := c.BodyParser(&in); err != nil {
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}

	out, err := h.uc.UpdateStatus(c.Context(), id, in)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(out)
}

func (h *Handler) Fulfill(c *fiber.Ctx) error {
	id := c.Params("id")
	out, err := h.uc.Fulfill(c.Context(), id)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(out)
}

func (h *Handler) UpdateItems(c *fiber.Ctx) error {
	id := c.Params("id")
	var in txuc.UpdateItemsInput
	if err := c.BodyParser(&in); err != nil {
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}

	out, err := h.uc.UpdateItems(c.Context(), id, in)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(out)
}

func (h *Handler) ListByCustomer(c *fiber.Ctx) error {
	customerID := c.Params("id")
	out, err := h.uc.ListByCustomer(c.Context(), customerID)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(out)
}

func mapErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, txuc.ErrInvalidInput):
		return apierr.BadRequest(c, apierr.CodeInvalidInput, err.Error())
	case errors.Is(err, txuc.ErrInvalidStatus):
		return apierr.BadRequest(c, apierr.CodeInvalidInput, err.Error())
	case errors.Is(err, txuc.ErrCustomerMissing),
		errors.Is(err, txuc.ErrProductMissing),
		errors.Is(err, txuc.ErrTransactionMissing):
		return apierr.NotFound(c, err.Error())
	case errors.Is(err, txuc.ErrInvalidTransition):
		return apierr.Conflict(c, apierr.CodeInvalidTransition, err.Error())
	case errors.Is(err, txuc.ErrTransactionNotEditable):
		return apierr.Conflict(c, apierr.CodeInvalidTransition, err.Error())
	case errors.Is(err, txuc.ErrInsufficientStock):
		return apierr.Conflict(c, apierr.CodeInsufficientStock, err.Error())
	case errors.Is(err, txuc.ErrAlreadyFulfilled):
		return apierr.Conflict(c, apierr.CodeAlreadyFulfilled, err.Error())
	case errors.Is(err, txuc.ErrTransactionCanceled):
		return apierr.Conflict(c, apierr.CodeTransactionCanceled, err.Error())
	default:
		return apierr.Internal(c)
	}
}

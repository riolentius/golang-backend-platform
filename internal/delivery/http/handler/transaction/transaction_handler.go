package transaction

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	txuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/transaction"
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
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}

	out, err := h.uc.Create(c.Context(), in)
	return writeOne(c, out, err, fiber.StatusCreated)
}

func (h *Handler) List(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	out, err := h.uc.List(c.Context(), txuc.ListInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return mapErr(err)
	}

	return c.JSON(fiber.Map{"items": out})
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	out, err := h.uc.GetByID(c.Context(), id)
	return writeOne(c, out, err, fiber.StatusOK)
}

func (h *Handler) UpdateStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	var in txuc.UpdateStatusInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}

	out, err := h.uc.UpdateStatus(c.Context(), id, in)
	return writeOne(c, out, err, fiber.StatusOK)
}

func writeOne(c *fiber.Ctx, out *txuc.Transaction, err error, okStatus int) error {
	if err != nil {
		return mapErr(err)
	}
	return c.Status(okStatus).JSON(out)
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, txuc.ErrInvalidInput):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, txuc.ErrInvalidStatus):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, txuc.ErrInvalidTransition):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, txuc.ErrInsufficientStock):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, txuc.ErrTransactionMissing):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) Fulfill(c *fiber.Ctx) error {
	id := c.Params("id")

	out, err := h.uc.Fulfill(c.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, txuc.ErrInvalidInput):
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, txuc.ErrTransactionMissing):
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, txuc.ErrAlreadyFulfilled):
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, txuc.ErrTransactionCanceled):
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, txuc.ErrInsufficientStock):
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(500).JSON(fiber.Map{"error": "internal error"})
		}
	}

	return c.JSON(out)
}

func (h *Handler) GetViewByID(c *fiber.Ctx) error {
	id := c.Params("id")
	out, err := h.uc.GetViewByID(c.Context(), id)
	if err != nil {
		return mapErr(err)
	}
	return c.JSON(out)
}

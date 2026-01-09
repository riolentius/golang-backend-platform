package payment

import (
	"github.com/gofiber/fiber/v2"

	payuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/payment"
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
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	req.TransactionID = trxID

	p, state, err := h.uc.Create(c.Context(), req)
	if err != nil {
		switch err {
		case payuc.ErrInvalidInput:
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		case payuc.ErrTransactionMissing:
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(500).JSON(fiber.Map{"error": "internal error"})
		}
	}

	return c.Status(201).JSON(fiber.Map{
		"payment":     p,
		"transaction": state,
	})
}

func (h *Handler) ListForTransaction(c *fiber.Ctx) error {
	trxID := c.Params("id")

	items, err := h.uc.ListByTransaction(c.Context(), trxID)
	if err != nil {
		if err == payuc.ErrInvalidInput {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(fiber.Map{"items": items})
}

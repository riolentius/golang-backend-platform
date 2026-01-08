package product

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	productuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product"
)

type Handler struct {
	uc *productuc.Usecase
}

func New(uc *productuc.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req productuc.CreateInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	out, err := h.uc.Create(c.Context(), req)
	if err != nil {
		if err == productuc.ErrInvalidInput {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(201).JSON(out)
}

func (h *Handler) List(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	out, err := h.uc.List(c.Context(), limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(fiber.Map{"items": out})
}

func (h *Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req productuc.UpdateInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	out, err := h.uc.Update(c.Context(), id, req)
	if err != nil {
		if err == productuc.ErrInvalidInput {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(out)
}

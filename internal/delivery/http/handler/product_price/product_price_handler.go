package product_price

import (
	"github.com/gofiber/fiber/v2"

	priceuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product_price"
)

type Handler struct {
	uc *priceuc.Usecase
}

func New(uc *priceuc.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) CreateForProduct(c *fiber.Ctx) error {
	productID := c.Params("id")

	var in priceuc.CreateInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}

	out, err := h.uc.CreateForProduct(c.Context(), productID, in)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *Handler) ListForProduct(c *fiber.Ctx) error {
	productID := c.Params("id")

	out, err := h.uc.ListForProduct(c.Context(), productID)
	if err != nil {
		return err
	}
	return c.JSON(out)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	priceID := c.Params("id")

	var in priceuc.UpdateInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}

	out, err := h.uc.Update(c.Context(), priceID, in)
	if err != nil {
		return err
	}
	return c.JSON(out)
}

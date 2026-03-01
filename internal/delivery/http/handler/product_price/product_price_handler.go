package product_price

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	priceuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product_price"
	"github.com/riolentius/cahaya-gading-backend/pkg/apierr"
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
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}

	out, err := h.uc.CreateForProduct(c.Context(), productID, in)
	if err != nil {
		return mapErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *Handler) ListForProduct(c *fiber.Ctx) error {
	productID := c.Params("id")

	out, err := h.uc.ListForProduct(c.Context(), productID)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(out)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	priceID := c.Params("id")

	var in priceuc.UpdateInput
	if err := c.BodyParser(&in); err != nil {
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}

	out, err := h.uc.Update(c.Context(), priceID, in)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(out)
}

func mapErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, priceuc.ErrInvalidInput):
		return apierr.BadRequest(c, apierr.CodeInvalidInput, err.Error())
	case errors.Is(err, priceuc.ErrNotFound):
		return apierr.NotFound(c, err.Error())
	case errors.Is(err, priceuc.ErrProductNotFound):
		return apierr.NotFound(c, err.Error())
	default:
		return apierr.Internal(c)
	}
}

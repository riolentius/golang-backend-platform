package product

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
	productuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product"
	"github.com/riolentius/cahaya-gading-backend/pkg/apierr"
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
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}

	out, err := h.uc.Create(c.Context(), req)
	if err != nil {
		return mapErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *Handler) List(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	out, err := h.uc.List(c.Context(), limit, offset)
	if err != nil {
		log.Printf("[product.list] unexpected error: %v", err)
		return apierr.Internal(c)
	}
	return c.JSON(fiber.Map{"items": out})
}

func (h *Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var req productuc.UpdateInput
	if err := c.BodyParser(&req); err != nil {
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}

	out, err := h.uc.Update(c.Context(), id, req)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(out)
}

func mapErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, productuc.ErrInvalidInput):
		return apierr.BadRequest(c, apierr.CodeInvalidInput, err.Error())
	case errors.Is(err, productuc.ErrNotFound):
		return apierr.NotFound(c, err.Error())
	default:
		log.Printf("[product] unexpected error: %v", err)
		return apierr.Internal(c)
	}
}

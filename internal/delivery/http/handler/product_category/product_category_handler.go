package productcategoryhandler

import (
	"context"
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
	prodcatuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product_category"
	"github.com/riolentius/cahaya-gading-backend/pkg/apierr"
)

type productCategoryUsecase interface {
	List(ctx context.Context) ([]prodcatuc.ProductCategory, error)
	Create(ctx context.Context, in prodcatuc.CreateInput) (*prodcatuc.ProductCategory, error)
	Update(ctx context.Context, id string, in prodcatuc.UpdateInput) (*prodcatuc.ProductCategory, error)
}

type Handler struct {
	uc productCategoryUsecase
}

func New(uc productCategoryUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) List(c *fiber.Ctx) error {
	items, err := h.uc.List(c.Context())
	if err != nil {
		return apierr.Internal(c)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req prodcatuc.CreateInput
	if err := c.BodyParser(&req); err != nil {
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}
	out, err := h.uc.Create(c.Context(), req)
	if err != nil {
		return mapErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var req prodcatuc.UpdateInput
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
	case errors.Is(err, prodcatuc.ErrInvalidInput):
		return apierr.BadRequest(c, apierr.CodeInvalidInput, err.Error())
	case errors.Is(err, prodcatuc.ErrNotFound):
		return apierr.NotFound(c, err.Error())
	default:
		log.Printf("[product_category] unexpected error: %v", err)
		return apierr.Internal(c)
	}
}

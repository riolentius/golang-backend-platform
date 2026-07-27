package product

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/riolentius/cahaya-gading-backend/internal/delivery/middleware"
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

	email := emailPtr(c)
	out, err := h.uc.Create(c.Context(), req, email)
	if err != nil {
		return mapErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *Handler) List(c *fiber.Ctx) error {
	params := productuc.ListParams{
		Search: c.Query("search"),
		Status: c.Query("status"),
		Limit:  c.QueryInt("limit", 20),
		Offset: c.QueryInt("offset", 0),
		Sort:   c.Query("sort", "alphabet"),
	}

	out, err := h.uc.List(c.Context(), params)
	if err != nil {
		log.Printf("[product.list] unexpected error: %v", err)
		return apierr.Internal(c)
	}
	return c.JSON(fiber.Map{"items": out.Items, "total": out.Total})
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	out, err := h.uc.GetByID(c.Context(), id)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(out)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.uc.Delete(c.Context(), id); err != nil {
		return mapErr(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Export(c *fiber.Ctx) error {
	rows, err := h.uc.Export(c.Context())
	if err != nil {
		log.Printf("[product.export] unexpected error: %v", err)
		return apierr.Internal(c)
	}
	return c.JSON(fiber.Map{"items": rows})
}

func (h *Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var req productuc.UpdateInput
	if err := c.BodyParser(&req); err != nil {
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}

	email := emailPtr(c)
	out, err := h.uc.Update(c.Context(), id, req, email)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(out)
}

func emailPtr(c *fiber.Ctx) *string {
	e := middleware.EmailFromContext(c)
	if e == "" {
		return nil
	}
	return &e
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

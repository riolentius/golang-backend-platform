package customer

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	customeruc "github.com/riolentius/cahaya-gading-backend/internal/usecase/customer"
	"github.com/riolentius/cahaya-gading-backend/pkg/apierr"
)

type Handler struct {
	uc *customeruc.Usecase
}

func New(uc *customeruc.Usecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var in customeruc.CreateInput
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
	out, err := h.uc.List(c.Context(), customeruc.ListQuery{
		Search: c.Query("search"),
		Limit:  c.QueryInt("limit", 50),
		Offset: c.QueryInt("offset", 0),
		Sort:   c.Query("sort", "alphabet"),
	})
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(fiber.Map{"items": out.Items, "total": out.Total})
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	out, err := h.uc.GetByID(c.Context(), id)
	if err != nil {
		return mapErr(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(out)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var in customeruc.UpdateInput
	if err := c.BodyParser(&in); err != nil {
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}

	out, err := h.uc.Update(c.Context(), id, in)
	if err != nil {
		return mapErr(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(out)
}

func mapErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, customeruc.ErrInvalidInput):
		return apierr.BadRequest(c, apierr.CodeInvalidInput, err.Error())
	case errors.Is(err, customeruc.ErrNotFound):
		return apierr.NotFound(c, err.Error())
	case errors.Is(err, customeruc.ErrEmailConflict):
		return apierr.Conflict(c, apierr.CodeConflict, err.Error())
	default:
		return apierr.Internal(c)
	}
}

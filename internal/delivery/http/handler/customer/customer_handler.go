package customer

import (
	"github.com/gofiber/fiber/v2"

	customeruc "github.com/riolentius/cahaya-gading-backend/internal/usecase/customer"
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
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}
	out, err := h.uc.Create(c.Context(), in)
	return writeOne(c, out, err, fiber.StatusCreated)
}

func (h *Handler) List(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	out, err := h.uc.List(c.Context(), customeruc.ListQuery{Limit: limit, Offset: offset})
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(fiber.Map{"items": out})
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	out, err := h.uc.GetByID(c.Context(), id)
	return writeOne(c, out, err, fiber.StatusOK)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var in customeruc.UpdateInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}

	out, err := h.uc.Update(c.Context(), id, in)
	return writeOne(c, out, err, fiber.StatusOK)
}

func writeOne(c *fiber.Ctx, out *customeruc.Customer, err error, okStatus int) error {
	if err != nil {
		return mapErr(c, err)
	}
	return c.Status(okStatus).JSON(out)
}

func mapErr(c *fiber.Ctx, err error) error {
	switch err {
	case customeruc.ErrInvalidInput:
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case customeruc.ErrNotFound:
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case customeruc.ErrEmailConflict:
		return fiber.NewError(fiber.StatusConflict, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "internal error")
	}
}

package customer_address

import (
	"context"

	"github.com/gofiber/fiber/v2"
	addruc "github.com/riolentius/cahaya-gading-backend/internal/usecase/customer_address"
)

type addressUsecase interface {
	Create(ctx context.Context, customerID string, in addruc.CreateInput) (*addruc.CustomerAddress, error)
	ListByCustomer(ctx context.Context, customerID string) ([]addruc.CustomerAddress, error)
	Update(ctx context.Context, id string, in addruc.UpdateInput) (*addruc.CustomerAddress, error)
	Delete(ctx context.Context, id string) error
}

type Handler struct {
	uc addressUsecase
}

func New(uc addressUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Create(c *fiber.Ctx) error {
	customerID := c.Params("id")

	var in addruc.CreateInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "invalid_input", "message": "invalid request body",
		})
	}

	addr, err := h.uc.Create(c.Context(), customerID, in)
	if err != nil {
		if err == addruc.ErrInvalidInput {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"code": "invalid_input", "message": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": "internal_error", "message": "an unexpected error occurred",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(addr)
}

func (h *Handler) List(c *fiber.Ctx) error {
	customerID := c.Params("id")

	items, err := h.uc.ListByCustomer(c.Context(), customerID)
	if err != nil {
		if err == addruc.ErrInvalidInput {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code": "invalid_input", "message": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": "internal_error", "message": "an unexpected error occurred",
		})
	}

	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) Update(c *fiber.Ctx) error {
	addrID := c.Params("addressId")

	var in addruc.UpdateInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code": "invalid_input", "message": "invalid request body",
		})
	}

	addr, err := h.uc.Update(c.Context(), addrID, in)
	if err != nil {
		if err == addruc.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"code": "not_found", "message": "address not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": "internal_error", "message": "an unexpected error occurred",
		})
	}

	return c.JSON(addr)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	addrID := c.Params("addressId")

	if err := h.uc.Delete(c.Context(), addrID); err != nil {
		if err == addruc.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"code": "not_found", "message": "address not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code": "internal_error", "message": "an unexpected error occurred",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

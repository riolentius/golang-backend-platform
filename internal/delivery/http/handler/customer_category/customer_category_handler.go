package customercategoryhandler

import (
	"context"

	"github.com/gofiber/fiber/v2"
	customercategory "github.com/riolentius/cahaya-gading-backend/internal/usecase/customer_category"
)

type categoryUsecase interface {
	List(ctx context.Context) ([]customercategory.CustomerCategory, error)
}

type Handler struct {
	uc categoryUsecase
}

func New(uc categoryUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) List(c *fiber.Ctx) error {
	items, err := h.uc.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "an unexpected error occurred",
		})
	}

	return c.JSON(fiber.Map{"items": items})
}

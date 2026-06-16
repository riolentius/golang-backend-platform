package dashboardhandler

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	dashboarduc "github.com/riolentius/cahaya-gading-backend/internal/usecase/dashboard"
	"github.com/riolentius/cahaya-gading-backend/pkg/apierr"
)

type dashboardUsecase interface {
	GetSummary(ctx context.Context) (*dashboarduc.Summary, error)
	GetTopProducts(ctx context.Context, limit int) ([]dashboarduc.TopProduct, error)
}

type Handler struct {
	uc dashboardUsecase
}

func New(uc dashboardUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) GetSummary(c *fiber.Ctx) error {
	summary, err := h.uc.GetSummary(c.Context())
	if err != nil {
		return apierr.Internal(c)
	}
	return c.JSON(summary)
}

func (h *Handler) GetTopProducts(c *fiber.Ctx) error {
	limit, err := strconv.Atoi(c.Query("limit", "5"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 5
	}

	items, err := h.uc.GetTopProducts(c.Context(), limit)
	if err != nil {
		return apierr.Internal(c)
	}
	return c.JSON(fiber.Map{"items": items})
}

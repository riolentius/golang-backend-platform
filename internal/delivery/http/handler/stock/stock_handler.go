package stockhandler

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/riolentius/cahaya-gading-backend/internal/delivery/middleware"
	stockuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/stock"
	"github.com/riolentius/cahaya-gading-backend/pkg/apierr"
)

type stockUsecase interface {
	StockIn(ctx context.Context, productID string, in stockuc.StockInInput, createdByEmail *string) (*stockuc.StockInResult, error)
	ListByProduct(ctx context.Context, productID string, filter stockuc.ListFilter) ([]stockuc.Movement, error)
}

type Handler struct {
	uc stockUsecase
}

func New(uc stockUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) StockIn(c *fiber.Ctx) error {
	productID := c.Params("id")

	var req stockuc.StockInInput
	if err := c.BodyParser(&req); err != nil {
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}

	email := middleware.EmailFromContext(c)
	var emailPtr *string
	if email != "" {
		emailPtr = &email
	}

	out, err := h.uc.StockIn(c.Context(), productID, req, emailPtr)
	if err != nil {
		return mapErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *Handler) ListByProduct(c *fiber.Ctx) error {
	productID := c.Params("id")

	limit, err := strconv.Atoi(c.Query("limit", "50"))
	if err != nil || limit <= 0 {
		limit = 50
	}
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	var direction *string
	if d := c.Query("direction"); d != "" {
		if d != "in" && d != "out" {
			return apierr.BadRequest(c, apierr.CodeInvalidInput, `direction must be "in" or "out"`)
		}
		direction = &d
	}

	var from, to *time.Time
	if v := c.Query("from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return apierr.BadRequest(c, apierr.CodeInvalidInput, `from must be a date in YYYY-MM-DD format`)
		}
		from = &t
	}
	if v := c.Query("to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return apierr.BadRequest(c, apierr.CodeInvalidInput, `to must be a date in YYYY-MM-DD format`)
		}
		to = &t
	}

	filter := stockuc.ListFilter{
		Direction: direction,
		From:      from,
		To:        to,
		Limit:     limit,
		Offset:    offset,
	}

	items, err := h.uc.ListByProduct(c.Context(), productID, filter)
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func mapErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, stockuc.ErrInvalidInput):
		return apierr.BadRequest(c, apierr.CodeInvalidInput, err.Error())
	case errors.Is(err, stockuc.ErrProductMissing):
		return apierr.NotFound(c, err.Error())
	case errors.Is(err, stockuc.ErrInvalidPackSize):
		return apierr.BadRequest(c, apierr.CodeInvalidInput, err.Error())
	default:
		log.Printf("[stock] unexpected error: %v", err)
		return apierr.Internal(c)
	}
}

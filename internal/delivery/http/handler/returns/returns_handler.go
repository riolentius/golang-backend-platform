package returnhandler

import (
	"context"
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/riolentius/cahaya-gading-backend/internal/delivery/middleware"
	returnsuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/returns"
	"github.com/riolentius/cahaya-gading-backend/pkg/apierr"
)

type returnsUsecase interface {
	Create(ctx context.Context, in returnsuc.CreateInput, createdByEmail *string) (*returnsuc.CreateResult, error)
	ListByTransaction(ctx context.Context, transactionID string) ([]returnsuc.Return, error)
	ListReturnableItems(ctx context.Context, transactionID string) ([]returnsuc.ReturnableItem, error)
}

type Handler struct {
	uc returnsUsecase
}

func New(uc returnsUsecase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) CreateForTransaction(c *fiber.Ctx) error {
	var req returnsuc.CreateInput
	if err := c.BodyParser(&req); err != nil {
		return apierr.BadRequest(c, apierr.CodeInvalidInput, "request body is not valid JSON")
	}
	req.TransactionID = c.Params("id")

	out, err := h.uc.Create(c.Context(), req, emailPtr(c))
	if err != nil {
		return mapErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *Handler) ListForTransaction(c *fiber.Ctx) error {
	items, err := h.uc.ListByTransaction(c.Context(), c.Params("id"))
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) ListReturnableItems(c *fiber.Ctx) error {
	items, err := h.uc.ListReturnableItems(c.Context(), c.Params("id"))
	if err != nil {
		return mapErr(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
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
	case errors.Is(err, returnsuc.ErrInvalidInput),
		errors.Is(err, returnsuc.ErrDuplicateItem),
		errors.Is(err, returnsuc.ErrItemNotInTransaction):
		return apierr.BadRequest(c, apierr.CodeInvalidInput, err.Error())
	case errors.Is(err, returnsuc.ErrTransactionMissing):
		return apierr.NotFound(c, err.Error())
	case errors.Is(err, returnsuc.ErrNotFulfilled):
		return apierr.Conflict(c, apierr.CodeInvalidTransition, err.Error())
	case errors.Is(err, returnsuc.ErrQtyExceedsReturnable):
		return apierr.Conflict(c, apierr.CodeConflict, err.Error())
	default:
		log.Printf("[returns] unexpected error: %v", err)
		return apierr.Internal(c)
	}
}

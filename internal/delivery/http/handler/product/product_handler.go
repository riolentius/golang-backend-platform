package product

import (
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	productuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product"
)

type Handler struct {
	uc *productuc.Usecase
}

func New(uc *productuc.Usecase) *Handler {
	return &Handler{uc: uc}
}

func isDev() bool {
	// set APP_ENV=dev in your local env
	return os.Getenv("APP_ENV") == "dev"
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req productuc.CreateInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	out, err := h.uc.Create(c.Context(), req)
	if err != nil {
		// 1) known validation error
		if errors.Is(err, productuc.ErrInvalidInput) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		// 2) log real error for debugging (server-side)
		log.Printf("[product.create] failed: %v", err)

		// 3) return safe message to client, but show details in dev
		if isDev() {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *Handler) List(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	out, err := h.uc.List(c.Context(), limit, offset)
	if err != nil {
		log.Printf("[product.list] failed: %v", err)
		if isDev() {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(fiber.Map{"items": out})
}

func (h *Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req productuc.UpdateInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	out, err := h.uc.Update(c.Context(), id, req)
	if err != nil {
		if errors.Is(err, productuc.ErrInvalidInput) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		log.Printf("[product.update] failed: %v", err)
		if isDev() {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(out)
}

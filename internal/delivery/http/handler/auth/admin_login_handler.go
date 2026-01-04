package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	authuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/auth"
)

type AdminLoginHandler struct {
	uc *authuc.AdminLoginUsecase
}

func NewAdminLoginHandler(uc *authuc.AdminLoginUsecase) *AdminLoginHandler {
	return &AdminLoginHandler{uc: uc}
}

func (h *AdminLoginHandler) Handle(c *fiber.Ctx) error {
	var req authuc.LoginInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	out, err := h.uc.Execute(c.Context(), req)
	if err != nil {
		if errors.Is(err, authuc.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
		}
		if errors.Is(err, authuc.ErrAdminInactive) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "admin inactive"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(out)
}

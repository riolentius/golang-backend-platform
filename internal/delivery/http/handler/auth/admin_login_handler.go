package auth

import (
	"github.com/gofiber/fiber/v2"

	authuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/auth"
)

type AdminLoginHandler struct {
	uc *authuc.AdminLoginUsecase
}

func NewAdminLoginHandler(uc *authuc.AdminLoginUsecase) *AdminLoginHandler {
	return &AdminLoginHandler{uc: uc}
}

type adminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AdminLoginHandler) Handle(c *fiber.Ctx) error {
	var req adminLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json body")
	}
	if req.Email == "" || req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}

	res, err := h.uc.Execute(c.Context(), req.Email, req.Password)
	if err == authuc.ErrInvalidCredentials {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}
	if err == authuc.ErrInactiveAdmin {
		return fiber.NewError(fiber.StatusForbidden, "admin inactive")
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal error")
	}

	return c.JSON(res)
}

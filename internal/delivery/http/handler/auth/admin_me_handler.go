package auth

import "github.com/gofiber/fiber/v2"

type AdminMeHandler struct{}

func NewAdminMeHandler() *AdminMeHandler {
	return &AdminMeHandler{}
}

func (h *AdminMeHandler) Handle(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"id":    c.Locals("admin_id"),
		"email": c.Locals("admin_email"),
	})
}

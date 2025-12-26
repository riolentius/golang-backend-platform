package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler"
)

func RegisterRoutes(app *fiber.App) {
	app.Get("/health", handler.Health)
}

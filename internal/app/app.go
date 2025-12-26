package app

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	httpdelivery "github.com/riolentius/cahaya-gading-backend/internal/delivery/http"
)

type App struct {
	f *fiber.App
}

func New() *App {
	f := fiber.New(fiber.Config{
		AppName: "cahaya-gading-backend",
	})

	f.Use(recover.New())
	f.Use(logger.New())

	httpdelivery.RegisterRoutes(f)

	return &App{f: f}
}

func (a *App) Run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return a.f.Listen(":" + port)
}

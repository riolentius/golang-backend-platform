package app

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/riolentius/cahaya-gading-backend/internal/config"
	"github.com/riolentius/cahaya-gading-backend/internal/db"
	httpdelivery "github.com/riolentius/cahaya-gading-backend/internal/delivery/http"
)

type App struct {
	f *fiber.App
}

func New() *App {
	cfg := config.Load()

	pool, err := db.NewPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}

	f := fiber.New(fiber.Config{
		AppName: "cahaya-gading-backend",
	})

	f.Use(recover.New())
	f.Use(logger.New())

	httpdelivery.RegisterRoutes(f, cfg, pool)

	return &App{f: f}
}

func (a *App) Run() error {
	return a.f.Listen(":" + config.Load().Port)
}

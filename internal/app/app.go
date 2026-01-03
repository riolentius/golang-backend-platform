package app

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"github.com/riolentius/cahaya-gading-backend/internal/config"
	"github.com/riolentius/cahaya-gading-backend/internal/db"
	httpdelivery "github.com/riolentius/cahaya-gading-backend/internal/delivery/http"
)

type App struct {
	f   *fiber.App
	cfg config.Config
}

func New() *App {
	_ = godotenv.Load()

	cfg := config.Load()

	pool, err := db.NewPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}

	f := fiber.New(fiber.Config{
		AppName: cfg.AppName,
	})

	f.Use(recover.New())
	f.Use(logger.New())

	// updated signature
	httpdelivery.RegisterRoutes(f, cfg, pool)

	return &App{f: f, cfg: cfg}
}

func (a *App) Run() error {
	return a.f.Listen(":" + a.cfg.Port)
}

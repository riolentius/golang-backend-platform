package http

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riolentius/cahaya-gading-backend/internal/config"
	authhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/auth"
	producthandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/product"
	pricehandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/product_price"
	trxhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/transaction"
	"github.com/riolentius/cahaya-gading-backend/internal/delivery/middleware"
	"github.com/riolentius/cahaya-gading-backend/internal/repository/postgres"
	authuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/auth"
	productuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product"
	priceuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product_price"
	trxuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/transaction"
)

func RegisterRoutes(app *fiber.App, cfg config.Config, db *pgxpool.Pool) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	api := app.Group("/api")

	// Auth wiring
	adminRepo := postgres.NewAdminRepo(db)
	adminFinder := &adminFinderAdapter{repo: adminRepo}
	loginUC := authuc.NewAdminLoginUsecase(adminFinder, cfg.JWTSecret, cfg.JWTExpiresMinutes)
	loginHandler := authhandler.NewAdminLoginHandler(loginUC)

	// Public route
	api.Post("/admin/login", loginHandler.Handle)

	// Protected admin group (MUST be defined before use)
	admin := api.Group("/admin", middleware.RequireAdminJWT(middleware.JWTConfig{
		Secret: cfg.JWTSecret,
	}))

	admin.Get("/me", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"ok":     true,
			"claims": c.Locals("claims"),
		})
	})

	// Products wiring
	productRepo := postgres.NewProductRepo(db)
	productStore := postgres.NewProductStoreAdapter(productRepo)
	productUC := productuc.New(productStore)
	productH := producthandler.New(productUC)

	// Prices wiring
	priceRepo := postgres.NewProductPriceRepo(db)
	priceStore := postgres.NewProductPriceStoreAdapter(priceRepo)
	priceUC := priceuc.New(priceStore)
	priceH := pricehandler.New(priceUC)

	// Transactions wiring
	trxRepo := postgres.NewTransactionRepo(db)
	trxStore := postgres.NewTransactionStoreAdapter(trxRepo)
	trxUC := trxuc.New(trxStore)
	trxH := trxhandler.New(trxUC)

	// Transaction routes
	admin.Post("/transactions", trxH.Create)
	admin.Get("/transactions", trxH.List)
	admin.Get("/transactions/:id", trxH.Get)
	admin.Patch("/transactions/:id/status", trxH.UpdateStatus)

	// Product routes
	admin.Post("/products", productH.Create)
	admin.Get("/products", productH.List)
	admin.Patch("/products/:id", productH.Update)

	// Product price routes
	admin.Post("/products/:id/prices", priceH.CreateForProduct)
	admin.Get("/products/:id/prices", priceH.ListForProduct)
	admin.Patch("/prices/:id", priceH.Update)
}

type adminFinderAdapter struct {
	repo *postgres.AdminRepo
}

func (a *adminFinderAdapter) FindByEmail(ctx context.Context, email string) (*authuc.Admin, error) {
	r, err := a.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return &authuc.Admin{
		ID:           r.ID,
		Email:        r.Email,
		PasswordHash: r.PasswordHash,
		IsActive:     r.IsActive,
	}, nil
}

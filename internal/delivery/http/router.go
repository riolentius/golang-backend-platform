package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riolentius/cahaya-gading-backend/internal/config"
	authhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/auth"
	customerhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/customer"
	payhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/payment"
	producthandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/product"
	pricehandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/product_price"
	trxhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/transaction"
	"github.com/riolentius/cahaya-gading-backend/internal/delivery/middleware"
	adminpg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/admin"
	postgres "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/admin"
	customerpg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/customer"
	paypg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/payment"
	productpg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/product"
	pricepg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/product_price"
	trxpg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/transaction"
	authuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/auth"
	customeruc "github.com/riolentius/cahaya-gading-backend/internal/usecase/customer"
	payuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/payment"
	productuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product"
	priceuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product_price"
	txuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/transaction"
)

func RegisterRoutes(app *fiber.App, cfg config.Config, db *pgxpool.Pool) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	api := app.Group("/api")

	// Auth wiring
	adminRepo := adminpg.NewAdminRepo(db)
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
	productRepo := productpg.NewProductRepo(db)
	productStore := productpg.NewProductStoreAdapter(productRepo)
	productUC := productuc.New(productStore)
	productH := producthandler.New(productUC)

	// Prices wiring
	priceRepo := pricepg.NewProductPriceRepo(db)
	priceStore := pricepg.NewProductPriceStoreAdapter(priceRepo)
	priceUC := priceuc.New(priceStore)
	priceH := pricehandler.New(priceUC)

	// Transactions wiring
	trxRepo := trxpg.NewTransactionRepo(db)
	trxStore := trxpg.NewTransactionStoreAdapter(trxRepo)
	trxUC := txuc.New(trxStore)
	trxH := trxhandler.New(trxUC)

	// Customer wiring
	customerRepo := customerpg.NewCustomerRepo(db)
	customerStore := customerpg.NewCustomerStoreAdapter(customerRepo)
	customerUC := customeruc.New(customerStore)
	customerH := customerhandler.New(customerUC)

	// Payments wiring
	paymentRepo := paypg.NewPaymentRepo(db)
	paymentStore := paypg.NewPaymentStoreAdapter(paymentRepo)
	paymentUC := payuc.New(paymentStore)
	paymentH := payhandler.New(paymentUC)

	// Endpoints
	admin.Get("/transactions/:id/view", trxH.GetViewByID)
	admin.Post("/transactions/:id/payments", paymentH.CreateForTransaction)
	admin.Get("/transactions/:id/payments", paymentH.ListForTransaction)
	admin.Post("/transactions/:id/fulfill", trxH.Fulfill)

	// Customer routes
	admin.Post("/customers", customerH.Create)
	admin.Get("/customers", customerH.List)
	admin.Get("/customers/:id", customerH.GetByID)
	admin.Patch("/customers/:id", customerH.Update)

	// Transaction routes
	admin.Post("/transactions", trxH.Create)
	admin.Get("/transactions", trxH.List)
	admin.Get("/transactions/:id", trxH.GetByID)
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

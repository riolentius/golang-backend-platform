package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riolentius/cahaya-gading-backend/internal/config"
	authhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/auth"
	customerhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/customer"
	addrhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/customer_address"
	categoryhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/customer_category"
	dashboardhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/dashboard"
	payhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/payment"
	producthandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/product"
	prodcathandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/product_category"
	pricehandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/product_price"
	trxhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/transaction"
	"github.com/riolentius/cahaya-gading-backend/internal/delivery/middleware"
	adminpg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/admin"
	customerpg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/customer"
	addrpg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/customer_address"
	categorypg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/customer_category"
	dashpg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/dashboard"
	paypg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/payment"
	productpg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/product"
	prodcatpg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/product_category"
	pricepg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/product_price"
	trxpg "github.com/riolentius/cahaya-gading-backend/internal/repository/postgres/transaction"
	authuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/auth"
	customeruc "github.com/riolentius/cahaya-gading-backend/internal/usecase/customer"
	addruc "github.com/riolentius/cahaya-gading-backend/internal/usecase/customer_address"
	categoryuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/customer_category"
	dashboarduc "github.com/riolentius/cahaya-gading-backend/internal/usecase/dashboard"
	payuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/payment"
	productuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product"
	prodcatuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product_category"
	priceuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/product_price"
	txuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/transaction"
)

func RegisterRoutes(app *fiber.App, cfg config.Config, db *pgxpool.Pool) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	api := app.Group("/api")

	// ── Auth ─────────────────────────────────────────────────
	adminRepo := adminpg.NewAdminRepo(db)
	adminFinder := &adminFinderAdapter{repo: adminRepo}
	loginUC := authuc.NewAdminLoginUsecase(adminFinder, cfg.JWTSecret, cfg.JWTExpiresMinutes)
	loginH := authhandler.NewAdminLoginHandler(loginUC)

	api.Post("/admin/login", loginH.Handle)

	admin := api.Group("/admin", middleware.RequireAdminJWT(middleware.JWTConfig{
		Secret: cfg.JWTSecret,
	}))

	admin.Get("/me", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true, "claims": c.Locals("claims")})
	})

	// ── Dashboard ─────────────────────────────────────────────
	dashboardRepo := dashpg.NewDashboardRepo(db)
	dashboardStore := dashpg.NewDashboardStoreAdapter(dashboardRepo)
	dashboardUC := dashboarduc.New(dashboardStore)
	dashboardH := dashboardhandler.New(dashboardUC)

	admin.Get("/dashboard", dashboardH.GetSummary)
	admin.Get("/dashboard/top-products", dashboardH.GetTopProducts)

	// ── Customer Categories ───────────────────────────────────
	categoryRepo := categorypg.NewCustomerCategoryRepo(db)
	categoryStore := categorypg.NewCustomerCategoryStoreAdapter(categoryRepo)
	categoryUC := categoryuc.New(categoryStore)
	categoryH := categoryhandler.New(categoryUC)

	admin.Get("/customer-categories", categoryH.List)

	// ── Customers ─────────────────────────────────────────────
	customerRepo := customerpg.NewCustomerRepo(db)
	customerStore := customerpg.NewCustomerStoreAdapter(customerRepo)
	customerUC := customeruc.New(customerStore)
	customerH := customerhandler.New(customerUC)

	admin.Post("/customers", customerH.Create)
	admin.Get("/customers", customerH.List)
	admin.Get("/customers/:id", customerH.GetByID)
	admin.Patch("/customers/:id", customerH.Update)

	// ── Customer Addresses ────────────────────────────────────
	addrRepo := addrpg.NewCustomerAddressRepo(db)
	addrStore := addrpg.NewCustomerAddressStoreAdapter(addrRepo)
	addrUC := addruc.New(addrStore)
	addrH := addrhandler.New(addrUC)

	admin.Get("/customers/:id/addresses", addrH.List)
	admin.Post("/customers/:id/addresses", addrH.Create)
	admin.Patch("/customers/:id/addresses/:addressId", addrH.Update)
	admin.Delete("/customers/:id/addresses/:addressId", addrH.Delete)

	// ── Products ──────────────────────────────────────────────
	productRepo := productpg.NewProductRepo(db)
	productStore := productpg.NewProductStoreAdapter(productRepo)
	productUC := productuc.New(productStore)
	productH := producthandler.New(productUC)

	admin.Post("/products", productH.Create)
	admin.Get("/products", productH.List)
	admin.Patch("/products/:id", productH.Update)

	// ── Product Categories ────────────────────────────────────
	prodCatRepo := prodcatpg.NewProductCategoryRepo(db)
	prodCatStore := prodcatpg.NewProductCategoryStoreAdapter(prodCatRepo)
	prodCatUC := prodcatuc.New(prodCatStore)
	prodCatH := prodcathandler.New(prodCatUC)

	admin.Get("/product-categories", prodCatH.List)
	admin.Post("/product-categories", prodCatH.Create)
	admin.Patch("/product-categories/:id", prodCatH.Update)

	// ── Product Prices ────────────────────────────────────────
	priceRepo := pricepg.NewProductPriceRepo(db)
	priceStore := pricepg.NewProductPriceStoreAdapter(priceRepo)
	priceUC := priceuc.New(priceStore)
	priceH := pricehandler.New(priceUC)

	admin.Post("/products/:id/prices", priceH.CreateForProduct)
	admin.Get("/products/:id/prices", priceH.ListForProduct)
	admin.Patch("/prices/:id", priceH.Update)

	// ── Transactions ──────────────────────────────────────────
	trxRepo := trxpg.NewTransactionRepo(db)
	trxStore := trxpg.NewTransactionStoreAdapter(trxRepo, db)
	trxUC := txuc.New(trxStore)
	trxH := trxhandler.New(trxUC)

	admin.Post("/transactions", trxH.Create)
	admin.Get("/transactions", trxH.List)
	admin.Get("/transactions/:id", trxH.GetByID)
	admin.Get("/transactions/:id/view", trxH.GetViewByID)
	admin.Post("/transactions/:id/fulfill", trxH.Fulfill)
	admin.Patch("/transactions/:id/status", trxH.UpdateStatus)

	// ── Payments ─────────────────────────────────────────────
	paymentRepo := paypg.NewPaymentRepo(db)
	paymentStore := paypg.NewPaymentStoreAdapter(paymentRepo)
	paymentUC := payuc.New(paymentStore)
	paymentH := payhandler.New(paymentUC)

	admin.Post("/transactions/:id/payments", paymentH.CreateForTransaction)
	admin.Get("/transactions/:id/payments", paymentH.ListForTransaction)
}

type adminFinderAdapter struct {
	repo *adminpg.AdminRepo
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
		Role:         r.Role,
	}, nil
}

package http

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/riolentius/cahaya-gading-backend/internal/config"
	authhandler "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/handler/auth"
	httpmw "github.com/riolentius/cahaya-gading-backend/internal/delivery/http/middleware"
	"github.com/riolentius/cahaya-gading-backend/internal/repository/postgres"
	authuc "github.com/riolentius/cahaya-gading-backend/internal/usecase/auth"
)

func RegisterRoutes(app *fiber.App, cfg config.Config, db *pgxpool.Pool) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	api := app.Group("/api")

	// --- Auth wiring (public login) ---
	adminRepo := postgres.NewAdminRepo(db)
	adminFinder := &adminFinderAdapter{repo: adminRepo}

	loginUC := authuc.NewAdminLoginUsecase(adminFinder, cfg.JWTSecret, cfg.JWTExpiresMinutes)
	loginHandler := authhandler.NewAdminLoginHandler(loginUC)

	api.Post("/admin/login", loginHandler.Handle)

	// --- JWT Middleware (protected admin routes) ---
	jwtMW := httpmw.NewJWTMiddleware(cfg.JWTSecret)

	admin := api.Group("/admin", jwtMW.Protect())

	meHandler := authhandler.NewAdminMeHandler()
	admin.Get("/me", meHandler.Handle)
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

package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type JWTMiddleware struct {
	secret []byte
}

func NewJWTMiddleware(secret string) *JWTMiddleware {
	return &JWTMiddleware{secret: []byte(secret)}
}

func (m *JWTMiddleware) Protect() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing authorization header")
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid authorization header")
		}

		tokenStr := parts[1]

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid token signing method")
			}
			return m.secret, nil
		})

		if err != nil || token == nil || !token.Valid {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid token claims")
		}

		// Optional token type check (we'll set typ=admin in login usecase)
		if typ, _ := claims["typ"].(string); typ != "" && typ != "admin" {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid token type")
		}

		if sub, _ := claims["sub"].(string); sub != "" {
			c.Locals("admin_id", sub)
		}
		if email, _ := claims["email"].(string); email != "" {
			c.Locals("admin_email", email)
		}

		return c.Next()
	}
}

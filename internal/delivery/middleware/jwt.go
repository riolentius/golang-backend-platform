package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type JWTConfig struct {
	Secret string
}

func RequireAdminJWT(cfg JWTConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}

		raw := strings.TrimPrefix(auth, "Bearer ")

		token, err := jwt.Parse(raw, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(cfg.Secret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token claims"})
		}

		role, _ := claims["role"].(string)
		if role != "admin" && role != "superadmin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}

		c.Locals("claims", claims)
		return c.Next()
	}
}

func RequireSuperAdmin(c *fiber.Ctx) error {
	claims, ok := c.Locals("claims").(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	role, _ := claims["role"].(string)
	if role != "superadmin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	return c.Next()
}

func RoleFromContext(c *fiber.Ctx) string {
	claims, ok := c.Locals("claims").(jwt.MapClaims)
	if !ok {
		return ""
	}
	role, _ := claims["role"].(string)
	return role
}

func EmailFromContext(c *fiber.Ctx) string {
	claims, ok := c.Locals("claims").(jwt.MapClaims)
	if !ok {
		return ""
	}
	email, _ := claims["email"].(string)
	return email
}

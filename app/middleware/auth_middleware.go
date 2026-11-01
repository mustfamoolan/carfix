package middleware

import (
	"strings"

	"carfix/config"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func getJWTSecret() []byte {
	secret := config.Global.App.Key
	if secret == "" {
		secret = "carfix_super_secret_jwt_key"
	}
	return []byte(strings.TrimPrefix(secret, "base64:"))
}

func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"message": "Invalid or expired token"})
		}

		c.Locals("user", token)

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if rawID, exists := claims["user_id"]; exists {
				switch v := rawID.(type) {
				case float64:
					c.Locals("user_id", uint(v))
				case int:
					c.Locals("user_id", uint(v))
				case uint:
					c.Locals("user_id", v)
				}
			}
			if role, exists := claims["role"]; exists {
				c.Locals("role", role)
			}
		}

		return c.Next()
	}
}

func AdminOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userToken, ok := c.Locals("user").(*jwt.Token)
		if !ok || userToken == nil {
			return c.Status(401).JSON(fiber.Map{"message": "Unauthorized"})
		}

		claims, ok := userToken.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(401).JSON(fiber.Map{"message": "Invalid token claims"})
		}

		role, _ := claims["role"].(string)
		if role != "admin" {
			return c.Status(403).JSON(fiber.Map{"message": "Forbidden: Admin access required"})
		}

		return c.Next()
	}
}

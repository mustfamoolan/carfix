package middleware

import (
	"carfix/app/services"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// QuotaCheckMiddleware verifies the authenticated user has sufficient remaining quota
func QuotaCheckMiddleware() fiber.Handler {
	quotaService := services.NewQuotaManagerService()

	return func(c *fiber.Ctx) error {
		userToken, ok := c.Locals("user").(*jwt.Token)
		if !ok || userToken == nil {
			return c.Status(401).JSON(fiber.Map{"message": "غير مصرح"})
		}

		claims, ok := userToken.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(401).JSON(fiber.Map{"message": "رمز الدخول غير صالح"})
		}

		rawID, exists := claims["user_id"]
		if !exists {
			return c.Status(401).JSON(fiber.Map{"message": "معرف المستخدم مفقود"})
		}

		var userID uint
		switch v := rawID.(type) {
		case float64:
			userID = uint(v)
		case int:
			userID = uint(v)
		case uint:
			userID = v
		default:
			return c.Status(401).JSON(fiber.Map{"message": "معرف المستخدم غير صالح"})
		}

		quota, err := quotaService.CheckAndPrepareQuota(userID)
		if err != nil {
			return c.Status(403).JSON(fiber.Map{
				"message": err.Error(),
				"quota":   quota,
			})
		}

		c.Locals("quota", quota)
		c.Locals("user_id", userID)
		return c.Next()
	}
}

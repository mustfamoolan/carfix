package controllers

import (
	"strings"
	"time"

	"carfix/app/models"
	"carfix/bootstrap"
	"carfix/config"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct{}

func getJWTSecret() []byte {
	secret := config.Global.App.Key
	if secret == "" {
		secret = "carfix_super_secret_jwt_key"
	}
	return []byte(strings.TrimPrefix(secret, "base64:"))
}

func (a *AuthController) Login(c *fiber.Ctx) error {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&input); err != nil || strings.TrimSpace(input.Username) == "" || strings.TrimSpace(input.Password) == "" {
		return c.Status(400).JSON(fiber.Map{"message": "يرجى إدخال اسم المستخدم وكلمة المرور"})
	}

	var user models.User
	if err := bootstrap.DB.Preload("Quota").Where("username = ?", strings.TrimSpace(input.Username)).First(&user).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "بيانات الدخول غير صحيحة"})
	}

	// Check if account is active
	if !user.IsActive {
		return c.Status(403).JSON(fiber.Map{
			"message": "حسابك متوقف حالياً، يرجى التواصل مع الإدارة",
		})
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "بيانات الدخول غير صحيحة"})
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString(getJWTSecret())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "فشل إنشاء رمز الدخول"})
	}

	return c.JSON(fiber.Map{
		"token": tokenString,
		"user":  user,
	})
}

func (a *AuthController) Me(c *fiber.Ctx) error {
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
		return c.Status(401).JSON(fiber.Map{"message": "رمز الدخول لا يحتوي على معرف المستخدم"})
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

	var dbUser models.User
	if err := bootstrap.DB.Preload("Quota").First(&dbUser, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "المستخدم غير موجود"})
	}

	if !dbUser.IsActive {
		return c.Status(403).JSON(fiber.Map{
			"message": "حسابك متوقف حالياً، يرجى التواصل مع الإدارة",
		})
	}

	return c.JSON(dbUser)
}

func (a *AuthController) Logout(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "تم تسجيل الخروج بنجاح",
	})
}

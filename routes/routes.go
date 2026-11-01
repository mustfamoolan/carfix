package routes

import (
	"github.com/gofiber/fiber/v2"
)

func Register(app *fiber.App) {
	// Register API routes first so they take precedence over web wildcards
	RegisterAPIRoutes(app)
	// Register Web & static fallback routes
	RegisterWebRoutes(app)
}

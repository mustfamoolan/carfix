package routes

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func RegisterWebRoutes(app *fiber.App) {
	// Serve static files from Vite build
	app.Static("/", "./public")

	// UI Showcase Route
	app.Get("/ui-showcase", func(c *fiber.Ctx) error {
		return c.SendFile("./public/index.html")
	})

	// Fallback to React for any other non-API web routes
	app.Get("/*", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api") {
			return c.Next()
		}
		return c.SendFile("./public/index.html")
	})
}

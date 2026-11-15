package routes

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func RegisterWebRoutes(app *fiber.App) {
	// Never cache index.html so all User and Admin browsers always load the latest UI bundle immediately
	sendFreshIndex := func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.SendFile("./public/index.html")
	}

	app.Get("/", sendFreshIndex)
	app.Get("/index.html", sendFreshIndex)

	// Serve hashed static assets from Vite build
	app.Static("/", "./public", fiber.Static{
		Index: "",
	})

	// UI Showcase Route
	app.Get("/ui-showcase", sendFreshIndex)

	// Fallback to React for any other non-API web routes
	app.Get("/*", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api") {
			return c.Next()
		}
		return sendFreshIndex(c)
	})
}

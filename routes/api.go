package routes

import (
	"carfix/app/controllers"
	"carfix/app/middleware"
	"carfix/app/services"
	"github.com/gofiber/fiber/v2"
)

func RegisterAPIRoutes(app *fiber.App) {
	api := app.Group("/api")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "app": "CarFix"})
	})

	// 1. Authentication Routes
	auth := api.Group("/auth")
	authController := &controllers.AuthController{}

	auth.Post("/login", authController.Login)
	auth.Post("/logout", authController.Logout)
	auth.Get("/me", middleware.AuthMiddleware(), authController.Me)

	// API v1 grouping support
	v1 := api.Group("/v1")
	v1.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "version": "v1"})
	})

	v1Auth := v1.Group("/auth")
	v1Auth.Post("/login", authController.Login)
	v1Auth.Post("/logout", authController.Logout)
	v1Auth.Get("/me", middleware.AuthMiddleware(), authController.Me)

	// 2. Search & Report Routes (Protected by Auth + QuotaCheck)
	vinSearchController := controllers.NewVinSearchController()

	api.Post("/search", middleware.AuthMiddleware(), middleware.QuotaCheckMiddleware(), vinSearchController.Search)
	v1.Post("/search", middleware.AuthMiddleware(), middleware.QuotaCheckMiddleware(), vinSearchController.Search)
	api.Get("/search/queue-status/:vin", middleware.AuthMiddleware(), vinSearchController.GetQueueStatus)
	v1.Get("/search/queue-status/:vin", middleware.AuthMiddleware(), vinSearchController.GetQueueStatus)

	api.Get("/reports/:vin", middleware.AuthMiddleware(), vinSearchController.GetCachedReport)
	api.Get("/reports/:vin/html", middleware.AuthMiddleware(), vinSearchController.GetReportHtml)
	api.Get("/reports/:vin/pdf", vinSearchController.GetReportPdf)
	api.Get("/user/searches", middleware.AuthMiddleware(), vinSearchController.GetUserSearchHistory)

	// 3. Admin & Command Center Protected Routes
	admin := api.Group("/admin", middleware.AuthMiddleware(), middleware.AdminOnly())
	adminUserController := &controllers.AdminUserController{}

	admin.Get("/check", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "message": "Admin authorized"})
	})
	admin.Get("/users", adminUserController.ListUsers)
	admin.Post("/users", adminUserController.CreateUser)
	admin.Put("/users/:id", adminUserController.UpdateUser)
	admin.Put("/users/:id/quota", adminUserController.UpdateUserQuota)
	admin.Get("/search-logs", adminUserController.ListSearchLogs)

	// Canada Worker Bridge & Guard Routes (Command Center Controls)
	workerController := controllers.NewWorkerController()
	admin.Get("/worker-status", workerController.GetWorkerStatus)
	admin.Post("/worker-inspect", workerController.RequestInspect)
	admin.Post("/worker-stop", workerController.EmergencyStop)
	admin.Post("/worker-reset", workerController.ResetCircuitBreaker)
	admin.Post("/worker-mode", workerController.SetWorkerMode)

	// Also allow authenticated dashboard to read worker status
	api.Get("/worker-status", middleware.AuthMiddleware(), workerController.GetWorkerStatus)

	// Remote Worker Secured Endpoints (Header: X-Worker-Secret)
	worker := api.Group("/worker")
	worker.Get("/poll", workerController.Poll)
	worker.Post("/telemetry", workerController.Telemetry)
	worker.Post("/progress", workerController.Progress)
	worker.Post("/complete", workerController.Complete)

	// 4. Public Tools & Services inspection endpoints
	tools := api.Group("/tools")
	vinValidator := services.NewVinValidatorService()
	translator := services.NewCarfaxTranslatorService()

	tools.Post("/vin-validate", func(c *fiber.Ctx) error {
		var body struct {
			VIN string `json:"vin"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "بيانات غير صالحة"})
		}
		cleanVin, isValid, errMsg := vinValidator.ValidateAndClean(body.VIN)

		var breakdown fiber.Map
		if len(cleanVin) == 17 {
			breakdown = fiber.Map{
				"wmi":             cleanVin[0:3],
				"vds":             cleanVin[3:8],
				"check_digit":     string(cleanVin[8]),
				"model_year_code": string(cleanVin[9]),
				"plant_code":      string(cleanVin[10]),
				"vis":             cleanVin[11:17],
			}
		}

		return c.JSON(fiber.Map{
			"vin":       cleanVin,
			"is_valid":  isValid,
			"error_msg": errMsg,
			"breakdown": breakdown,
		})
	})

	tools.Post("/translate-test", func(c *fiber.Ctx) error {
		var body struct {
			VIN  string `json:"vin"`
			HTML string `json:"html"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "بيانات غير صالحة"})
		}
		translatedHtml, vehicleTitle, summaryJson := translator.TranslateAndExtract(body.HTML, body.VIN)
		return c.JSON(fiber.Map{
			"vehicle_title":      vehicleTitle,
			"summary_json":       summaryJson,
			"translated_html_ar": translatedHtml,
			"translated_len":     len(translatedHtml),
		})
	})
}

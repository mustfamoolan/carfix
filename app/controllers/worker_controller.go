package controllers

import (
	"encoding/json"
	"strings"
	"time"

	"carfix/app/models"
	"carfix/app/services"
	"carfix/bootstrap"
	"github.com/gofiber/fiber/v2"
)

type WorkerController struct {
	guard      *services.WorkerGuardService
	translator *services.CarfaxTranslatorService
}

func NewWorkerController() *WorkerController {
	return &WorkerController{
		guard:      services.NewWorkerGuardService(),
		translator: services.NewCarfaxTranslatorService(),
	}
}

// verifySecret validates the secret key from the worker header
func (ctrl *WorkerController) verifySecret(c *fiber.Ctx) bool {
	secret := c.Get("X-Worker-Secret")
	return ctrl.guard.ValidateSecret(secret)
}

// Poll delivers the next queued VIN or an inspect request to the Canada node
func (ctrl *WorkerController) Poll(c *fiber.Ctx) error {
	if !ctrl.verifySecret(c) {
		return c.Status(401).JSON(fiber.Map{
			"error": "غير مصرح - مفتاح عامل كندا غير صحيح أو مفقود",
		})
	}

	inspectRequested := ctrl.guard.ConsumeInspectRequest()

	vin, cooldownRemaining, err := ctrl.guard.PollNextTask()
	if err != nil {
		if err == services.ErrCooldownActive {
			return c.JSON(fiber.Map{
				"has_task":           false,
				"inspect_requested":  inspectRequested,
				"cooldown_remaining": cooldownRemaining,
				"message":            "فترة تهدئة لحماية الحساب من الحظر",
			})
		}
		if err == services.ErrQueueEmpty {
			return c.JSON(fiber.Map{
				"has_task":          false,
				"inspect_requested": inspectRequested,
				"message":           "لا توجد مهام حالياً",
			})
		}
		if err == services.ErrDailyLimit {
			return c.JSON(fiber.Map{
				"has_task":          false,
				"inspect_requested": inspectRequested,
				"daily_limit_hit":   true,
				"message":           err.Error(),
			})
		}
		return c.JSON(fiber.Map{
			"has_task":          false,
			"inspect_requested": inspectRequested,
			"circuit_breaker":   true,
			"message":           err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"has_task":          true,
		"inspect_requested": inspectRequested,
		"vin":               vin,
	})
}

// Telemetry receives read-only browser state and compressed JPEG screenshot from Canada node
func (ctrl *WorkerController) Telemetry(c *fiber.Ctx) error {
	if !ctrl.verifySecret(c) {
		return c.Status(401).JSON(fiber.Map{"error": "غير مصرح"})
	}

	var payload services.WorkerTelemetry
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "بيانات الفحص غير صالحة"})
	}

	ctrl.guard.SaveTelemetry(&payload)
	return c.JSON(fiber.Map{"status": "ok"})
}

// Progress receives step-by-step live status while Canada node is harvesting a report
func (ctrl *WorkerController) Progress(c *fiber.Ctx) error {
	if !ctrl.verifySecret(c) {
		return c.Status(401).JSON(fiber.Map{"error": "غير مصرح"})
	}

	var payload services.WorkerProgress
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "بيانات التقدم غير صالحة"})
	}

	ctrl.guard.SaveProgress(&payload)
	return c.JSON(fiber.Map{"status": "ok"})
}

// Complete receives fetched report from Canada node or One-Shot safety freeze alert
func (ctrl *WorkerController) Complete(c *fiber.Ctx) error {
	if !ctrl.verifySecret(c) {
		return c.Status(401).JSON(fiber.Map{
			"error": "غير مصرح",
		})
	}

	var payload services.WorkerTaskResult
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "بنية البيانات غير صالحة"})
	}

	if payload.VIN == "" {
		return c.Status(400).JSON(fiber.Map{"error": "رقم الشاصي مطلوب"})
	}

	// Stage 2 Silent Background Enrichment: Update existing ReportCache with full Carfax HTML + Official PDF
	if payload.IsBackgroundEnrich && payload.ErrorCode == "" && payload.RawHtmlEn != "" {
		cleanVin := strings.ToUpper(strings.TrimSpace(payload.VIN))
		translatedHtmlAr, extractedTitle, fallbackSummary := ctrl.translator.TranslateAndExtract(payload.RawHtmlEn, cleanVin)
		title := payload.VehicleTitle
		if title == "" {
			title = extractedTitle
		}

		summaryJson := payload.FastSummaryJson
		if summaryJson == "" {
			var existing models.ReportCache
			if err := bootstrap.DB.Where("vin = ?", cleanVin).First(&existing).Error; err == nil && existing.SummaryJson != "" {
				summaryJson = existing.SummaryJson
			} else {
				summaryJson = fallbackSummary
			}
		}

		// Mark background_pending = false and background_ready = true inside summary_json
		var summaryMap map[string]interface{}
		if json.Unmarshal([]byte(summaryJson), &summaryMap) == nil && summaryMap != nil {
			summaryMap["background_pending"] = false
			summaryMap["background_ready"] = true
			if updatedBytes, mErr := json.Marshal(summaryMap); mErr == nil {
				summaryJson = string(updatedBytes)
			}
		}

		bootstrap.DB.Model(&models.ReportCache{}).Where("vin = ?", cleanVin).Updates(map[string]interface{}{
			"vehicle_title":      title,
			"raw_html_en":        payload.RawHtmlEn,
			"translated_html_ar": translatedHtmlAr,
			"summary_json":       summaryJson,
			"updated_at":         time.Now(),
		})

		return c.JSON(fiber.Map{
			"status":  "enriched",
			"message": "تم تحديث التقرير بالنسخة الكاملة وملف الطباعة الرسمي بصمت",
		})
	}

	if err := ctrl.guard.CompleteTask(&payload); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status":  "received",
		"message": "تم استلام النتيجة وتحديث حالة أمان الحساب بنجاح",
	})
}

// GetWorkerStatus returns the live status, telemetry, and screenshot of Canada Node
func (ctrl *WorkerController) GetWorkerStatus(c *fiber.Ctx) error {
	status := ctrl.guard.GetStatus()
	return c.JSON(status)
}

// RequestInspect triggers an immediate screenshot + DOM check on the Canada Node
func (ctrl *WorkerController) RequestInspect(c *fiber.Ctx) error {
	ctrl.guard.RequestImmediateInspect()
	return c.JSON(fiber.Map{
		"status":  "requested",
		"message": "تم إرسال أمر فحص الشاشة والتقاط Screenshot مضغوط إلى حاسبة كندا",
	})
}

// EmergencyStop manually trips the circuit breaker and clears pending tasks
func (ctrl *WorkerController) EmergencyStop(c *fiber.Ctx) error {
	_ = ctrl.guard.EmergencyStop()
	return c.JSON(fiber.Map{
		"status":  "stopped",
		"message": "تم تفعيل الإيقاف الاحترازي وتفريغ الطابور لحماية حساب كندا",
	})
}

// ResetCircuitBreaker clears alarm after manual intervention
func (ctrl *WorkerController) ResetCircuitBreaker(c *fiber.Ctx) error {
	if err := ctrl.guard.ResetCircuitBreaker(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status":  "reset",
		"message": "تمت إعادة تفعيل قاطع الحماية واستئناف استقبال المهام بنجاح",
	})
}

// SetWorkerMode toggles between Strict Live Canada Mode and Mock Simulation Fallback
func (ctrl *WorkerController) SetWorkerMode(c *fiber.Ctx) error {
	var body struct {
		UseMockFallback bool `json:"use_mock_fallback"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "بيانات غير صالحة"})
	}
	ctrl.guard.SetMockFallback(body.UseMockFallback)
	modeText := "وضع حاسبة كندا الحقيقي فقط (Strict Live Mode)"
	if body.UseMockFallback {
		modeText = "وضع المحاكاة الاحتياطي عند إغلاق كندا"
	}
	return c.JSON(fiber.Map{
		"status":            "updated",
		"use_mock_fallback": body.UseMockFallback,
		"message":           "تم تغيير وضع التشغيل إلى: " + modeText,
	})
}

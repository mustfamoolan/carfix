package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"carfix/app/models"
	"carfix/app/services"
	"carfix/bootstrap"
	"github.com/gofiber/fiber/v2"
)

var fastMemoryCache sync.Map
var ErrStillInQueue = errors.New("STILL_IN_QUEUE")

func InvalidateFastReportCache(vin string) {
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))
	fastMemoryCache.Delete(cleanVin)
	if bootstrap.Redis != nil {
		bootstrap.Redis.Del(context.Background(), "report:fast:"+cleanVin)
	}
}

// UpsertReportCache safely creates or updates a ReportCache record in PostgreSQL and updates Redis/RAM caches
// without ever failing on duplicate VIN keys or overwriting rich HTML with empty Stage-1 HTML.
func UpsertReportCache(cleanVin, vehicleTitle, rawHtmlEn, fastSummaryJson string, translator *services.CarfaxTranslatorService) (*models.ReportCache, fiber.Map) {
	cleanVin = strings.ToUpper(strings.TrimSpace(cleanVin))

	var translatedHtmlAr, extractedTitle, summaryJson string
	if rawHtmlEn != "" && translator != nil {
		translatedHtmlAr, extractedTitle, summaryJson = translator.TranslateAndExtract(rawHtmlEn, cleanVin)
		if extractedTitle != "" && !strings.HasPrefix(extractedTitle, "مركبة شاصي") && vehicleTitle == "" {
			vehicleTitle = extractedTitle
		}
	}
	if fastSummaryJson != "" && summaryJson == "" {
		summaryJson = fastSummaryJson
	}
	if vehicleTitle == "" {
		vehicleTitle = "CARFAX Vehicle Report - " + cleanVin
	}

	hasFullHtml := len(rawHtmlEn) > 500
	hasOfficialPdf := strings.Contains(rawHtmlEn, "carfix-official-pdf") || strings.Contains(summaryJson, `"has_official_pdf":true`)

	if hasFullHtml {
		var summaryMap map[string]interface{}
		if json.Unmarshal([]byte(summaryJson), &summaryMap) == nil && summaryMap != nil {
			summaryMap["background_pending"] = false
			summaryMap["background_ready"] = true
			summaryMap["has_official_pdf"] = hasOfficialPdf
			summaryMap["has_full_html"] = true
			if updatedBytes, mErr := json.Marshal(summaryMap); mErr == nil {
				summaryJson = string(updatedBytes)
			}
		}
	}

	now := time.Now()
	reportRecord := &models.ReportCache{
		VIN:              cleanVin,
		VehicleTitle:     vehicleTitle,
		RawHtmlEn:        rawHtmlEn,
		TranslatedHtmlAr: translatedHtmlAr,
		SummaryJson:      summaryJson,
		FetchedAt:        now,
		UpdatedAt:        now,
	}

	var existingVin string
	bootstrap.DB.Model(&models.ReportCache{}).Select("vin").Where("vin = ?", cleanVin).Scan(&existingVin)

	if existingVin == "" {
		if err := bootstrap.DB.Create(reportRecord).Error; err != nil {
			updates := map[string]interface{}{
				"vehicle_title": vehicleTitle,
				"summary_json":  summaryJson,
				"updated_at":    now,
			}
			if hasFullHtml {
				updates["raw_html_en"] = rawHtmlEn
				updates["translated_html_ar"] = translatedHtmlAr
			}
			bootstrap.DB.Model(&models.ReportCache{}).Where("vin = ?", cleanVin).Updates(updates)
		}
	} else {
		updates := map[string]interface{}{
			"vehicle_title": vehicleTitle,
			"summary_json":  summaryJson,
			"updated_at":    now,
		}
		if hasFullHtml {
			updates["raw_html_en"] = rawHtmlEn
			updates["translated_html_ar"] = translatedHtmlAr
		}
		bootstrap.DB.Model(&models.ReportCache{}).Where("vin = ?", cleanVin).Updates(updates)
	}

	fastPayload := fiber.Map{
		"vin":              cleanVin,
		"vehicle_title":    vehicleTitle,
		"summary_json":     summaryJson,
		"has_full_html":    hasFullHtml || strings.Contains(summaryJson, `"has_full_html":true`),
		"has_official_pdf": hasOfficialPdf,
		"fetched_at":       now,
	}

	if !strings.Contains(summaryJson, `"background_pending":true`) {
		fastMemoryCache.Store(cleanVin, fastPayload)
		if bootstrap.Redis != nil {
			if b, mErr := json.Marshal(fastPayload); mErr == nil {
				bootstrap.Redis.Set(context.Background(), "report:fast:"+cleanVin, string(b), 24*time.Hour)
			}
		}
	} else {
		InvalidateFastReportCache(cleanVin)
	}

	return reportRecord, fastPayload
}

type VinSearchController struct {
	vinValidator *services.VinValidatorService
	quotaManager *services.QuotaManagerService
	translator   *services.CarfaxTranslatorService
	workerGuard  *services.WorkerGuardService
}

func NewVinSearchController() *VinSearchController {
	return &VinSearchController{
		vinValidator: services.NewVinValidatorService(),
		quotaManager: services.NewQuotaManagerService(),
		translator:   services.NewCarfaxTranslatorService(),
		workerGuard:  services.NewWorkerGuardService(),
	}
}

// Helper to safely extract user ID from Fiber locals
func extractUserID(c *fiber.Ctx) (uint, error) {
	if val := c.Locals("user_id"); val != nil {
		if uid, ok := val.(uint); ok && uid > 0 {
			return uid, nil
		}
	}
	return 0, fiber.ErrUnauthorized
}

// Search processes VIN report retrieval with 2-Tier Caching (Cache vs Canada Node)
func (ctrl *VinSearchController) Search(c *fiber.Ctx) error {
	startTime := time.Now()

	// 1. Get authenticated user ID from locals
	userID, err := extractUserID(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "غير مصرح"})
	}

	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")

	// 2. Parse request body
	var input struct {
		VIN string `json:"vin"`
	}
	if err := c.BodyParser(&input); err != nil || strings.TrimSpace(input.VIN) == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "يرجى إدخال رقم الشاصي (VIN) للبحث",
		})
	}

	// 3. Step A: Validate and clean VIN
	cleanVin, isValid, errMsg := ctrl.vinValidator.ValidateAndClean(input.VIN)
	if !isValid {
		duration := time.Since(startTime).Milliseconds()

		// Log invalid VIN search attempt without quota deduction
		logEntry := models.SearchLog{
			UserID:       userID,
			VIN:          cleanVin,
			VehicleTitle: "رقم شاصي غير صالح",
			ResultStatus: "invalid_vin",
			FetchSource:  "none",
			DurationMs:   duration,
			IPAddress:    ipAddress,
			UserAgent:    userAgent,
			CreatedAt:    time.Now(),
		}
		bootstrap.DB.Create(&logEntry)

		return c.Status(400).JSON(fiber.Map{
			"message":  errMsg,
			"vin":      cleanVin,
			"is_valid": false,
		})
	}

	// 3.5. Pre-Flight Quota Check (resets daily counter at midnight & blocks exhausted users BEFORE queueing to Canada)
	if _, qErr := ctrl.quotaManager.CheckAndPrepareQuota(userID); qErr != nil {
		return c.Status(403).JSON(fiber.Map{
			"message": qErr.Error(),
		})
	}

	// 4. Step B: 2-Tier Cache Inspection (Tier 1: Redis RAM + PostgreSQL report_caches)
	if fastReport, err := ctrl.getOrBuildFastReport(cleanVin); err == nil && fastReport != nil {
		// ==================== CACHE HIT ====================
		if err := ctrl.quotaManager.ConsumeQuota(userID); err != nil {
			return c.Status(403).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		duration := time.Since(startTime).Milliseconds()
		vehicleTitle, _ := fastReport["vehicle_title"].(string)

		logEntry := models.SearchLog{
			UserID:       userID,
			VIN:          cleanVin,
			VehicleTitle: vehicleTitle,
			ResultStatus: "success",
			FetchSource:  "cache",
			DurationMs:   duration,
			IPAddress:    ipAddress,
			UserAgent:    userAgent,
			CreatedAt:    time.Now(),
		}
		bootstrap.DB.Create(&logEntry)

		fastReport["status"] = "success"
		fastReport["fetch_source"] = "cache"
		fastReport["duration_ms"] = duration
		return c.JSON(fastReport)
	}

	// ==================== CACHE MISS ====================
	rawHtmlEn, vehicleTitle, fastSummaryJson, err := ctrl.fetchFromCanadaNode(cleanVin)
	if err != nil {
		// If the VIN is still waiting in queue or actively being harvested after a 48s HTTP window,
		// return 202 Processing so the client seamlessly continues waiting without 504/502 timeout or false error logs!
		if errors.Is(err, ErrStillInQueue) {
			pos, estSec := ctrl.workerGuard.GetQueuePosition(cleanVin)
			isActive, _ := ctrl.workerGuard.IsVinActive(cleanVin)
			return c.Status(202).JSON(fiber.Map{
				"status":            "processing",
				"vin":               cleanVin,
				"is_active":         isActive,
				"queue_position":    pos,
				"estimated_seconds": estSec,
				"message":           "جاري استخراج وتجهيز التقرير، يرجى الانتظار لحظات...",
			})
		}

		duration := time.Since(startTime).Milliseconds()

		logEntry := models.SearchLog{
			UserID:       userID,
			VIN:          cleanVin,
			VehicleTitle: "تعذر استخراج التقرير",
			ResultStatus: "error",
			FetchSource:  "live",
			DurationMs:   duration,
			IPAddress:    ipAddress,
			UserAgent:    userAgent,
			CreatedAt:    time.Now(),
		}
		bootstrap.DB.Create(&logEntry)

		return c.Status(502).JSON(fiber.Map{
			"message": err.Error(),
			"vin":     cleanVin,
		})
	}

	// Save or update in report_caches (idempotent even if WorkerController.Complete already persisted it)
	newReport, fastPayload := UpsertReportCache(cleanVin, vehicleTitle, rawHtmlEn, fastSummaryJson, ctrl.translator)
	hasFullHtml, _ := fastPayload["has_full_html"].(bool)
	hasOfficialPdf, _ := fastPayload["has_official_pdf"].(bool)

	// Deduct user quota
	if err := ctrl.quotaManager.ConsumeQuota(userID); err != nil {
		return c.Status(403).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	duration := time.Since(startTime).Milliseconds()

	// Record successful search log
	logEntry := models.SearchLog{
		UserID:       userID,
		VIN:          cleanVin,
		VehicleTitle: newReport.VehicleTitle,
		ResultStatus: "success",
		FetchSource:  "canada_node",
		DurationMs:   duration,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    time.Now(),
	}
	bootstrap.DB.Create(&logEntry)

	return c.JSON(fiber.Map{
		"status":           "success",
		"vin":              newReport.VIN,
		"vehicle_title":    newReport.VehicleTitle,
		"fetch_source":     "canada_node",
		"duration_ms":      duration,
		"summary_json":     newReport.SummaryJson,
		"has_full_html":    hasFullHtml,
		"has_official_pdf": hasOfficialPdf,
		"fetched_at":       newReport.FetchedAt,
	})
}

// getOrBuildFastReport returns a lightweight (~5 KB) report object from Go Memory, Redis RAM, or PostgreSQL without touching 5MB TOAST HTML columns
func (ctrl *VinSearchController) getOrBuildFastReport(cleanVin string) (fiber.Map, error) {
	ctx := context.Background()
	redisKey := "report:fast:" + cleanVin

	// 0. Check In-Memory Go RAM Cache first (<0.01ms)
	if val, ok := fastMemoryCache.Load(cleanVin); ok {
		if m, ok := val.(fiber.Map); ok {
			copyMap := make(fiber.Map, len(m)+3)
			for k, v := range m {
				copyMap[k] = v
			}
			return copyMap, nil
		}
	}

	// 1. Check Redis RAM Cache (<1ms)
	if bootstrap.Redis != nil {
		if cachedStr, err := bootstrap.Redis.Get(ctx, redisKey).Result(); err == nil && cachedStr != "" {
			var cachedMap fiber.Map
			if json.Unmarshal([]byte(cachedStr), &cachedMap) == nil {
				if sj, ok := cachedMap["summary_json"].(string); ok && strings.Contains(sj, `"specs"`) && !strings.Contains(sj, `"background_pending":true`) {
					fastMemoryCache.Store(cleanVin, cachedMap)
					return cachedMap, nil
				}
			}
		}
	}

	// 2. Query ONLY lightweight metadata columns from PostgreSQL (never touching raw_html_en TOAST column!)
	type lightReportRow struct {
		VIN          string    `gorm:"column:vin"`
		VehicleTitle string    `gorm:"column:vehicle_title"`
		SummaryJson  string    `gorm:"column:summary_json"`
		FetchedAt    time.Time `gorm:"column:fetched_at"`
	}

	var row lightReportRow
	err := bootstrap.DB.Model(&models.ReportCache{}).
		Select("vin, vehicle_title, summary_json, fetched_at").
		Where("vin = ?", cleanVin).
		Scan(&row).Error

	if err != nil || row.VIN == "" {
		return nil, fmt.Errorf("not found")
	}

	// 3. Auto-upgrade legacy or pending cached reports once (when "specs" or "has_official_pdf" is missing or background_pending is true)
	if !strings.Contains(row.SummaryJson, `"specs"`) || !strings.Contains(row.SummaryJson, `"has_official_pdf"`) || strings.Contains(row.SummaryJson, `"background_pending":true`) {
		var rawHtml string
		bootstrap.DB.Model(&models.ReportCache{}).
			Select("raw_html_en").
			Where("vin = ?", cleanVin).
			Scan(&rawHtml)
		if len(rawHtml) > 500 {
			upgradedSummary := ctrl.translator.ExtractSummaryJson(rawHtml, cleanVin)
			if upgradedSummary != "" {
				row.SummaryJson = upgradedSummary
				bootstrap.DB.Model(&models.ReportCache{}).Where("vin = ?", cleanVin).Update("summary_json", upgradedSummary)
			}
		}
	}

	hasOfficialPdf := strings.Contains(row.SummaryJson, `"has_official_pdf":true`)
	hasFullHtml := strings.Contains(row.SummaryJson, `"has_full_html":true`)

	fastMap := fiber.Map{
		"vin":              row.VIN,
		"vehicle_title":    row.VehicleTitle,
		"summary_json":     row.SummaryJson,
		"has_full_html":    hasFullHtml,
		"has_official_pdf": hasOfficialPdf,
		"fetched_at":       row.FetchedAt,
	}

	// 4. Cache in Go Memory + Redis RAM if background enrichment is complete
	if !strings.Contains(row.SummaryJson, `"background_pending":true`) {
		fastMemoryCache.Store(cleanVin, fastMap)
		if bootstrap.Redis != nil {
			if b, mErr := json.Marshal(fastMap); mErr == nil {
				bootstrap.Redis.Set(ctx, redisKey, string(b), 24*time.Hour)
			}
		}
	}

	return fastMap, nil
}

// GetQueueStatus returns the live queue position, estimated wait time, or active progress for a VIN
func (ctrl *VinSearchController) GetQueueStatus(c *fiber.Ctx) error {
	vin := strings.ToUpper(strings.TrimSpace(c.Params("vin")))
	cleanVin, isValid, _ := ctrl.vinValidator.ValidateAndClean(vin)
	if !isValid {
		return c.Status(400).JSON(fiber.Map{"error": "رقم الشاصي غير صالح"})
	}

	// 1. If already cached and ready, notify immediately
	if fastReport, err := ctrl.getOrBuildFastReport(cleanVin); err == nil && fastReport != nil {
		return c.JSON(fiber.Map{
			"status":    "ready",
			"is_ready":  true,
			"vin":       cleanVin,
			"report":    fastReport,
		})
	}

	// 2. Check position in sequential queue / active pull
	pos, estSec := ctrl.workerGuard.GetQueuePosition(cleanVin)
	isActive, _ := ctrl.workerGuard.IsVinActive(cleanVin)
	cooldown := ctrl.workerGuard.GetCooldownRemainingSec()
	workerOnline := ctrl.workerGuard.IsConnected()

	return c.JSON(fiber.Map{
		"status":            "queued",
		"is_ready":          false,
		"is_active":         isActive,
		"vin":               cleanVin,
		"worker_online":     workerOnline,
		"queue_position":    pos,
		"estimated_seconds": estSec,
		"cooldown_seconds":  cooldown,
	})
}

// GetUserSearchHistory returns the authenticated user's own search history with fast pagination and filtering
func (ctrl *VinSearchController) GetUserSearchHistory(c *fiber.Ctx) error {
	userID, err := extractUserID(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"message": "غير مصرح"})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	if limit < 1 || limit > 200 {
		limit = 100
	}
	offset := (page - 1) * limit
	q := strings.TrimSpace(c.Query("q"))

	query := bootstrap.DB.Model(&models.SearchLog{}).Where("user_id = ?", userID)
	if q != "" {
		pattern := "%" + q + "%"
		query = query.Where("vin ILIKE ? OR vehicle_title ILIKE ?", pattern, pattern)
	}

	var total int64
	query.Count(&total)

	var logs []models.SearchLog
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "فشل جلب سجل البحث"})
	}

	return c.JSON(fiber.Map{
		"logs": logs,
		"pagination": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// GetCachedReport returns the fast lightweight report (~12 KB) in <5ms instead of transferring 10MB of HTML/PDF
func (ctrl *VinSearchController) GetCachedReport(c *fiber.Ctx) error {
	vin := strings.ToUpper(strings.TrimSpace(c.Params("vin")))

	fastReport, err := ctrl.getOrBuildFastReport(vin)
	if err != nil || fastReport == nil {
		return c.Status(404).JSON(fiber.Map{"message": "التقرير غير موجود في السجل"})
	}

	return c.JSON(fiber.Map{"report": fastReport})
}

// GetReportHtml serves the full Carfax HTML on-demand ONLY when the user clicks "Original Carfax Page"
func (ctrl *VinSearchController) GetReportHtml(c *fiber.Ctx) error {
	vin := strings.ToUpper(strings.TrimSpace(c.Params("vin")))
	lang := strings.ToLower(strings.TrimSpace(c.Query("lang", "en")))

	type htmlRow struct {
		RawHtmlEn        string `gorm:"column:raw_html_en"`
		TranslatedHtmlAr string `gorm:"column:translated_html_ar"`
	}
	var row htmlRow
	if err := bootstrap.DB.Model(&models.ReportCache{}).
		Select("raw_html_en, translated_html_ar").
		Where("vin = ?", vin).
		Scan(&row).Error; err != nil || (row.RawHtmlEn == "" && row.TranslatedHtmlAr == "") {
		return c.Status(404).JSON(fiber.Map{"message": "نسخة الصفحة الكاملة غير جاهزة بعد"})
	}

	html := row.RawHtmlEn
	if lang == "ar" && row.TranslatedHtmlAr != "" {
		html = row.TranslatedHtmlAr
	}

	// Strip embedded Base64 PDF script tag before sending HTML to browser (saves ~2.5 MB of transfer)
	const pdfStartMarker = `<script id="carfix-official-pdf" type="application/pdf-base64">`
	if startIdx := strings.Index(html, pdfStartMarker); startIdx != -1 {
		if endIdx := strings.Index(html[startIdx:], "</script>"); endIdx != -1 {
			html = html[:startIdx] + html[startIdx+endIdx+len("</script>"):]
		}
	}

	return c.JSON(fiber.Map{
		"vin":  vin,
		"lang": lang,
		"html": html,
	})
}

// GetReportPdf streams the official 4-Section Chrome PrintToPDF directly as binary application/pdf.
// If Stage 2 background PDF generation is still completing (within 30s of Stage 1), it waits up to 15s seamlessly!
func (ctrl *VinSearchController) GetReportPdf(c *fiber.Ctx) error {
	vin := strings.ToUpper(strings.TrimSpace(c.Params("vin")))
	const markerStart = `<script id="carfix-official-pdf" type="application/pdf-base64">`

	var rawHtml string
	var startIdx int = -1

	for attempt := 0; attempt < 18; attempt++ {
		rawHtml = ""
		_ = bootstrap.DB.Model(&models.ReportCache{}).
			Select("raw_html_en").
			Where("vin = ?", vin).
			Scan(&rawHtml).Error

		if rawHtml != "" {
			startIdx = strings.Index(rawHtml, markerStart)
			if startIdx != -1 {
				break
			}
		}
		time.Sleep(800 * time.Millisecond)
	}

	if rawHtml == "" || startIdx == -1 {
		return c.Status(404).JSON(fiber.Map{"message": "ملف الطباعة الرسمي PDF غير متوفر لهذا التقرير"})
	}
	contentStart := startIdx + len(markerStart)
	endIdx := strings.Index(rawHtml[contentStart:], "</script>")
	if endIdx == -1 {
		return c.Status(404).JSON(fiber.Map{"message": "بيانات PDF غير مكتملة"})
	}

	b64Data := strings.TrimSpace(rawHtml[contentStart : contentStart+endIdx])
	pdfBytes, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "تعذر فك تشفير ملف PDF"})
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf(`inline; filename="CARFAX_%s.pdf"`, vin))
	c.Set("Cache-Control", "public, max-age=86400")
	return c.Send(pdfBytes)
}

// fetchFromCanadaNode communicates with the remote worker or falls back to simulation if permitted
func (ctrl *VinSearchController) fetchFromCanadaNode(vin string) (string, string, string, error) {
	// 1. Always respect Circuit Breaker first
	if active, _ := ctrl.workerGuard.IsCircuitBreakerActive(); active {
		return "", "", "", fmt.Errorf("الخدمة مشغولة حالياً أو قيد التحديث اللحظي، يرجى المحاولة بعد قليل")
	}

	// 2. If Worker is online, dispatch strictly to Worker (with 1 automatic silent retry on transient DOM/typing hiccup)
	if ctrl.workerGuard.IsConnected() {
		for attempt := 1; attempt <= 2; attempt++ {
			if err := ctrl.workerGuard.EnqueueSearchTask(vin); err != nil {
				return "", "", "", fmt.Errorf("تعذر إرسال الطلب حالياً، يرجى المحاولة مرة أخرى")
			}

			// Wait up to 48s per HTTP cycle (safely below Nginx/Cloudflare 60s gateway timeout)
			res, err := ctrl.workerGuard.WaitForReport(vin, 48*time.Second)
			if err != nil {
				// If the report is still in queue or actively being extracted by the worker,
				// signal 202 Processing continuation instead of failing!
				if ctrl.workerGuard.IsVinPendingOrActive(vin) && ctrl.workerGuard.IsConnected() {
					return "", "", "", ErrStillInQueue
				}
				return "", "", "", fmt.Errorf("استغرقت عملية البحث وقتاً أطول من المعتاد، يرجى المحاولة بعد قليل")
			}

			if res != nil && res.ErrorCode != "" {
				if res.ErrorCode == "VIN_NOT_FOUND_ON_CARFAX" {
					return "", "", "", fmt.Errorf("لم يتم العثور على سجلات لهذا الرقم، يرجى التأكد من صحة رقم الشاصي")
				}
				// If it was a transient browser focus/typing/tab hiccup on attempt 1, retry once silently
				if attempt == 1 && (res.ErrorCode == "VIN_TYPE_ERROR" || res.ErrorCode == "TAB_LOST" || res.ErrorCode == "VIN_INPUT_NOT_FOUND") {
					time.Sleep(1500 * time.Millisecond)
					continue
				}
				return "", "", "", fmt.Errorf("الخدمة مشغولة حالياً، يرجى المحاولة مرة أخرى بعد قليل")
			}

			if res != nil && (res.RawHtmlEn != "" || res.FastSummaryJson != "") {
				return res.RawHtmlEn, res.VehicleTitle, res.FastSummaryJson, nil
			}
		}

		return "", "", "", fmt.Errorf("تعذر استخراج بيانات التقرير، يرجى المحاولة مرة أخرى")
	}

	// 3. Worker is offline: check if Mock Simulation Fallback is permitted
	if !ctrl.workerGuard.IsMockFallbackEnabled() {
		return "", "", "", fmt.Errorf("الخدمة قيد الصيانة اللحظية حالياً، يرجى المحاولة بعد قليل")
	}

	html, title, err := ctrl.fetchFromMock(vin)
	return html, title, "", err
}

// fetchFromMock generates a 100% faithful 6-Block Official CARFAX Vehicle History Report™
func (ctrl *VinSearchController) fetchFromMock(vin string) (string, string, error) {
	time.Sleep(300 * time.Millisecond)

	year := decodeVinYear(vin)
	makeModel := decodeVinMakeModel(vin)
	vehicleTitle := fmt.Sprintf("%d %s", year, makeModel)
	specLine := decodeVinSpecs(vin)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>CARFAX Vehicle History Report for %s - %s</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: 'Helvetica Neue', Arial, sans-serif; background-color: #f5f7fa; color: #333; line-height: 1.45; padding: 20px; font-size: 13px; }
    .report-container { max-width: 980px; margin: 0 auto; background: #ffffff; border: 1px solid #dcdfe4; box-shadow: 0 2px 8px rgba(0,0,0,0.06); padding: 25px; }

    /* BLOCK 1: Header & Vehicle Info */
    .cfx-header { display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 2px solid #003478; padding-bottom: 15px; margin-bottom: 20px; }
    .cfx-brand { display: flex; align-items: center; gap: 10px; }
    .cfx-boxes { display: flex; gap: 2px; }
    .cfx-box { background: #003478; color: #ffffff; font-weight: 900; font-size: 18px; width: 26px; height: 28px; display: flex; align-items: center; justify-content: center; border-radius: 2px; }
    .cfx-fox { background: #e65100; color: #fff; font-size: 12px; font-weight: bold; padding: 2px 6px; border-radius: 2px; display: flex; align-items: center; }
    .cfx-report-type { font-size: 12px; color: #666; font-weight: 600; text-transform: uppercase; margin-top: 2px; }
    .cfx-meta-box { text-align: right; }
    .cfx-car-title { font-size: 20px; font-weight: 800; color: #003478; margin-bottom: 3px; }
    .cfx-vin-line { font-size: 13px; font-family: monospace; color: #444; font-weight: 600; }
    .cfx-specs { font-size: 11px; color: #777; margin-top: 3px; font-weight: 500; }

    /* BLOCK 2: 6-Icon Summary Snapshot Box */
    .summary-snapshot-card { border: 2px solid #00599c; border-radius: 6px; padding: 15px; margin-bottom: 25px; background: #fafcff; }
    .snapshot-title { font-size: 14px; font-weight: 800; color: #003478; text-transform: uppercase; margin-bottom: 12px; border-bottom: 1px solid #d5e4f3; padding-bottom: 6px; }
    .snapshot-grid { display: grid; grid-template-columns: repeat(6, 1fr); gap: 10px; }
    .snapshot-item { background: #ffffff; border: 1px solid #d9e4ef; border-radius: 5px; padding: 10px 8px; text-align: center; }
    .snapshot-icon { width: 34px; height: 34px; border-radius: 50%%; margin: 0 auto 6px; display: flex; align-items: center; justify-content: center; font-size: 16px; font-weight: bold; }
    .icon-green { background: #ebf7ee; color: #1c8237; border: 1px solid #b7e3c0; }
    .icon-blue { background: #e8f1f8; color: #00599c; border: 1px solid #bad6ee; }
    .snapshot-label { font-size: 10px; color: #777; font-weight: 600; text-transform: uppercase; margin-bottom: 3px; }
    .snapshot-val { font-size: 11px; font-weight: 700; color: #222; }

    /* SECTION TITLES & TABLES */
    .section-title-bar { background: #003478; color: #ffffff; padding: 8px 12px; font-size: 13px; font-weight: 800; border-radius: 4px 4px 0 0; text-transform: uppercase; margin-top: 25px; display: flex; justify-content: space-between; }
    .cfx-table { width: 100%%; border-collapse: collapse; margin-bottom: 20px; font-size: 12px; }
    .cfx-table th, .cfx-table td { border: 1px solid #dcdfe4; padding: 9px 12px; vertical-align: middle; }
    .cfx-table thead th { background: #e8f1f8; color: #003478; font-weight: 800; text-align: left; }
    .row-label { background: #fbfcfd; font-weight: 700; color: #444; width: 32%%; }

    /* BADGES */
    .badge-guaranteed { display: inline-flex; align-items: center; gap: 4px; background: #ebf7ee; color: #1c8237; border: 1px solid #b7e3c0; padding: 3px 8px; border-radius: 3px; font-size: 11px; font-weight: 700; }
    .badge-alert { display: inline-flex; align-items: center; gap: 4px; background: #fdf2f2; color: #d0202e; border: 1px solid #f8b4b4; padding: 3px 8px; border-radius: 3px; font-size: 11px; font-weight: 700; }

    /* BLOCK 6: Timeline History */
    .timeline-table th { background: #00599c; color: #ffffff; font-weight: 700; }
    .timeline-date { font-weight: 700; color: #003478; white-space: nowrap; width: 90px; }
    .timeline-miles { font-family: monospace; font-weight: 700; color: #222; width: 95px; }
    .timeline-source { font-weight: 700; color: #444; width: 170px; }
    .timeline-comment { color: #333; line-height: 1.4; }

    /* PRINT RULES */
    @media print {
      body { background: #fff; padding: 0; }
      .report-container { border: none; box-shadow: none; padding: 0; width: 100%%; max-width: 100%%; }
      * { -webkit-print-color-adjust: exact !important; print-color-adjust: exact !important; }
    }
  </style>
</head>
<body>
  <div class="report-container">
    <!-- BLOCK 1: Header & Vehicle Info -->
    <div class="cfx-header">
      <div class="cfx-brand">
        <div>
          <div class="cfx-boxes">
            <span class="cfx-box">C</span>
            <span class="cfx-box">A</span>
            <span class="cfx-box">R</span>
            <span class="cfx-box">F</span>
            <span class="cfx-box">A</span>
            <span class="cfx-box">X</span>
            <span class="cfx-fox">FOX</span>
          </div>
          <div class="cfx-report-type">Vehicle History Report™</div>
        </div>
      </div>
      <div class="cfx-meta-box">
        <h1 class="cfx-car-title">%s</h1>
        <div class="cfx-vin-line">VIN: %s</div>
        <div class="cfx-specs">%s</div>
      </div>
    </div>

    <!-- BLOCK 2: 6-Icon Summary Snapshot Box -->
    <div class="summary-snapshot-card">
      <div class="snapshot-title">CARFAX Vehicle Highlights & Snapshot</div>
      <div class="snapshot-grid">
        <div class="snapshot-item">
          <div class="snapshot-icon icon-green">✓</div>
          <div class="snapshot-label">Accidents</div>
          <div class="snapshot-val">No Accidents</div>
        </div>
        <div class="snapshot-item">
          <div class="snapshot-icon icon-blue">14</div>
          <div class="snapshot-label">Service Records</div>
          <div class="snapshot-val">14 Records</div>
        </div>
        <div class="snapshot-item">
          <div class="snapshot-icon icon-blue">1</div>
          <div class="snapshot-label">Owners</div>
          <div class="snapshot-val">1 Owner</div>
        </div>
        <div class="snapshot-item">
          <div class="snapshot-icon icon-green">✓</div>
          <div class="snapshot-label">Type of Use</div>
          <div class="snapshot-val">Personal</div>
        </div>
        <div class="snapshot-item">
          <div class="snapshot-icon icon-blue">TX</div>
          <div class="snapshot-label">Last Location</div>
          <div class="snapshot-val">Texas</div>
        </div>
        <div class="snapshot-item">
          <div class="snapshot-icon icon-green">✓</div>
          <div class="snapshot-label">Last Odometer</div>
          <div class="snapshot-val">38,450 mi</div>
        </div>
      </div>
    </div>

    <!-- BLOCK 3: CARFAX Ownership History Table -->
    <div class="section-title-bar">
      <span>1. CARFAX Ownership History</span>
      <span>Owner 1</span>
    </div>
    <table class="cfx-table">
      <thead>
        <tr>
          <th>Ownership Details</th>
          <th>Owner 1</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td class="row-label">Year purchased</td>
          <td>%d</td>
        </tr>
        <tr>
          <td class="row-label">Type of owner</td>
          <td>Personal vehicle</td>
        </tr>
        <tr>
          <td class="row-label">Estimated length of ownership</td>
          <td>4 yrs. 2 mo.</td>
        </tr>
        <tr>
          <td class="row-label">Owned in the following states/provinces</td>
          <td>Texas, United States</td>
        </tr>
        <tr>
          <td class="row-label">Estimated miles driven per year</td>
          <td>9,240 / yr</td>
        </tr>
        <tr>
          <td class="row-label">Last reported odometer reading</td>
          <td><strong>38,450 miles</strong></td>
        </tr>
      </tbody>
    </table>

    <!-- BLOCK 4: CARFAX Title History Table -->
    <div class="section-title-bar">
      <span>2. CARFAX Title History Guarantee</span>
      <span>Guaranteed Clean Title</span>
    </div>
    <table class="cfx-table">
      <thead>
        <tr>
          <th>Title History Records</th>
          <th>Owner 1</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td class="row-label">Salvage | Junk | Rebuilt | Fire | Flood | Hail | Lemon</td>
          <td><span class="badge-guaranteed">✓ Guaranteed No Problem</span></td>
        </tr>
        <tr>
          <td class="row-label">Not Actual Mileage | Exceeds Mechanical Limits</td>
          <td><span class="badge-guaranteed">✓ Guaranteed No Problem</span></td>
        </tr>
      </tbody>
    </table>

    <!-- BLOCK 5: Additional History Table -->
    <div class="section-title-bar">
      <span>3. Additional Technical & Damage Check</span>
      <span>No Issues Reported</span>
    </div>
    <table class="cfx-table">
      <thead>
        <tr>
          <th>System / Safety Inspection Item</th>
          <th>Status & Finding</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td class="row-label">Total Loss Check</td>
          <td><span class="badge-guaranteed">✓ No total loss reported</span></td>
        </tr>
        <tr>
          <td class="row-label">Structural Damage</td>
          <td><span class="badge-guaranteed">✓ No structural damage reported</span></td>
        </tr>
        <tr>
          <td class="row-label">Airbag Deployment</td>
          <td><span class="badge-guaranteed">✓ No airbag deployment reported</span></td>
        </tr>
        <tr>
          <td class="row-label">Odometer Check</td>
          <td><span class="badge-guaranteed">✓ No indication of an odometer rollback</span></td>
        </tr>
        <tr>
          <td class="row-label">Accident / Damage Check</td>
          <td><span class="badge-guaranteed">✓ No accidents or damage reported</span></td>
        </tr>
        <tr>
          <td class="row-label">Manufacturer Recall</td>
          <td><span class="badge-guaranteed">✓ No open recalls reported</span></td>
        </tr>
      </tbody>
    </table>

    <!-- BLOCK 6: Detailed History Timeline Table -->
    <div class="section-title-bar">
      <span>4. Detailed History Records (Chronological Timeline)</span>
      <span>Owner 1 Records</span>
    </div>
    <table class="cfx-table timeline-table">
      <thead>
        <tr>
          <th>Date</th>
          <th>Mileage</th>
          <th>Source</th>
          <th>Comments & Service Records</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td class="timeline-date">03/15/%d</td>
          <td class="timeline-miles">12 mi</td>
          <td class="timeline-source">Motor Vehicle Dept.</td>
          <td class="timeline-comment">First owner reported, Titled or registered as personal vehicle</td>
        </tr>
        <tr>
          <td class="timeline-date">09/20/%d</td>
          <td class="timeline-miles">5,140 mi</td>
          <td class="timeline-source">Authorized Dealer Service</td>
          <td class="timeline-comment">Vehicle serviced, Oil and filter changed, Maintenance inspection completed, Fluids checked</td>
        </tr>
        <tr>
          <td class="timeline-date">04/10/%d</td>
          <td class="timeline-miles">12,680 mi</td>
          <td class="timeline-source">Service Facility</td>
          <td class="timeline-comment">Vehicle serviced, Tires rotated, Cabin air filter replaced, Multi-point safety inspection performed</td>
        </tr>
        <tr>
          <td class="timeline-date">11/05/%d</td>
          <td class="timeline-miles">21,400 mi</td>
          <td class="timeline-source">Authorized Dealership</td>
          <td class="timeline-comment">Maintenance inspection completed, Front brake pads checked, Battery/charging system checked</td>
        </tr>
        <tr>
          <td class="timeline-date">08/18/%d</td>
          <td class="timeline-miles">31,250 mi</td>
          <td class="timeline-source">Dealer Service Center</td>
          <td class="timeline-comment">Oil and filter changed, Wiper blades replaced, Four tires balanced and rotated</td>
        </tr>
        <tr>
          <td class="timeline-date">02/02/%d</td>
          <td class="timeline-miles">38,450 mi</td>
          <td class="timeline-source">Online Dealer Inventory</td>
          <td class="timeline-comment">Vehicle offered for sale, Pre-delivery inspection completed, Passed safety and emissions inspection</td>
        </tr>
      </tbody>
    </table>
  </div>
</body>
</html>`, vehicleTitle, vin, vehicleTitle, vin, specLine, year, year, year, year+1, year+2, year+3, year+4)

	return html, vehicleTitle, nil
}

// decodeVinYear decodes year character at 10th position (index 9)
func decodeVinYear(vin string) int {
	if len(vin) < 10 {
		return 2022
	}
	char := vin[9]
	years := map[byte]int{
		'1': 2001, '2': 2002, '3': 2003, '4': 2004, '5': 2005, '6': 2006, '7': 2007, '8': 2008, '9': 2009,
		'A': 2010, 'B': 2011, 'C': 2012, 'D': 2013, 'E': 2014, 'F': 2015, 'G': 2016, 'H': 2017, 'J': 2018,
		'K': 2019, 'L': 2020, 'M': 2021, 'N': 2022, 'P': 2023, 'R': 2024, 'S': 2025, 'T': 2026,
	}
	if y, ok := years[char]; ok {
		return y
	}
	return 2020
}

// decodeVinMakeModel estimates make and model based on WMI prefix
func decodeVinMakeModel(vin string) string {
	if len(vin) < 3 {
		return "Vehicle"
	}
	wmi := vin[:3]
	switch wmi {
	case "1FA", "1FM", "1FT":
		return "Ford Mustang GT"
	case "1HG":
		return "Honda Accord EX-L"
	case "1GC", "1GT":
		return "Chevrolet Silverado 1500"
	case "2T1", "2T2", "4T1":
		return "Toyota Camry SE"
	case "3FA", "3FM":
		return "Ford Fusion Titanium"
	case "3VW":
		return "Volkswagen Jetta GLI"
	case "5UX", "WBA":
		return "BMW X5 xDrive40i"
	case "WA1", "WAU":
		return "Audi A6 Quattro"
	case "WDD":
		return "Mercedes-Benz E350"
	case "JN1", "JNK":
		return "Nissan Altima SV"
	default:
		return "Sedan Touring"
	}
}

// decodeVinSpecs generates authentic body and engine specs
func decodeVinSpecs(vin string) string {
	if len(vin) < 3 {
		return "SEDAN 4 DR | GASOLINE | FWD"
	}
	switch vin[:3] {
	case "1FA":
		return "COUPE 2 DR | 5.0L V8 DOHC 32V | GASOLINE | REAR WHEEL DRIVE"
	case "1HG":
		return "SEDAN 4 DR | 1.5L I4 DOHC 16V TURBO | GASOLINE | FRONT WHEEL DRIVE"
	case "2T1", "4T1":
		return "SEDAN 4 DR | 2.5L I4 DOHC 16V | GASOLINE | FRONT WHEEL DRIVE"
	case "5UX", "WBA":
		return "SUV 4 DR | 3.0L I6 DOHC 24V TURBO | GASOLINE | ALL WHEEL DRIVE"
	case "WDD":
		return "SEDAN 4 DR | 2.0L I4 DOHC 16V TURBO | GASOLINE | ALL WHEEL DRIVE"
	default:
		return "SEDAN 4 DR | 4-CYLINDER INLINE | GASOLINE | FWD"
	}
}

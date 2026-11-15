package controllers

import (
	"net/url"
	"strings"

	"carfix/app/services"
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

// extractNodeInfo extracts worker ID and Name from request headers (with URL-unescaping for Arabic names)
func extractNodeInfo(c *fiber.Ctx) (string, string) {
	nodeID := strings.TrimSpace(c.Get("X-Worker-ID"))
	if nodeID == "" {
		nodeID = strings.TrimSpace(c.Query("node_id"))
	}
	nodeName := strings.TrimSpace(c.Get("X-Worker-Name"))
	if nodeName == "" {
		nodeName = strings.TrimSpace(c.Query("node_name"))
	}
	if strings.Contains(nodeName, "%") {
		if unescaped, err := url.QueryUnescape(nodeName); err == nil && strings.TrimSpace(unescaped) != "" {
			nodeName = strings.TrimSpace(unescaped)
		}
	}
	return nodeID, nodeName
}

// Poll delivers 1 to 4 queued VINs (fairly load-balanced across online nodes) or an inspect request to the Canada node
func (ctrl *WorkerController) Poll(c *fiber.Ctx) error {
	if !ctrl.verifySecret(c) {
		return c.Status(401).JSON(fiber.Map{
			"error": "غير مصرح - مفتاح عامل كندا غير صحيح أو مفقود",
		})
	}

	nodeID, nodeName := extractNodeInfo(c)
	inspectRequested := ctrl.guard.ConsumeInspectRequest(nodeID)

	maxBatch := c.QueryInt("max_batch", 1)
	if maxBatch < 1 {
		maxBatch = 1
	}
	if maxBatch > 2 {
		maxBatch = 2
	}

	vins, cooldownRemaining, err := ctrl.guard.PollBatchTasks(nodeID, nodeName, maxBatch)
	if err != nil {
		if err == services.ErrCooldownActive {
			return c.JSON(fiber.Map{
				"has_task":           false,
				"inspect_requested":  inspectRequested,
				"cooldown_remaining": cooldownRemaining,
				"node_id":            nodeID,
				"message":            "فترة تهدئة لحماية الحساب من الحظر",
			})
		}
		if err == services.ErrQueueEmpty {
			return c.JSON(fiber.Map{
				"has_task":          false,
				"inspect_requested": inspectRequested,
				"node_id":           nodeID,
				"message":           "لا توجد مهام حالياً",
			})
		}
		if err == services.ErrDailyLimit {
			return c.JSON(fiber.Map{
				"has_task":          false,
				"inspect_requested": inspectRequested,
				"daily_limit_hit":   true,
				"node_id":           nodeID,
				"message":           err.Error(),
			})
		}
		return c.JSON(fiber.Map{
			"has_task":          false,
			"inspect_requested": inspectRequested,
			"circuit_breaker":   true,
			"node_id":           nodeID,
			"message":           err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"has_task":          true,
		"inspect_requested": inspectRequested,
		"node_id":           nodeID,
		"vin":               vins[0],
		"vins":              vins,
		"batch_size":        len(vins),
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

	nodeID, nodeName := extractNodeInfo(c)
	ctrl.guard.SaveTelemetry(&payload, nodeID, nodeName)
	return c.JSON(fiber.Map{"status": "ok", "node_id": nodeID})
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

	nodeID, nodeName := extractNodeInfo(c)
	ctrl.guard.SaveProgress(&payload, nodeID, nodeName)
	return c.JSON(fiber.Map{"status": "ok", "node_id": nodeID})
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

	nodeID, nodeName := extractNodeInfo(c)
	if payload.WorkerID == "" {
		payload.WorkerID = nodeID
	}
	if payload.WorkerName == "" {
		payload.WorkerName = nodeName
	}

	if payload.VIN == "" {
		return c.Status(400).JSON(fiber.Map{"error": "رقم الشاصي مطلوب"})
	}

	// Stage 2 Silent Background Enrichment: Update existing ReportCache with full Carfax HTML + Official PDF
	// Always handle IsBackgroundEnrich here so it NEVER falls through to CompleteTask a second time!
	if payload.IsBackgroundEnrich {
		if payload.ErrorCode == "" && (payload.RawHtmlEn != "" || payload.FastSummaryJson != "") {
			UpsertReportCache(payload.VIN, payload.VehicleTitle, payload.RawHtmlEn, payload.FastSummaryJson, ctrl.translator)
		}
		ctrl.guard.ClearNodeActiveVin(payload.WorkerID, payload.VIN)

		return c.JSON(fiber.Map{
			"status":  "enriched",
			"node_id": payload.WorkerID,
			"message": "تم تحديث التقرير بالنسخة الكاملة وملف الطباعة الرسمي بصمت",
		})
	}

	// Stage 1 Immediate Worker-Side Persistence: Save completed report directly to PostgreSQL report_caches
	if payload.ErrorCode == "" && (payload.RawHtmlEn != "" || payload.FastSummaryJson != "") {
		UpsertReportCache(payload.VIN, payload.VehicleTitle, payload.RawHtmlEn, payload.FastSummaryJson, ctrl.translator)
		RecordRecentLiveAttribution(payload.VIN, payload.WorkerID, payload.WorkerName)
	}

	if err := ctrl.guard.CompleteTask(&payload); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status":  "received",
		"node_id": payload.WorkerID,
		"message": "تم استلام النتيجة وتحديث حالة أمان الحساب بنجاح",
	})
}

// Heartbeat is a lightweight keep-alive endpoint called by the Canada node every ~10s.
func (ctrl *WorkerController) Heartbeat(c *fiber.Ctx) error {
	if !ctrl.verifySecret(c) {
		return c.Status(401).JSON(fiber.Map{"error": "غير مصرح"})
	}
	nodeID, nodeName := extractNodeInfo(c)
	ctrl.guard.RecordHeartbeat(nodeID, nodeName)
	return c.JSON(fiber.Map{
		"status":  "ok",
		"node_id": nodeID,
		"ts":      c.Context().Time().Unix(),
	})
}

// GetWorkerStatus returns the live status, telemetry, and screenshot of Canada Node
func (ctrl *WorkerController) GetWorkerStatus(c *fiber.Ctx) error {
	status := ctrl.guard.GetStatus()
	return c.JSON(status)
}

// RequestInspect triggers an immediate screenshot + DOM check on Canada Node (node-specific or all)
func (ctrl *WorkerController) RequestInspect(c *fiber.Ctx) error {
	nodeID := c.Query("node_id")
	if nodeID == "" {
		var body struct {
			NodeID string `json:"node_id"`
		}
		_ = c.BodyParser(&body)
		nodeID = body.NodeID
	}
	ctrl.guard.RequestImmediateInspect(nodeID)
	return c.JSON(fiber.Map{
		"status":  "requested",
		"node_id": nodeID,
		"message": "تم إرسال أمر فحص الشاشة والتقاط Screenshot مضغوط إلى العقدة",
	})
}

// EmergencyStop manually trips the circuit breaker for all nodes or a specific node
func (ctrl *WorkerController) EmergencyStop(c *fiber.Ctx) error {
	nodeID := c.Query("node_id")
	if nodeID == "" {
		var body struct {
			NodeID string `json:"node_id"`
		}
		_ = c.BodyParser(&body)
		nodeID = body.NodeID
	}

	if nodeID != "" {
		_ = ctrl.guard.EmergencyStopNode(nodeID)
		return c.JSON(fiber.Map{
			"status":  "stopped",
			"node_id": nodeID,
			"message": "تم تفعيل الإيقاف الاحترازي للعقدة " + nodeID,
		})
	}

	_ = ctrl.guard.EmergencyStop()
	return c.JSON(fiber.Map{
		"status":  "stopped",
		"message": "تم تفعيل الإيقاف الاحترازي وتفريغ الطابور لكافة العقد",
	})
}

// ResetCircuitBreaker clears alarm after manual intervention
func (ctrl *WorkerController) ResetCircuitBreaker(c *fiber.Ctx) error {
	nodeID := c.Query("node_id")
	if nodeID == "" {
		var body struct {
			NodeID string `json:"node_id"`
		}
		_ = c.BodyParser(&body)
		nodeID = body.NodeID
	}

	if nodeID != "" {
		if err := ctrl.guard.ResetNodeCircuitBreaker(nodeID); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{
			"status":  "reset",
			"node_id": nodeID,
			"message": "تمت إعادة تشغيل قاطع الحماية للعقدة " + nodeID,
		})
	}

	if err := ctrl.guard.ResetCircuitBreaker(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"status":  "reset",
		"message": "تمت إعادة تفعيل قاطع الحماية واستئناف استقبال المهام بنجاح",
	})
}

// DeleteNode removes a dead or decommissioned node from the cluster registry
func (ctrl *WorkerController) DeleteNode(c *fiber.Ctx) error {
	nodeID := c.Params("id")
	if nodeID == "" {
		nodeID = c.Query("node_id")
	}
	if nodeID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "معرف العقدة مطلوب"})
	}
	if err := ctrl.guard.DeleteNode(nodeID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"status":  "deleted",
		"node_id": nodeID,
		"message": "تم حذف العقدة من سجل الشبكة بنجاح",
	})
}

// GetQueueDetails returns the active tasks across all nodes and full list of pending tasks in queue
func (ctrl *WorkerController) GetQueueDetails(c *fiber.Ctx) error {
	activeVin, activeProg, queueList := ctrl.guard.GetDetailedQueue()
	activeTasks := ctrl.guard.GetAllActiveTasks()
	cooldown := ctrl.guard.GetCooldownRemainingSec()
	connected := ctrl.guard.IsConnected()

	return c.JSON(fiber.Map{
		"connected":        connected,
		"active_vin":       activeVin,
		"active_progress":  activeProg,
		"active_tasks":     activeTasks,
		"queue":            queueList,
		"queue_count":      len(queueList),
		"cooldown_seconds": cooldown,
	})
}

// RemoveQueueTask removes a VIN from the waiting queue
func (ctrl *WorkerController) RemoveQueueTask(c *fiber.Ctx) error {
	vin := c.Params("vin")
	if vin == "" {
		return c.Status(400).JSON(fiber.Map{"error": "رقم الشاصي مطلوب"})
	}
	if err := ctrl.guard.RemoveFromQueue(vin); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok", "message": "تم حذف الشاصي من قائمة الانتظار بنجاح"})
}

// PromoteQueueTask moves a VIN to the front of the queue
func (ctrl *WorkerController) PromoteQueueTask(c *fiber.Ctx) error {
	vin := c.Params("vin")
	if vin == "" {
		return c.Status(400).JSON(fiber.Map{"error": "رقم الشاصي مطلوب"})
	}
	if err := ctrl.guard.PromoteInQueue(vin); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok", "message": "تم تقديم الشاصي إلى مقدمة الطابور كأولوية قصوى"})
}

// PurgeAllQueue clears all pending tasks
func (ctrl *WorkerController) PurgeAllQueue(c *fiber.Ctx) error {
	if err := ctrl.guard.PurgeQueue(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "ok", "message": "تم تفريغ قائمة الانتظار بالكامل"})
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

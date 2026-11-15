package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"carfix/app/models"
	"carfix/bootstrap"
)

var (
	ErrCircuitBreaker = errors.New("قاطع الدائرة مفعل لحماية حساب كندا من الحظر")
	ErrDailyLimit     = errors.New("تم بلوغ الحد الأقصى للسحب اليومي الآمن من كندا")
	ErrCooldownActive = errors.New("فترة التهدئة الإلزامية لا زالت نشطة لحماية الحساب")
	ErrQueueEmpty     = errors.New("لا توجد مهام في طابور السحب حالياً")
	ErrWorkerTimedOut = errors.New("انتهت مهلة استجابة عقدة كندا")
)

type WorkerTelemetry struct {
	ChromeConnected  bool      `json:"chrome_connected"`
	CarfaxTabFound   bool      `json:"carfax_tab_found"`
	TabID            string    `json:"tab_id,omitempty"`
	CurrentURL       string    `json:"current_url"`
	PageTitle        string    `json:"page_title"`
	VinInputFound    bool      `json:"vin_input_found"`
	VinInputSelector string    `json:"vin_input_selector"`
	SubmitBtnFound   bool      `json:"submit_btn_found"`
	SessionActive    bool      `json:"session_active"`
	IsLoginPage      bool      `json:"is_login_page"`
	HasChallenge     bool      `json:"has_challenge"`
	StatusMessage    string    `json:"status_message"`
	ScreenshotBase64 string    `json:"screenshot_base64,omitempty"`
	InspectedAt      time.Time `json:"inspected_at"`
}

type WorkerProgress struct {
	WorkerID         string    `json:"worker_id,omitempty"`
	WorkerName       string    `json:"worker_name,omitempty"`
	VIN              string    `json:"vin"`
	Step             string    `json:"step"`
	StepAr           string    `json:"step_ar"`
	ScreenshotBase64 string    `json:"screenshot_base64,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type WorkerNodeInfo struct {
	WorkerID             string           `json:"worker_id"`
	WorkerName           string           `json:"worker_name"`
	Connected            bool             `json:"connected"`
	LastHeartbeat        time.Time        `json:"last_heartbeat"`
	LastHeartbeatSecAgo  int64            `json:"last_heartbeat_sec_ago"`
	CooldownRemainingSec int              `json:"cooldown_remaining_sec"`
	DailyPulls           int              `json:"daily_pulls"`
	CircuitBreakerActive bool             `json:"circuit_breaker_active"`
	CircuitBreakerReason string           `json:"circuit_breaker_reason"`
	InspectPending       bool             `json:"inspect_pending"`
	ActiveVIN            string           `json:"active_vin,omitempty"`
	ActiveProgress       *WorkerProgress  `json:"active_progress,omitempty"`
	Telemetry            *WorkerTelemetry `json:"telemetry,omitempty"`
	LastScreenshotBase64 string           `json:"last_screenshot_base64,omitempty"`
	LastScreenshotAt     time.Time        `json:"last_screenshot_at,omitempty"`
}

type WorkerStatus struct {
	Connected            bool             `json:"connected"`
	CircuitBreakerActive bool             `json:"circuit_breaker_active"`
	CircuitBreakerReason string           `json:"circuit_breaker_reason"`
	DailyPulls           int              `json:"daily_pulls"`
	MaxDailyPulls        int              `json:"max_daily_pulls"`
	CooldownRemainingSec int              `json:"cooldown_remaining_sec"`
	QueueLength          int              `json:"queue_length"`
	LastHeartbeat        time.Time        `json:"last_heartbeat"`
	UseMockFallback      bool             `json:"use_mock_fallback"`
	InspectPending       bool             `json:"inspect_pending"`
	Telemetry            *WorkerTelemetry `json:"telemetry,omitempty"`
	ActiveProgress       *WorkerProgress  `json:"active_progress,omitempty"`
	ActiveTasks          []WorkerProgress `json:"active_tasks,omitempty"`
	LastScreenshotBase64 string           `json:"last_screenshot_base64,omitempty"`
	LastScreenshotAt     time.Time        `json:"last_screenshot_at,omitempty"`
	TotalNodes           int              `json:"total_nodes"`
	OnlineNodes          int              `json:"online_nodes"`
	Nodes                []WorkerNodeInfo `json:"nodes,omitempty"`
}

type WorkerTaskResult struct {
	WorkerID           string    `json:"worker_id,omitempty"`
	WorkerName         string    `json:"worker_name,omitempty"`
	VIN                string    `json:"vin"`
	VehicleTitle       string    `json:"vehicle_title"`
	RawHtmlEn          string    `json:"raw_html_en"`
	FastSummaryJson    string    `json:"fast_summary_json,omitempty"`
	IsBackgroundEnrich bool      `json:"is_background_enrich,omitempty"`
	ErrorCode          string    `json:"error_code,omitempty"`
	ErrorMessage       string    `json:"error_message,omitempty"`
	ScreenshotBase64   string    `json:"screenshot_base64,omitempty"`
	CompletedAt        time.Time `json:"completed_at"`
}

type WorkerGuardService struct {
	secretKey       string
	maxDailyPulls   int
	minCooldownSec  int
	useMockFallback bool
	mu              sync.Mutex
}

var (
	workerGuardInstance *WorkerGuardService
	workerGuardOnce     sync.Once
)

func NewWorkerGuardService() *WorkerGuardService {
	workerGuardOnce.Do(func() {
		secretKey := os.Getenv("WORKER_SECRET_KEY")
		if secretKey == "" {
			secretKey = "almoq3-canada-secret-2026"
		}

		maxDaily := 1000000
		if val := os.Getenv("CANADA_MAX_DAILY_PULLS"); val != "" {
			if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
				maxDaily = parsed
			}
		}

		minCooldown := 13
		if val := os.Getenv("CANADA_MIN_COOLDOWN_SEC"); val != "" {
			if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
				minCooldown = parsed
			}
		}
		if minCooldown > 15 {
			minCooldown = 13
		}

		mockFallback := true
		if val := os.Getenv("USE_MOCK_FALLBACK"); val != "" {
			mockFallback = val == "true" || val == "1"
		}

		workerGuardInstance = &WorkerGuardService{
			secretKey:       secretKey,
			maxDailyPulls:   maxDaily,
			minCooldownSec:  minCooldown,
			useMockFallback: mockFallback,
		}
	})

	return workerGuardInstance
}

func (s *WorkerGuardService) ValidateSecret(token string) bool {
	return token == s.secretKey
}

func (s *WorkerGuardService) IsMockFallbackEnabled() bool {
	if bootstrap.Redis != nil {
		val, err := bootstrap.Redis.Get(context.Background(), "canada:use_mock_fallback").Result()
		if err == nil {
			return val == "1" || val == "true"
		}
	}
	return s.useMockFallback
}

func (s *WorkerGuardService) SetMockFallback(enabled bool) {
	s.useMockFallback = enabled
	if bootstrap.Redis != nil {
		val := "0"
		if enabled {
			val = "1"
		}
		bootstrap.Redis.Set(context.Background(), "canada:use_mock_fallback", val, 0)
	}
}

func (s *WorkerGuardService) getDailyKey() string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf("canada:daily_pulls:%s", today)
}

func (s *WorkerGuardService) getNodeDailyKey(nodeID string) string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf("canada:worker:%s:daily_pulls:%s", nodeID, today)
}

// isCorruptedNodeName detects HTTP header mojibake (e.g. "('D'-*J'7J) 2 '/F3'-") from unescaped legacy headers
func isCorruptedNodeName(s string) bool {
	if s == "" || !utf8.ValidString(s) {
		return true
	}
	if strings.Contains(s, "('D") || strings.Contains(s, "'/F") || strings.Contains(s, "*J'") || strings.Contains(s, "3'-") {
		return true
	}
	return false
}

// defaultArabicNodeName returns a clean Arabic display name for a node ID
func defaultArabicNodeName(cleanID string) string {
	switch cleanID {
	case "canada-node-1":
		return "حاسبة كندا 1 (الأساسية)"
	case "canada-node-2":
		return "حاسبة كندا 2 (الاحتياطية)"
	case "canada-node-3":
		return "حاسبة كندا 3"
	case "canada-node-4":
		return "حاسبة كندا 4"
	default:
		return "حاسبة كندا (" + cleanID + ")"
	}
}

// sanitizeNodeID ensures worker ID and Arabic name are clean, URL-unescaped, and free of header mojibake
func sanitizeNodeID(id, name string) (string, string) {
	cleanID := strings.TrimSpace(strings.ToLower(id))
	if cleanID == "" {
		cleanID = "canada-node-1"
	}
	cleanName := strings.TrimSpace(name)
	if strings.Contains(cleanName, "%") {
		if unescaped, err := url.QueryUnescape(cleanName); err == nil && strings.TrimSpace(unescaped) != "" {
			cleanName = strings.TrimSpace(unescaped)
		}
	}
	if cleanName == "" || isCorruptedNodeName(cleanName) {
		cleanName = defaultArabicNodeName(cleanID)
	}
	return cleanID, cleanName
}

// TouchWaiterLease marks that a customer is actively waiting for this VIN right now (TTL 60s)
func (s *WorkerGuardService) TouchWaiterLease(vin string) {
	if bootstrap.Redis == nil || vin == "" {
		return
	}
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))
	bootstrap.Redis.Set(context.Background(), "canada:waiter:"+cleanVin, "1", 60*time.Second)
}

// IsVinCancelled checks if the customer explicitly closed/refreshed their page and cancelled this VIN
func (s *WorkerGuardService) IsVinCancelled(vin string) bool {
	if bootstrap.Redis == nil || vin == "" {
		return false
	}
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))
	val, err := bootstrap.Redis.Get(context.Background(), "canada:cancelled:"+cleanVin).Result()
	return err == nil && val == "1"
}

// CancelWaiterAndQueue immediately removes one or more VINs from the waiting queue when a user closes their browser page
func (s *WorkerGuardService) CancelWaiterAndQueue(vins []string) int {
	if bootstrap.Redis == nil || len(vins) == 0 {
		return 0
	}
	ctx := context.Background()
	removed := 0
	for _, rawVin := range vins {
		cleanVin := strings.ToUpper(strings.TrimSpace(rawVin))
		if cleanVin == "" {
			continue
		}
		bootstrap.Redis.Set(ctx, "canada:cancelled:"+cleanVin, "1", 20*time.Second)
		bootstrap.Redis.Del(ctx, "canada:waiter:"+cleanVin)
		// Only remove from waiting queue if not already completed
		if state, _ := bootstrap.Redis.Get(ctx, "canada:task:state:"+cleanVin).Result(); state != "DONE" {
			bootstrap.Redis.Del(ctx, "canada:task:state:"+cleanVin)
		}
		if n, err := bootstrap.Redis.LRem(ctx, "canada:tasks:queue", 0, cleanVin).Result(); err == nil && n > 0 {
			removed += int(n)
		}
	}
	return removed
}

// HasActiveWaiter returns true if a customer browser or active HTTP request is still waiting for this VIN
func (s *WorkerGuardService) HasActiveWaiter(vin string) bool {
	if bootstrap.Redis == nil || vin == "" {
		return true
	}
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))
	if s.IsVinCancelled(cleanVin) {
		return false
	}
	val, err := bootstrap.Redis.Get(context.Background(), "canada:waiter:"+cleanVin).Result()
	return err == nil && val == "1"
}

// RegisterNode records worker node in registry set and sets metadata
func (s *WorkerGuardService) RegisterNode(nodeID, nodeName string) {
	if bootstrap.Redis == nil {
		return
	}
	ctx := context.Background()
	cleanID, cleanName := sanitizeNodeID(nodeID, nodeName)
	bootstrap.Redis.SAdd(ctx, "canada:workers:registry", cleanID)
	if nodeName != "" && !isCorruptedNodeName(nodeName) {
		bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":name", cleanName, 30*24*time.Hour)
	} else {
		existing, _ := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":name").Result()
		if existing == "" || isCorruptedNodeName(existing) {
			bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":name", defaultArabicNodeName(cleanID), 30*24*time.Hour)
		}
	}
}

// RecordHeartbeat logs a ping from Canada Worker (legacy global + per-node)
func (s *WorkerGuardService) RecordHeartbeat(nodeID, nodeName string) {
	if bootstrap.Redis == nil {
		return
	}
	ctx := context.Background()
	now := time.Now().Unix()

	// Global legacy key for backward compatibility
	bootstrap.Redis.Set(ctx, "canada:last_heartbeat", now, 3*time.Minute)

	cleanID, cleanName := sanitizeNodeID(nodeID, nodeName)
	s.RegisterNode(cleanID, cleanName)
	bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":last_heartbeat", now, 3*time.Minute)
}

// IsConnected checks if worker sent heartbeat within last 60 seconds (checks any online node)
func (s *WorkerGuardService) IsConnected() bool {
	if bootstrap.Redis == nil {
		return false
	}
	ctx := context.Background()

	// Check any registered node
	nodes, err := bootstrap.Redis.SMembers(ctx, "canada:workers:registry").Result()
	if err == nil && len(nodes) > 0 {
		now := time.Now().Unix()
		for _, n := range nodes {
			val, nErr := bootstrap.Redis.Get(ctx, "canada:worker:"+n+":last_heartbeat").Int64()
			if nErr == nil && (now-val) <= 60 {
				return true
			}
		}
	}

	// Fallback legacy check
	val, err := bootstrap.Redis.Get(ctx, "canada:last_heartbeat").Int64()
	if err != nil {
		return false
	}
	return (time.Now().Unix() - val) <= 60
}

// IsNodeConnected checks if a specific worker node is connected
func (s *WorkerGuardService) IsNodeConnected(nodeID string) bool {
	if bootstrap.Redis == nil {
		return false
	}
	cleanID, _ := sanitizeNodeID(nodeID, "")
	val, err := bootstrap.Redis.Get(context.Background(), "canada:worker:"+cleanID+":last_heartbeat").Int64()
	if err != nil {
		return false
	}
	return (time.Now().Unix() - val) <= 60
}

// RequestImmediateInspect sets a flag telling Canada Worker to take a fresh JPEG screenshot on next poll
func (s *WorkerGuardService) RequestImmediateInspect(nodeID string) {
	if bootstrap.Redis == nil {
		return
	}
	ctx := context.Background()
	bootstrap.Redis.Set(ctx, "canada:inspect_requested", "1", 1*time.Minute)
	if nodeID != "" {
		cleanID, _ := sanitizeNodeID(nodeID, "")
		bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":inspect_requested", "1", 1*time.Minute)
	}
}

// ConsumeInspectRequest checks and clears the inspect flag (checks node-specific or global)
func (s *WorkerGuardService) ConsumeInspectRequest(nodeID string) bool {
	if bootstrap.Redis == nil {
		return false
	}
	ctx := context.Background()
	if nodeID != "" {
		cleanID, _ := sanitizeNodeID(nodeID, "")
		val, err := bootstrap.Redis.GetDel(ctx, "canada:worker:"+cleanID+":inspect_requested").Result()
		if err == nil && val == "1" {
			return true
		}
	}
	val, err := bootstrap.Redis.GetDel(ctx, "canada:inspect_requested").Result()
	return err == nil && val == "1"
}

// SaveTelemetry stores the browser state and updates the latest screenshot if included (node-aware)
func (s *WorkerGuardService) SaveTelemetry(t *WorkerTelemetry, nodeID, nodeName string) {
	cleanID, cleanName := sanitizeNodeID(nodeID, nodeName)
	s.RecordHeartbeat(cleanID, cleanName)
	if bootstrap.Redis == nil || t == nil {
		return
	}
	ctx := context.Background()

	// Self-Healing: if browser is healthy, logged in, and showing the VIN input with zero security challenge,
	// automatically clear any prior non-manual circuit breaker state
	if t.ChromeConnected && t.SessionActive && t.VinInputFound && !t.HasChallenge && !t.IsLoginPage {
		if active, reason := s.IsNodeCircuitBreakerActive(cleanID); active {
			if !strings.Contains(reason, "MANUAL_STOP") {
				_ = s.ResetNodeCircuitBreaker(cleanID)
			}
		}
	}

	if t.ScreenshotBase64 != "" {
		bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":last_screenshot", t.ScreenshotBase64, 24*time.Hour)
		bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":last_screenshot_at", time.Now().Unix(), 24*time.Hour)
		bootstrap.Redis.Set(ctx, "canada:last_screenshot", t.ScreenshotBase64, 24*time.Hour)
		bootstrap.Redis.Set(ctx, "canada:last_screenshot_at", time.Now().Unix(), 24*time.Hour)
		tCopy := *t
		tCopy.ScreenshotBase64 = ""
		if b, err := json.Marshal(&tCopy); err == nil {
			bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":telemetry", b, 10*time.Minute)
			bootstrap.Redis.Set(ctx, "canada:telemetry", b, 10*time.Minute)
		}
		return
	}

	if b, err := json.Marshal(t); err == nil {
		bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":telemetry", b, 10*time.Minute)
		bootstrap.Redis.Set(ctx, "canada:telemetry", b, 10*time.Minute)
	}
}

// SaveProgress updates live step progress for the active VIN pull (node-aware and per-VIN inflight-aware)
func (s *WorkerGuardService) SaveProgress(p *WorkerProgress, nodeID, nodeName string) {
	cleanID, cleanName := sanitizeNodeID(nodeID, nodeName)
	s.RecordHeartbeat(cleanID, cleanName)
	if bootstrap.Redis == nil || p == nil {
		return
	}
	ctx := context.Background()
	p.UpdatedAt = time.Now()
	p.WorkerID = cleanID
	p.WorkerName = cleanName

	cleanVin := strings.ToUpper(strings.TrimSpace(p.VIN))
	if cleanVin != "" {
		p.VIN = cleanVin
		bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":active_vin", cleanVin, 90*time.Second)
		bootstrap.Redis.Set(ctx, "canada:active_vin", cleanVin, 90*time.Second)
		bootstrap.Redis.Set(ctx, "canada:task:state:"+cleanVin, "ACTIVE", 3*time.Minute)
		bootstrap.Redis.SAdd(ctx, "canada:inflight:set", cleanVin)
	}

	if p.ScreenshotBase64 != "" {
		bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":last_screenshot", p.ScreenshotBase64, 24*time.Hour)
		bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":last_screenshot_at", time.Now().Unix(), 24*time.Hour)
		bootstrap.Redis.Set(ctx, "canada:last_screenshot", p.ScreenshotBase64, 24*time.Hour)
		bootstrap.Redis.Set(ctx, "canada:last_screenshot_at", time.Now().Unix(), 24*time.Hour)
		p.ScreenshotBase64 = ""
	}

	if b, err := json.Marshal(p); err == nil {
		bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":active_progress", b, 90*time.Second)
		bootstrap.Redis.Set(ctx, "canada:active_progress", b, 90*time.Second)
		if cleanVin != "" {
			bootstrap.Redis.Set(ctx, "canada:inflight:"+cleanVin, b, 90*time.Second)
		}
	}
}

// getNodeTargetCooldown calculates the smart adaptive cooldown for a specific node:
// - 10s after a single-VIN pull (fast response!)
// - 15s after a 2-VIN batch pull
// - Anti-Burst Shield: adds +7s if the same node pulled 6-7 reports in 2 mins, or +14s if >=8 reports in 2 mins
func (s *WorkerGuardService) getNodeTargetCooldown(ctx context.Context, cleanID string) int {
	lastBatch, _ := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":last_batch_size").Int()
	burst2m, _ := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":burst_2m").Int()

	targetCooldown := 10
	if lastBatch >= 2 {
		targetCooldown = 15
	}
	if s.minCooldownSec > targetCooldown && lastBatch >= 2 {
		targetCooldown = s.minCooldownSec
	}

	if burst2m >= 8 {
		targetCooldown += 14
	} else if burst2m >= 6 {
		targetCooldown += 7
	}
	return targetCooldown
}

// GetNodeCooldownRemainingSec returns cooldown seconds left for a specific node
func (s *WorkerGuardService) GetNodeCooldownRemainingSec(nodeID string) int {
	if bootstrap.Redis == nil {
		return 0
	}
	ctx := context.Background()
	cleanID, _ := sanitizeNodeID(nodeID, "")
	lastEnd, err := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":last_pull_ended_at").Int64()
	if err != nil {
		return 0
	}
	targetCooldown := s.getNodeTargetCooldown(ctx, cleanID)
	elapsed := int(time.Now().Unix() - lastEnd)
	if elapsed < targetCooldown {
		return targetCooldown - elapsed
	}
	return 0
}

// GetCooldownRemainingSec returns seconds left in mandatory anti-ban rest (minimum cooldown across online healthy nodes!)
func (s *WorkerGuardService) GetCooldownRemainingSec() int {
	if bootstrap.Redis == nil {
		return 0
	}
	ctx := context.Background()

	nodes, err := bootstrap.Redis.SMembers(ctx, "canada:workers:registry").Result()
	if err == nil && len(nodes) > 0 {
		now := time.Now().Unix()
		minCooldown := -1
		for _, n := range nodes {
			cleanID := strings.TrimSpace(n)
			if cleanID == "" {
				continue
			}
			hb, hErr := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":last_heartbeat").Int64()
			if hErr != nil || (now-hb) > 60 {
				continue
			}
			if cb, _ := s.IsNodeCircuitBreakerActive(cleanID); cb {
				continue
			}
			cd := s.GetNodeCooldownRemainingSec(cleanID)
			if minCooldown == -1 || cd < minCooldown {
				minCooldown = cd
			}
		}
		if minCooldown >= 0 {
			return minCooldown
		}
	}

	lastEnd, err := bootstrap.Redis.Get(ctx, "canada:last_pull_ended_at").Int64()
	if err != nil {
		return 0
	}
	elapsed := int(time.Now().Unix() - lastEnd)
	if elapsed < s.minCooldownSec {
		return s.minCooldownSec - elapsed
	}
	return 0
}

// CountHealthyOnlineNodes returns the number of currently online nodes that do not have their circuit breaker tripped.
// If excludeNodeID is provided, it counts other healthy online nodes (for failover decisions).
func (s *WorkerGuardService) CountHealthyOnlineNodes(excludeNodeID string) int {
	if bootstrap.Redis == nil {
		return 0
	}
	ctx := context.Background()
	nodes, err := bootstrap.Redis.SMembers(ctx, "canada:workers:registry").Result()
	if err != nil || len(nodes) == 0 {
		return 0
	}
	now := time.Now().Unix()
	count := 0
	for _, n := range nodes {
		cleanID := strings.TrimSpace(n)
		if cleanID == "" || (excludeNodeID != "" && cleanID == excludeNodeID) {
			continue
		}
		hb, hErr := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":last_heartbeat").Int64()
		if hErr != nil || (now-hb) > 60 {
			continue
		}
		if cb, _ := s.IsNodeCircuitBreakerActive(cleanID); cb {
			continue
		}
		count++
	}
	return count
}

// GetAdminMaxDailyPulls returns the Admin user's configured daily limit (or total limit) from UserQuota in DB
func (s *WorkerGuardService) GetAdminMaxDailyPulls() int {
	if bootstrap.DB != nil {
		var quota models.UserQuota
		err := bootstrap.DB.Table("user_quotas").
			Select("user_quotas.daily_limit, user_quotas.total_limit").
			Joins("JOIN users ON users.id = user_quotas.user_id").
			Where("users.role = ?", "admin").
			Order("users.id ASC").
			First(&quota).Error
		if err == nil {
			if quota.DailyLimit > 0 {
				return quota.DailyLimit
			}
			if quota.TotalLimit > 0 {
				return quota.TotalLimit
			}
		}
	}
	return s.maxDailyPulls
}

// GetDailyPulls returns count of reports fetched today (globally)
func (s *WorkerGuardService) GetDailyPulls() int {
	if bootstrap.Redis == nil {
		return 0
	}
	ctx := context.Background()
	val, err := bootstrap.Redis.Get(ctx, s.getDailyKey()).Int()
	if err != nil {
		return 0
	}
	return val
}

// GetNodeDailyPulls returns count of reports fetched today by a specific node
func (s *WorkerGuardService) GetNodeDailyPulls(nodeID string) int {
	if bootstrap.Redis == nil {
		return 0
	}
	ctx := context.Background()
	cleanID, _ := sanitizeNodeID(nodeID, "")
	val, err := bootstrap.Redis.Get(ctx, s.getNodeDailyKey(cleanID)).Int()
	if err != nil {
		return 0
	}
	return val
}

// IsNodeCircuitBreakerActive checks if safety tripwire tripped for a specific node
func (s *WorkerGuardService) IsNodeCircuitBreakerActive(nodeID string) (bool, string) {
	if bootstrap.Redis == nil {
		return false, ""
	}
	ctx := context.Background()
	cleanID, _ := sanitizeNodeID(nodeID, "")
	active, err := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":circuit_breaker_active").Bool()
	if err != nil || !active {
		return false, ""
	}
	reason, _ := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":circuit_breaker_reason").Result()
	return true, reason
}

// IsCircuitBreakerActive checks if safety tripwire tripped (global or if ALL online nodes are tripped)
func (s *WorkerGuardService) IsCircuitBreakerActive() (bool, string) {
	if bootstrap.Redis == nil {
		return false, ""
	}
	ctx := context.Background()
	active, err := bootstrap.Redis.Get(ctx, "canada:circuit_breaker_active").Bool()
	if err == nil && active {
		reason, _ := bootstrap.Redis.Get(ctx, "canada:circuit_breaker_reason").Result()
		return true, reason
	}
	return false, ""
}

// TriggerNodeCircuitBreaker trips safety alarm for a specific node
func (s *WorkerGuardService) TriggerNodeCircuitBreaker(nodeID, errorCode, errorMsg string) {
	if bootstrap.Redis == nil {
		return
	}
	ctx := context.Background()
	cleanID, _ := sanitizeNodeID(nodeID, "")
	bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":circuit_breaker_active", true, 0)
	reason := fmt.Sprintf("[%s] %s", errorCode, errorMsg)
	bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":circuit_breaker_reason", reason, 0)
}

// TriggerCircuitBreaker safely stops automated pulls upon global freeze
func (s *WorkerGuardService) TriggerCircuitBreaker(errorCode, errorMsg string) {
	if bootstrap.Redis == nil {
		return
	}
	ctx := context.Background()
	bootstrap.Redis.Set(ctx, "canada:circuit_breaker_active", true, 0)
	reason := fmt.Sprintf("[%s] %s", errorCode, errorMsg)
	bootstrap.Redis.Set(ctx, "canada:circuit_breaker_reason", reason, 0)
}

// ResetNodeCircuitBreaker clears alarm for a specific node (and clears global breaker if it was blocking)
func (s *WorkerGuardService) ResetNodeCircuitBreaker(nodeID string) error {
	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()
	cleanID, _ := sanitizeNodeID(nodeID, "")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":circuit_breaker_active")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":circuit_breaker_reason")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_progress")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_vin")
	bootstrap.Redis.Del(ctx, "canada:circuit_breaker_active")
	bootstrap.Redis.Del(ctx, "canada:circuit_breaker_reason")
	return nil
}

// EmergencyStopNode allows operator to stop a specific worker node
func (s *WorkerGuardService) EmergencyStopNode(nodeID string) error {
	cleanID, _ := sanitizeNodeID(nodeID, "")
	s.TriggerNodeCircuitBreaker(cleanID, "MANUAL_STOP", "تم إيقاف هذه العقدة يدوياً من غرفة العمليات")
	return nil
}

// DeleteNode removes a stale/decommissioned node from the registry
func (s *WorkerGuardService) DeleteNode(nodeID string) error {
	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()
	cleanID, _ := sanitizeNodeID(nodeID, "")
	bootstrap.Redis.SRem(ctx, "canada:workers:registry", cleanID)
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":last_heartbeat")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":name")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":circuit_breaker_active")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":circuit_breaker_reason")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":telemetry")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_progress")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_vin")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":last_screenshot")
	return nil
}

// EmergencyStop allows operator to manually trip the circuit breaker and clear pending queue
func (s *WorkerGuardService) EmergencyStop() error {
	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()
	bootstrap.Redis.Del(ctx, "canada:tasks:queue")
	bootstrap.Redis.Del(ctx, "canada:active_progress")
	bootstrap.Redis.Del(ctx, "canada:active_vin")
	bootstrap.Redis.Del(ctx, "canada:inflight:set")
	s.TriggerCircuitBreaker("MANUAL_STOP", "تم إيقاف السحب الآلي يدوياً من غرفة العمليات")
	return nil
}

// ResetCircuitBreaker clears alarm after manual intervention across global AND all registered nodes
func (s *WorkerGuardService) ResetCircuitBreaker() error {
	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()
	bootstrap.Redis.Del(ctx, "canada:circuit_breaker_active")
	bootstrap.Redis.Del(ctx, "canada:circuit_breaker_reason")
	bootstrap.Redis.Del(ctx, "canada:active_progress")
	bootstrap.Redis.Del(ctx, "canada:active_vin")

	if nodes, err := bootstrap.Redis.SMembers(ctx, "canada:workers:registry").Result(); err == nil {
		for _, n := range nodes {
			cleanID := strings.TrimSpace(n)
			if cleanID != "" {
				bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":circuit_breaker_active")
				bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":circuit_breaker_reason")
				bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_progress")
				bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_vin")
			}
		}
	}
	return nil
}

// GetAllActiveTasks returns all currently active VIN pulls across all worker nodes and multi-tab batches
func (s *WorkerGuardService) GetAllActiveTasks() []WorkerProgress {
	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()
	var activeList []WorkerProgress
	seen := make(map[string]bool)

	if members, err := bootstrap.Redis.SMembers(ctx, "canada:inflight:set").Result(); err == nil {
		for _, vin := range members {
			cleanVin := strings.ToUpper(strings.TrimSpace(vin))
			if cleanVin == "" {
				continue
			}
			if pBytes, gErr := bootstrap.Redis.Get(ctx, "canada:inflight:"+cleanVin).Bytes(); gErr == nil && len(pBytes) > 0 {
				var p WorkerProgress
				if json.Unmarshal(pBytes, &p) == nil && time.Since(p.UpdatedAt) < 95*time.Second {
					activeList = append(activeList, p)
					seen[cleanVin] = true
					continue
				}
			}
			// Stale member cleanup
			bootstrap.Redis.SRem(ctx, "canada:inflight:set", vin)
		}
	}

	// Fallback check on global active_progress
	if pBytes, err := bootstrap.Redis.Get(ctx, "canada:active_progress").Bytes(); err == nil && len(pBytes) > 0 {
		var p WorkerProgress
		if json.Unmarshal(pBytes, &p) == nil && p.VIN != "" && !seen[p.VIN] && time.Since(p.UpdatedAt) < 90*time.Second {
			activeList = append(activeList, p)
		}
	}

	return activeList
}

// IsVinActive checks if the given VIN is currently being harvested right now by ANY worker node
func (s *WorkerGuardService) IsVinActive(vin string) (bool, *WorkerProgress) {
	if bootstrap.Redis == nil || vin == "" {
		return false, nil
	}
	ctx := context.Background()
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))

	// 1. Check per-VIN inflight key first (immune to multi-node overwrites!)
	if pBytes, err := bootstrap.Redis.Get(ctx, "canada:inflight:"+cleanVin).Bytes(); err == nil && len(pBytes) > 0 {
		var p WorkerProgress
		if json.Unmarshal(pBytes, &p) == nil && time.Since(p.UpdatedAt) < 95*time.Second {
			return true, &p
		}
	}

	// 2. Fallback global check
	if pBytes, err := bootstrap.Redis.Get(ctx, "canada:active_progress").Bytes(); err == nil && len(pBytes) > 0 {
		var p WorkerProgress
		if json.Unmarshal(pBytes, &p) == nil && p.VIN == cleanVin && time.Since(p.UpdatedAt) < 90*time.Second {
			return true, &p
		}
	}

	if activeVin, err := bootstrap.Redis.Get(ctx, "canada:active_vin").Result(); err == nil && activeVin == cleanVin {
		return true, nil
	}

	return false, nil
}

// IsVinPendingOrActive returns true if the VIN is either waiting in the FIFO queue or actively being harvested
func (s *WorkerGuardService) IsVinPendingOrActive(vin string) bool {
	if bootstrap.Redis == nil || vin == "" {
		return false
	}
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))
	if active, _ := s.IsVinActive(cleanVin); active {
		return true
	}
	ctx := context.Background()

	// Check atomic task state key (prevents 48s boundary race condition)
	if state, err := bootstrap.Redis.Get(ctx, "canada:task:state:"+cleanVin).Result(); err == nil {
		if state == "QUEUED" || state == "ACTIVE" {
			return true
		}
	}

	list, err := bootstrap.Redis.LRange(ctx, "canada:tasks:queue", 0, -1).Result()
	if err == nil {
		for _, v := range list {
			if v == cleanVin {
				return true
			}
		}
	}
	return false
}

// RemoveFromQueue removes a specific VIN from the FIFO waiting queue
func (s *WorkerGuardService) RemoveFromQueue(vin string) error {
	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))
	bootstrap.Redis.Del(ctx, "canada:task:state:"+cleanVin)
	bootstrap.Redis.Del(ctx, "canada:waiter:"+cleanVin)
	return bootstrap.Redis.LRem(ctx, "canada:tasks:queue", 0, cleanVin).Err()
}

// PromoteInQueue moves a specific VIN to the front (top) of the waiting queue
func (s *WorkerGuardService) PromoteInQueue(vin string) error {
	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))
	s.TouchWaiterLease(cleanVin)
	// Remove from current position and LPUSH to the front
	bootstrap.Redis.LRem(ctx, "canada:tasks:queue", 0, cleanVin)
	return bootstrap.Redis.LPush(ctx, "canada:tasks:queue", cleanVin).Err()
}

// PurgeQueue completely clears all pending tasks in the waiting queue
func (s *WorkerGuardService) PurgeQueue() error {
	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()
	if list, err := bootstrap.Redis.LRange(ctx, "canada:tasks:queue", 0, -1).Result(); err == nil {
		for _, v := range list {
			bootstrap.Redis.Del(ctx, "canada:task:state:"+v)
		}
	}
	return bootstrap.Redis.Del(ctx, "canada:tasks:queue").Err()
}

// GetDetailedQueue returns the active VIN, active progress, all active tasks across nodes, and the ordered waiting queue
func (s *WorkerGuardService) GetDetailedQueue() (string, *WorkerProgress, []string) {
	if bootstrap.Redis == nil {
		return "", nil, nil
	}
	ctx := context.Background()
	activeTasks := s.GetAllActiveTasks()
	var activeVin string
	var activeProg *WorkerProgress
	if len(activeTasks) > 0 {
		activeVin = activeTasks[0].VIN
		activeProg = &activeTasks[0]
	} else {
		activeVin, _ = bootstrap.Redis.Get(ctx, "canada:active_vin").Result()
		if pBytes, err := bootstrap.Redis.Get(ctx, "canada:active_progress").Bytes(); err == nil && len(pBytes) > 0 {
			var p WorkerProgress
			if json.Unmarshal(pBytes, &p) == nil {
				activeProg = &p
			}
		}
	}
	list, _ := bootstrap.Redis.LRange(ctx, "canada:tasks:queue", 0, -1).Result()
	return activeVin, activeProg, list
}

// EnqueueSearchTask pushes VIN into the sequential FIFO queue with Singleflight In-Flight Deduplication & Customer Lease
func (s *WorkerGuardService) EnqueueSearchTask(vin string) error {
	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))

	// Clear any prior tab-close cancellation flag when a fresh search request arrives
	bootstrap.Redis.Del(ctx, "canada:cancelled:"+cleanVin)

	// Always refresh customer waiter lease so worker knows a live user is waiting right now
	s.TouchWaiterLease(cleanVin)

	// 1. In-Flight Singleflight Lock: if ANY worker is already harvesting this VIN right now,
	// do not push a duplicate task; both requests will receive the report simultaneously via Pub/Sub!
	if active, _ := s.IsVinActive(cleanVin); active {
		return nil
	}

	// 2. Queue Deduplication: if the VIN is already waiting in the FIFO queue, do not push a second copy
	list, err := bootstrap.Redis.LRange(ctx, "canada:tasks:queue", 0, -1).Result()
	if err == nil {
		for _, v := range list {
			if v == cleanVin {
				bootstrap.Redis.Set(ctx, "canada:task:state:"+cleanVin, "QUEUED", 10*time.Minute)
				return nil
			}
		}
	}

	bootstrap.Redis.Set(ctx, "canada:task:state:"+cleanVin, "QUEUED", 10*time.Minute)
	return bootstrap.Redis.RPush(ctx, "canada:tasks:queue", cleanVin).Err()
}

// estimateStepRemainingSec maps a WorkerProgress Step to realistic remaining harvest seconds
func estimateStepRemainingSec(p *WorkerProgress) int {
	if p == nil {
		return 12
	}
	switch p.Step {
	case "DISPATCHED", "STARTING", "pre_flight":
		return 14
	case "TYPING_VIN", "typing_vin", "pasting_vin":
		return 10
	case "SUBMITTING", "submitting_search":
		return 8
	case "WAIT_REPORT":
		return 6
	case "HARVESTING", "extracting_data":
		return 3
	default:
		return 10
	}
}

// GetQueuePosition returns the 1-based position of a VIN in the waiting/active pipeline and estimated wait seconds (multi-node aware)
func (s *WorkerGuardService) GetQueuePosition(vin string) (int, int) {
	if bootstrap.Redis == nil {
		return 0, 0
	}
	ctx := context.Background()
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))

	// Refresh waiter lease whenever customer's browser polls queue status
	s.TouchWaiterLease(cleanVin)

	// 1. Check if this VIN is actively being pulled right now (Position 1)
	if active, prog := s.IsVinActive(cleanVin); active {
		return 1, estimateStepRemainingSec(prog)
	}

	// 2. Check position in the waiting FIFO queue
	list, err := bootstrap.Redis.LRange(ctx, "canada:tasks:queue", 0, -1).Result()
	if err != nil || len(list) == 0 {
		return 0, 0
	}

	idxInQueue := -1
	for idx, v := range list {
		if v == cleanVin {
			idxInQueue = idx
			break
		}
	}

	if idxInQueue == -1 {
		return 0, 0
	}

	activeTasks := s.GetAllActiveTasks()
	activeOffset := len(activeTasks)
	pos := idxInQueue + 1 + activeOffset

	healthyNodes := s.CountHealthyOnlineNodes("")
	if healthyNodes < 1 {
		healthyNodes = 1
	}

	cooldown := s.GetCooldownRemainingSec()
	// With Multi-Tab Rapid Paste & Multi-Node farm, each batch of up to 4 VINs per node completes Stage-1 in ~10s
	effectiveWaveIdx := idxInQueue / (healthyNodes * 3)
	estimatedSec := cooldown + 10 + (effectiveWaveIdx * 16) + ((idxInQueue % 3) * 3)
	if estimatedSec < 5 {
		estimatedSec = 5
	}

	return pos, estimatedSec
}

// recoverOrphanedNodeTasks checks if any disconnected node (>35s without heartbeat) left an in-flight VIN unfinished,
// and immediately requeues that VIN to the front of the queue so a healthy online node can finish it.
func (s *WorkerGuardService) recoverOrphanedNodeTasks(ctx context.Context) {
	nodes, err := bootstrap.Redis.SMembers(ctx, "canada:workers:registry").Result()
	if err != nil || len(nodes) == 0 {
		return
	}
	now := time.Now().Unix()
	for _, n := range nodes {
		cleanID := strings.TrimSpace(n)
		if cleanID == "" {
			continue
		}
		hb, hErr := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":last_heartbeat").Int64()
		if hErr == nil && (now-hb) <= 35 {
			continue // Node is alive
		}
		// Node is silent for >35s: check if it was holding an active VIN
		orphanVin, vErr := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":active_vin").Result()
		if vErr == nil && orphanVin != "" {
			bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_vin")
			bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_progress")
			for _, singleVin := range strings.Split(orphanVin, ",") {
				v := strings.ToUpper(strings.TrimSpace(singleVin))
				if v == "" {
					continue
				}
				bootstrap.Redis.Del(ctx, "canada:inflight:"+v)
				bootstrap.Redis.SRem(ctx, "canada:inflight:set", v)
				if s.HasActiveWaiter(v) {
					if ok, _ := bootstrap.Redis.SetNX(ctx, "canada:requeued:orphan:"+v, "1", 5*time.Minute).Result(); ok {
						bootstrap.Redis.LRem(ctx, "canada:tasks:queue", 0, v)
						bootstrap.Redis.LPush(ctx, "canada:tasks:queue", v)
						bootstrap.Redis.Set(ctx, "canada:task:state:"+v, "QUEUED", 5*time.Minute)
					}
				}
			}
		}
	}
}

// PollBatchTasks gives the remote Canada node 1 to 4 VINs at once (fairly divided across online nodes)
// while automatically skipping any abandoned Zombie VINs whose customer closed the page!
func (s *WorkerGuardService) PollBatchTasks(nodeID, nodeName string, maxBatch int) ([]string, int, error) {
	cleanID, cleanName := sanitizeNodeID(nodeID, nodeName)
	s.RecordHeartbeat(cleanID, cleanName)

	if bootstrap.Redis == nil {
		return nil, 0, ErrQueueEmpty
	}
	ctx := context.Background()

	// Recover any tasks stranded on a dead node before polling
	s.recoverOrphanedNodeTasks(ctx)

	// Gate 1: Check Global Circuit Breaker
	if active, reason := s.IsCircuitBreakerActive(); active {
		return nil, 0, fmt.Errorf("%w: %s", ErrCircuitBreaker, reason)
	}

	// Gate 1.5: Check Node-Specific Circuit Breaker
	if active, reason := s.IsNodeCircuitBreakerActive(cleanID); active {
		return nil, 0, fmt.Errorf("قاطع الحماية مفعل لهذه العقدة (%s): %s", cleanName, reason)
	}

	// Gate 2: Daily limit is governed by each user's / Admin's UserQuota in PostgreSQL (no artificial 45 node lock!)

	// Gate 3: Check Node-Specific Mandatory Cooldown
	nodeCooldown := s.GetNodeCooldownRemainingSec(cleanID)
	if nodeCooldown > 0 {
		return nil, nodeCooldown, ErrCooldownActive
	}

	// Calculate fair batch size across healthy online nodes (max 2 per wave per PC for 100% human safety):
	// - 1 PC facing 2+ tasks takes 2 VINs with a natural 2.2s gap
	// - 2 PCs facing 4 tasks take 2 VINs each simultaneously!
	// - Anti-Burst Shield: if this node already pulled >=6 reports in the last 2 minutes, step down to 1 VIN at a time
	if maxBatch < 1 {
		maxBatch = 1
	}
	if maxBatch > 2 {
		maxBatch = 2
	}
	qLen, _ := bootstrap.Redis.LLen(ctx, "canada:tasks:queue").Result()
	if qLen == 0 {
		return nil, 0, ErrQueueEmpty
	}

	healthyNodes := s.CountHealthyOnlineNodes("")
	if healthyNodes < 1 {
		healthyNodes = 1
	}
	fairBatch := int((int(qLen) + healthyNodes - 1) / healthyNodes)
	if fairBatch < 1 {
		fairBatch = 1
	}
	if fairBatch > maxBatch {
		fairBatch = maxBatch
	}
	if burst2m, _ := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":burst_2m").Int(); burst2m >= 6 {
		fairBatch = 1
	}

	// Gate 4: Pop valid VINs (skipping Zombie tasks where customer closed browser)
	var batchVins []string
	for len(batchVins) < fairBatch {
		vin, err := bootstrap.Redis.LPop(ctx, "canada:tasks:queue").Result()
		if err != nil {
			break
		}
		cleanVin := strings.ToUpper(strings.TrimSpace(vin))
		if cleanVin == "" {
			continue
		}
		// Zombie Queue Protection: if no customer is waiting for this VIN anymore, discard it immediately!
		if !s.HasActiveWaiter(cleanVin) {
			bootstrap.Redis.Del(ctx, "canada:task:state:"+cleanVin)
			continue
		}
		batchVins = append(batchVins, cleanVin)
	}

	if len(batchVins) == 0 {
		return nil, 0, ErrQueueEmpty
	}

	primaryVin := batchVins[0]
	joinedVins := strings.Join(batchVins, ",")
	bootstrap.Redis.Set(ctx, "canada:active_vin", primaryVin, 90*time.Second)
	bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":active_vin", joinedVins, 90*time.Second)
	bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":last_batch_size", len(batchVins), 10*time.Minute)

	now := time.Now()
	for _, v := range batchVins {
		bootstrap.Redis.Set(ctx, "canada:vin_node_map:"+v, cleanID, 30*time.Minute)
		bootstrap.Redis.Set(ctx, "canada:task:state:"+v, "ACTIVE", 3*time.Minute)
		bootstrap.Redis.SAdd(ctx, "canada:inflight:set", v)

		initProg := WorkerProgress{
			WorkerID:   cleanID,
			WorkerName: cleanName,
			VIN:        v,
			Step:       "DISPATCHED",
			StepAr:     fmt.Sprintf("تم توجيه الطلب إلى %s لبدء الفحص الفوري", cleanName),
			UpdatedAt:  now,
		}
		if b, mErr := json.Marshal(&initProg); mErr == nil {
			bootstrap.Redis.Set(ctx, "canada:inflight:"+v, b, 90*time.Second)
			if v == primaryVin {
				bootstrap.Redis.Set(ctx, "canada:active_progress", b, 90*time.Second)
				bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":active_progress", b, 90*time.Second)
			}
		}
	}

	return batchVins, 0, nil
}

// PollNextTask backward-compatible single-VIN wrapper around PollBatchTasks
func (s *WorkerGuardService) PollNextTask(nodeID, nodeName string) (string, int, error) {
	vins, cd, err := s.PollBatchTasks(nodeID, nodeName, 1)
	if err != nil || len(vins) == 0 {
		return "", cd, err
	}
	return vins[0], cd, nil
}

// TouchNodeCooldownWithJitter sets the anti-ban cooldown timestamp for a specific node with natural human jitter
func (s *WorkerGuardService) TouchNodeCooldownWithJitter(nodeID string) {
	if bootstrap.Redis == nil {
		return
	}
	ctx := context.Background()
	cleanID, _ := sanitizeNodeID(nodeID, "")
	jitter := int64(rand.Intn(4))
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_progress")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_vin")
	bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":last_pull_ended_at", time.Now().Unix()-jitter, 0)

	// Also update global legacy
	bootstrap.Redis.Del(ctx, "canada:active_progress")
	bootstrap.Redis.Del(ctx, "canada:active_vin")
	bootstrap.Redis.Set(ctx, "canada:last_pull_ended_at", time.Now().Unix()-jitter, 0)
}

// TouchCooldownWithJitter sets legacy cooldown
func (s *WorkerGuardService) TouchCooldownWithJitter() {
	s.TouchNodeCooldownWithJitter("canada-node-1")
}

// ClearNodeActiveVin clears only the finished VIN from the node's active tracking without resetting cooldown twice
func (s *WorkerGuardService) ClearNodeActiveVin(nodeID, vin string) {
	if bootstrap.Redis == nil {
		return
	}
	ctx := context.Background()
	cleanID, _ := sanitizeNodeID(nodeID, "")
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))
	if cleanVin != "" {
		bootstrap.Redis.Del(ctx, "canada:inflight:"+cleanVin)
		bootstrap.Redis.SRem(ctx, "canada:inflight:set", cleanVin)
		bootstrap.Redis.Set(ctx, "canada:task:state:"+cleanVin, "DONE", 10*time.Minute)
	}
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_progress")
	bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_vin")
	bootstrap.Redis.Del(ctx, "canada:active_progress")
	bootstrap.Redis.Del(ctx, "canada:active_vin")
}

// CompleteTask handles Stage-1 or final response from Canada node (multi-node aware + automatic failover requeue!)
func (s *WorkerGuardService) CompleteTask(res *WorkerTaskResult) error {
	cleanID, cleanName := sanitizeNodeID(res.WorkerID, res.WorkerName)
	res.WorkerID = cleanID
	res.WorkerName = cleanName
	res.VIN = strings.ToUpper(strings.TrimSpace(res.VIN))

	s.RecordHeartbeat(cleanID, cleanName)

	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()

	// Save screenshot if sent with completion
	if res.ScreenshotBase64 != "" {
		bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":last_screenshot", res.ScreenshotBase64, 24*time.Hour)
		bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":last_screenshot_at", time.Now().Unix(), 24*time.Hour)
		bootstrap.Redis.Set(ctx, "canada:last_screenshot", res.ScreenshotBase64, 24*time.Hour)
		bootstrap.Redis.Set(ctx, "canada:last_screenshot_at", time.Now().Unix(), 24*time.Hour)
	}

	// Handle Errors & Smart Multi-Node Failover Requeue
	if res.ErrorCode != "" {
		bootstrap.Redis.Del(ctx, "canada:inflight:"+res.VIN)
		bootstrap.Redis.SRem(ctx, "canada:inflight:set", res.VIN)
		bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_progress")
		bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_vin")

		// Trip ONLY this node's breaker if CAPTCHA or SESSION_EXPIRED
		if res.ErrorCode == "CAPTCHA_DETECTED" || res.ErrorCode == "SESSION_EXPIRED" {
			s.TriggerNodeCircuitBreaker(cleanID, res.ErrorCode, res.ErrorMessage)
		}

		// If the error is NOT an explicit Carfax "Invalid/No Records VIN", check if we can seamlessly requeue!
		if res.ErrorCode != "VIN_NOT_FOUND_ON_CARFAX" && s.HasActiveWaiter(res.VIN) {
			canFailover := false
			if res.ErrorCode == "CAPTCHA_DETECTED" || res.ErrorCode == "SESSION_EXPIRED" || res.ErrorCode == "CHROME_NOT_CONNECTED" {
				// Requeue if there is another healthy online node ready to take over
				canFailover = s.CountHealthyOnlineNodes(cleanID) > 0
			} else {
				// Transient browser/tab/timeout hiccup: any healthy node (including this one) can retry once immediately
				canFailover = s.CountHealthyOnlineNodes("") > 0
			}

			if canFailover {
				if ok, _ := bootstrap.Redis.SetNX(ctx, "canada:requeued:"+res.VIN, cleanID, 4*time.Minute).Result(); ok {
					// Clear cooldown on this node if it was just a transient DOM/tab hiccup so it can retry immediately
					if res.ErrorCode != "CAPTCHA_DETECTED" && res.ErrorCode != "SESSION_EXPIRED" {
						bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":last_pull_ended_at")
					}
					bootstrap.Redis.LRem(ctx, "canada:tasks:queue", 0, res.VIN)
					bootstrap.Redis.LPush(ctx, "canada:tasks:queue", res.VIN)
					bootstrap.Redis.Set(ctx, "canada:task:state:"+res.VIN, "QUEUED", 5*time.Minute)
					// Do NOT publish error to Pub/Sub — let WaitForReport seamlessly receive the retry result!
					return nil
				}
			}
		}

		bootstrap.Redis.Set(ctx, "canada:task:state:"+res.VIN, "FAILED", 2*time.Minute)
	} else {
		// SUCCESS!
		// If Stage 1 completed without RawHtmlEn, Stage 2 continues in background for ~2-3s
		if res.RawHtmlEn == "" {
			bootstrap.Redis.Expire(ctx, "canada:inflight:"+res.VIN, 15*time.Second)
			bootstrap.Redis.Expire(ctx, "canada:active_vin", 15*time.Second)
			bootstrap.Redis.Expire(ctx, "canada:active_progress", 15*time.Second)
			bootstrap.Redis.Expire(ctx, "canada:worker:"+cleanID+":active_vin", 15*time.Second)
			bootstrap.Redis.Expire(ctx, "canada:worker:"+cleanID+":active_progress", 15*time.Second)
		} else {
			bootstrap.Redis.Del(ctx, "canada:inflight:"+res.VIN)
			bootstrap.Redis.SRem(ctx, "canada:inflight:set", res.VIN)
			bootstrap.Redis.Del(ctx, "canada:active_progress")
			bootstrap.Redis.Del(ctx, "canada:active_vin")
			bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_progress")
			bootstrap.Redis.Del(ctx, "canada:worker:"+cleanID+":active_vin")
		}
		bootstrap.Redis.Set(ctx, "canada:task:state:"+res.VIN, "DONE", 10*time.Minute)

		// Increment daily counter, 2-min burst shield counter, & set cooldown ONLY ONCE per VIN (using SetNX lock per VIN per 2 minutes)
		if counted, _ := bootstrap.Redis.SetNX(ctx, "canada:counted:"+res.VIN, cleanID, 2*time.Minute).Result(); counted {
			jitter := int64(rand.Intn(3))
			burstKey := "canada:worker:" + cleanID + ":burst_2m"
			burstCount, _ := bootstrap.Redis.Incr(ctx, burstKey).Result()
			if burstCount <= 1 {
				bootstrap.Redis.Expire(ctx, burstKey, 120*time.Second)
			} else if ttl, err := bootstrap.Redis.TTL(ctx, burstKey).Result(); err == nil && ttl < 0 {
				bootstrap.Redis.Expire(ctx, burstKey, 120*time.Second)
			}

			pipe := bootstrap.Redis.Pipeline()

			globalDailyKey := s.getDailyKey()
			pipe.Incr(ctx, globalDailyKey)
			pipe.Expire(ctx, globalDailyKey, 24*time.Hour)
			pipe.Set(ctx, "canada:last_pull_ended_at", time.Now().Unix()-jitter, 0)

			nodeDailyKey := s.getNodeDailyKey(cleanID)
			pipe.Incr(ctx, nodeDailyKey)
			pipe.Expire(ctx, nodeDailyKey, 24*time.Hour)
			pipe.Set(ctx, "canada:worker:"+cleanID+":last_pull_ended_at", time.Now().Unix()-jitter, 0)

			_, _ = pipe.Exec(ctx)
		}
	}

	// Store result in Redis if successful
	resForWaiter := *res
	resForWaiter.ScreenshotBase64 = ""
	resBytes, _ := json.Marshal(&resForWaiter)
	resultKey := fmt.Sprintf("canada:result:%s", res.VIN)
	if res.ErrorCode == "" {
		bootstrap.Redis.Set(ctx, resultKey, resBytes, 15*time.Minute)
	} else {
		bootstrap.Redis.Del(ctx, resultKey)
	}

	channel := fmt.Sprintf("canada:pubsub:%s", res.VIN)
	bootstrap.Redis.Publish(ctx, channel, resBytes)

	return nil
}

// WaitForReport waits synchronously for Canada worker to deliver report, respecting client context cancellation
// and refreshing the customer waiter lease every second so the worker knows the customer is still waiting.
func (s *WorkerGuardService) WaitForReport(reqCtx context.Context, vin string, timeout time.Duration) (*WorkerTaskResult, error) {
	if bootstrap.Redis == nil {
		return nil, errors.New("redis unavailable")
	}
	if reqCtx == nil {
		reqCtx = context.Background()
	}
	ctx := context.Background()
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))
	s.TouchWaiterLease(cleanVin)

	channel := fmt.Sprintf("canada:pubsub:%s", cleanVin)
	sub := bootstrap.Redis.Subscribe(ctx, channel)
	defer sub.Close()
	ch := sub.Channel()

	resultKey := fmt.Sprintf("canada:result:%s", cleanVin)
	checkCachedResult := func() *WorkerTaskResult {
		val, err := bootstrap.Redis.Get(ctx, resultKey).Result()
		if err == nil && val != "" {
			var res WorkerTaskResult
			if json.Unmarshal([]byte(val), &res) == nil && res.ErrorCode == "" {
				return &res
			}
			bootstrap.Redis.Del(ctx, resultKey)
		}
		return nil
	}

	if res := checkCachedResult(); res != nil {
		return res, nil
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	deadline := time.After(timeout)

	for {
		select {
		case <-reqCtx.Done():
			// Customer closed browser tab or cancelled request
			return nil, reqCtx.Err()

		case msg := <-ch:
			if msg == nil {
				return nil, ErrWorkerTimedOut
			}
			var res WorkerTaskResult
			if err := json.Unmarshal([]byte(msg.Payload), &res); err != nil {
				return nil, err
			}
			return &res, nil

		case <-ticker.C:
			if res := checkCachedResult(); res != nil {
				return res, nil
			}
			// If customer explicitly closed their browser tab (triggered pagehide/beforeunload cancel), abort and dequeue immediately!
			if s.IsVinCancelled(cleanVin) {
				_ = s.RemoveFromQueue(cleanVin)
				return nil, context.Canceled
			}
			// Otherwise keep the waiter lease alive while the HTTP search request is waiting
			s.TouchWaiterLease(cleanVin)

		case <-deadline:
			if res := checkCachedResult(); res != nil {
				return res, nil
			}
			if !s.IsVinCancelled(cleanVin) {
				s.TouchWaiterLease(cleanVin)
			}
			return nil, ErrWorkerTimedOut
		}
	}
}

// GetClusterNodes returns live status of all registered worker nodes
func (s *WorkerGuardService) GetClusterNodes() []WorkerNodeInfo {
	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()

	// Ensure at least canada-node-1 is in registry
	bootstrap.Redis.SAdd(ctx, "canada:workers:registry", "canada-node-1")

	nodeIDs, err := bootstrap.Redis.SMembers(ctx, "canada:workers:registry").Result()
	if err != nil || len(nodeIDs) == 0 {
		return nil
	}

	now := time.Now().Unix()
	var nodes []WorkerNodeInfo

	for _, n := range nodeIDs {
		cleanID := strings.TrimSpace(n)
		if cleanID == "" {
			continue
		}

		rawName, _ := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":name").Result()
		_, name := sanitizeNodeID(cleanID, rawName)
		if rawName != name {
			bootstrap.Redis.Set(ctx, "canada:worker:"+cleanID+":name", name, 30*24*time.Hour)
		}

		var lastHb time.Time
		var lastHbSecAgo int64 = 99999
		if val, hErr := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":last_heartbeat").Int64(); hErr == nil && val > 0 {
			lastHb = time.Unix(val, 0)
			lastHbSecAgo = now - val
		}

		// Also check global fallback for canada-node-1
		if cleanID == "canada-node-1" && lastHb.IsZero() {
			if gVal, gErr := bootstrap.Redis.Get(ctx, "canada:last_heartbeat").Int64(); gErr == nil && gVal > 0 {
				lastHb = time.Unix(gVal, 0)
				lastHbSecAgo = now - gVal
			}
		}

		connected := lastHbSecAgo <= 60

		cbActive, cbReason := s.IsNodeCircuitBreakerActive(cleanID)
		cooldown := s.GetNodeCooldownRemainingSec(cleanID)
		dailyPulls := s.GetNodeDailyPulls(cleanID)

		var inspectPending bool
		if ip, ipErr := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":inspect_requested").Result(); ipErr == nil && ip == "1" {
			inspectPending = true
		}

		activeVin, _ := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":active_vin").Result()
		var activeProg *WorkerProgress
		if pBytes, pErr := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":active_progress").Bytes(); pErr == nil && len(pBytes) > 0 {
			var p WorkerProgress
			if json.Unmarshal(pBytes, &p) == nil {
				activeProg = &p
			}
		}

		var telemetry *WorkerTelemetry
		if tBytes, tErr := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":telemetry").Bytes(); tErr == nil && len(tBytes) > 0 {
			var t WorkerTelemetry
			if json.Unmarshal(tBytes, &t) == nil {
				telemetry = &t
			}
		}

		lastSc, _ := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":last_screenshot").Result()
		var lastScAt time.Time
		if scAtVal, scErr := bootstrap.Redis.Get(ctx, "canada:worker:"+cleanID+":last_screenshot_at").Int64(); scErr == nil && scAtVal > 0 {
			lastScAt = time.Unix(scAtVal, 0)
		}

		nodes = append(nodes, WorkerNodeInfo{
			WorkerID:             cleanID,
			WorkerName:           name,
			Connected:            connected,
			LastHeartbeat:        lastHb,
			LastHeartbeatSecAgo:  lastHbSecAgo,
			CooldownRemainingSec: cooldown,
			DailyPulls:           dailyPulls,
			CircuitBreakerActive: cbActive,
			CircuitBreakerReason: cbReason,
			InspectPending:       inspectPending,
			ActiveVIN:            activeVin,
			ActiveProgress:       activeProg,
			Telemetry:            telemetry,
			LastScreenshotBase64: lastSc,
			LastScreenshotAt:     lastScAt,
		})
	}

	return nodes
}

// GetStatus returns the full live status, telemetry, active progress, screenshot, AND multi-node cluster status
func (s *WorkerGuardService) GetStatus() *WorkerStatus {
	ctx := context.Background()
	var qLen int
	var lastHb time.Time
	var inspectPending bool
	var telemetry *WorkerTelemetry
	var progress *WorkerProgress
	var lastScreenshot string
	var lastScreenshotAt time.Time

	if bootstrap.Redis != nil {
		l, _ := bootstrap.Redis.LLen(ctx, "canada:tasks:queue").Result()
		qLen = int(l)

		if val, err := bootstrap.Redis.Get(ctx, "canada:last_heartbeat").Int64(); err == nil && val > 0 {
			lastHb = time.Unix(val, 0)
		}

		if ip, err := bootstrap.Redis.Get(ctx, "canada:inspect_requested").Result(); err == nil && ip == "1" {
			inspectPending = true
		}

		if tBytes, err := bootstrap.Redis.Get(ctx, "canada:telemetry").Bytes(); err == nil && len(tBytes) > 0 {
			var t WorkerTelemetry
			if json.Unmarshal(tBytes, &t) == nil {
				telemetry = &t
			}
		}

		if pBytes, err := bootstrap.Redis.Get(ctx, "canada:active_progress").Bytes(); err == nil && len(pBytes) > 0 {
			var p WorkerProgress
			if json.Unmarshal(pBytes, &p) == nil {
				progress = &p
			}
		}

		if sc, err := bootstrap.Redis.Get(ctx, "canada:last_screenshot").Result(); err == nil {
			lastScreenshot = sc
		}
		if scAt, err := bootstrap.Redis.Get(ctx, "canada:last_screenshot_at").Int64(); err == nil && scAt > 0 {
			lastScreenshotAt = time.Unix(scAt, 0)
		}
	}

	cbActive, cbReason := s.IsCircuitBreakerActive()
	clusterNodes := s.GetClusterNodes()
	activeTasks := s.GetAllActiveTasks()
	if progress == nil && len(activeTasks) > 0 {
		progress = &activeTasks[0]
	}

	totalNodes := len(clusterNodes)
	onlineNodes := 0
	for _, n := range clusterNodes {
		if n.Connected {
			onlineNodes++
			// Prefer telemetry from a connected node so top bar is always accurate
			if n.Telemetry != nil {
				telemetry = n.Telemetry
			}
		}
	}

	return &WorkerStatus{
		Connected:            s.IsConnected(),
		CircuitBreakerActive: cbActive,
		CircuitBreakerReason: cbReason,
		DailyPulls:           s.GetDailyPulls(),
		MaxDailyPulls:        s.GetAdminMaxDailyPulls(),
		CooldownRemainingSec: s.GetCooldownRemainingSec(),
		QueueLength:          qLen,
		LastHeartbeat:        lastHb,
		UseMockFallback:      s.IsMockFallbackEnabled(),
		InspectPending:       inspectPending,
		Telemetry:            telemetry,
		ActiveProgress:       progress,
		ActiveTasks:          activeTasks,
		LastScreenshotBase64: lastScreenshot,
		LastScreenshotAt:     lastScreenshotAt,
		TotalNodes:           totalNodes,
		OnlineNodes:          onlineNodes,
		Nodes:                clusterNodes,
	}
}

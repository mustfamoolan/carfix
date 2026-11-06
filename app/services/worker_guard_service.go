package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"carfix/bootstrap"
	"github.com/redis/go-redis/v9"
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
	VIN              string    `json:"vin"`
	Step             string    `json:"step"`
	StepAr           string    `json:"step_ar"`
	ScreenshotBase64 string    `json:"screenshot_base64,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
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
	LastScreenshotBase64 string           `json:"last_screenshot_base64,omitempty"`
	LastScreenshotAt     time.Time        `json:"last_screenshot_at,omitempty"`
}

type WorkerTaskResult struct {
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

		maxDaily := 45
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

// RecordHeartbeat logs a ping from Canada Worker
func (s *WorkerGuardService) RecordHeartbeat() {
	if bootstrap.Redis == nil {
		return
	}
	ctx := context.Background()
	now := time.Now().Unix()
	bootstrap.Redis.Set(ctx, "canada:last_heartbeat", now, 2*time.Minute)
}

// IsConnected checks if worker sent heartbeat within last 20 seconds
func (s *WorkerGuardService) IsConnected() bool {
	if bootstrap.Redis == nil {
		return false
	}
	ctx := context.Background()
	val, err := bootstrap.Redis.Get(ctx, "canada:last_heartbeat").Int64()
	if err != nil {
		return false
	}
	return (time.Now().Unix() - val) <= 20
}

// RequestImmediateInspect sets a flag telling Canada Worker to take a fresh JPEG screenshot on next poll
func (s *WorkerGuardService) RequestImmediateInspect() {
	if bootstrap.Redis == nil {
		return
	}
	ctx := context.Background()
	bootstrap.Redis.Set(ctx, "canada:inspect_requested", "1", 1*time.Minute)
}

// ConsumeInspectRequest checks and clears the inspect flag
func (s *WorkerGuardService) ConsumeInspectRequest() bool {
	if bootstrap.Redis == nil {
		return false
	}
	ctx := context.Background()
	val, err := bootstrap.Redis.GetDel(ctx, "canada:inspect_requested").Result()
	return err == nil && val == "1"
}

// SaveTelemetry stores the browser state and updates the latest screenshot if included
func (s *WorkerGuardService) SaveTelemetry(t *WorkerTelemetry) {
	s.RecordHeartbeat()
	if bootstrap.Redis == nil || t == nil {
		return
	}
	ctx := context.Background()

	// Self-Healing: if browser is healthy, logged in, and showing the VIN input with zero security challenge,
	// automatically clear any prior non-manual circuit breaker state so users are never stuck blocked!
	if t.ChromeConnected && t.SessionActive && t.VinInputFound && !t.HasChallenge && !t.IsLoginPage {
		if active, reason := s.IsCircuitBreakerActive(); active {
			if !strings.Contains(reason, "MANUAL_STOP") {
				_ = s.ResetCircuitBreaker()
			}
		}
	}

	if t.ScreenshotBase64 != "" {
		bootstrap.Redis.Set(ctx, "canada:last_screenshot", t.ScreenshotBase64, 24*time.Hour)
		bootstrap.Redis.Set(ctx, "canada:last_screenshot_at", time.Now().Unix(), 24*time.Hour)
		// Strip screenshot from the lightweight telemetry JSON to avoid duplication
		tCopy := *t
		tCopy.ScreenshotBase64 = ""
		if b, err := json.Marshal(&tCopy); err == nil {
			bootstrap.Redis.Set(ctx, "canada:telemetry", b, 10*time.Minute)
		}
		return
	}

	if b, err := json.Marshal(t); err == nil {
		bootstrap.Redis.Set(ctx, "canada:telemetry", b, 10*time.Minute)
	}
}

// SaveProgress updates live step progress for the active VIN pull
func (s *WorkerGuardService) SaveProgress(p *WorkerProgress) {
	s.RecordHeartbeat()
	if bootstrap.Redis == nil || p == nil {
		return
	}
	ctx := context.Background()
	p.UpdatedAt = time.Now()

	if p.VIN != "" {
		bootstrap.Redis.Set(ctx, "canada:active_vin", p.VIN, 90*time.Second)
	}

	if p.ScreenshotBase64 != "" {
		bootstrap.Redis.Set(ctx, "canada:last_screenshot", p.ScreenshotBase64, 24*time.Hour)
		bootstrap.Redis.Set(ctx, "canada:last_screenshot_at", time.Now().Unix(), 24*time.Hour)
		p.ScreenshotBase64 = ""
	}

	if b, err := json.Marshal(p); err == nil {
		bootstrap.Redis.Set(ctx, "canada:active_progress", b, 2*time.Minute)
	}
}

// GetCooldownRemainingSec returns seconds left in mandatory anti-ban rest
func (s *WorkerGuardService) GetCooldownRemainingSec() int {
	if bootstrap.Redis == nil {
		return 0
	}
	ctx := context.Background()
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

// GetDailyPulls returns count of reports fetched today
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

// IsCircuitBreakerActive checks if safety tripwire tripped
func (s *WorkerGuardService) IsCircuitBreakerActive() (bool, string) {
	if bootstrap.Redis == nil {
		return false, ""
	}
	ctx := context.Background()
	active, err := bootstrap.Redis.Get(ctx, "canada:circuit_breaker_active").Bool()
	if err != nil || !active {
		return false, ""
	}
	reason, _ := bootstrap.Redis.Get(ctx, "canada:circuit_breaker_reason").Result()
	return true, reason
}

// TriggerCircuitBreaker safely stops automated pulls upon any One-Shot Freeze or security challenge
func (s *WorkerGuardService) TriggerCircuitBreaker(errorCode, errorMsg string) {
	if bootstrap.Redis == nil {
		return
	}
	ctx := context.Background()
	bootstrap.Redis.Set(ctx, "canada:circuit_breaker_active", true, 0) // Persist until manual reset
	reason := fmt.Sprintf("[%s] %s", errorCode, errorMsg)
	bootstrap.Redis.Set(ctx, "canada:circuit_breaker_reason", reason, 0)
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
	s.TriggerCircuitBreaker("MANUAL_STOP", "تم إيقاف السحب الآلي يدوياً من غرفة العمليات")
	return nil
}

// ResetCircuitBreaker clears alarm after manual intervention
func (s *WorkerGuardService) ResetCircuitBreaker() error {
	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()
	bootstrap.Redis.Del(ctx, "canada:circuit_breaker_active")
	bootstrap.Redis.Del(ctx, "canada:circuit_breaker_reason")
	bootstrap.Redis.Del(ctx, "canada:active_progress")
	bootstrap.Redis.Del(ctx, "canada:active_vin")
	return nil
}

// IsVinActive checks if the given VIN is currently being harvested right now by the worker node
func (s *WorkerGuardService) IsVinActive(vin string) (bool, *WorkerProgress) {
	if bootstrap.Redis == nil || vin == "" {
		return false, nil
	}
	ctx := context.Background()

	if pBytes, err := bootstrap.Redis.Get(ctx, "canada:active_progress").Bytes(); err == nil && len(pBytes) > 0 {
		var p WorkerProgress
		if json.Unmarshal(pBytes, &p) == nil && p.VIN == vin && time.Since(p.UpdatedAt) < 90*time.Second {
			return true, &p
		}
	}

	if activeVin, err := bootstrap.Redis.Get(ctx, "canada:active_vin").Result(); err == nil && activeVin == vin {
		return true, nil
	}

	return false, nil
}

// IsVinPendingOrActive returns true if the VIN is either waiting in the FIFO queue or actively being harvested
func (s *WorkerGuardService) IsVinPendingOrActive(vin string) bool {
	if active, _ := s.IsVinActive(vin); active {
		return true
	}
	if bootstrap.Redis == nil {
		return false
	}
	ctx := context.Background()
	list, err := bootstrap.Redis.LRange(ctx, "canada:tasks:queue", 0, -1).Result()
	if err == nil {
		for _, v := range list {
			if v == vin {
				return true
			}
		}
	}
	return false
}

// EnqueueSearchTask pushes VIN into the sequential FIFO queue with Singleflight In-Flight Deduplication
func (s *WorkerGuardService) EnqueueSearchTask(vin string) error {
	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()

	// 1. In-Flight Singleflight Lock: if the worker is already harvesting this VIN right now,
	// do not push a duplicate task; both requests will receive the report simultaneously via Pub/Sub!
	if active, _ := s.IsVinActive(vin); active {
		return nil
	}

	// 2. Queue Deduplication: if the VIN is already waiting in the FIFO queue, do not push a second copy
	list, err := bootstrap.Redis.LRange(ctx, "canada:tasks:queue", 0, -1).Result()
	if err == nil {
		for _, v := range list {
			if v == vin {
				return nil
			}
		}
	}

	return bootstrap.Redis.RPush(ctx, "canada:tasks:queue", vin).Err()
}

// estimateStepRemainingSec maps a WorkerProgress Step to realistic remaining harvest seconds
func estimateStepRemainingSec(p *WorkerProgress) int {
	if p == nil {
		return 18
	}
	switch p.Step {
	case "DISPATCHED", "STARTING":
		return 20
	case "TYPING_VIN":
		return 16
	case "SUBMITTING":
		return 12
	case "WAIT_REPORT":
		return 8
	case "HARVESTING":
		return 4
	default:
		return 15
	}
}

// GetQueuePosition returns the 1-based position of a VIN in the waiting/active pipeline and estimated wait seconds
func (s *WorkerGuardService) GetQueuePosition(vin string) (int, int) {
	if bootstrap.Redis == nil {
		return 0, 0
	}
	ctx := context.Background()

	// 1. Check if this VIN is actively being pulled right now (Position 1)
	if active, prog := s.IsVinActive(vin); active {
		return 1, estimateStepRemainingSec(prog)
	}

	// 2. Check position in the waiting FIFO queue
	list, err := bootstrap.Redis.LRange(ctx, "canada:tasks:queue", 0, -1).Result()
	if err != nil || len(list) == 0 {
		return 0, 0
	}

	idxInQueue := -1
	for idx, v := range list {
		if v == vin {
			idxInQueue = idx
			break
		}
	}

	if idxInQueue == -1 {
		return 0, 0
	}

	// Check if another VIN is currently active in the worker
	activeOffset := 0
	activeRemainingSec := 0
	if activeVin, err := bootstrap.Redis.Get(ctx, "canada:active_vin").Result(); err == nil && activeVin != "" && activeVin != vin {
		activeOffset = 1
		activeRemainingSec = 16
		if pBytes, pErr := bootstrap.Redis.Get(ctx, "canada:active_progress").Bytes(); pErr == nil && len(pBytes) > 0 {
			var p WorkerProgress
			if json.Unmarshal(pBytes, &p) == nil {
				activeRemainingSec = estimateStepRemainingSec(&p)
			}
		}
	}

	pos := idxInQueue + 1 + activeOffset
	cooldown := s.GetCooldownRemainingSec()

	// Each full pull cycle takes ~20s harvest + minCooldownSec (15s) = ~35s
	cycleSec := s.minCooldownSec + 20
	var estimatedSec int
	if activeOffset > 0 {
		estimatedSec = activeRemainingSec + s.minCooldownSec + (idxInQueue * cycleSec) + 20
	} else {
		estimatedSec = cooldown + (idxInQueue * cycleSec) + 20
	}

	if estimatedSec < 5 {
		estimatedSec = 5
	}

	return pos, estimatedSec
}

// PollNextTask gives the remote Canada node the next VIN to fetch if safety gates pass
func (s *WorkerGuardService) PollNextTask() (string, int, error) {
	s.RecordHeartbeat()

	if bootstrap.Redis == nil {
		return "", 0, ErrQueueEmpty
	}
	ctx := context.Background()

	// Gate 1: Check Circuit Breaker
	if active, reason := s.IsCircuitBreakerActive(); active {
		return "", 0, fmt.Errorf("%w: %s", ErrCircuitBreaker, reason)
	}

	// Gate 2: Check Daily Limit
	dailyPulls := s.GetDailyPulls()
	if dailyPulls >= s.maxDailyPulls {
		return "", 0, ErrDailyLimit
	}

	// Gate 3: Check Mandatory Cooldown
	cooldown := s.GetCooldownRemainingSec()
	if cooldown > 0 {
		return "", cooldown, ErrCooldownActive
	}

	// Gate 4: Pop next VIN from queue
	vin, err := bootstrap.Redis.LPop(ctx, "canada:tasks:queue").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", 0, ErrQueueEmpty
		}
		return "", 0, err
	}

	// Immediately mark VIN as active so concurrent requests coalesce onto this in-flight pull
	bootstrap.Redis.Set(ctx, "canada:active_vin", vin, 90*time.Second)
	initProg := WorkerProgress{
		VIN:       vin,
		Step:      "DISPATCHED",
		StepAr:    "تم توجيه الطلب لبدء الفحص",
		UpdatedAt: time.Now(),
	}
	if b, mErr := json.Marshal(&initProg); mErr == nil {
		bootstrap.Redis.Set(ctx, "canada:active_progress", b, 90*time.Second)
	}

	return vin, 0, nil
}

// TouchCooldownWithJitter sets the anti-ban cooldown timestamp with natural human jitter (10s - 13s)
// and clears active_vin after Stage 2 background PDF enrichment finishes.
func (s *WorkerGuardService) TouchCooldownWithJitter() {
	if bootstrap.Redis == nil {
		return
	}
	ctx := context.Background()
	jitter := int64(rand.Intn(4)) // 0, 1, 2, or 3 seconds jitter
	bootstrap.Redis.Del(ctx, "canada:active_progress")
	bootstrap.Redis.Del(ctx, "canada:active_vin")
	bootstrap.Redis.Set(ctx, "canada:last_pull_ended_at", time.Now().Unix()-jitter, 0)
}

// CompleteTask handles response from Canada node
func (s *WorkerGuardService) CompleteTask(res *WorkerTaskResult) error {
	s.RecordHeartbeat()

	if bootstrap.Redis == nil {
		return nil
	}
	ctx := context.Background()

	// If Stage 1 completed without RawHtmlEn, Stage 2 (human reading + PDF) continues for ~10s on the worker;
	// keep active_vin alive with a 15s TTL until Stage 2 calls TouchCooldownWithJitter!
	if res.ErrorCode == "" && res.RawHtmlEn == "" {
		bootstrap.Redis.Expire(ctx, "canada:active_vin", 15*time.Second)
		bootstrap.Redis.Expire(ctx, "canada:active_progress", 15*time.Second)
	} else {
		bootstrap.Redis.Del(ctx, "canada:active_progress")
		bootstrap.Redis.Del(ctx, "canada:active_vin")
	}

	// Save screenshot if sent with completion
	if res.ScreenshotBase64 != "" {
		bootstrap.Redis.Set(ctx, "canada:last_screenshot", res.ScreenshotBase64, 24*time.Hour)
		bootstrap.Redis.Set(ctx, "canada:last_screenshot_at", time.Now().Unix(), 24*time.Hour)
	}

	// Only trip the global Circuit Breaker on actual account/security blocks (CAPTCHA_DETECTED or SESSION_EXPIRED),
	// NEVER on transient DOM/typing/tab hiccups (VIN_TYPE_ERROR, TAB_LOST, VIN_INPUT_NOT_FOUND, CHROME_NOT_CONNECTED)
	if res.ErrorCode != "" {
		if res.ErrorCode == "CAPTCHA_DETECTED" || res.ErrorCode == "SESSION_EXPIRED" {
			s.TriggerCircuitBreaker(res.ErrorCode, res.ErrorMessage)
		}
	} else {
		// Successful report fetch
		jitter := int64(rand.Intn(4))
		pipe := bootstrap.Redis.Pipeline()
		dailyKey := s.getDailyKey()
		pipe.Incr(ctx, dailyKey)
		pipe.Expire(ctx, dailyKey, 24*time.Hour)
		pipe.Set(ctx, "canada:last_pull_ended_at", time.Now().Unix()-jitter, 0)
		_, _ = pipe.Exec(ctx)
	}

	// Store result in Redis only if successful so retries don't read stale errors
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

// WaitForReport waits synchronously for Canada worker to deliver report or times out.
// Subscribes to Pub/Sub BEFORE checking Redis key and also polls every 1s to guarantee zero missed completions.
func (s *WorkerGuardService) WaitForReport(vin string, timeout time.Duration) (*WorkerTaskResult, error) {
	if bootstrap.Redis == nil {
		return nil, errors.New("redis unavailable")
	}
	ctx := context.Background()

	channel := fmt.Sprintf("canada:pubsub:%s", vin)
	sub := bootstrap.Redis.Subscribe(ctx, channel)
	defer sub.Close()
	ch := sub.Channel()

	resultKey := fmt.Sprintf("canada:result:%s", vin)
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

		case <-deadline:
			if res := checkCachedResult(); res != nil {
				return res, nil
			}
			return nil, ErrWorkerTimedOut
		}
	}
}

// GetStatus returns the full live status, telemetry, active progress, and screenshot for the Command Center UI
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

	return &WorkerStatus{
		Connected:            s.IsConnected(),
		CircuitBreakerActive: cbActive,
		CircuitBreakerReason: cbReason,
		DailyPulls:           s.GetDailyPulls(),
		MaxDailyPulls:        s.maxDailyPulls,
		CooldownRemainingSec: s.GetCooldownRemainingSec(),
		QueueLength:          qLen,
		LastHeartbeat:        lastHb,
		UseMockFallback:      s.IsMockFallbackEnabled(),
		InspectPending:       inspectPending,
		Telemetry:            telemetry,
		ActiveProgress:       progress,
		LastScreenshotBase64: lastScreenshot,
		LastScreenshotAt:     lastScreenshotAt,
	}
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

type PollResponse struct {
	HasTask           bool   `json:"has_task"`
	VIN               string `json:"vin"`
	InspectRequested  bool   `json:"inspect_requested,omitempty"`
	CooldownRemaining int    `json:"cooldown_remaining,omitempty"`
	CircuitBreaker    bool   `json:"circuit_breaker,omitempty"`
	DailyLimitHit     bool   `json:"daily_limit_hit,omitempty"`
	Message           string `json:"message"`
}

type CompleteRequest struct {
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

type ProgressUpdate struct {
	VIN              string `json:"vin"`
	Step             string `json:"step"`
	StepAr           string `json:"step_ar"`
	ScreenshotBase64 string `json:"screenshot_base64,omitempty"`
}

type CanadaWorker struct {
	serverURL    string
	secretKey    string
	cdpWsURL     string
	pollInterval time.Duration
	client       *http.Client
	stealth      *StealthHarvester
	pollCounter  int
}

func main() {
	_ = godotenv.Load()

	serverURL := strings.TrimRight(strings.TrimSpace(os.Getenv("SERVER_BASE_URL")), "/")
	if serverURL == "" {
		serverURL = "https://vinreport.tech"
	}

	secretKey := os.Getenv("WORKER_SECRET_KEY")
	if secretKey == "" {
		secretKey = "almoq3-canada-secret-2026"
	}

	cdpWsURL := os.Getenv("CDP_WS_URL")
	if cdpWsURL == "" {
		cdpWsURL = "http://127.0.0.1:9222"
	}

	pollSec := 1
	if val := os.Getenv("POLL_INTERVAL_SEC"); val != "" {
		if s, err := strconv.Atoi(val); err == nil && s > 0 {
			pollSec = s
		}
	}

	worker := &CanadaWorker{
		serverURL:    serverURL,
		secretKey:    secretKey,
		cdpWsURL:     cdpWsURL,
		pollInterval: time.Duration(pollSec) * time.Second,
		client:       &http.Client{Timeout: 60 * time.Second},
		stealth:      NewStealthHarvester(),
	}

	fmt.Println("=================================================================")
	fmt.Println("   🇨🇦 CarFix Stealth Harvester Node (Canada One-Shot Anti-Ban)")
	fmt.Println("=================================================================")
	fmt.Printf("[*] Master Server URL : %s\n", serverURL)
	fmt.Printf("[*] Local Chrome CDP  : %s\n", cdpWsURL)
	fmt.Printf("[*] Polling Cadence   : %v\n", worker.pollInterval)
	fmt.Println("[*] Target Portal     : https://www.carfaxonline.com")
	fmt.Println("[*] Safety Policy     : One-Shot Or Freeze (Zero Auto-Retry)")
	fmt.Println("=================================================================")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Auto-Recovery: Ensure Chrome is launched in Off-Screen mode with fresh profile sync
	_ = worker.stealth.EnsureChromeRunning(worker.cdpWsURL)

	// Perform an initial browser inspection + screenshot on startup
	worker.reportTelemetry(ctx, true)

	worker.Run(ctx)
}

// sendHeartbeat pings the server to signal the worker is alive.
// Called independently of the poll loop so the server never shows "disconnected"
// during long harvest operations or mandatory cooldown gaps.
func (w *CanadaWorker) sendHeartbeat() {
	req, err := http.NewRequest("POST", w.serverURL+"/api/worker/heartbeat", nil)
	if err != nil {
		return
	}
	req.Header.Set("X-Worker-Secret", w.secretKey)
	resp, err := w.client.Do(req)
	if err == nil && resp != nil {
		resp.Body.Close()
	}
}

func (w *CanadaWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Independent heartbeat goroutine: fires every 10s regardless of poll/harvest state.
	// This prevents false "disconnected" alerts during long Carfax harvests or cooldowns.
	go func() {
		hbTicker := time.NewTicker(10 * time.Second)
		defer hbTicker.Stop()
		w.sendHeartbeat() // immediate first beat on startup
		for {
			select {
			case <-ctx.Done():
				return
			case <-hbTicker.C:
				w.sendHeartbeat()
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\n[*] Shutting down Canada Stealth Worker safely...")
			return
		case <-ticker.C:
			w.pollAndExecute(ctx)
		}
	}
}

func (w *CanadaWorker) pollAndExecute(ctx context.Context) {
	w.pollCounter++

	// Every 12th poll (~12s), refresh lightweight browser state (no screenshot unless requested)
	if w.pollCounter%12 == 0 {
		w.reportTelemetry(ctx, false)
	}

	pollRes, err := w.sendPoll()
	if err != nil {
		log.Printf("⚠️ [POLL FAILED] Unable to reach master server: %v", err)
		return
	}

	// 1. Check if operator requested an immediate live screenshot & DOM inspection from dashboard
	if pollRes.InspectRequested {
		log.Printf("📸 [INSPECT REQUESTED] Capturing live JPEG 60%% screenshot & checking carfaxonline.com DOM...")
		w.reportTelemetry(ctx, true)
	}

	// 2. Guard check: Circuit breaker or daily limit
	if pollRes.CircuitBreaker || pollRes.DailyLimitHit {
		return
	}

	// 3. Anti-ban mandatory cooldown in effect
	if pollRes.CooldownRemaining > 0 {
		return
	}

	// 4. No pending task in queue
	if !pollRes.HasTask || pollRes.VIN == "" {
		return
	}

	// 5. Task received: Execute strict One-Shot Harvest with 2-Stage Split Delivery (Instant UI + Background Official PDF)
	vin := pollRes.VIN
	log.Printf("🚀 [TASK RECEIVED] Starting One-Shot Carfax pull for VIN: %s", vin)

	stage1Delivered := false
	harvestRes := w.stealth.HarvestReport(
		ctx,
		w.cdpWsURL,
		vin,
		func(step, stepAr, screenshot string) {
			log.Printf("   -> [%s] %s", step, stepAr)
			_ = w.sendProgress(&ProgressUpdate{
				VIN:              vin,
				Step:             step,
				StepAr:           stepAr,
				ScreenshotBase64: screenshot,
			})
		},
		func(fastRes *HarvestResult) {
			if fastRes != nil && fastRes.Success && fastRes.FastSummaryJson != "" {
				stage1Req := CompleteRequest{
					VIN:             vin,
					VehicleTitle:    fastRes.VehicleTitle,
					FastSummaryJson: fastRes.FastSummaryJson,
					CompletedAt:     time.Now(),
				}
				if err := w.sendComplete(&stage1Req); err == nil {
					stage1Delivered = true
					log.Printf("⚡ [STAGE 1 DELIVERED] Instant structured report sent to customer for %s in ~7s! Finishing human reading & Official PDF in background...", vin)
				}
			}
		},
	)

	completeReq := CompleteRequest{
		VIN:                vin,
		VehicleTitle:       harvestRes.VehicleTitle,
		RawHtmlEn:          harvestRes.RawHtmlEn,
		FastSummaryJson:    harvestRes.FastSummaryJson,
		IsBackgroundEnrich: stage1Delivered && harvestRes.Success,
		ErrorCode:          harvestRes.ErrorCode,
		ErrorMessage:       harvestRes.ErrorMessage,
		ScreenshotBase64:   harvestRes.ScreenshotBase64,
		CompletedAt:        time.Now(),
	}

	if err := w.sendComplete(&completeReq); err != nil {
		log.Printf("❌ [COMPLETE FAILED] Error delivering result to server: %v", err)
	} else {
		if !harvestRes.Success {
			log.Printf("🛑 [ONE-SHOT FREEZE] Safety stop [%s]: %s", harvestRes.ErrorCode, harvestRes.ErrorMessage)
		} else {
			log.Printf("✅ [STAGE 2 COMPLETED] Human reading + Official PDF for %s finalized and enriched!", vin)
		}
	}

	// Refresh telemetry state after task completion
	w.reportTelemetry(ctx, false)
}

func (w *CanadaWorker) reportTelemetry(ctx context.Context, withScreenshot bool) {
	state := w.stealth.InspectBrowserState(ctx, w.cdpWsURL, withScreenshot)
	payload, err := json.Marshal(state)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", w.serverURL+"/api/worker/telemetry", bytes.NewBuffer(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Worker-Secret", w.secretKey)

	resp, err := w.client.Do(req)
	if err == nil && resp != nil {
		resp.Body.Close()
	}
}

func (w *CanadaWorker) sendProgress(prog *ProgressUpdate) error {
	payload, err := json.Marshal(prog)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", w.serverURL+"/api/worker/progress", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Worker-Secret", w.secretKey)

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (w *CanadaWorker) sendPoll() (*PollResponse, error) {
	req, err := http.NewRequest("GET", w.serverURL+"/api/worker/poll", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Worker-Secret", w.secretKey)

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pollRes PollResponse
	if err := json.Unmarshal(body, &pollRes); err != nil {
		return nil, err
	}

	return &pollRes, nil
}

func (w *CanadaWorker) sendComplete(res *CompleteRequest) error {
	payload, err := json.Marshal(res)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", w.serverURL+"/api/worker/complete", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Worker-Secret", w.secretKey)

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server responded with status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

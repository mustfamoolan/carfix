package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

// CDPTab represents an open tab returned by http://127.0.0.1:9222/json/list
type CDPTab struct {
	ID                   string `json:"id"`
	Title                string `json:"title"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// InspectResult holds the complete read-only state of the Canada Chrome browser + JPEG screenshot
type InspectResult struct {
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

// HarvestResult holds the result of a single One-Shot report harvest
type HarvestResult struct {
	Success          bool
	VehicleTitle     string
	RawHtmlEn        string
	FastSummaryJson  string
	ErrorCode        string
	ErrorMessage     string
	ScreenshotBase64 string
}

type StealthHarvester struct {
	rnd               *rand.Rand
	httpClient        *http.Client
	assetCache        sync.Map
	recentModes       []int
	mu                sync.Mutex
	activeTabs        map[string]time.Time
	isHarvesting      bool
	lastInspectResult *InspectResult
}

func NewStealthHarvester() *StealthHarvester {
	return &StealthHarvester{
		rnd:         rand.New(rand.NewSource(time.Now().UnixNano())),
		httpClient:  &http.Client{Timeout: 8 * time.Second},
		recentModes: make([]int, 0, 5),
		activeTabs:  make(map[string]time.Time),
	}
}

func (sh *StealthHarvester) setHarvesting(active bool) {
	sh.mu.Lock()
	sh.isHarvesting = active
	sh.mu.Unlock()
}

func (sh *StealthHarvester) IsHarvesting() bool {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	return sh.isHarvesting
}

func (sh *StealthHarvester) markTabActive(tabID string) {
	if tabID == "" {
		return
	}
	sh.mu.Lock()
	sh.activeTabs[tabID] = time.Now()
	sh.mu.Unlock()
}

func (sh *StealthHarvester) unmarkTabActive(tabID string) {
	if tabID == "" {
		return
	}
	sh.mu.Lock()
	delete(sh.activeTabs, tabID)
	sh.mu.Unlock()
}

func (sh *StealthHarvester) isTabActive(tabID string) bool {
	if tabID == "" {
		return false
	}
	sh.mu.Lock()
	defer sh.mu.Unlock()
	ts, ok := sh.activeTabs[tabID]
	if !ok {
		return false
	}
	if time.Since(ts) > 2*time.Minute {
		delete(sh.activeTabs, tabID)
		return false
	}
	return true
}

func (sh *StealthHarvester) randomDelay(minMs, maxMs int) {
	delta := maxMs - minMs
	if delta <= 0 {
		delta = 50
	}
	sleepMs := minMs + sh.rnd.Intn(delta)
	time.Sleep(time.Duration(sleepMs) * time.Millisecond)
}

// normalizeCDPHttpBase converts ws://127.0.0.1:9222 or http://127.0.0.1:9222 into http://127.0.0.1:9222
func normalizeCDPHttpBase(cdpEndpoint string) string {
	u := strings.TrimSpace(cdpEndpoint)
	u = strings.TrimPrefix(u, "ws://")
	u = strings.TrimPrefix(u, "wss://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimPrefix(u, "https://")
	if idx := strings.Index(u, "/"); idx != -1 {
		u = u[:idx]
	}
	if u == "" {
		u = "127.0.0.1:9222"
	}
	return "http://" + u
}

// ListOpenTabs queries Chrome's local debugging port for all open page tabs
func (sh *StealthHarvester) ListOpenTabs(cdpEndpoint string) ([]CDPTab, error) {
	base := normalizeCDPHttpBase(cdpEndpoint)
	resp, err := sh.httpClient.Get(base + "/json/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var all []CDPTab
	if err := json.Unmarshal(body, &all); err != nil {
		return nil, err
	}

	var pages []CDPTab
	for _, t := range all {
		if t.Type == "page" && t.WebSocketDebuggerURL != "" && !strings.HasPrefix(t.URL, "devtools://") {
			pages = append(pages, t)
		}
	}
	return pages, nil
}

// CloseTabByID closes only a specific popup/report tab without touching the main search tab
func (sh *StealthHarvester) CloseTabByID(cdpEndpoint, tabID string) {
	if tabID == "" {
		return
	}
	base := normalizeCDPHttpBase(cdpEndpoint)
	resp, err := sh.httpClient.Get(base + "/json/close/" + tabID)
	if err == nil && resp != nil {
		resp.Body.Close()
	}
}

// EnsureChromeRunning checks if Chrome CDP port 9222 is alive; if closed or crashed, it auto-launches it
// using the dedicated, isolated C:\CarfaxDealerProfile directory (NEVER copying or overwriting from Main Chrome
// so Chrome 127+ App-Bound Encryption keys and CARFAX login cookies + 2FA device trust remain 100% permanent).
func (sh *StealthHarvester) EnsureChromeRunning(cdpEndpoint string) error {
	// Wait briefly if Chrome is in the middle of starting up (up to 2s) before spawning a new window
	for w := 0; w < 4; w++ {
		if _, err := sh.ListOpenTabs(cdpEndpoint); err == nil {
			sh.deduplicateSearchTabs(cdpEndpoint)
			return nil // Already running and healthy
		}
		time.Sleep(400 * time.Millisecond)
	}

	log.Printf("⚠️ [AUTO-RECOVERY] Chrome is closed or disconnected on 9222. Launching dedicated Chrome profile automatically...")

	_ = os.MkdirAll(`C:\CarfaxDealerProfile`, 0755)

	chromePaths := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		filepath.Join(os.Getenv("LOCALAPPDATA"), `Google\Chrome\Application\chrome.exe`),
	}

	for _, cp := range chromePaths {
		if _, stat := os.Stat(cp); stat == nil {
			cmd := exec.Command(cp,
				"--remote-debugging-port=9222",
				"--remote-allow-origins=*",
				"--user-data-dir=C:\\CarfaxDealerProfile",
				"--profile-directory=Default",
				"--disable-popup-blocking",
				"--disable-session-crashed-bubble",
				"--disable-blink-features=AutomationControlled",
				"--no-first-run",
				"--no-default-browser-check",
				"https://www.carfaxonline.com",
			)
			// Launch Chrome directly with zero CMD windows (Chrome is GUI so opens visible window)
			cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
			_ = cmd.Start()
			break
		}
	}

	// Wait up to 10 seconds for port 9222 to become available
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		if _, checkErr := sh.ListOpenTabs(cdpEndpoint); checkErr == nil {
			sh.deduplicateSearchTabs(cdpEndpoint)
			log.Printf("✅ [AUTO-RECOVERY SUCCESS] Chrome host connected on 9222 (C:\\CarfaxDealerProfile)!")
			return nil
		}
	}

	return fmt.Errorf("تعذر إعادة تشغيل كروم تلقائياً")
}

// deduplicateSearchTabs closes any duplicate carfaxonline.com home/search tabs (or extra New Tabs)
// so Chrome always keeps just ONE clean search tab open!
func (sh *StealthHarvester) deduplicateSearchTabs(cdpEndpoint string) {
	if sh.IsHarvesting() {
		return
	}
	tabs, err := sh.ListOpenTabs(cdpEndpoint)
	if err != nil || len(tabs) <= 1 {
		return
	}

	keptSearchTabID := ""
	for _, t := range tabs {
		u := strings.ToLower(t.URL)
		if (strings.Contains(u, "carfaxonline.com") || strings.Contains(u, "carfax.ca") || strings.Contains(u, "carfax.com")) &&
			!strings.Contains(u, "/vhr/") && !strings.Contains(u, "/cfm/") {
			if keptSearchTabID == "" {
				keptSearchTabID = t.ID
			} else if !sh.isTabActive(t.ID) {
				sh.CloseTabByID(cdpEndpoint, t.ID)
			}
		} else if keptSearchTabID != "" && (strings.HasPrefix(u, "chrome://newtab") || u == "about:blank") {
			sh.CloseTabByID(cdpEndpoint, t.ID)
		}
	}
}

// KeepAliveSession gently refreshes the active carfaxonline.com dealer session in the background
// when the worker is idle so CARFAX server-side session cookies & 2FA trust never expire due to inactivity.
// It NEVER touches the browser if a harvest is running or if the page is on a login screen.
func (sh *StealthHarvester) KeepAliveSession(parentCtx context.Context, cdpEndpoint string) {
	if sh.IsHarvesting() {
		return
	}

	tab, isCarfax, err := sh.FindCarfaxSearchTab(cdpEndpoint)
	if err != nil || tab == nil || !isCarfax {
		return
	}

	u := strings.ToLower(tab.URL)
	if strings.Contains(u, "/login") || strings.Contains(u, "/signin") {
		return
	}

	tabCtx, cleanup, attachErr := sh.attachExistingTab(parentCtx, cdpEndpoint, tab.ID)
	if attachErr != nil {
		return
	}
	defer cleanup()

	keepAliveJS := `
(() => {
    const passInput = document.querySelector('input[type="password"]');
    if (passInput && passInput.offsetParent !== null) {
        return 'login_page_skipped';
    }
    // Dispatch subtle activity and background credentialed ping to keep CARFAX rolling session cookie fresh
    document.dispatchEvent(new MouseEvent('mousemove', { clientX: 120, clientY: 120, bubbles: true }));
    fetch('https://www.carfaxonline.com/', { method: 'GET', credentials: 'include', cache: 'no-store' }).catch(() => {});
    return 'kept_alive';
})();
`
	var status string
	if err := chromedp.Run(tabCtx, chromedp.Evaluate(keepAliveJS, &status)); err == nil && status == "kept_alive" {
		log.Printf("🔄 [SESSION KEEP-ALIVE] Refreshed carfaxonline.com dealer session cookies silently to prevent idle logout.")
	}
}

// FindCarfaxSearchTab finds the open carfaxonline.com tab, or falls back to the first visible page tab.
// Never closes any user tabs.
func (sh *StealthHarvester) FindCarfaxSearchTab(cdpEndpoint string) (*CDPTab, bool, error) {
	_ = sh.EnsureChromeRunning(cdpEndpoint)
	tabs, err := sh.ListOpenTabs(cdpEndpoint)
	if err != nil {
		return nil, false, err
	}
	if len(tabs) == 0 {
		return nil, false, nil
	}

	// 1. Prefer a tab on carfaxonline.com that is NOT an already-opened report popup (/vhr/ or /cfm/)
	for i := range tabs {
		t := tabs[i]
		u := strings.ToLower(t.URL)
		if (strings.Contains(u, "carfaxonline.com") || strings.Contains(u, "carfax.ca") || strings.Contains(u, "carfax.com")) &&
			!strings.Contains(u, "/vhr/") && !strings.Contains(u, "/cfm/") {
			return &t, true, nil
		}
	}

	// 1b. Fallback to any carfax tab if only a report tab is open
	for i := range tabs {
		t := tabs[i]
		u := strings.ToLower(t.URL)
		if strings.Contains(u, "carfaxonline.com") || strings.Contains(u, "carfax.ca") || strings.Contains(u, "carfax.com") {
			return &t, true, nil
		}
	}

	// 2. Prefer any non-blank tab
	for i := range tabs {
		t := tabs[i]
		u := strings.TrimSpace(strings.ToLower(t.URL))
		if u != "about:blank" && u != "" {
			return &t, false, nil
		}
	}

	// 3. Fallback to first tab
	return &tabs[0], false, nil
}

// attachExistingTab attaches to an existing Chrome tab by ID and immediately clears
// c.Target.TargetID in chromedp's internal struct so that when the context ends,
// chromedp only detaches the debugger session (Target.detachFromTarget) and NEVER
// sends Target.closeTarget to close the user's open tab!
func (sh *StealthHarvester) attachExistingTab(
	parentCtx context.Context,
	cdpEndpoint string,
	tabID string,
) (context.Context, context.CancelFunc, error) {
	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(parentCtx, normalizeCDPHttpBase(cdpEndpoint))
	tabCtx, cancelTab := chromedp.NewContext(allocCtx, chromedp.WithTargetID(target.ID(tabID)))

	err := chromedp.Run(tabCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		if c := chromedp.FromContext(ctx); c != nil && c.Target != nil {
			c.Target.TargetID = ""
		}
		return nil
	}))
	if err != nil {
		if c := chromedp.FromContext(tabCtx); c != nil && c.Target != nil {
			c.Target.TargetID = ""
		}
		cancelTab()
		cancelAlloc()
		return nil, nil, err
	}

	cleanup := func() {
		if c := chromedp.FromContext(tabCtx); c != nil && c.Target != nil {
			c.Target.TargetID = ""
		}
		cancelTab()
		cancelAlloc()
	}

	return tabCtx, cleanup, nil
}

// captureCompressedJPEG captures the current visible viewport as a 60% quality JPEG (40KB-70KB)
// without resizing the window or touching device metrics.
func (sh *StealthHarvester) captureCompressedJPEG(tabCtx context.Context) string {
	var buf []byte
	err := chromedp.Run(tabCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		buf, err = page.CaptureScreenshot().
			WithFormat(page.CaptureScreenshotFormatJpeg).
			WithQuality(60).
			WithFromSurface(true).
			Do(ctx)
		return err
	}))
	if err != nil || len(buf) == 0 {
		return ""
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf)
}

// InspectBrowserState performs a 100% passive, read-only inspection of Chrome + takes an optimized JPEG screenshot
func (sh *StealthHarvester) InspectBrowserState(parentCtx context.Context, cdpEndpoint string, takeScreenshot bool) *InspectResult {
	// Never interfere with active tab focus during an ongoing VIN harvest unless an explicit screenshot was requested
	if !takeScreenshot && sh.IsHarvesting() {
		sh.mu.Lock()
		cached := sh.lastInspectResult
		sh.mu.Unlock()
		if cached != nil {
			copyRes := *cached
			copyRes.InspectedAt = time.Now()
			copyRes.SessionActive = true
			copyRes.StatusMessage = "⚡ جاري سحب ومعالجة تقارير كارفاكس حالياً..."
			return &copyRes
		}
	}

	res := &InspectResult{
		InspectedAt: time.Now(),
	}

	tab, isCarfax, err := sh.FindCarfaxSearchTab(cdpEndpoint)
	if err != nil {
		res.ChromeConnected = false
		res.StatusMessage = "متصفح كروم في حاسبة كندا غير متصل على المنفذ 9222 (يرجى تشغيل start_chrome_stealth.bat)"
		return res
	}
	if tab == nil {
		res.ChromeConnected = true
		res.StatusMessage = "متصفح كروم مفتوح ولكن لا توجد أي نافذة صفحة مفتوحة بداخله"
		return res
	}

	res.ChromeConnected = true
	res.CarfaxTabFound = isCarfax
	res.TabID = tab.ID
	res.CurrentURL = tab.URL
	res.PageTitle = tab.Title

	// Attach to the existing tab safely without EVER sending Target.closeTarget on disconnect
	tabCtx, cleanupTab, attachErr := sh.attachExistingTab(parentCtx, cdpEndpoint, tab.ID)
	if attachErr != nil {
		return res
	}
	defer cleanupTab()

	// Read-only DOM inspection script tailored for carfaxonline.com
	inspectJS := `
(() => {
    const url = (window.location.href || '').toLowerCase();
    const titleLower = (document.title || '').toLowerCase();
    const bodyText = (document.body ? document.body.innerText : '').toLowerCase();

    // 1. Check if we are on a login screen
    const passInput = document.querySelector('input[type="password"]');
    const isLogin = Boolean(passInput && passInput.offsetParent !== null) || url.includes('/login') || url.includes('/signin');

    // 2. Locate VIN input box on carfaxonline.com
    const vinCandidates = [
        'input[name="vin"]',
        'input[name="VIN"]',
        'input[id="vin"]',
        'input[id="vinInput"]',
        'input[id*="vin" i]',
        'input[name*="vin" i]',
        'input[placeholder*="VIN" i]',
        'input[aria-label*="VIN" i]',
        'input[data-testid*="vin" i]',
        'input[class*="vin" i]',
        'input[maxlength="17"]',
        'input[type="text"]',
        'input[type="search"]',
        'input:not([type])',
        'input:not([type="hidden"]):not([type="password"]):not([type="checkbox"]):not([type="radio"]):not([type="submit"]):not([type="button"])'
    ];

    let matchedVinSelector = '';
    for (const sel of vinCandidates) {
        const el = document.querySelector(sel);
        if (el && el.offsetParent !== null && !el.disabled && el.type !== 'hidden') {
            matchedVinSelector = sel;
            break;
        }
    }

    const isReportPage = url.includes('/vhr/') || bodyText.includes('ownership history') || bodyText.includes('detailed history');

    // 3. Check for bot/captcha challenges (only if VIN box and report content are absent)
    let hasChallenge = false;
    if (!matchedVinSelector && !isReportPage && !isLogin) {
        const challengeWords = ['verify you are human', 'cf-chl', 'datadome', 'access denied', 'unusual traffic'];
        for (const w of challengeWords) {
            if (url.includes(w) || titleLower.includes(w) || bodyText.includes(w)) {
                hasChallenge = true;
                break;
            }
        }
    }

    // 4. Locate Submit / Run Report button
    const btnCandidates = [
        'button[type="submit"]',
        'input[type="submit"]',
        'button[id*="run" i]',
        'button[id*="search" i]',
        'button[id*="vhr" i]',
        'button[class*="search" i]',
        'button[class*="submit" i]',
        'a[id*="run" i]',
        'button'
    ];
    let submitFound = false;
    for (const sel of btnCandidates) {
        const btn = document.querySelector(sel);
        if (btn && btn.offsetParent !== null) {
            submitFound = true;
            break;
        }
    }

    return {
        url: window.location.href || '',
        title: document.title || '',
        vinFound: matchedVinSelector !== '',
        vinSelector: matchedVinSelector,
        submitFound: submitFound,
        isLogin: isLogin,
        hasChallenge: hasChallenge
    };
})();
`
	var domState struct {
		URL          string `json:"url"`
		Title        string `json:"title"`
		VinFound     bool   `json:"vinFound"`
		VinSelector  string `json:"vinSelector"`
		SubmitFound  bool   `json:"submitFound"`
		IsLogin      bool   `json:"isLogin"`
		HasChallenge bool   `json:"hasChallenge"`
	}

	if err := chromedp.Run(tabCtx, chromedp.Evaluate(inspectJS, &domState)); err == nil {
		if domState.URL != "" && domState.URL != "about:blank" {
			res.CurrentURL = domState.URL
		}
		if domState.Title != "" {
			res.PageTitle = domState.Title
		}
		res.VinInputFound = domState.VinFound
		res.VinInputSelector = domState.VinSelector
		res.SubmitBtnFound = domState.SubmitFound
		res.IsLoginPage = domState.IsLogin
		res.HasChallenge = domState.HasChallenge
	}

	if takeScreenshot {
		res.ScreenshotBase64 = sh.captureCompressedJPEG(tabCtx)
	}

	// Determine overall session readiness and Arabic status message
	switch {
	case res.HasChallenge:
		res.SessionActive = false
		res.StatusMessage = "⚠️ تحذير: تم رصد شاشة تحقق (Captcha / Security) في متصفح كندا"
	case !res.CarfaxTabFound:
		res.SessionActive = false
		res.StatusMessage = "⚠️ متصفح كندا مفتوح ولكن موقع carfaxonline.com غير مفتوح حالياً"
	case res.IsLoginPage:
		res.SessionActive = false
		res.StatusMessage = "🔒 حساب carfaxonline.com في كندا يحتاج لتسجيل الدخول يدوياً"
	case res.VinInputFound:
		res.SessionActive = true
		res.StatusMessage = "✅ متصفح كندا متصل وجاهز على صفحة البحث (حقل الـ VIN متاح)"
	default:
		// Logged in on a Carfax report tab (/vhr/) or sub-page — session is active!
		res.SessionActive = true
		res.StatusMessage = "✅ متصفح كندا متصل وجلسة كارفاكس فعالة (جاهز للبحث)"
	}

	sh.mu.Lock()
	sh.lastInspectResult = res
	sh.mu.Unlock()

	return res
}

// BatchTabTicket tracks an opened report tab bound to a specific VIN during a single or multi-tab harvest
type BatchTabTicket struct {
	VIN           string
	ReportTab     *CDPTab
	OpenedNewTab  bool
	FastTitle     string
	FastSummary   string
	Stage1Success bool
	FailedResult  *HarvestResult
}

// HarvestReport executes a single-VIN harvest by delegating to the unified HarvestBatchReports pipeline.
func (sh *StealthHarvester) HarvestReport(
	parentCtx context.Context,
	cdpEndpoint string,
	vin string,
	onProgress func(step, stepAr string, screenshot string),
	onFastData func(fastRes *HarvestResult),
) *HarvestResult {
	var finalOut *HarvestResult
	sh.HarvestBatchReports(
		parentCtx,
		cdpEndpoint,
		[]string{vin},
		func(v, step, stepAr, screenshot string) {
			if onProgress != nil {
				onProgress(step, stepAr, screenshot)
			}
		},
		func(v string, fastRes *HarvestResult) {
			if onFastData != nil {
				onFastData(fastRes)
			}
		},
		func(v string, finalRes *HarvestResult) {
			finalOut = finalRes
		},
	)
	if finalOut == nil {
		return &HarvestResult{
			Success:      false,
			ErrorCode:    "REPORT_LOAD_TIMEOUT",
			ErrorMessage: "انتهت مهلة معالجة التقرير وسيتم إعادة المحاولة تلقائياً.",
		}
	}
	return finalOut
}

// HarvestBatchReports executes the Multi-Tab Rapid Paste & Harvest Pipeline:
// - If 1 VIN is passed, it pastes the VIN instantly (~0.15s), opens the report tab, delivers Stage-1 JSON in ~3-5s,
//   and captures the 4-Checkbox Official PDF in ~2s via event-driven polling.
// - If 2..4 VINs are passed (when queue builds up), it mimics a real CARFAX dealer:
//   pastes VIN #1 -> Run Report (new tab opens) -> switches back to search tab -> pastes VIN #2 -> Run Report, etc.,
//   then harvests Stage-1 JSON for all open tabs immediately, followed by Stage-2 4-Checkbox Official PDF for each tab!
func (sh *StealthHarvester) HarvestBatchReports(
	parentCtx context.Context,
	cdpEndpoint string,
	vins []string,
	onProgress func(vin, step, stepAr, screenshot string),
	onFastData func(vin string, fastRes *HarvestResult),
	onComplete func(vin string, finalRes *HarvestResult),
) {
	if len(vins) == 0 {
		return
	}

	sh.setHarvesting(true)
	defer sh.setHarvesting(false)

	// Auto-Recovery: Respawn Chrome if someone closed it before task arrived
	_ = sh.EnsureChromeRunning(cdpEndpoint)

	firstVin := vins[0]
	if onProgress != nil {
		onProgress(firstVin, "pre_flight", "1/4 فحص أمان وجاهزية متصفح كندا (Pre-Flight Check)...", "")
	}

	// Temporarily allow InspectBrowserState to run real check for pre-flight
	sh.setHarvesting(false)
	state := sh.InspectBrowserState(parentCtx, cdpEndpoint, false)
	sh.setHarvesting(true)

	failAll := func(res *HarvestResult) {
		for _, v := range vins {
			if onComplete != nil {
				onComplete(v, res)
			}
		}
	}

	if !state.ChromeConnected {
		failAll(&HarvestResult{
			Success:      false,
			ErrorCode:    "CHROME_NOT_CONNECTED",
			ErrorMessage: state.StatusMessage,
		})
		return
	}

	if state.HasChallenge {
		sh.setHarvesting(false)
		snapState := sh.InspectBrowserState(parentCtx, cdpEndpoint, true)
		sh.setHarvesting(true)
		failAll(&HarvestResult{
			Success:          false,
			ErrorCode:        "CAPTCHA_DETECTED",
			ErrorMessage:     state.StatusMessage,
			ScreenshotBase64: snapState.ScreenshotBase64,
		})
		return
	}

	if state.IsLoginPage {
		log.Printf("🔒 [LOGIN REQUIRED] Browser is on login screen; leaving Chrome untouched for manual operator login + 2FA.")
		sh.setHarvesting(false)
		snapState := sh.InspectBrowserState(parentCtx, cdpEndpoint, true)
		sh.setHarvesting(true)
		failAll(&HarvestResult{
			Success:          false,
			ErrorCode:        "SESSION_EXPIRED",
			ErrorMessage:     "🔒 حساب carfaxonline.com يحتاج لتسجيل الدخول يدوياً في متصفح كروم على حاسبة كندا",
			ScreenshotBase64: snapState.ScreenshotBase64,
		})
		return
	}

	// If Chrome is open (and not on a login/captcha page) but not currently showing the VIN input box
	// (e.g. briefly showing "Verifying Device" on /landingPage or left on a report tab),
	// first wait gently if it's on /landingPage so Auth0 device verification can finish undisturbed!
	if !state.VinInputFound && state.TabID != "" {
		lowerCurrURL := strings.ToLower(state.CurrentURL)
		if strings.Contains(lowerCurrURL, "landingpage") || strings.Contains(lowerCurrURL, "auth.") {
			sh.randomDelay(3500, 4500)
			sh.setHarvesting(false)
			state = sh.InspectBrowserState(parentCtx, cdpEndpoint, false)
			sh.setHarvesting(true)
		}
		if !state.VinInputFound && !state.IsLoginPage && !state.HasChallenge {
			if navTabCtx, cleanupNav, err := sh.attachExistingTab(parentCtx, cdpEndpoint, state.TabID); err == nil {
				_ = chromedp.Run(navTabCtx, chromedp.Navigate("https://www.carfaxonline.com/"))
				cleanupNav()
				sh.randomDelay(2000, 2800)
				sh.setHarvesting(false)
				state = sh.InspectBrowserState(parentCtx, cdpEndpoint, false)
				sh.setHarvesting(true)
			}
		}
	}

	if !state.CarfaxTabFound || state.IsLoginPage {
		sh.setHarvesting(false)
		snapState := sh.InspectBrowserState(parentCtx, cdpEndpoint, true)
		sh.setHarvesting(true)
		failAll(&HarvestResult{
			Success:          false,
			ErrorCode:        "SESSION_EXPIRED",
			ErrorMessage:     state.StatusMessage,
			ScreenshotBase64: snapState.ScreenshotBase64,
		})
		return
	}

	if !state.VinInputFound || state.VinInputSelector == "" {
		sh.setHarvesting(false)
		snapState := sh.InspectBrowserState(parentCtx, cdpEndpoint, true)
		sh.setHarvesting(true)
		failAll(&HarvestResult{
			Success:          false,
			ErrorCode:        "VIN_INPUT_NOT_FOUND",
			ErrorMessage:     "توقف احترازي: حقل إدخال الشاصي غير ظاهر في الصفحة الحالية داخل حاسبة كندا.",
			ScreenshotBase64: snapState.ScreenshotBase64,
		})
		return
	}

	searchTab, _, err := sh.FindCarfaxSearchTab(cdpEndpoint)
	if err != nil || searchTab == nil {
		failAll(&HarvestResult{
			Success:      false,
			ErrorCode:    "TAB_LOST",
			ErrorMessage: "فقد الاتصال بتبويب البحث قبل البدء مباشرة.",
		})
		return
	}
	sh.markTabActive(searchTab.ID)
	defer sh.unmarkTabActive(searchTab.ID)

	// Safely clean up only stale /vhr/ popup tabs that are NOT currently tracked in activeTabs
	tabsBefore, _ := sh.ListOpenTabs(cdpEndpoint)
	knownTabIDs := make(map[string]bool)
	targetVinSet := make(map[string]bool)
	for _, v := range vins {
		targetVinSet[strings.ToUpper(strings.TrimSpace(v))] = true
	}

	for _, t := range tabsBefore {
		if t.ID != searchTab.ID && strings.Contains(strings.ToLower(t.URL), "/vhr/") && !sh.isTabActive(t.ID) {
			matchesTarget := false
			upperURL := strings.ToUpper(t.URL)
			for tv := range targetVinSet {
				if tv != "" && strings.Contains(upperURL, tv) {
					matchesTarget = true
					break
				}
			}
			if !matchesTarget {
				sh.CloseTabByID(cdpEndpoint, t.ID)
				continue
			}
		}
		knownTabIDs[t.ID] = true
	}

	searchTabCtx, cleanupSearchTab, attachErr := sh.attachExistingTab(parentCtx, cdpEndpoint, searchTab.ID)
	if attachErr != nil {
		failAll(&HarvestResult{
			Success:      false,
			ErrorCode:    "TAB_LOST",
			ErrorMessage: fmt.Sprintf("تعذر الارتباط بتبويب البحث: %v", attachErr),
		})
		return
	}
	defer cleanupSearchTab()

	// =========================================================================
	// ⚡ PHASE 1: RAPID CLIPBOARD PASTE & MULTI-TAB SPAWNING
	// =========================================================================
	tickets := make([]*BatchTabTicket, 0, len(vins))

	for idx, rawVin := range vins {
		vin := strings.ToUpper(strings.TrimSpace(rawVin))
		ticket := &BatchTabTicket{VIN: vin}
		tickets = append(tickets, ticket)

		if onProgress != nil {
			if len(vins) > 1 {
				onProgress(vin, "typing_vin", fmt.Sprintf("2/4 لصق فوري للشاصي (%s) [تقرير %d من %d]...", vin, idx+1, len(vins)), "")
			} else {
				onProgress(vin, "typing_vin", fmt.Sprintf("2/4 لصق فوري لرقم الشاصي (%s) وطلب التقرير...", vin), "")
			}
		}

		if err := sh.pasteVinInstantly(searchTabCtx, state.VinInputSelector, vin); err != nil {
			snap := sh.captureCompressedJPEG(searchTabCtx)
			ticket.FailedResult = &HarvestResult{
				Success:          false,
				ErrorCode:        "VIN_TYPE_ERROR",
				ErrorMessage:     fmt.Sprintf("تعذر لصق رقم الشاصي في حقل البحث (%v)", err),
				ScreenshotBase64: snap,
			}
			continue
		}

		if onProgress != nil {
			onProgress(vin, "submitting_search", fmt.Sprintf("3/4 جاري فتح تبويب تقرير كارفاكس للشاصي (%s)...", vin), "")
		}

		sh.submitVinSearch(searchTabCtx, state.VinInputSelector)

		// Poll local CDP every 250ms (up to 10s = 40 attempts) for the new report tab to open,
		// with an automatic retry click at 3.0s (attempt 12) if React missed the first click.
		var reportTab *CDPTab
		openedNewTab := false

		for attempt := 0; attempt < 40; attempt++ {
			time.Sleep(250 * time.Millisecond)
			tabsAfter, _ := sh.ListOpenTabs(cdpEndpoint)

			// 1. Prefer a brand-new tab ID not seen before
			for i := range tabsAfter {
				t := tabsAfter[i]
				if !knownTabIDs[t.ID] && t.URL != "about:blank" && t.ID != searchTab.ID {
					reportTab = &t
					openedNewTab = true
					break
				}
			}

			// 2. Fallback: tab whose URL or Title explicitly contains this VIN
			if reportTab == nil {
				for i := range tabsAfter {
					t := tabsAfter[i]
					if t.ID != searchTab.ID && !sh.isTabActive(t.ID) &&
						(strings.Contains(strings.ToUpper(t.URL), vin) || strings.Contains(strings.ToUpper(t.Title), vin)) {
						reportTab = &t
						openedNewTab = true
						break
					}
				}
			}

			if reportTab != nil {
				knownTabIDs[reportTab.ID] = true
				sh.markTabActive(reportTab.ID)
				break
			}

			// 3. Check if CARFAX showed "Popup Blocked — View Your Report" modal!
			// If so, extract the report URL (or click "View Your Report"), open it via CDP target.CreateTarget, and close the modal!
			var popupBlockedInfo struct {
				Found bool   `json:"found"`
				Href  string `json:"href"`
			}
			_ = chromedp.Run(searchTabCtx, chromedp.Evaluate(`
(() => {
    const links = Array.from(document.querySelectorAll('a, button, [role="button"]'));
    let viewLink = null;
    for (const el of links) {
        if (el.offsetParent === null) continue;
        const txt = (el.innerText || el.textContent || '').trim().toLowerCase();
        const href = el.href || el.getAttribute('href') || '';
        if (txt === 'view your report' || txt.includes('view your report') || (href && href.toLowerCase().includes('/vhr/'))) {
            viewLink = el;
            break;
        }
    }
    if (!viewLink) return { found: false, href: '' };

    const rawHref = viewLink.href || viewLink.getAttribute('href') || '';
    // Also click the link in case it uses an onClick handler
    viewLink.click();

    // Dismiss the "Popup Blocked" modal close button (×) so the search page stays clean for the next VIN
    const closeBtns = Array.from(document.querySelectorAll('button, span, a, [role="button"], [aria-label*="close" i]'));
    for (const cb of closeBtns) {
        if (cb.offsetParent === null) continue;
        const t = (cb.innerText || cb.textContent || '').trim();
        const aria = (cb.getAttribute('aria-label') || '').toLowerCase();
        if (t === '×' || t === '✕' || t === 'X' || aria.includes('close')) {
            cb.click();
            break;
        }
    }

    return { found: true, href: rawHref };
})()
`, &popupBlockedInfo))

			if popupBlockedInfo.Found && popupBlockedInfo.Href != "" && !strings.HasPrefix(strings.ToLower(popupBlockedInfo.Href), "javascript:") {
				_ = chromedp.Run(searchTabCtx, chromedp.ActionFunc(func(ctx context.Context) error {
					_, err := target.CreateTarget(popupBlockedInfo.Href).Do(ctx)
					return err
				}))
			}

			// Automatic retry click at ~3.0s if no tab opened yet
			if attempt == 12 {
				_ = sh.pasteVinInstantly(searchTabCtx, state.VinInputSelector, vin)
				sh.submitVinSearch(searchTabCtx, state.VinInputSelector)
			}
		}

		ticket.ReportTab = reportTab
		ticket.OpenedNewTab = openedNewTab

		// If there are more VINs in the batch, pause 2.0s-2.8s like a real human dealer switching back to tab 1
		if idx < len(vins)-1 {
			sh.randomDelay(2000, 2800)
		}
	}

	// =========================================================================
	// ⚡ PHASE 2: STAGE-1 INSTANT STRUCTURED EXTRACTION ACROSS OPEN TABS
	// =========================================================================
	for _, ticket := range tickets {
		if ticket.FailedResult != nil {
			continue
		}

		vin := ticket.VIN
		if onProgress != nil {
			onProgress(vin, "extracting_data", fmt.Sprintf("4/4 جاري قراءة جداول التقرير للشاصي (%s)...", vin), "")
		}

		activeReportCtx := searchTabCtx
		var cleanupReportTab context.CancelFunc

		if ticket.OpenedNewTab && ticket.ReportTab != nil {
			if rTabCtx, rCleanup, rErr := sh.attachExistingTab(parentCtx, cdpEndpoint, ticket.ReportTab.ID); rErr == nil {
				activeReportCtx = rTabCtx
				cleanupReportTab = rCleanup
			}
		}

		fastSummaryJson, _, fastTitle, errCode, fastErrMsg, fastErr := sh.extractFastReportData(activeReportCtx, vin)
		if cleanupReportTab != nil {
			cleanupReportTab()
		}

		if fastErr != nil || errCode != "" {
			var errScreenshot string
			if ticket.OpenedNewTab && ticket.ReportTab != nil {
				if rTabCtx, rCleanup, rErr := sh.attachExistingTab(parentCtx, cdpEndpoint, ticket.ReportTab.ID); rErr == nil {
					errScreenshot = sh.captureCompressedJPEG(rTabCtx)
					rCleanup()
				}
				sh.CloseTabByID(cdpEndpoint, ticket.ReportTab.ID)
				sh.unmarkTabActive(ticket.ReportTab.ID)
			} else {
				errScreenshot = sh.captureCompressedJPEG(searchTabCtx)
			}

			if errCode == "" {
				errCode = "REPORT_LOAD_TIMEOUT"
			}
			if fastErrMsg == "" && fastErr != nil {
				fastErrMsg = fmt.Sprintf("تعذر قراءة بيانات التقرير: %v", fastErr)
			}
			ticket.FailedResult = &HarvestResult{
				Success:          false,
				ErrorCode:        errCode,
				ErrorMessage:     fastErrMsg,
				ScreenshotBase64: errScreenshot,
			}
			continue
		}

		ticket.FastTitle = fastTitle
		ticket.FastSummary = fastSummaryJson
		ticket.Stage1Success = true

		// Deliver Stage 1 immediately so the waiting customer gets the full report UI right away!
		if onFastData != nil {
			onFastData(vin, &HarvestResult{
				Success:         true,
				VehicleTitle:    fastTitle,
				FastSummaryJson: fastSummaryJson,
			})
		}
	}

	// =========================================================================
	// 🖨️ PHASE 3: ZERO-SLEEP EVENT-DRIVEN 4-CHECKBOX OFFICIAL PDF & TAB CLEANUP
	// =========================================================================
	for _, ticket := range tickets {
		vin := ticket.VIN
		if ticket.FailedResult != nil {
			if onComplete != nil {
				onComplete(vin, ticket.FailedResult)
			}
			continue
		}

		activeReportCtx := searchTabCtx
		var cleanupReportTab context.CancelFunc
		if ticket.OpenedNewTab && ticket.ReportTab != nil {
			if rTabCtx, rCleanup, rErr := sh.attachExistingTab(parentCtx, cdpEndpoint, ticket.ReportTab.ID); rErr == nil {
				activeReportCtx = rTabCtx
				cleanupReportTab = rCleanup
			}
		}

		if onProgress != nil {
			onProgress(vin, "generating_pdf", fmt.Sprintf("جاري تجهيز ملف الطباعة الرسمي الكامل (4 أقسام) للشاصي (%s)...", vin), "")
		}

		// Brief natural scroll before triggering Print Options modal
		_ = chromedp.Run(activeReportCtx, chromedp.Evaluate(`window.scrollBy({ top: 260, behavior: 'smooth' });`, nil))
		sh.randomDelay(350, 600)
		_ = chromedp.Run(activeReportCtx, chromedp.Evaluate(`window.scrollTo({ top: 0, behavior: 'smooth' });`, nil))
		sh.randomDelay(200, 350)

		finalReportCtx, cleanupPrintTab, printPopupTabID := sh.prepareOfficialPrintModal(parentCtx, cdpEndpoint, activeReportCtx)
		officialPdfBase64 := sh.captureOfficialPrintPDF(finalReportCtx, vin)

		var fullPageHtml string
		_ = chromedp.Run(finalReportCtx, chromedp.Evaluate(`
(() => {
    const baseTag = '<base href="https://www.carfaxonline.com/">';
    let html = document.documentElement ? document.documentElement.outerHTML : '';
    if (html && !html.includes('<base ')) {
        html = html.replace(/<head([^>]*)>/i, '<head$1>' + baseTag);
    }
    return html;
})()
`, &fullPageHtml))

		if cleanupPrintTab != nil {
			cleanupPrintTab()
		}
		if cleanupReportTab != nil {
			cleanupReportTab()
		}

		// Clean up any extra print popup tab if Carfax opened one
		if printPopupTabID != "" && (ticket.ReportTab == nil || printPopupTabID != ticket.ReportTab.ID) && printPopupTabID != searchTab.ID {
			sh.CloseTabByID(cdpEndpoint, printPopupTabID)
			sh.unmarkTabActive(printPopupTabID)
		}

		// Close the report tab cleanly and unmark from activeTabs
		if ticket.OpenedNewTab && ticket.ReportTab != nil {
			sh.CloseTabByID(cdpEndpoint, ticket.ReportTab.ID)
			sh.unmarkTabActive(ticket.ReportTab.ID)
		} else {
			var stillHasVinInput bool
			_ = chromedp.Run(searchTabCtx, chromedp.Evaluate(fmt.Sprintf(`Boolean(document.querySelector(%q))`, state.VinInputSelector), &stillHasVinInput))
			if !stillHasVinInput {
				_ = chromedp.Run(searchTabCtx, chromedp.Evaluate(`window.history.back();`, nil))
			}
		}

		rawHtmlWithPdf := fullPageHtml
		if officialPdfBase64 != "" {
			rawHtmlWithPdf += `<script id="carfix-official-pdf" type="application/pdf-base64">` + officialPdfBase64 + `</script>`
		}

		if onComplete != nil {
			onComplete(vin, &HarvestResult{
				Success:          true,
				VehicleTitle:     ticket.FastTitle,
				RawHtmlEn:        rawHtmlWithPdf,
				FastSummaryJson:  ticket.FastSummary,
				ScreenshotBase64: "",
			})
		}
	}

	// =========================================================================
	// 🔄 PHASE 4: PRE-WARM SEARCH TAB FOR ZERO-LATENCY NEXT REQUEST
	// =========================================================================
	_ = chromedp.Run(searchTabCtx, chromedp.Evaluate(fmt.Sprintf(`
(() => {
    const el = document.querySelector(%q) || document.querySelector('input[name="VIN"]') || document.querySelector('input[name="vin"]') || document.querySelector('input[placeholder*="VIN" i]');
    if (el) {
        const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')?.set;
        if (setter) setter.call(el, '');
        else el.value = '';
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.focus();
    }
})();
`, state.VinInputSelector), nil))
}

// extractFastReportData waits up to 12 seconds (polling every 250ms) for the newly opened Carfax report tab
// to render its tables, verifies the tab belongs to the requested VIN, distinguishes true "Invalid VIN"
// from slow network loads ("REPORT_LOAD_TIMEOUT"), and extracts all structured report data in <25ms.
func (sh *StealthHarvester) extractFastReportData(tabCtx context.Context, vin string) (string, string, string, string, string, error) {
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))

	// Wait up to 12 seconds (48 * 250ms) for report DOM tables or explicit CARFAX error banner
	for domWait := 0; domWait < 48; domWait++ {
		var domReady bool
		_ = chromedp.Run(tabCtx, chromedp.Evaluate(`
			(() => {
				const txt = (document.body ? document.body.innerText : '').toLowerCase();
				const hasTables = document.querySelectorAll('table tbody tr').length >= 2;
				return (
					hasTables ||
					txt.includes('ownership history') ||
					txt.includes('detailed history') ||
					txt.includes('history-based value') ||
					txt.includes('additional history') ||
					txt.includes('title history') ||
					txt.includes('invalid vin') ||
					txt.includes('not a valid vin') ||
					txt.includes('no records found')
				);
			})()
		`, &domReady))
		if domReady {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	fastJS := fmt.Sprintf(`
(() => {
    const reqVin = %q.toUpperCase().trim();
    const url = (window.location.href || '');
    const bodyText = (document.body ? document.body.innerText : '');
    const lowerText = bodyText.toLowerCase();

    // 1. Check if session expired or login screen appeared inside the report tab
    const passInput = document.querySelector('input[type="password"]');
    if ((passInput && passInput.offsetParent !== null) || url.toLowerCase().includes('/login') || url.toLowerCase().includes('/signin')) {
        return {
            isError: true,
            errorCode: 'SESSION_EXPIRED',
            errorMsg: 'انتهت جلسة تسجيل الدخول في كارفاكس أثناء فتح التقرير.'
        };
    }

    // 2. Check for genuine CARFAX invalid VIN / no records message
    if (lowerText.includes('invalid vin') || lowerText.includes('not a valid vin') || lowerText.includes('no records found for this vin') || lowerText.includes('cannot find any records')) {
        return {
            isError: true,
            errorCode: 'VIN_NOT_FOUND_ON_CARFAX',
            errorMsg: 'أفاد موقع كارفاكس أن رقم الشاصي هذا غير صحيح أو لا توجد له سجلات في قاعدة بيانات كارفاكس.'
        };
    }

    // 3. Check if the report DOM did not finish loading yet (transient timeout, NOT invalid VIN)
    const hasReportContent = url.toLowerCase().includes('/vhr/') ||
        lowerText.includes('vehicle history report') ||
        lowerText.includes('ownership history') ||
        lowerText.includes('detailed history') ||
        document.querySelectorAll('table tbody tr').length >= 2;

    if (!hasReportContent) {
        return {
            isError: true,
            errorCode: 'REPORT_LOAD_TIMEOUT',
            errorMsg: 'استغرق تحميل صفحة التقرير وقتاً أطول من المعتاد، جاري إعادة المحاولة تلقائياً.'
        };
    }

    // 4. Verify VIN match if a 17-character VIN is explicitly printed in the report body
    const vinMatch = bodyText.match(/\b([A-HJ-NPR-Z0-9]{17})\b/);
    if (vinMatch && vinMatch[1] && reqVin.length === 17 && vinMatch[1].toUpperCase() !== reqVin && !bodyText.toUpperCase().includes(reqVin)) {
        return {
            isError: true,
            errorCode: 'REPORT_LOAD_TIMEOUT',
            errorMsg: 'تعارض مؤقت في تبويب التقرير، جاري إعادة الجلب تلقائياً.'
        };
    }

    // 5. Extract Title (Year Make Model)
    let title = '';
    const titleSelectors = [
        '.vehicle-information-year-make-model',
        '.header-vehicle-name',
        '[class*="year-make-model" i]',
        'h1',
        'h2.vehicle-title'
    ];
    for (const sel of titleSelectors) {
        const el = document.querySelector(sel);
        if (el && el.innerText && el.innerText.trim().length > 3) {
            title = el.innerText.trim().replace(/\s+/g, ' ');
            break;
        }
    }
    if (!title) {
        title = (document.title || '').replace('CARFAX Vehicle History Report for this ', '').split(':')[0].trim() || ('CARFAX Report ' + reqVin);
    }

    // 6. Odometer
    let odometer = 'غير مسجل';
    const odoMatch = bodyText.match(/(\d{1,3}(?:,\d{3})+|\d+)\s*(?:mi|miles|km)\b/i);
    if (odoMatch) {
        odometer = odoMatch[0].trim();
    }

    // 7. Carfax Value / Retail Value
    let carfaxValue = '';
    const valMatch = bodyText.match(/(?:CARFAX\s*VALUE|Retail\s*Value|History-Based\s*Value)[^\$\n]{0,40}(\$[0-9,]+)/i) || bodyText.match(/(\$[0-9]{1,3}(?:,[0-9]{3})+)/);
    if (valMatch && valMatch[1]) {
        carfaxValue = valMatch[1].trim();
    }

    // 8. Vehicle Specifications (Body, Engine, Fuel, Drivetrain)
    const specs = [];
    const specLines = bodyText.split('\n').map(s => s.trim()).filter(Boolean);
    for (const line of specLines) {
        if (/^(Sedan|SUV|Coupe|Convertible|Hatchback|Pickup|Wagon|Van|Minivan|Truck|\d\.\dL\s|Gasoline|Diesel|Hybrid|Electric|Front wheel drive|Rear wheel drive|All wheel drive|4WD|AWD|FWD|RWD)/i.test(line) && line.length < 65) {
            if (!specs.includes(line)) specs.push(line);
        }
    }

    // 9. Owners Count
    let owners = 1;
    const ownerBanner = bodyText.match(/CARFAX\s+(\d+)-Owner/i);
    if (ownerBanner && ownerBanner[1]) {
        owners = parseInt(ownerBanner[1], 10) || 1;
    } else {
        const ownerCols = bodyText.match(/Owner\s+(\d+)/gi) || [];
        for (const oc of ownerCols) {
            const num = parseInt(oc.replace(/\D+/g, ''), 10);
            if (num > owners) owners = num;
        }
    }

    // 10. Accident / Damage Status
    let accidentsReported = 0;
    let accidentSummary = 'No accidents or damage reported to CARFAX';
    let damageSeverity = 'None';
    if (!lowerText.includes('no accidents or damage reported')) {
        if (lowerText.includes('severe damage')) {
            accidentsReported = 1;
            accidentSummary = 'Severe Damage Reported';
            damageSeverity = 'Severe';
        } else if (lowerText.includes('moderate damage')) {
            accidentsReported = 1;
            accidentSummary = 'Moderate Damage Reported';
            damageSeverity = 'Moderate';
        } else if (lowerText.includes('minor damage')) {
            accidentsReported = 1;
            accidentSummary = 'Minor Damage Reported';
            damageSeverity = 'Minor';
        } else if (lowerText.includes('vandalism damage')) {
            accidentsReported = 1;
            accidentSummary = 'Vandalism Damage Reported';
            damageSeverity = 'Moderate';
        } else if (lowerText.includes('accident reported') || lowerText.includes('damage reported')) {
            accidentsReported = 1;
            accidentSummary = 'Accident / Damage Reported';
            damageSeverity = 'Reported';
        }
    }

    // 11. Service Records, Vehicle Use, Last Location, Detailed Records Count
    let serviceRecordsText = '';
    const srvMatch = bodyText.match(/(\d+\s+Service\s+History\s+Records?)/i);
    if (srvMatch) serviceRecordsText = srvMatch[1].trim();

    let vehicleUse = 'Personal Vehicle';
    const useMatch = bodyText.match(/(Personal\s+Vehicle|Rental\s+Vehicle|Commercial\s+Vehicle|Personal\s+Lease|Fleet\s+Vehicle|Lease\s+Vehicle)/i);
    if (useMatch) vehicleUse = useMatch[1].trim();

    let lastLocation = '';
    const locMatch = bodyText.match(/Last\s+Owned\s+in\s+([A-Za-z\s]+?)(?:\n|$|\r)/i);
    if (locMatch) lastLocation = 'Last Owned in ' + locMatch[1].trim();

    let detailedCountText = '';
    const detMatch = bodyText.match(/(\d+\s+Detailed\s+Records?\s+Available)/i);
    if (detMatch) detailedCountText = detMatch[1].trim();

    // Helper to extract clean text from a cell without screen-reader duplicate spans
    const cleanCellText = (el, multiline = false) => {
        if (!el) return '';
        const c = el.cloneNode(true);
        c.querySelectorAll('.visually-hidden, .narrow-record-odometer-reading, caption, svg, script, style').forEach(n => n.remove());
        if (multiline) {
            c.querySelectorAll('p, li, div, strong, h1, h2, h3, h4, br').forEach(n => {
                if (n.tagName === 'BR') n.replaceWith('\n');
                else n.append('\n');
            });
            return (c.innerText || c.textContent || '')
                .split('\n')
                .map(s => s.replace(/\s+/g, ' ').trim())
                .filter(Boolean)
                .join('\n');
        }
        return (c.innerText || c.textContent || '').replace(/\s+/g, ' ').trim();
    };

    // 12. Extract Structured Tables (Ownership, Title, Additional History, and Detailed History Timeline)
    const tables = [];
    const detailedRecords = [];
    const allTables = Array.from(document.querySelectorAll('table'));

    for (const tbl of allTables) {
        const headers = Array.from(tbl.querySelectorAll('thead th, thead td')).map(th => cleanCellText(th, false)).filter(Boolean);
        if (headers.length > 0 && headers[0].toLowerCase().includes('service & comments')) continue;
        const rows = [];
        const trs = Array.from(tbl.querySelectorAll('tbody tr'));
        for (const tr of trs) {
            if (tr.classList.contains('detailed-history-row-supplementary')) continue;
            const rawCells = Array.from(tr.querySelectorAll('th, td'));
            const cells = rawCells.map(cell => cleanCellText(cell, false));
            const multiCells = rawCells.map(cell => cleanCellText(cell, true));
            if (cells.length === 0 || cells.every(c => !c)) continue;

            const firstCell = cells[0] || '';
            if (cells.length >= 3 && (/^\d{1,2}\/\d{1,2}\/\d{2,4}$/.test(firstCell) || /^Not Reported$/i.test(firstCell))) {
                if (cells.length >= 5) {
                    detailedRecords.push({
                        date: cells[0] || '',
                        mileage: cells[1] || '',
                        source: multiCells[2] || '',
                        comments: multiCells[4] || ''
                    });
                } else {
                    detailedRecords.push({
                        date: cells[0] || '',
                        mileage: cells.length >= 4 ? (cells[1] || '') : '',
                        source: cells.length >= 4 ? (multiCells[2] || '') : (multiCells[1] || ''),
                        comments: cells.length >= 4 ? multiCells.slice(3).join('\n') : (multiCells[2] || '')
                    });
                }
            } else {
                rows.push(multiCells);
            }
        }
        if (rows.length > 0) {
            tables.push({
                title: headers[0] || 'CARFAX Section',
                headers: headers,
                rows: rows
            });
        }
    }

    // 13. Extract Accident / Damage Events if present
    const accidentEvents = [];
    const accSec = document.getElementById('accident-damage-section');
    if (accSec) {
        const evRecords = Array.from(accSec.querySelectorAll('.accident-damage-record'));
        evRecords.forEach((ev, idx) => {
            accidentEvents.push({
                title: 'Event ' + (idx + 1),
                details: cleanCellText(ev, true)
            });
        });
    }

    const cleanTitle = !(lowerText.includes('salvage title') || lowerText.includes('rebuilt title') || lowerText.includes('total loss reported'));

    const summaryObj = {
        vin: reqVin,
        vehicle_title: title,
        carfax_score: accidentsReported > 0 ? 82 : 95,
        carfax_value: carfaxValue,
        total_owners: owners,
        accidents_reported: accidentsReported,
        accident_summary: accidentSummary,
        damage_severity: damageSeverity,
        clean_title: cleanTitle,
        last_reported_odometer: odometer,
        service_records_count: serviceRecordsText,
        vehicle_use: vehicleUse,
        last_owned_location: lastLocation,
        detailed_records_count: detailedCountText,
        specs: specs,
        tables: tables,
        detailed_records: detailedRecords,
        accident_events: accidentEvents,
        background_pending: true,
        background_ready: false,
        analyzed_at: new Date().toISOString()
    };

    return {
        isError: false,
        errorCode: '',
        errorMsg: '',
        title: title,
        summaryJson: JSON.stringify(summaryObj),
        lightHtml: ''
    };
})();
`, cleanVin)

	var out struct {
		IsError     bool   `json:"isError"`
		ErrorCode   string `json:"errorCode"`
		ErrorMsg    string `json:"errorMsg"`
		Title       string `json:"title"`
		SummaryJson string `json:"summaryJson"`
		LightHtml   string `json:"lightHtml"`
	}

	if err := chromedp.Run(tabCtx, chromedp.Evaluate(fastJS, &out)); err != nil {
		return "", "", "", "REPORT_LOAD_TIMEOUT", "", err
	}

	if out.IsError {
		code := out.ErrorCode
		if code == "" {
			code = "REPORT_LOAD_TIMEOUT"
		}
		return "", "", "", code, out.ErrorMsg, nil
	}

	return out.SummaryJson, out.LightHtml, out.Title, "", "", nil
}

// pasteVinInstantly mimics a dealer pressing Ctrl+V into the CARFAX VIN box (~0.18s instead of 3.5s)
// using the native HTMLInputElement value setter for the first 16 chars + a real trusted CDP KeyEvent for the 17th char
// so Chrome grants Transient User Activation (preventing Chrome's Popup Blocker from blocking window.open!)
func (sh *StealthHarvester) pasteVinInstantly(tabCtx context.Context, selector, vin string) error {
	cleanVin := strings.ToUpper(strings.TrimSpace(vin))
	if len(cleanVin) < 17 {
		return fmt.Errorf("invalid VIN length: %s", cleanVin)
	}
	sh.randomDelay(50, 100)

	prefix16 := cleanVin[:16]
	lastChar := cleanVin[16:]

	// 1. Focus and paste the first 16 characters instantly
	pastePrefixJS := fmt.Sprintf(`
(() => {
    // Dismiss any leftover "Popup Blocked" modal if open
    const closeBtns = Array.from(document.querySelectorAll('button, span, a, [role="button"], [aria-label*="close" i]'));
    for (const cb of closeBtns) {
        if (cb.offsetParent === null) continue;
        const t = (cb.innerText || cb.textContent || '').trim();
        const aria = (cb.getAttribute('aria-label') || '').toLowerCase();
        if (t === '×' || t === '✕' || aria.includes('close')) {
            cb.click();
        }
    }

    const el = document.querySelector(%q) ||
        document.querySelector('input[name="VIN"]') ||
        document.querySelector('input[name="vin"]') ||
        document.querySelector('input[placeholder*="VIN" i]') ||
        document.querySelector('input[id*="vin" i]');
    if (!el) return false;
    el.focus();
    const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')?.set;
    if (setter) {
        setter.call(el, %q);
    } else {
        el.value = %q;
    }
    el.dispatchEvent(new InputEvent('input', { bubbles: true, cancelable: true, inputType: 'insertFromPaste', data: %q }));
    return true;
})();
`, selector, prefix16, prefix16, prefix16)

	var okPrefix bool
	if err := chromedp.Run(tabCtx, chromedp.Evaluate(pastePrefixJS, &okPrefix)); err != nil || !okPrefix {
		return fmt.Errorf("لم يتم العثور على حقل الشاصي (%s)", selector)
	}

	// 2. Send the 17th character via a real trusted CDP KeyEvent (event.isTrusted = true -> grants Chrome User Activation for popups!)
	_ = chromedp.Run(tabCtx, chromedp.KeyEvent(lastChar))

	// 3. Verify all 17 characters are in the input box
	verifyJS := fmt.Sprintf(`
(() => {
    const el = document.querySelector(%q) ||
        document.querySelector('input[name="VIN"]') ||
        document.querySelector('input[name="vin"]') ||
        document.querySelector('input[placeholder*="VIN" i]') ||
        document.querySelector('input[id*="vin" i]');
    if (!el) return '';
    el.focus();
    if ((el.value || '').trim().toUpperCase() !== %q) {
        const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')?.set;
        if (setter) {
            setter.call(el, %q);
        } else {
            el.value = %q;
        }
        el.dispatchEvent(new InputEvent('input', { bubbles: true, cancelable: true, inputType: 'insertFromPaste', data: %q }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
    }
    return (el.value || '').trim().toUpperCase();
})();
`, selector, cleanVin, cleanVin, cleanVin, cleanVin)

	var finalVal string
	if err := chromedp.Run(tabCtx, chromedp.Evaluate(verifyJS, &finalVal)); err != nil {
		return err
	}
	if finalVal != cleanVin {
		return fmt.Errorf("القيمة الملصقة (%s) لا تطابق الشاصي المطلوب (%s)", finalVal, cleanVin)
	}

	sh.randomDelay(60, 120)
	return nil
}

// submitVinSearch submits the VIN using a real trusted CDP Enter KeyEvent first (preserving Chrome User Activation
// so window.open popup tabs are NEVER blocked), and also clicks the Run Report button if still on the form.
func (sh *StealthHarvester) submitVinSearch(tabCtx context.Context, selector string) {
	// 1. Focus the VIN input and press a trusted CDP Enter key (event.isTrusted = true allows popups!)
	_ = chromedp.Run(tabCtx,
		chromedp.Evaluate(fmt.Sprintf(`
(() => {
    const input = document.querySelector(%q) ||
        document.querySelector('input[name="VIN"]') ||
        document.querySelector('input[name="vin"]') ||
        document.querySelector('input[placeholder*="VIN" i]');
    if (input) input.focus();
})()
`, selector), nil),
		chromedp.KeyEvent("\r"),
	)

	// 2. Also click the explicit Run Report button if needed
	submitJS := fmt.Sprintf(`
(() => {
    const input = document.querySelector(%q) ||
        document.querySelector('input[name="VIN"]') ||
        document.querySelector('input[name="vin"]') ||
        document.querySelector('input[placeholder*="VIN" i]');

    if (input && input.form) {
        const formBtn = input.form.querySelector('button[type="submit"], input[type="submit"], button:not([type="button"]):not([type="reset"])');
        if (formBtn && formBtn.offsetParent !== null && !formBtn.disabled) {
            formBtn.click();
            return 'clicked_form_btn';
        }
    }

    const allBtns = Array.from(document.querySelectorAll('button, input[type="submit"], input[type="button"], a[role="button"]'));
    for (const b of allBtns) {
        if (b.offsetParent === null || b.disabled) continue;
        const txt = (b.innerText || b.value || b.textContent || '').trim().toLowerCase();
        const id = (b.id || '').toLowerCase();
        const aria = (b.getAttribute('aria-label') || '').toLowerCase();
        if (txt === 'run report' || txt === 'get report' || txt === 'search' || txt === 'submit' ||
            txt.includes('run report') || id.includes('run') || id.includes('vhr') || aria.includes('run report')) {
            b.click();
            return 'clicked_named_btn';
        }
    }
    return 'enter_sent';
})();
`, selector)

	var submitMethod string
	_ = chromedp.Run(tabCtx, chromedp.Evaluate(submitJS, &submitMethod))
}

// prepareOfficialPrintModal clicks the top "Print" button on the Carfax report page,
// polls every 120ms for the "Printing Options" modal, checks all 4 checkboxes
// (CARFAX Report, History-Based Value, Warranty Check, Buyback Guarantee) and clicks Print in the same tick,
// then polls every 150ms until window.print() is triggered or all 4 sections render (~1.8s-2.5s total instead of ~8s).
func (sh *StealthHarvester) prepareOfficialPrintModal(
	parentCtx context.Context,
	cdpEndpoint string,
	activeReportCtx context.Context,
) (context.Context, context.CancelFunc, string) {
	tabsBeforePrint, _ := sh.ListOpenTabs(cdpEndpoint)
	knownBeforePrint := make(map[string]bool)
	for _, t := range tabsBeforePrint {
		knownBeforePrint[t.ID] = true
	}

	// 1. Intercept window.print() on current document and any future document in this tab
	overridePrintJS := `
(() => {
    window.__carfaxPrintTriggered = false;
    window.print = function() {
        window.__carfaxPrintTriggered = true;
    };
})();
`
	_ = chromedp.Run(activeReportCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, _ = page.AddScriptToEvaluateOnNewDocument(overridePrintJS).Do(ctx)
			return nil
		}),
		chromedp.Evaluate(overridePrintJS, nil),
	)

	// 2. Click the top-right "Print" button on the Carfax report page to open "Printing Options"
	clickTopPrintJS := `
(() => {
    const els = Array.from(document.querySelectorAll('button, a, [role="button"]'));
    for (const el of els) {
        if (el.offsetParent === null) continue;
        const txt = (el.innerText || el.textContent || '').trim();
        const id = (el.id || '').toLowerCase();
        const cls = (typeof el.className === 'string' ? el.className : '').toLowerCase();
        if (txt === 'Print' || txt.includes('Print') || id.includes('print') || cls.includes('print-btn') || cls.includes('print-button')) {
            el.click();
            return true;
        }
    }
    return false;
})();
`
	var clickedTopPrint bool
	if err := chromedp.Run(activeReportCtx, chromedp.Evaluate(clickTopPrintJS, &clickedTopPrint)); err != nil || !clickedTopPrint {
		return activeReportCtx, nil, ""
	}

	// 3. Two-Step Visible & React-Safe Modal Handling:
	// Step 3A: Wait for "Printing Options" modal to open, then check all 4 checkboxes (without clicking Print yet!)
	checkAllFourBoxesJS := `
(() => {
    const allContainers = Array.from(document.querySelectorAll('[role="dialog"], [class*="modal" i], [class*="dialog" i], div'));
    let modal = null;
    for (const d of allContainers) {
        if (d.offsetParent === null) continue;
        const t = (d.innerText || '');
        if ((t.includes('Printing Options') || t.includes('Print Selected Reports')) && t.includes('Cancel')) {
            modal = d;
        }
    }
    if (!modal) {
        return { ready: false, checked: 0 };
    }

    // Click "Select All" if present and visible
    const links = Array.from(modal.querySelectorAll('a, button, span, [role="button"]'));
    for (const l of links) {
        const txt = (l.innerText || l.textContent || '').trim().toLowerCase();
        if (txt === 'select all' && l.offsetParent !== null) {
            l.click();
            break;
        }
    }

    // Ensure all 4 report checkboxes (CARFAX Report, History-Based Value, Warranty Check, Buyback Guarantee) are checked
    const checkboxes = Array.from(modal.querySelectorAll('input[type="checkbox"]'));
    let checkedCount = 0;
    for (const cb of checkboxes) {
        const labelText = ((cb.closest('label') ? cb.closest('label').innerText : '') || (cb.parentElement ? cb.parentElement.innerText : '') || '').toLowerCase();
        if (labelText.includes('display carfax in online listings')) {
            continue;
        }
        if (!cb.checked) {
            cb.click();
        }
        if (cb.checked) {
            checkedCount++;
        }
    }

    return { ready: true, checked: checkedCount };
})();
`
	modalReady := false
	for i := 0; i < 20; i++ {
		time.Sleep(150 * time.Millisecond)
		var mOut struct {
			Ready   bool `json:"ready"`
			Checked int  `json:"checked"`
		}
		if err := chromedp.Run(activeReportCtx, chromedp.Evaluate(checkAllFourBoxesJS, &mOut)); err == nil && mOut.Ready {
			modalReady = true
			break
		}
	}

	// Step 3B: Pause 750ms-950ms while the 4 checkboxes are visibly checked on screen!
	// This makes the action clearly visible to the human eye AND lets Carfax's React state mount all 4 sections cleanly!
	sh.randomDelay(750, 950)

	// Step 3C: Now click the modal's "Print" button (with a second pass on checkboxes just in case React re-rendered)
	clickModalPrintBtnJS := `
(() => {
    const checkboxes = Array.from(document.querySelectorAll('input[type="checkbox"]'));
    for (const cb of checkboxes) {
        const labelText = ((cb.closest('label') ? cb.closest('label').innerText : '') || (cb.parentElement ? cb.parentElement.innerText : '') || '').toLowerCase();
        if (labelText.includes('display carfax in online listings')) continue;
        if (!cb.checked) cb.click();
    }

    const allContainers = Array.from(document.querySelectorAll('[role="dialog"], [class*="modal" i], [class*="dialog" i], div'));
    let modal = null;
    for (const d of allContainers) {
        if (d.offsetParent === null) continue;
        const t = (d.innerText || '');
        if ((t.includes('Printing Options') || t.includes('Print Selected Reports')) && t.includes('Cancel')) {
            modal = d;
        }
    }
    const root = modal || document;
    const buttons = Array.from(root.querySelectorAll('button, input[type="button"], input[type="submit"], a[role="button"]'));
    let targetBtn = null;
    for (const b of buttons) {
        if (b.offsetParent === null) continue;
        const txt = (b.innerText || b.value || b.textContent || '').trim();
        if (txt === 'Print') {
            targetBtn = b;
        }
    }
    if (targetBtn) {
        targetBtn.click();
        return true;
    }
    return false;
})();
`
	var modalPrinted bool
	_ = chromedp.Run(activeReportCtx, chromedp.Evaluate(clickModalPrintBtnJS, &modalPrinted))
	_ = modalReady

	// 4. Event-Driven Print Readiness Poll (every 150ms, up to 24 attempts = 3.6s max):
	// Return as soon as window.__carfaxPrintTriggered is true OR a new print popup tab opens!
	for i := 0; i < 24; i++ {
		time.Sleep(150 * time.Millisecond)

		// Check if Carfax opened a separate print tab
		tabsAfterPrint, _ := sh.ListOpenTabs(cdpEndpoint)
		for idx := range tabsAfterPrint {
			t := tabsAfterPrint[idx]
			if !knownBeforePrint[t.ID] && t.URL != "about:blank" {
				sh.markTabActive(t.ID)
				if pCtx, pCleanup, pErr := sh.attachExistingTab(parentCtx, cdpEndpoint, t.ID); pErr == nil {
					_ = chromedp.Run(pCtx, chromedp.Evaluate(overridePrintJS, nil))
					sh.randomDelay(850, 1150)
					return pCtx, pCleanup, t.ID
				}
			}
		}

		// Check if window.print() was triggered in the current report tab
		var printTriggered bool
		_ = chromedp.Run(activeReportCtx, chromedp.Evaluate(`Boolean(window.__carfaxPrintTriggered)`, &printTriggered))
		if printTriggered {
			break
		}
	}

	// Give the 4 mounted sections (History-Based Value, Warranty Check, Buyback Guarantee) 850ms-1100ms to finish rendering images/SVGs
	sh.randomDelay(850, 1100)

	// 5. Hide any remaining "Printing Options" modal dialog / backdrop so the report & PDF are 100% unobstructed
	hideModalJS := `
(() => {
    const allContainers = Array.from(document.querySelectorAll('[role="dialog"], [class*="modal" i], [class*="dialog" i], [class*="backdrop" i], [class*="overlay" i]'));
    for (const el of allContainers) {
        const t = (el.innerText || '');
        const cls = (typeof el.className === 'string' ? el.className : '').toLowerCase();
        if (t.includes('Printing Options') || t.includes('Print Selected Reports') || cls.includes('backdrop') || cls.includes('modal-overlay') || el.getAttribute('role') === 'dialog') {
            el.style.setProperty('display', 'none', 'important');
            el.style.setProperty('visibility', 'hidden', 'important');
        }
    }
    if (document.body) {
        document.body.style.overflow = 'visible';
    }
})();
`
	_ = chromedp.Run(activeReportCtx, chromedp.Evaluate(hideModalJS, nil))
	sh.randomDelay(180, 280)

	return activeReportCtx, nil, ""
}

// captureOfficialPrintPDF captures the full 4-section Carfax report via Chrome's native Page.printToPDF
// and saves a local copy in C:\CarfaxReports\CARFAX_<VIN>.pdf
func (sh *StealthHarvester) captureOfficialPrintPDF(tabCtx context.Context, vin string) string {
	var pdfBase64 string
	_ = chromedp.Run(tabCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		pdfBuf, _, pdfErr := page.PrintToPDF().
			WithPrintBackground(true).
			WithPreferCSSPageSize(true).
			Do(ctx)
		if pdfErr == nil && len(pdfBuf) > 0 {
			pdfBase64 = base64.StdEncoding.EncodeToString(pdfBuf)
			saveDir := `C:\CarfaxReports`
			if mkErr := os.MkdirAll(saveDir, 0755); mkErr == nil {
				cleanVin := strings.ToUpper(strings.TrimSpace(vin))
				if cleanVin == "" {
					cleanVin = "REPORT"
				}
				pdfPath := filepath.Join(saveDir, fmt.Sprintf("CARFAX_%s.pdf", cleanVin))
				if wErr := os.WriteFile(pdfPath, pdfBuf, 0644); wErr == nil {
					log.Printf("[💾] Saved Official Carfax PDF (%d KB) to %s", len(pdfBuf)/1024, pdfPath)
				}
			}
		} else if pdfErr != nil {
			log.Printf("⚠️ [PDF CAPTURE WARNING] PrintToPDF error: %v", pdfErr)
		}
		return nil
	}))

	return pdfBase64
}

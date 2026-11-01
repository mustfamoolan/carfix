package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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
	rnd        *rand.Rand
	httpClient *http.Client
	assetCache sync.Map
}

func NewStealthHarvester() *StealthHarvester {
	return &StealthHarvester{
		rnd:        rand.New(rand.NewSource(time.Now().UnixNano())),
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
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

// FindCarfaxSearchTab finds the open carfaxonline.com tab, or falls back to the first visible page tab.
// Never closes any user tabs.
func (sh *StealthHarvester) FindCarfaxSearchTab(cdpEndpoint string) (*CDPTab, bool, error) {
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
    const bodyText = (document.body ? document.body.innerText : '').toLowerCase();

    // 1. Check for bot/captcha challenges
    const challengeWords = ['verify you are human', 'cloudflare', 'cf-chl', 'datadome', 'recaptcha', 'access denied', 'unusual traffic'];
    let hasChallenge = false;
    for (const w of challengeWords) {
        if (url.includes(w) || bodyText.includes(w)) {
            hasChallenge = true;
            break;
        }
    }

    // 2. Check if we are on a login screen
    const passInput = document.querySelector('input[type="password"]');
    const isLogin = Boolean(passInput && passInput.offsetParent !== null) || url.includes('/login') || url.includes('/signin');

    // 3. Locate VIN input box on carfaxonline.com
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
		URL         string `json:"url"`
		Title       string `json:"title"`
		VinFound    bool   `json:"vinFound"`
		VinSelector string `json:"vinSelector"`
		SubmitFound bool   `json:"submitFound"`
		IsLogin     bool   `json:"isLogin"`
		HasChallenge bool  `json:"hasChallenge"`
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
		res.SessionActive = false
		res.StatusMessage = "ℹ️ صفحة carfaxonline.com مفتوحة ولكن لم يتم رصد حقل إدخال الـ VIN في هذه الشاشة"
	}

	return res
}

// HarvestReport executes the strict One-Shot Or Freeze protocol on carfaxonline.com
// using Two-Stage Progressive Delivery:
//   - Stage 1 (Fast): Immediately extracts structured report data + lightweight HTML (~2s) and triggers onFastData so the user views the report right away in our design.
//   - Stage 2 (Silent Background): Opens Printing Options -> Select All -> PrintToPDF (saves to C:\CarfaxReports\CARFAX_<VIN>.pdf) + inlines all CSS/Images and returns the full bundle.
func (sh *StealthHarvester) HarvestReport(
	parentCtx context.Context,
	cdpEndpoint string,
	vin string,
	onProgress func(step, stepAr string, screenshot string),
	onFastData func(fastRes *HarvestResult),
) *HarvestResult {
	// 1. Pre-Flight Silent Inspection
	if onProgress != nil {
		onProgress("pre_flight", "1/4 فحص أمان وجاهزية متصفح كندا (Pre-Flight Check)...", "")
	}

	state := sh.InspectBrowserState(parentCtx, cdpEndpoint, true)

	if !state.ChromeConnected {
		return &HarvestResult{
			Success:      false,
			ErrorCode:    "CHROME_NOT_CONNECTED",
			ErrorMessage: state.StatusMessage,
		}
	}

	if state.HasChallenge {
		return &HarvestResult{
			Success:          false,
			ErrorCode:        "CAPTCHA_DETECTED",
			ErrorMessage:     state.StatusMessage,
			ScreenshotBase64: state.ScreenshotBase64,
		}
	}

	if state.IsLoginPage {
		return &HarvestResult{
			Success:          false,
			ErrorCode:        "SESSION_EXPIRED",
			ErrorMessage:     state.StatusMessage,
			ScreenshotBase64: state.ScreenshotBase64,
		}
	}

	// If Chrome is open (and not on a login/captcha page) but not currently showing the VIN input box
	// (e.g. user left it on a sub-page or New Tab), navigate once automatically to https://www.carfaxonline.com/
	if !state.VinInputFound && state.TabID != "" {
		if navTabCtx, cleanupNav, err := sh.attachExistingTab(parentCtx, cdpEndpoint, state.TabID); err == nil {
			_ = chromedp.Run(navTabCtx, chromedp.Navigate("https://www.carfaxonline.com/"))
			cleanupNav()
			sh.randomDelay(2500, 3500)
			state = sh.InspectBrowserState(parentCtx, cdpEndpoint, true)
		}
	}

	if !state.CarfaxTabFound || state.IsLoginPage {
		return &HarvestResult{
			Success:          false,
			ErrorCode:        "SESSION_EXPIRED",
			ErrorMessage:     state.StatusMessage,
			ScreenshotBase64: state.ScreenshotBase64,
		}
	}

	if !state.VinInputFound || state.VinInputSelector == "" {
		return &HarvestResult{
			Success:          false,
			ErrorCode:        "VIN_INPUT_NOT_FOUND",
			ErrorMessage:     "توقف احترازي (One-Shot Protection): حقل إدخال الشاصي غير ظاهر في الصفحة الحالية داخل حاسبة كندا.",
			ScreenshotBase64: state.ScreenshotBase64,
		}
	}

	// Connect to the search tab without closing it on disconnect
	searchTab, _, err := sh.FindCarfaxSearchTab(cdpEndpoint)
	if err != nil || searchTab == nil {
		return &HarvestResult{
			Success:          false,
			ErrorCode:        "TAB_LOST",
			ErrorMessage:     "فقد الاتصال بتبويب البحث قبل البدء مباشرة.",
			ScreenshotBase64: state.ScreenshotBase64,
		}
	}

	// Clean up any leftover /vhr/ popup tabs from previous searches (except the main search tab)
	// so there is zero chance of reading an older VIN's popup tab
	tabsBefore, _ := sh.ListOpenTabs(cdpEndpoint)
	knownTabIDs := make(map[string]bool)
	upperTargetVin := strings.ToUpper(strings.TrimSpace(vin))
	for _, t := range tabsBefore {
		if t.ID != searchTab.ID && strings.Contains(strings.ToLower(t.URL), "/vhr/") && !strings.Contains(strings.ToUpper(t.URL), upperTargetVin) {
			sh.CloseTabByID(cdpEndpoint, t.ID)
			continue
		}
		knownTabIDs[t.ID] = true
	}

	tabCtx, cleanupSearchTab, attachErr := sh.attachExistingTab(parentCtx, cdpEndpoint, searchTab.ID)
	if attachErr != nil {
		return &HarvestResult{
			Success:          false,
			ErrorCode:        "TAB_LOST",
			ErrorMessage:     fmt.Sprintf("تعذر الارتباط بتبويب البحث: %v", attachErr),
			ScreenshotBase64: state.ScreenshotBase64,
		}
	}
	defer cleanupSearchTab()

	// 2. Human VIN Typing
	if onProgress != nil {
		onProgress("typing_vin", fmt.Sprintf("2/4 جاري كتابة رقم الشاصي (%s) بمحاكاة بشرية...", vin), state.ScreenshotBase64)
	}

	if err := sh.typeVinHumanly(tabCtx, state.VinInputSelector, vin); err != nil {
		snap := sh.captureCompressedJPEG(tabCtx)
		return &HarvestResult{
			Success:          false,
			ErrorCode:        "VIN_TYPE_ERROR",
			ErrorMessage:     fmt.Sprintf("توقف احترازي: تعذر الكتابة في حقل الشاصي (%v)", err),
			ScreenshotBase64: snap,
		}
	}

	// 3. Submit Search (Click button or press Enter)
	if onProgress != nil {
		onProgress("submitting_search", "3/4 تم إدخال الشاصي، جاري طلب التقرير من carfaxonline.com...", "")
	}

	submitJS := fmt.Sprintf(`
(() => {
    const input = document.querySelector(%q);
    const btnSelectors = [
        'button[type="submit"]',
        'input[type="submit"]',
        'button[id*="run" i]',
        'button[id*="search" i]',
        'button[id*="vhr" i]',
        'button[class*="search" i]',
        'button[class*="submit" i]'
    ];
    for (const sel of btnSelectors) {
        const b = document.querySelector(sel);
        if (b && b.offsetParent !== null && !b.disabled) {
            b.click();
            return 'clicked_button';
        }
    }
    if (input && input.form) {
        input.form.submit();
        return 'submitted_form';
    }
    return 'enter_fallback';
})();
`, state.VinInputSelector)

	var submitMethod string
	_ = chromedp.Run(tabCtx, chromedp.Evaluate(submitJS, &submitMethod))
	if submitMethod == "enter_fallback" {
		_ = chromedp.Run(tabCtx, chromedp.KeyEvent("\r"))
	}

	// Wait 3.2 to 4.2 seconds for the report tab to open
	sh.randomDelay(3200, 4200)

	// Check if a new tab was opened for the report (standard on carfaxonline.com)
	tabsAfter, _ := sh.ListOpenTabs(cdpEndpoint)
	var reportTab *CDPTab
	openedNewTab := false
	for i := range tabsAfter {
		t := tabsAfter[i]
		if !knownTabIDs[t.ID] && t.URL != "about:blank" {
			reportTab = &t
			openedNewTab = true
			break
		}
	}
	// If no brand-new tab opened, check if an existing tab holds this exact VIN's report (/vhr/<VIN>)
	if reportTab == nil {
		for i := range tabsAfter {
			t := tabsAfter[i]
			if t.ID != searchTab.ID && (strings.Contains(strings.ToUpper(t.URL), upperTargetVin) || strings.Contains(strings.ToUpper(t.Title), upperTargetVin)) {
				reportTab = &t
				openedNewTab = true
				break
			}
		}
	}

	// If no new tab opened, the report is in the current searchTab
	activeReportCtx := tabCtx
	if openedNewTab && reportTab != nil {
		if rTabCtx, cleanupReportTab, rErr := sh.attachExistingTab(parentCtx, cdpEndpoint, reportTab.ID); rErr == nil {
			defer cleanupReportTab()
			activeReportCtx = rTabCtx
			// Wait 1.6s to 2.2s for the report DOM text & tables to render
			sh.randomDelay(1600, 2200)
		}
	}

	// =========================================================================
	// ⚡ STAGE 1 (FAST INSTANT DELIVERY): Extract Structured Data & Light HTML
	// =========================================================================
	if onProgress != nil {
		onProgress("extracting_fast", "4/4 تم فتح التقرير! جاري إرسال بيانات الفحص الفورية لتصميمنا وإكمال الطباعة بصمت...", "")
	}

	reportScreenshot := sh.captureCompressedJPEG(activeReportCtx)
	fastSummaryJson, lightHtml, fastTitle, isFastErr, fastErrMsg, fastErr := sh.extractFastReportData(activeReportCtx, vin)

	if fastErr == nil && isFastErr {
		if openedNewTab && reportTab != nil {
			sh.CloseTabByID(cdpEndpoint, reportTab.ID)
		}
		return &HarvestResult{
			Success:          false,
			ErrorCode:        "VIN_NOT_FOUND_ON_CARFAX",
			ErrorMessage:     fastErrMsg,
			ScreenshotBase64: reportScreenshot,
		}
	}

	if fastErr == nil && fastSummaryJson != "" && onFastData != nil {
		onFastData(&HarvestResult{
			Success:          true,
			VehicleTitle:     fastTitle,
			RawHtmlEn:        lightHtml,
			FastSummaryJson:  fastSummaryJson,
			ScreenshotBase64: reportScreenshot,
		})
	}

	// =========================================================================
	// 🔇 STAGE 2 (SILENT BACKGROUND): Gentle Scroll + Printing Options (Select All)
	//                                + Official PDF + Full Self-Contained HTML
	// =========================================================================
	sh.simulateGentleScroll(activeReportCtx)

	finalReportCtx, cleanupPrintTab, printPopupTabID := sh.prepareOfficialPrintModal(parentCtx, cdpEndpoint, activeReportCtx)
	if cleanupPrintTab != nil {
		defer cleanupPrintTab()
	}

	html, title, isErrorPage, errPageMsg, err := sh.extractClonedReportHTML(finalReportCtx, vin)
	if title == "" && fastTitle != "" {
		title = fastTitle
	}

	// Clean up any extra print popup tab if Carfax opened one
	if printPopupTabID != "" && (reportTab == nil || printPopupTabID != reportTab.ID) && printPopupTabID != searchTab.ID {
		sh.CloseTabByID(cdpEndpoint, printPopupTabID)
	}

	// Clean up: if a popup tab was opened for the report, close ONLY the popup tab so the main search tab stays clean
	if openedNewTab && reportTab != nil {
		sh.CloseTabByID(cdpEndpoint, reportTab.ID)
	} else {
		var stillHasVinInput bool
		_ = chromedp.Run(activeReportCtx, chromedp.Evaluate(fmt.Sprintf(`Boolean(document.querySelector(%q))`, state.VinInputSelector), &stillHasVinInput))
		if !stillHasVinInput {
			_ = chromedp.Run(activeReportCtx, chromedp.Evaluate(`window.history.back();`, nil))
		}
	}

	if err != nil {
		return &HarvestResult{
			Success:          false,
			ErrorCode:        "DOM_EXTRACT_ERROR",
			ErrorMessage:     fmt.Sprintf("تعذر نسخ محتوى التقرير من الصفحة: %v", err),
			ScreenshotBase64: reportScreenshot,
		}
	}

	if isErrorPage {
		return &HarvestResult{
			Success:          false,
			ErrorCode:        "VIN_NOT_FOUND_ON_CARFAX",
			ErrorMessage:     errPageMsg,
			ScreenshotBase64: reportScreenshot,
		}
	}

	return &HarvestResult{
		Success:          true,
		VehicleTitle:     title,
		RawHtmlEn:        html,
		FastSummaryJson:  fastSummaryJson,
		ScreenshotBase64: reportScreenshot,
	}
}

// extractFastReportData runs in <25ms on the newly opened Carfax report tab to extract all structured
// vehicle specs, Carfax Value, highlights, ownership history, title guarantee, technical checks,
// accident events, and chronological service records so the user sees the report immediately in our UI.
func (sh *StealthHarvester) extractFastReportData(tabCtx context.Context, vin string) (string, string, string, bool, string, error) {
	fastJS := fmt.Sprintf(`
(() => {
    const reqVin = %q.toUpperCase().trim();
    const url = (window.location.href || '');
    const bodyText = (document.body ? document.body.innerText : '');
    const lowerText = bodyText.toLowerCase();

    if (lowerText.includes('invalid vin') || lowerText.includes('not a valid vin') || lowerText.includes('no records found for this vin')) {
        return {
            isError: true,
            errorMsg: 'أفاد موقع كارفاكس أن رقم الشاصي هذا غير صحيح أو لا توجد له سجلات في قاعدة بيانات كارفاكس.'
        };
    }

    // Check if we are still stuck on the search home page without a report
    if (!url.toLowerCase().includes('/vhr/') && !lowerText.includes('vehicle history report') && !lowerText.includes('ownership history')) {
        return {
            isError: true,
            errorMsg: 'لم يفتح موقع كارفاكس صفحة تقرير لهذا الشاصي (يرجى التأكد من صحة الرقم أو عدم وجود تنبيه في صفحة البحث).'
        };
    }

    // 1. Extract Title (Year Make Model)
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

    // 2. Odometer
    let odometer = 'غير مسجل';
    const odoMatch = bodyText.match(/(\d{1,3}(?:,\d{3})+|\d+)\s*(?:mi|miles|km)\b/i);
    if (odoMatch) {
        odometer = odoMatch[0].trim();
    }

    // 3. Carfax Value / Retail Value
    let carfaxValue = '';
    const valMatch = bodyText.match(/(?:CARFAX\s*VALUE|Retail\s*Value|History-Based\s*Value)[^\$\n]{0,40}(\$[0-9,]+)/i) || bodyText.match(/(\$[0-9]{1,3}(?:,[0-9]{3})+)/);
    if (valMatch && valMatch[1]) {
        carfaxValue = valMatch[1].trim();
    }

    // 4. Vehicle Specifications (Body, Engine, Fuel, Drivetrain)
    const specs = [];
    const specLines = bodyText.split('\n').map(s => s.trim()).filter(Boolean);
    for (const line of specLines) {
        if (/^(Sedan|SUV|Coupe|Convertible|Hatchback|Pickup|Wagon|Van|Minivan|Truck|\d\.\dL\s|Gasoline|Diesel|Hybrid|Electric|Front wheel drive|Rear wheel drive|All wheel drive|4WD|AWD|FWD|RWD)/i.test(line) && line.length < 65) {
            if (!specs.includes(line)) specs.push(line);
        }
    }

    // 5. Owners Count
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

    // 6. Accident / Damage Status
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

    // 7. Service Records, Vehicle Use, Last Location, Detailed Records Count
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

    // 8. Extract Structured Tables (Ownership, Title, Additional History, and Detailed History Timeline)
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

            // Check if this row is a chronological Detailed History row (Date | Mileage | Source | Icon | Comments)
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

    // 9. Extract Accident / Damage Events if present
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
        analyzed_at: new Date().toISOString()
    };

    // Create a lightweight clean HTML snapshot (no heavy scripts/SVGs) for immediate storage
    const clone = document.documentElement.cloneNode(true);
    clone.querySelectorAll('script, noscript, iframe, svg').forEach(el => el.remove());

    return {
        isError: false,
        errorMsg: '',
        title: title,
        summaryJson: JSON.stringify(summaryObj),
        lightHtml: '<!DOCTYPE html>\n' + clone.outerHTML
    };
})();
`, vin)

	var out struct {
		IsError     bool   `json:"isError"`
		ErrorMsg    string `json:"errorMsg"`
		Title       string `json:"title"`
		SummaryJson string `json:"summaryJson"`
		LightHtml   string `json:"lightHtml"`
	}

	if err := chromedp.Run(tabCtx, chromedp.Evaluate(fastJS, &out)); err != nil {
		return "", "", "", false, "", err
	}

	if out.IsError {
		return "", "", "", true, out.ErrorMsg, nil
	}

	return out.SummaryJson, out.LightHtml, out.Title, false, "", nil
}

// prepareOfficialPrintModal safely clicks the top "Print" button on the Carfax report page,
// selects all 4 report sections inside the "Printing Options" modal (CARFAX Report, History-Based Value,
// Warranty Check, Buyback Guarantee), clicks the modal's "Print" button while intercepting window.print()
// so no native OS print dialog blocks Chrome, and hides the modal overlay once the full sections render.
func (sh *StealthHarvester) prepareOfficialPrintModal(
	parentCtx context.Context,
	cdpEndpoint string,
	activeReportCtx context.Context,
) (context.Context, context.CancelFunc, string) {
	// Record tabs before clicking Print in case Carfax opens a separate print tab
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

	// Wait 1.3s to 1.9s like a human for the "Printing Options" modal to appear
	sh.randomDelay(1300, 1900)

	// 3. In "Printing Options" modal: click "Select All" AND ensure every checkbox is checked
	selectAllOptionsJS := `
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
    const root = modal || document;

    // Click "Select All" if present and visible
    const links = Array.from(root.querySelectorAll('a, button, span, [role="button"]'));
    for (const l of links) {
        const txt = (l.innerText || l.textContent || '').trim().toLowerCase();
        if (txt === 'select all' && l.offsetParent !== null) {
            l.click();
            break;
        }
    }

    // Ensure all checkboxes inside the modal (CARFAX Report, History-Based Value, Warranty Check, Buyback Guarantee) are checked
    const checkboxes = Array.from(root.querySelectorAll('input[type="checkbox"]'));
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
    return checkedCount;
})();
`
	var checkedCount int
	_ = chromedp.Run(activeReportCtx, chromedp.Evaluate(selectAllOptionsJS, &checkedCount))
	sh.randomDelay(500, 850)

	// 4. Click the blue "Print" button inside the "Printing Options" modal
	clickModalPrintJS := `
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
    const root = modal || document;
    const buttons = Array.from(root.querySelectorAll('button, input[type="button"], input[type="submit"], a[ role="button" ]'));
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
	var clickedModalPrint bool
	_ = chromedp.Run(activeReportCtx, chromedp.Evaluate(clickModalPrintJS, &clickedModalPrint))

	// Wait 2.6s to 3.6s for Carfax to assemble & render all 4 selected sections (or open a print tab)
	sh.randomDelay(2600, 3600)

	// 5. Check if clicking Print in the modal opened a brand-new print tab
	tabsAfterPrint, _ := sh.ListOpenTabs(cdpEndpoint)
	for i := range tabsAfterPrint {
		t := tabsAfterPrint[i]
		if !knownBeforePrint[t.ID] && t.URL != "about:blank" {
			if pCtx, pCleanup, pErr := sh.attachExistingTab(parentCtx, cdpEndpoint, t.ID); pErr == nil {
				_ = chromedp.Run(pCtx, chromedp.Evaluate(overridePrintJS, nil))
				sh.randomDelay(1500, 2200)
				return pCtx, pCleanup, t.ID
			}
		}
	}

	// 6. Hide any remaining "Printing Options" modal dialog / backdrop so the report & PDF are 100% unobstructed
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
	sh.randomDelay(300, 500)

	return activeReportCtx, nil, ""
}

// typeVinHumanly focuses the VIN input, clears it, types the 17 characters with human rhythm, and verifies the value
func (sh *StealthHarvester) typeVinHumanly(tabCtx context.Context, selector, vin string) error {
	sh.randomDelay(250, 500)

	// Clear field and focus via JS first
	clearJS := fmt.Sprintf(`
(() => {
    const el = document.querySelector(%q);
    if (!el) return false;
    el.focus();
    el.value = '';
    el.dispatchEvent(new Event('input', { bubbles: true }));
    return true;
})();
`, selector)
	var focused bool
	if err := chromedp.Run(tabCtx, chromedp.Evaluate(clearJS, &focused)); err != nil || !focused {
		return fmt.Errorf("لم يتم العثور على العنصر %s", selector)
	}

	for i, c := range []rune(vin) {
		sh.randomDelay(75, 195)
		if err := chromedp.Run(tabCtx, chromedp.KeyEvent(string(c))); err != nil {
			return err
		}
		if (i+1)%5 == 0 {
			sh.randomDelay(250, 480)
		}
	}

	// Verify the input value matches the full 17-char VIN (in case a keystroke was dropped by React/Angular state)
	verifyJS := fmt.Sprintf(`
(() => {
    const el = document.querySelector(%q);
    if (!el) return '';
    if (el.value.trim().toUpperCase() !== %q) {
        const nativeInputValueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
        nativeInputValueSetter.call(el, %q);
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
    }
    return el.value.trim().toUpperCase();
})();
`, selector, vin, vin)

	var finalVal string
	if err := chromedp.Run(tabCtx, chromedp.Evaluate(verifyJS, &finalVal)); err != nil {
		return err
	}
	if finalVal != vin {
		return fmt.Errorf("القيمة المكتوبة (%s) لا تطابق الشاصي المطلوب (%s)", finalVal, vin)
	}

	sh.randomDelay(400, 750)
	return nil
}

func (sh *StealthHarvester) simulateGentleScroll(tabCtx context.Context) {
	steps := 2 + sh.rnd.Intn(2) // 2 to 3 gentle scrolls
	for s := 0; s < steps; s++ {
		scrollBy := 280 + sh.rnd.Intn(250)
		_ = chromedp.Run(tabCtx, chromedp.Evaluate(fmt.Sprintf(`window.scrollBy({ top: %d, behavior: 'smooth' });`, scrollBy), nil))
		sh.randomDelay(1100, 1900)
	}
	_ = chromedp.Run(tabCtx, chromedp.Evaluate(`window.scrollTo({ top: 0, behavior: 'smooth' });`, nil))
	sh.randomDelay(500, 900)
}

// extractClonedReportHTML clones the document in memory without modifying the live page,
// inlines all loaded CSS stylesheets and images from Chrome's memory cache (bypassing geo-blocks),
// and embeds the official native Chrome PrintToPDF output.
func (sh *StealthHarvester) extractClonedReportHTML(tabCtx context.Context, vin string) (string, string, bool, string, error) {
	extractJS := `
(() => {
    const bodyText = (document.body ? document.body.innerText : '');
    const lowerText = bodyText.toLowerCase();

    // Check if Carfax returned an invalid VIN or no record message on the search page
    if (lowerText.includes('invalid vin') || lowerText.includes('not a valid vin') || lowerText.includes('no records found for this vin')) {
        return {
            html: '',
            title: '',
            isError: true,
            errorMsg: 'أفاد موقع كارفاكس أن رقم الشاصي هذا غير صحيح أو لا توجد له سجلات في قاعدة بيانات كارفاكس.'
        };
    }

    // Clone the entire document in memory so the live page is NEVER touched
    const clone = document.documentElement.cloneNode(true);

    // 1. Remove scripts, noscript, iframes, and <picture><source> tags from the clone
    clone.querySelectorAll('script, noscript, iframe, picture source').forEach(el => el.remove());

    // 2. Sync every cloned <img> src to the exact currentSrc loaded by Chrome in the live DOM
    const liveImgs = Array.from(document.querySelectorAll('img'));
    const cloneImgs = Array.from(clone.querySelectorAll('img'));
    const origin = window.location.origin;
    for (let i = 0; i < cloneImgs.length; i++) {
        const cImg = cloneImgs[i];
        const lImg = liveImgs[i];
        cImg.removeAttribute('srcset');
        if (lImg && lImg.currentSrc) {
            cImg.setAttribute('src', lImg.currentSrc);
        } else {
            const src = cImg.getAttribute('src');
            if (src && src.startsWith('/')) {
                cImg.setAttribute('src', origin + src);
            }
        }
    }

    // 3. Also convert any relative stylesheet links to absolute URLs
    clone.querySelectorAll('link[rel="stylesheet"]').forEach(link => {
        const href = link.getAttribute('href');
        if (href && href.startsWith('/')) {
            link.setAttribute('href', origin + href);
        }
    });

    // 4. Extract any accessible runtime CSSOM rules from document.styleSheets
    try {
        let cssomText = '';
        for (const sheet of Array.from(document.styleSheets)) {
            try {
                if (sheet.ownerNode && sheet.ownerNode.tagName === 'STYLE' && sheet.ownerNode.textContent.trim().length > 0) {
                    continue;
                }
                const rules = Array.from(sheet.cssRules || []);
                if (rules.length > 0) {
                    cssomText += rules.map(r => r.cssText).join('\n') + '\n';
                }
            } catch (e) {}
        }
        if (cssomText.length > 0) {
            const st = document.createElement('style');
            st.setAttribute('id', 'carfax-runtime-cssom');
            st.textContent = cssomText;
            const head = clone.querySelector('head');
            if (head) head.appendChild(st);
        }
    } catch (e) {}

    // 5. Extract vehicle title (Year Make Model)
    let title = document.title || '';
    const titleSelectors = [
        '.vehicle-information-year-make-model',
        '.header-vehicle-name',
        '[data-testid*="vehicle" i]',
        'h1',
        'h2.vehicle-title',
        '.vehicle-header h1'
    ];
    for (const sel of titleSelectors) {
        const el = document.querySelector(sel);
        if (el && el.innerText && el.innerText.trim().length > 3) {
            title = el.innerText.trim().replace(/\s+/g, ' ');
            break;
        }
    }

    return {
        html: '<!DOCTYPE html>\n' + clone.outerHTML,
        title: title,
        isError: false,
        errorMsg: ''
    };
})();
`
	var out struct {
		HTML     string `json:"html"`
		Title    string `json:"title"`
		IsError  bool   `json:"isError"`
		ErrorMsg string `json:"errorMsg"`
	}

	if err := chromedp.Run(tabCtx, chromedp.Evaluate(extractJS, &out)); err != nil {
		return "", "", false, "", err
	}

	if out.IsError {
		return "", "", true, out.ErrorMsg, nil
	}

	if len(out.HTML) < 400 {
		return "", "", true, "محتوى الصفحة المستخرج فارغ أو لم يكتمل تحميل التقرير في حاسبة كندا.", nil
	}

	if out.Title == "" {
		out.Title = fmt.Sprintf("CARFAX Vehicle History Report (%s)", vin)
	}

	// Inline all CSS stylesheets and images from Chrome's C++ cache + attach official PrintToPDF
	out.HTML = sh.bundleSelfContainedReportAndPDF(tabCtx, out.HTML, vin)

	return out.HTML, out.Title, false, "", nil
}

// bundleSelfContainedReportAndPDF reads all cached CSS stylesheets and images directly from
// Chrome's C++ memory cache via CDP (Page.getResourceTree / Page.getResourceContent) with zero
// extra network requests, inlines them into the HTML, saves the official multi-page PDF locally
// on the Canada PC (C:\CarfaxReports\CARFAX_<VIN>.pdf), and appends it to the HTML payload.
func (sh *StealthHarvester) bundleSelfContainedReportAndPDF(tabCtx context.Context, rawHTML string, vin string) string {
	var inlinedCSSBuilder strings.Builder
	imageDataURIs := make(map[string]string)
	var pdfBase64 string

	_ = chromedp.Run(tabCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		_ = page.Enable().Do(ctx)

		tree, err := page.GetResourceTree().Do(ctx)
		if err == nil && tree != nil && tree.Frame != nil {
			frameID := tree.Frame.ID
			for _, res := range tree.Resources {
				if res == nil || res.Failed || res.Canceled || res.URL == "" {
					continue
				}
				lowerURL := strings.ToLower(res.URL)
				lowerMime := strings.ToLower(res.MimeType)

				// 1. Inline CSS Stylesheets (e.g. vhr.css)
				if string(res.Type) == "Stylesheet" || strings.Contains(lowerMime, "css") || strings.Contains(lowerURL, ".css") {
					contentBytes, cErr := page.GetResourceContent(frameID, res.URL).Do(ctx)
					if cErr == nil && len(contentBytes) > 0 {
						inlinedCSSBuilder.WriteString("\n/* Inlined from " + res.URL + " */\n")
						inlinedCSSBuilder.Write(contentBytes)
						inlinedCSSBuilder.WriteString("\n")
					}
					continue
				}

				// 2. Inline Images (SVG, PNG, GIF, WebP, AVIF)
				if string(res.Type) == "Image" || strings.HasPrefix(lowerMime, "image/") ||
					strings.HasSuffix(lowerURL, ".svg") || strings.HasSuffix(lowerURL, ".png") ||
					strings.HasSuffix(lowerURL, ".gif") || strings.HasSuffix(lowerURL, ".webp") {
					contentBytes, cErr := page.GetResourceContent(frameID, res.URL).Do(ctx)
					if cErr == nil && len(contentBytes) > 0 {
						mime := res.MimeType
						if mime == "" {
							switch {
							case strings.Contains(lowerURL, ".svg"):
								mime = "image/svg+xml"
							case strings.Contains(lowerURL, ".gif"):
								mime = "image/gif"
							case strings.Contains(lowerURL, ".webp"):
								mime = "image/webp"
							case strings.Contains(lowerURL, ".avif"):
								mime = "image/avif"
							default:
								mime = "image/png"
							}
						}
						b64Data := base64.StdEncoding.EncodeToString(contentBytes)
						imageDataURIs[res.URL] = "data:" + mime + ";base64," + b64Data
					}
				}
			}
		}

		// 3. Generate Official Native Chrome PrintToPDF & save locally to C:\CarfaxReports\CARFAX_<VIN>.pdf
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
					fmt.Printf("[💾] Saved Official PDF (%d KB) to %s\n", len(pdfBuf)/1024, pdfPath)
				}
			}
		}

		return nil
	}))

	// Fallback: if any <link rel="stylesheet"> was not in GetResourceContent, fetch once on Canada PC and cache in RAM
	linkRe := regexp.MustCompile(`(?i)<link[^>]+rel=["']stylesheet["'][^>]*href=["'](https?://[^"']+)["'][^>]*>|<link[^>]+href=["'](https?://[^"']+)["'][^>]*rel=["']stylesheet["'][^>]*>`)
	linkMatches := linkRe.FindAllStringSubmatch(rawHTML, -1)
	for _, m := range linkMatches {
		cssURL := m[1]
		if cssURL == "" {
			cssURL = m[2]
		}
		cssURL = strings.ReplaceAll(cssURL, "&amp;", "&")
		if cssURL == "" || strings.Contains(cssURL, "fonts.googleapis.com") {
			continue
		}
		if !strings.Contains(inlinedCSSBuilder.String(), cssURL) {
			if bodyBytes, _, err := sh.fetchAssetCached(cssURL); err == nil && len(bodyBytes) > 0 {
				inlinedCSSBuilder.WriteString("\n/* Inlined fallback from " + cssURL + " */\n")
				inlinedCSSBuilder.Write(bodyBytes)
				inlinedCSSBuilder.WriteString("\n")
			}
		}
	}

	// Inject all captured CSS directly into <head> and remove external carfax <link> tags so browsers outside Canada don't hang waiting on blocked CDNs
	if inlinedCSSBuilder.Len() > 0 {
		carfaxLinkRe := regexp.MustCompile(`(?i)<link[^>]+href=["']https?://[^"']*carfax[^"']*["'][^>]*>`)
		rawHTML = carfaxLinkRe.ReplaceAllString(rawHTML, "")
		styleBlock := `<style id="carfax-cdp-inlined-css">` + inlinedCSSBuilder.String() + `</style>`
		if strings.Contains(rawHTML, "</head>") {
			rawHTML = strings.Replace(rawHTML, "</head>", styleBlock+"</head>", 1)
		} else {
			rawHTML = styleBlock + rawHTML
		}
	}

	// Replace image URLs (both raw and HTML-entity encoded &amp;) with Base64 data URIs
	for origURL, dataURI := range imageDataURIs {
		rawHTML = strings.ReplaceAll(rawHTML, `"`+origURL+`"`, `"`+dataURI+`"`)
		rawHTML = strings.ReplaceAll(rawHTML, `'`+origURL+`'`, `'`+dataURI+`'`)
		escapedURL := strings.ReplaceAll(origURL, "&", "&amp;")
		if escapedURL != origURL {
			rawHTML = strings.ReplaceAll(rawHTML, `"`+escapedURL+`"`, `"`+dataURI+`"`)
			rawHTML = strings.ReplaceAll(rawHTML, `'`+escapedURL+`'`, `'`+dataURI+`'`)
		}
	}

	// Fallback for any remaining <img src="https://..."> not in GetResourceContent
	imgRe := regexp.MustCompile(`(?i)<img[^>]+src=["'](https?://[^"']+)["']`)
	imgMatches := imgRe.FindAllStringSubmatch(rawHTML, -1)
	seenImgs := make(map[string]bool)
	for _, m := range imgMatches {
		rawImgURL := m[1]
		cleanImgURL := strings.ReplaceAll(rawImgURL, "&amp;", "&")
		if seenImgs[cleanImgURL] {
			continue
		}
		seenImgs[cleanImgURL] = true
		if bodyBytes, mime, err := sh.fetchAssetCached(cleanImgURL); err == nil && len(bodyBytes) > 0 {
			dataURI := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(bodyBytes)
			rawHTML = strings.ReplaceAll(rawHTML, `"`+rawImgURL+`"`, `"`+dataURI+`"`)
			rawHTML = strings.ReplaceAll(rawHTML, `'`+rawImgURL+`'`, `'`+dataURI+`'`)
			rawHTML = strings.ReplaceAll(rawHTML, `"`+cleanImgURL+`"`, `"`+dataURI+`"`)
		}
	}

	// Embed the official native PDF at the end of </body> so the platform can download/print it 1:1
	if pdfBase64 != "" {
		pdfTag := `<script id="carfix-official-pdf" type="application/pdf-base64">` + pdfBase64 + `</script>`
		if strings.Contains(rawHTML, "</body>") {
			rawHTML = strings.Replace(rawHTML, "</body>", pdfTag+"</body>", 1)
		} else {
			rawHTML = rawHTML + pdfTag
		}
	}

	return rawHTML
}

type cachedAsset struct {
	data []byte
	mime string
}

func (sh *StealthHarvester) fetchAssetCached(assetURL string) ([]byte, string, error) {
	if val, ok := sh.assetCache.Load(assetURL); ok {
		ca := val.(cachedAsset)
		return ca.data, ca.mime, nil
	}
	req, err := http.NewRequest("GET", assetURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.carfaxonline.com/")
	resp, err := sh.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil || len(data) == 0 {
		return nil, "", fmt.Errorf("empty body")
	}
	mime := resp.Header.Get("Content-Type")
	if idx := strings.Index(mime, ";"); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}
	lowerURL := strings.ToLower(assetURL)
	if mime == "" || mime == "application/octet-stream" {
		switch {
		case strings.Contains(lowerURL, ".svg"):
			mime = "image/svg+xml"
		case strings.Contains(lowerURL, ".gif"):
			mime = "image/gif"
		case strings.Contains(lowerURL, ".webp"):
			mime = "image/webp"
		case strings.Contains(lowerURL, ".avif"):
			mime = "image/avif"
		case strings.Contains(lowerURL, ".css"):
			mime = "text/css"
		default:
			mime = "image/png"
		}
	}
	sh.assetCache.Store(assetURL, cachedAsset{data: data, mime: mime})
	return data, mime, nil
}

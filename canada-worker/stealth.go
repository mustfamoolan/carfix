package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
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

	// 1. Prefer a tab on carfaxonline.com that is NOT an already-opened report popup
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
func (sh *StealthHarvester) HarvestReport(
	parentCtx context.Context,
	cdpEndpoint string,
	vin string,
	onProgress func(step, stepAr string, screenshot string),
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

	// Record existing open tabs before clicking search so we can detect if a new tab pops up
	tabsBefore, _ := sh.ListOpenTabs(cdpEndpoint)
	knownTabIDs := make(map[string]bool)
	for _, t := range tabsBefore {
		knownTabIDs[t.ID] = true
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

	// Wait 3.5 to 5 seconds for either a new report tab to open OR current tab to load report
	sh.randomDelay(3500, 5000)

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

	// If no new tab opened, the report is in the current searchTab
	activeReportCtx := tabCtx
	if openedNewTab && reportTab != nil {
		if rTabCtx, cleanupReportTab, rErr := sh.attachExistingTab(parentCtx, cdpEndpoint, reportTab.ID); rErr == nil {
			defer cleanupReportTab()
			activeReportCtx = rTabCtx
			// Give the new tab 2.5 more seconds to finish rendering styles & records
			sh.randomDelay(2200, 3400)
		}
	}

	// 4. Natural Human Reading & Extraction
	if onProgress != nil {
		onProgress("extracting_report", "4/4 تم فتح التقرير، جاري القراءة البشرية وسحب المحتوى...", "")
	}

	// Gentle scroll like a human dealer reading the report
	sh.simulateGentleScroll(activeReportCtx)

	// Capture screenshot of the opened report
	reportScreenshot := sh.captureCompressedJPEG(activeReportCtx)

	// Non-destructive clone & extraction (never alters the live DOM!)
	html, title, isErrorPage, errPageMsg, err := sh.extractClonedReportHTML(activeReportCtx, vin)

	// Clean up: if a popup tab was opened for the report, close ONLY the popup tab so the main search tab stays clean
	if openedNewTab && reportTab != nil {
		sh.CloseTabByID(cdpEndpoint, reportTab.ID)
	} else {
		// If it navigated in the same tab, go back to the search screen if URL changed away from the main search form
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
		ScreenshotBase64: reportScreenshot,
	}
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
	out.HTML = sh.bundleSelfContainedReportAndPDF(tabCtx, out.HTML)

	return out.HTML, out.Title, false, "", nil
}

// bundleSelfContainedReportAndPDF reads all cached CSS stylesheets and images directly from
// Chrome's C++ memory cache via CDP (Page.getResourceTree / Page.getResourceContent) with zero
// extra network requests, inlines them into the HTML, and appends Chrome's native PrintToPDF.
func (sh *StealthHarvester) bundleSelfContainedReportAndPDF(tabCtx context.Context, rawHTML string) string {
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

		// 3. Generate Official Native Chrome PrintToPDF
		pdfBuf, _, pdfErr := page.PrintToPDF().
			WithPrintBackground(true).
			WithPreferCSSPageSize(true).
			Do(ctx)
		if pdfErr == nil && len(pdfBuf) > 0 {
			pdfBase64 = base64.StdEncoding.EncodeToString(pdfBuf)
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

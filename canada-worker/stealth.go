package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
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

// FindCarfaxSearchTab finds the open carfaxonline.com tab, or falls back to the first visible page tab
func (sh *StealthHarvester) FindCarfaxSearchTab(cdpEndpoint string) (*CDPTab, bool, error) {
	tabs, err := sh.ListOpenTabs(cdpEndpoint)
	if err != nil {
		return nil, false, err
	}
	if len(tabs) == 0 {
		return nil, false, nil
	}

	// 1. Prefer a tab on carfaxonline.com that is NOT an already-opened report popup
	for _, t := range tabs {
		u := strings.ToLower(t.URL)
		if strings.Contains(u, "carfaxonline.com") || strings.Contains(u, "carfax.ca") || strings.Contains(u, "carfax.com") {
			return &t, true, nil
		}
	}

	// 2. If no carfax tab is open, return the first tab so we can still screenshot what Chrome is showing
	return &tabs[0], false, nil
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

	// Connect directly to the tab's WebSocket without closing the tab on disconnect
	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(parentCtx, tab.WebSocketDebuggerURL, chromedp.NoModifyURL)
	defer cancelAlloc()

	evalCtx, cancelEval := context.WithTimeout(allocCtx, 12*time.Second)
	defer cancelEval()

	// Note: We do NOT call cancelTab() before cancelAlloc() so chromedp never sends Target.closeTarget
	tabCtx, _ := chromedp.NewContext(evalCtx)

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
        'input[maxlength="17"]',
        'input[type="text"]'
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
        'a[id*="run" i]'
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
		if domState.URL != "" {
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

	// Connect to the search tab
	searchTab, _, err := sh.FindCarfaxSearchTab(cdpEndpoint)
	if err != nil || searchTab == nil {
		return &HarvestResult{
			Success:          false,
			ErrorCode:        "TAB_LOST",
			ErrorMessage:     "فقد الاتصال بتبويب البحث قبل البدء مباشرة.",
			ScreenshotBase64: state.ScreenshotBase64,
		}
	}

	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(parentCtx, searchTab.WebSocketDebuggerURL, chromedp.NoModifyURL)
	defer cancelAlloc()

	taskCtx, cancelTask := context.WithTimeout(allocCtx, 55*time.Second)
	defer cancelTask()

	tabCtx, _ := chromedp.NewContext(taskCtx)

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
		if !knownTabIDs[t.ID] {
			reportTab = &t
			openedNewTab = true
			break
		}
	}

	// If no new tab opened, the report is in the current searchTab
	activeReportCtx := tabCtx
	var cancelReportAlloc context.CancelFunc
	if openedNewTab && reportTab != nil {
		rAllocCtx, rCancel := chromedp.NewRemoteAllocator(parentCtx, reportTab.WebSocketDebuggerURL, chromedp.NoModifyURL)
		cancelReportAlloc = rCancel
		defer cancelReportAlloc()

		rTaskCtx, rTaskCancel := context.WithTimeout(rAllocCtx, 35*time.Second)
		defer rTaskCancel()

		rTabCtx, _ := chromedp.NewContext(rTaskCtx)
		activeReportCtx = rTabCtx
		// Give the new tab 2.5 more seconds to finish rendering styles & records
		sh.randomDelay(2200, 3400)
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
// resolves relative links/images to absolute URLs, and extracts the full HTML.
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

    // 1. Remove scripts and noscript tags from the clone
    clone.querySelectorAll('script, noscript, iframe').forEach(el => el.remove());

    // 2. Convert all relative img src and link href attributes to absolute URLs
    const origin = window.location.origin;
    clone.querySelectorAll('img').forEach(img => {
        const src = img.getAttribute('src');
        if (src && src.startsWith('/')) {
            img.setAttribute('src', origin + src);
        }
    });
    clone.querySelectorAll('link[rel="stylesheet"]').forEach(link => {
        const href = link.getAttribute('href');
        if (href && href.startsWith('/')) {
            link.setAttribute('href', origin + href);
        }
    });

    // 3. Extract vehicle title (Year Make Model)
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

	return out.HTML, out.Title, false, "", nil
}

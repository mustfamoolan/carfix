package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CarfaxTranslatorService struct {
	dictionary []translationPair
	milesRegex *regexp.Regexp
}

type translationPair struct {
	en string
	ar string
}

func NewCarfaxTranslatorService() *CarfaxTranslatorService {
	s := &CarfaxTranslatorService{
		milesRegex: regexp.MustCompile(`\b(\d[\d,]*)\s+mi\b`),
	}
	s.initDictionary()
	return s
}

func (s *CarfaxTranslatorService) initDictionary() {
	dict := map[string]string{
		// Official CARFAXOnline VHR Headers & Badges
		"CARFAX Vehicle History Report":                        "تقرير كارفاكس لتاريخ المركبة",
		"Vehicle History Report™":                              "تقرير تاريخ المركبة™",
		"Vehicle History Report":                               "تقرير تاريخ المركبة",
		"This report provided free of charge by:":              "تم توفير هذا التقرير مجاناً بواسطة:",
		"This report provided free of charge by":               "تم توفير هذا التقرير مجاناً بواسطة",
		"CARFAX Advantage Dealer":                              "وكيل كارفاكس المعتمد (Advantage Dealer)",
		"Display CARFAX in online listings:":                   "عرض التقرير في الإعلانات الإلكترونية:",
		"Original Window Sticker":                              "ملصق الوكالة الأصلي (Window Sticker)",
		"Vehicle Details":                                      "تفاصيل ومواصفات المركبة",
		"CARFAX Buyback Guarantee":                             "ضمان إعادة الشراء من كارفاكس",
		"Buyback Guarantee":                                    "ضمان إعادة الشراء",
		"CARFAX Homegrown":                                     "صيانة الوكالة المعتمدة (CARFAX Homegrown)",

		// Summary Snapshot Box & Severity Scale
		"ACCIDENT":                                             "حادث مسجل",
		"NO ACCIDENTS OR DAMAGE REPORTED TO CARFAX":            "لم يتم الإبلاغ عن أي حوادث أو أضرار لكارفاكس",
		"No accidents or damage reported to CARFAX":            "لم يتم الإبلاغ عن أي حوادث أو أضرار لكارفاكس",
		"Severe Damage":                                        "أضرار شديدة",
		"Moderate Damage":                                      "أضرار متوسطة",
		"Minor Damage":                                         "أضرار طفيفة",
		"Severe":                                               "شديدة",
		"Moderate":                                             "متوسطة",
		"Minor":                                                "طفيفة",
		"1 Service History Record":                             "سجل صيانة واحد مسجل",
		"Service History Records":                              "سجلات صيانة مسجلة",
		"Service History Record":                               "سجل صيانة مسجل",
		"CARFAX 1-Owner Vehicle":                               "مركبة مالك أول (1-Owner)",
		"1-Owner Vehicle":                                      "مركبة مالك أول",
		"Personal Vehicle":                                     "مركبة استخدام شخصي",
		"Personal Lease":                                       "مركبة تأجير شخصي",
		"Commercial Vehicle":                                   "مركبة استخدام تجاري",
		"Rental Vehicle":                                       "مركبة تأجير",
		"Last Owned in":                                        "آخر ملكية مسجلة في",

		// Section Headers
		"Accident / Damage History":                            "سجل الحوادث والأضرار",
		"Not all accidents / issues are reported to CARFAX":    "لا يتم الإبلاغ عن جميع الحوادث أو المشاكل إلى كارفاكس",
		"Additional History":                                   "السجل الفني الإضافي",
		"Title History":                                        "سجل وثيقة الملكية (Title)",
		"CARFAX guarantees the information in this section":    "تضمن كارفاكس صحة البيانات الواردة في هذا القسم",
		"Ownership History":                                    "سجل تاريخ الملكية",
		"The number of owners is estimated":                    "العدد التقديري للمالكين",
		"Detailed History":                                     "السجل التاريخي الزمني المفصل",
		"Glossary":                                             "دليل المصطلحات",

		// Ownership History Rows
		"Year purchased":                                       "سنة الشراء",
		"Type of owner":                                        "نوع المالك والاستخدام",
		"Estimated length of ownership":                        "المدة التقديرية للملكية",
		"Owned in the following states/provinces":              "الولايات أو المقاطعات المسجل بها",
		"Estimated miles driven per year":                      "المسافة التقديرية المقطوعة سنوياً",
		"Last reported odometer reading":                       "آخر قراءة مسجلة لعداد المسافات",
		"Personal":                                             "شخصي",
		"Owner 1":                                              "المالك 1",
		"Owner 2":                                              "المالك 2",
		"Owner 3":                                              "المالك 3",
		"Owner 4":                                              "المالك 4",

		// Title History & Additional History Rows
		"Damage Brands":                                        "علامات الأضرار في الوثيقة",
		"Salvage | Junk | Rebuilt | Fire | Flood | Hail | Lemon": "سالفج | سكراب | معاد تصليحها | حريق | غرق | بَرَد | عيوب مصنعية",
		"Odometer Brands":                                      "علامات عداد المسافات",
		"Not Actual Mileage | Exceeds Mechanical Limits":       "المسافة غير حقيقية | تجاوز الحدود الميكانيكية",
		"Guaranteed No Problem":                                "مضمون - لا توجد مشكلة",
		"Problem Found":                                        "تم رصد مشكلة",
		"No Problem":                                           "لا توجد مشكلة",
		"Total Loss":                                           "خسارة كلية (شطب)",
		"No Total Loss Reported to CARFAX":                     "لم يتم الإبلاغ عن خسارة كلية لكارفاكس",
		"Structural Damage":                                    "أضرار الهيكل الإنشائي (الشاصي)",
		"No Structural Damage Reported to CARFAX":              "لم يتم الإبلاغ عن أضرار هيكلية لكارفاكس",
		"Airbag Deployment":                                    "انفتاح الوسائد الهوائية (الإيرباج)",
		"No Airbag Deployment Reported to CARFAX":              "لم يتم الإبلاغ عن انفتاح الإيرباج لكارفاكس",
		"Odometer Check":                                       "فحص سلامة العداد",
		"No Indication of an Odometer Rollback":                "لا توجد مؤشرات على التلاعب بالعداد",
		"Accident / Damage":                                    "الحوادث والأضرار",
		"Manufacturer Recall":                                  "استدعاءات المصنع",
		"No Open Recalls Reported to CARFAX":                   "لا توجد استدعاءات مفتوحة مسجلة",
		"Basic Warranty":                                       "الضمان الأساسي للمصنع",
		"Warranty Active":                                      "الضمان ساري المفعول",
		"Warranty Expired":                                     "انتهى الضمان الأساسي",
		"No Issues Reported":                                   "سليم - لم تسجل أي مشاكل",
		"No New Issues Reported":                               "لم تسجل أي مشاكل جديدة",

		// Detailed History Table Columns & Common Entries
		"Date":                                                 "التاريخ",
		"Mileage":                                              "المسافة (ميل)",
		"Source":                                               "المصدر / الجهة",
		"Comments":                                             "التفاصيل والملاحظات",
		"Motor Vehicle Dept.":                                  "دائرة ترخيص المركبات (DMV)",
		"Vehicle serviced":                                     "تمت صيانة المركبة",
		"Pre-delivery inspection completed":                    "إتمام فحص ما قبل التسليم",
		"Oil and filter changed":                               "تغيير زيت المحرك والفلتر",
		"Tires rotated":                                        "تدوير الإطارات",
		"Vehicle offered for sale":                             "عرض المركبة للبيع",
		"Vehicle sold":                                         "تم بيع المركبة",
		"Title issued or updated":                              "إصدار أو تحديث وثيقة الملكية (Title)",
		"Registration issued or renewed":                       "إصدار أو تجديد رخصة السير",
		"First owner reported":                                 "تسجيل المالك الأول للمركبة",
		"New owner reported":                                   "تسجيل مالك جديد للمركبة",
		"Loan or lien reported":                                "مسجل عليها تمويل أو رهن بنكي",
		"Accident reported":                                    "تم الإبلاغ عن حادث",
		"Damage reported":                                      "تم الإبلاغ عن أضرار",
		"Airbag deployed":                                      "انفتاح الوسائد الهوائية (الإيرباج)",
		"Vehicle towed":                                        "تم سحب المركبة (ونش)",

		// Common Specifications
		"Sedan 4 DR":                                           "سيدان 4 أبواب",
		"SEDAN 4 DR":                                           "سيدان 4 أبواب",
		"Coupe 2 DR":                                           "كوبيه بابين",
		"SUV 4 DR":                                             "دفع رباعي عائلي (SUV)",
		"Gasoline":                                             "بنزين",
		"GASOLINE":                                             "بنزين",
		"Front wheel drive":                                    "دفع أمامي (FWD)",
		"FRONT WHEEL DRIVE":                                    "دفع أمامي (FWD)",
		"Rear wheel drive":                                     "دفع خلفي (RWD)",
		"All wheel drive":                                      "دفع كلي (AWD)",
	}

	pairs := make([]translationPair, 0, len(dict))
	for en, ar := range dict {
		pairs = append(pairs, translationPair{en: en, ar: ar})
	}

	// Sort descending by length so longer multi-word phrases match before individual words
	sort.Slice(pairs, func(i, j int) bool {
		return len(pairs[i].en) > len(pairs[j].en)
	})

	s.dictionary = pairs
}

// TranslateAndExtract translates only visible text nodes in the Carfax HTML body (never touching
// <head>, <style>, <script>, or HTML tag attributes) and extracts real vehicle summary metadata.
func (s *CarfaxTranslatorService) TranslateAndExtract(rawHtml, vin string) (string, string, string) {
	translated := s.translateVisibleTextNodesOnly(rawHtml)

	// Inject gentle Arabic font support without breaking Carfax's native vhr.css flex/grid layout
	rtlStyles := `
<style id="carfix-arabic-font-support">
  @import url('https://fonts.googleapis.com/css2?family=Cairo:wght@400;600;700&display=swap');
  body { font-family: 'Cairo', 'Roboto', 'Helvetica Neue', Arial, sans-serif; }
</style>
`
	if strings.Contains(translated, "</head>") {
		translated = strings.Replace(translated, "</head>", rtlStyles+"</head>", 1)
	}

	title := s.extractTitle(rawHtml, vin)
	summaryJson := s.extractSummaryJson(rawHtml, vin)

	return translated, title, summaryJson
}

// translateVisibleTextNodesOnly walks the HTML body and translates only text outside of <...> tags
// and outside of <style>, <script>, and <svg> blocks so CSS, URLs, and Base64 data are 100% preserved.
func (s *CarfaxTranslatorService) translateVisibleTextNodesOnly(rawHtml string) string {
	lower := strings.ToLower(rawHtml)
	bodyStart := strings.Index(lower, "<body")
	if bodyStart == -1 {
		bodyStart = 0
	}

	// Do not duplicate <script id="carfix-official-pdf"...> into translated_html_ar (saves ~2 MB per report!)
	pdfScriptIdx := strings.Index(rawHtml, `<script id="carfix-official-pdf"`)
	endLimit := len(rawHtml)
	suffix := ""
	if pdfScriptIdx != -1 && pdfScriptIdx > bodyStart {
		endLimit = pdfScriptIdx
		if closeScriptIdx := strings.Index(rawHtml[pdfScriptIdx:], "</script>"); closeScriptIdx != -1 {
			suffix = rawHtml[pdfScriptIdx+closeScriptIdx+len("</script>"):]
		}
	}

	prefix := rawHtml[:bodyStart]
	bodySegment := rawHtml[bodyStart:endLimit]

	var out strings.Builder
	out.Grow(len(rawHtml) + 2048)
	out.WriteString(prefix)

	i := 0
	n := len(bodySegment)
	for i < n {
		if bodySegment[i] == '<' {
			// Find end of tag
			tagEnd := strings.IndexByte(bodySegment[i:], '>')
			if tagEnd == -1 {
				out.WriteString(bodySegment[i:])
				break
			}
			tagStr := bodySegment[i : i+tagEnd+1]
			out.WriteString(tagStr)
			i += tagEnd + 1

			// If this is an opening <style, <script, or <svg tag, copy everything until its closing tag untouched
			lowerTag := strings.ToLower(tagStr)
			var closeToken string
			switch {
			case strings.HasPrefix(lowerTag, "<style") && !strings.HasSuffix(lowerTag, "/>"):
				closeToken = "</style>"
			case strings.HasPrefix(lowerTag, "<script") && !strings.HasSuffix(lowerTag, "/>"):
				closeToken = "</script>"
			case strings.HasPrefix(lowerTag, "<svg") && !strings.HasSuffix(lowerTag, "/>"):
				closeToken = "</svg>"
			}
			if closeToken != "" {
				closeIdx := strings.Index(strings.ToLower(bodySegment[i:]), closeToken)
				if closeIdx == -1 {
					out.WriteString(bodySegment[i:])
					i = n
				} else {
					out.WriteString(bodySegment[i : i+closeIdx+len(closeToken)])
					i += closeIdx + len(closeToken)
				}
			}
			continue
		}

		// We are in a visible text node up to the next '<'
		nextTag := strings.IndexByte(bodySegment[i:], '<')
		var textChunk string
		if nextTag == -1 {
			textChunk = bodySegment[i:]
			i = n
		} else {
			textChunk = bodySegment[i : i+nextTag]
			i += nextTag
		}

		if strings.TrimSpace(textChunk) != "" {
			for _, pair := range s.dictionary {
				if strings.Contains(textChunk, pair.en) {
					textChunk = strings.ReplaceAll(textChunk, pair.en, pair.ar)
				}
			}
			textChunk = s.milesRegex.ReplaceAllString(textChunk, "$1 ميل")
		}
		out.WriteString(textChunk)
	}

	out.WriteString(suffix)
	return out.String()
}

func (s *CarfaxTranslatorService) extractTitle(html, vin string) string {
	// 1. Check Carfax VHR year-make-model class
	ymmRe := regexp.MustCompile(`(?i)class=["'][^"']*vehicle-information-year-make-model[^"']*["'][^>]*>([^<]+)<`)
	if m := ymmRe.FindStringSubmatch(html); len(m) > 1 {
		if t := strings.TrimSpace(m[1]); len(t) > 2 {
			return t
		}
	}

	// 2. Check <title>CARFAX Vehicle History Report for this 2025 KIA K5 GT-LINE: KNAG...
	titleTagRe := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	if m := titleTagRe.FindStringSubmatch(html); len(m) > 1 {
		t := strings.TrimSpace(m[1])
		t = strings.TrimPrefix(t, "CARFAX Vehicle History Report for this ")
		if colonIdx := strings.Index(t, ":"); colonIdx != -1 {
			t = strings.TrimSpace(t[:colonIdx])
		}
		if len(t) > 2 {
			return t
		}
	}

	// 3. Fallback to first <h1>
	h1Re := regexp.MustCompile(`(?i)<h1[^>]*>([^<]+)</h1>`)
	if m := h1Re.FindStringSubmatch(html); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}

	return fmt.Sprintf("CARFAX Report (%s)", vin)
}

func (s *CarfaxTranslatorService) ExtractSummaryJson(html, vin string) string {
	return s.extractSummaryJson(html, vin)
}

var (
	tagStripRe   = regexp.MustCompile(`(?s)<[^>]*>`)
	styleBlockRe = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	scriptBlkRe  = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	svgBlkRe     = regexp.MustCompile(`(?is)<svg[^>]*>.*?</svg>`)
	tableBlkRe   = regexp.MustCompile(`(?is)<table[^>]*>(.*?)</table>`)
	trBlkRe      = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	thCellRe     = regexp.MustCompile(`(?is)<th[^>]*>(.*?)</th>`)
	tdCellRe     = regexp.MustCompile(`(?is)<td[^>]*>(.*?)</td>`)
	wsCollapseRe = regexp.MustCompile(`\s+`)
	dateRowRe    = regexp.MustCompile(`^(\d{1,2}/\d{1,2}/\d{2,4}|Not Reported)$`)
)

func cleanHtmlText(fragment string) string {
	t := tagStripRe.ReplaceAllString(fragment, " ")
	t = strings.ReplaceAll(t, "&nbsp;", " ")
	t = strings.ReplaceAll(t, "&amp;", "&")
	t = strings.ReplaceAll(t, "&#39;", "'")
	t = strings.ReplaceAll(t, "&quot;", `"`)
	return strings.TrimSpace(wsCollapseRe.ReplaceAllString(t, " "))
}

func (s *CarfaxTranslatorService) extractSummaryJson(html, vin string) string {
	hasOfficialPdf := strings.Contains(html, `id="carfix-official-pdf"`)

	// Strip heavy <style>, <script>, and <svg> blocks before regex extraction so parsing takes <2ms
	lightHtml := scriptBlkRe.ReplaceAllString(html, "")
	lightHtml = styleBlockRe.ReplaceAllString(lightHtml, "")
	lightHtml = svgBlkRe.ReplaceAllString(lightHtml, "")
	lower := strings.ToLower(lightHtml)
	plainText := cleanHtmlText(lightHtml)

	title := s.extractTitle(lightHtml, vin)

	// 1. Extract real odometer reading (e.g. <span><strong>1,483</strong> mi</span>)
	odometer := "غير مسجل"
	odoRe := regexp.MustCompile(`(?i)<strong>([\d,]+)</strong>\s*(?:mi|km|miles)`)
	if m := odoRe.FindStringSubmatch(lightHtml); len(m) > 1 {
		odometer = m[1] + " mi"
	} else {
		odoFallbackRe := regexp.MustCompile(`\b(\d{1,3}(?:,\d{3})+)\s*mi\b`)
		if m2 := odoFallbackRe.FindStringSubmatch(plainText); len(m2) > 1 {
			odometer = m2[1] + " mi"
		}
	}

	// 2. Extract Carfax Value
	carfaxValue := ""
	valRe := regexp.MustCompile(`(?i)(?:CARFAX\s*VALUE|Retail\s*Value|History-Based\s*Value)[^\$\d]{0,40}(\$[0-9,]+)`)
	if m := valRe.FindStringSubmatch(plainText); len(m) > 1 {
		carfaxValue = strings.TrimSpace(m[1])
	}

	// 3. Extract real owners count
	owners := 1
	ownerRe := regexp.MustCompile(`(?i)CARFAX\s+(\d+)-Owner`)
	if m := ownerRe.FindStringSubmatch(plainText); len(m) > 1 {
		if v, err := strconv.Atoi(m[1]); err == nil && v > 0 {
			owners = v
		}
	} else {
		ownerColRe := regexp.MustCompile(`(?i)\bOwner\s+(\d+)\b`)
		allMatches := ownerColRe.FindAllStringSubmatch(plainText, -1)
		maxOwner := 0
		for _, om := range allMatches {
			if len(om) > 1 {
				if v, err := strconv.Atoi(om[1]); err == nil && v > maxOwner {
					maxOwner = v
				}
			}
		}
		if maxOwner > 0 {
			owners = maxOwner
		}
	}

	// 4. Extract real accident status & summary
	accidents := 0
	accidentSummary := "No accidents or damage reported to CARFAX"
	damageSeverity := "None"
	if !strings.Contains(lower, "no accidents or damage reported") {
		switch {
		case strings.Contains(lower, "severe damage"):
			accidents = 1
			accidentSummary = "Severe Damage Reported"
			damageSeverity = "Severe"
		case strings.Contains(lower, "vandalism damage"):
			accidents = 1
			accidentSummary = "Vandalism Damage Reported"
			damageSeverity = "Moderate"
		case strings.Contains(lower, "moderate damage"):
			accidents = 1
			accidentSummary = "Moderate Damage Reported"
			damageSeverity = "Moderate"
		case strings.Contains(lower, "minor damage"):
			accidents = 1
			accidentSummary = "Minor Damage Reported"
			damageSeverity = "Minor"
		case strings.Contains(lower, "accident reported") || strings.Contains(lower, "damage reported"):
			accidents = 1
			accidentSummary = "Accident / Damage Reported"
			damageSeverity = "Reported"
		}
	}

	// 5. Extract Service Records, Vehicle Use, Last Location, Detailed Count
	serviceRecordsText := ""
	if m := regexp.MustCompile(`(?i)(\d+\s+Service\s+History\s+Records?)`).FindStringSubmatch(plainText); len(m) > 1 {
		serviceRecordsText = m[1]
	}
	vehicleUse := "Personal Vehicle"
	if m := regexp.MustCompile(`(?i)(Personal\s+Vehicle|Rental\s+Vehicle|Commercial\s+Vehicle|Personal\s+Lease|Fleet\s+Vehicle)`).FindStringSubmatch(plainText); len(m) > 1 {
		vehicleUse = m[1]
	}
	lastLocation := ""
	if m := regexp.MustCompile(`(?i)Last\s+Owned\s+in\s+([A-Za-z]+(?:\s+[A-Za-z]+)?)`).FindStringSubmatch(plainText); len(m) > 1 {
		lastLocation = "Last Owned in " + strings.TrimSpace(m[1])
	}
	detailedCountText := ""
	if m := regexp.MustCompile(`(?i)(\d+\s+Detailed\s+Records?\s+Available)`).FindStringSubmatch(plainText); len(m) > 1 {
		detailedCountText = m[1]
	}

	// 6. Clean title check
	cleanTitle := true
	if strings.Contains(lower, "salvage title") || strings.Contains(lower, "rebuilt title") || strings.Contains(lower, "total loss reported") {
		cleanTitle = false
	}

	// 7. Extract all structured <table> blocks & chronological Detailed History records
	var tables []map[string]interface{}
	var detailedRecords []map[string]string

	tblMatches := tableBlkRe.FindAllStringSubmatch(lightHtml, -1)
	for tIdx, tm := range tblMatches {
		if len(tm) < 2 {
			continue
		}
		tblInner := tm[1]
		var headers []string
		var rows [][]string

		trMatches := trBlkRe.FindAllStringSubmatch(tblInner, -1)
		for _, trm := range trMatches {
			if len(trm) < 2 {
				continue
			}
			trInner := trm[1]
			thMatches := thCellRe.FindAllStringSubmatch(trInner, -1)
			if len(thMatches) > 0 && len(rows) == 0 && len(headers) == 0 {
				for _, thm := range thMatches {
					if h := cleanHtmlText(thm[1]); h != "" {
						headers = append(headers, h)
					}
				}
				continue
			}

			tdMatches := tdCellRe.FindAllStringSubmatch(trInner, -1)
			if len(tdMatches) == 0 {
				continue
			}
			var cells []string
			nonEmpty := false
			for _, tdm := range tdMatches {
				c := cleanHtmlText(tdm[1])
				if c != "" {
					nonEmpty = true
				}
				cells = append(cells, c)
			}
			if !nonEmpty {
				continue
			}

			firstCell := cells[0]
			if len(cells) >= 3 && dateRowRe.MatchString(firstCell) {
				rec := map[string]string{
					"date": firstCell,
				}
				if len(cells) >= 4 {
					rec["mileage"] = cells[1]
					rec["source"] = cells[2]
					rec["comments"] = strings.Join(cells[3:], " | ")
				} else {
					rec["mileage"] = ""
					rec["source"] = cells[1]
					rec["comments"] = cells[2]
				}
				detailedRecords = append(detailedRecords, rec)
			} else {
				rows = append(rows, cells)
			}
		}

		if len(rows) > 0 {
			secTitle := fmt.Sprintf("Section %d", tIdx+1)
			if len(headers) > 0 && headers[0] != "" {
				secTitle = headers[0]
			}
			tables = append(tables, map[string]interface{}{
				"title":   secTitle,
				"headers": headers,
				"rows":    rows,
			})
		}
	}

	score := 95
	if accidents > 0 {
		score = 82
	}

	summary := map[string]interface{}{
		"vin":                    vin,
		"vehicle_title":          title,
		"carfax_score":           score,
		"carfax_value":           carfaxValue,
		"total_owners":           owners,
		"accidents_reported":     accidents,
		"accident_summary":       accidentSummary,
		"damage_severity":        damageSeverity,
		"clean_title":            cleanTitle,
		"last_reported_odometer": odometer,
		"service_records_count":  serviceRecordsText,
		"vehicle_use":            vehicleUse,
		"last_owned_location":    lastLocation,
		"detailed_records_count": detailedCountText,
		"tables":                 tables,
		"detailed_records":       detailedRecords,
		"has_official_pdf":       hasOfficialPdf,
		"has_full_html":          len(html) > 500,
		"analyzed_at":            time.Now().Format(time.RFC3339),
	}

	bytes, _ := json.Marshal(summary)
	return string(bytes)
}

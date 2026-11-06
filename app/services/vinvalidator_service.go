package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type VinValidatorService struct {
	httpClient *http.Client
}

var (
	nhtsaValidCache sync.Map // cleanVin -> nhtsaCheckResult
)

type nhtsaCheckResult struct {
	IsValid  bool
	ErrorMsg string
	CachedAt time.Time
}

func NewVinValidatorService() *VinValidatorService {
	return &VinValidatorService{
		httpClient: &http.Client{Timeout: 3500 * time.Millisecond},
	}
}

// Weights for positions 1 through 17
var vinWeights = [17]int{8, 7, 6, 5, 4, 3, 2, 10, 0, 9, 8, 7, 6, 5, 4, 3, 2}

// Character transliteration values (ISO 3779 / NHTSA standard)
var vinCharValues = map[rune]int{
	'A': 1, 'B': 2, 'C': 3, 'D': 4, 'E': 5, 'F': 6, 'G': 7, 'H': 8,
	'J': 1, 'K': 2, 'L': 3, 'M': 4, 'N': 5, 'P': 7, 'R': 9,
	'S': 2, 'T': 3, 'U': 4, 'V': 5, 'W': 6, 'X': 7, 'Y': 8, 'Z': 9,
	'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9,
}

var vinRegex = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)
var lastFourDigitsRegex = regexp.MustCompile(`^[0-9]{4}$`)

// ValidateAndClean strictly sanitizes the VIN and verifies length, forbidden chars,
// check digit (position 9), model year code (position 10), serial digits, and real manufacturer registration.
func (s *VinValidatorService) ValidateAndClean(vin string) (string, bool, string) {
	// 1. Remove all spaces, hyphens, and convert to uppercase
	clean := strings.ToUpper(strings.TrimSpace(vin))
	clean = strings.ReplaceAll(clean, " ", "")
	clean = strings.ReplaceAll(clean, "-", "")
	clean = strings.ReplaceAll(clean, "_", "")

	// 2. Check exact length
	if len(clean) != 17 {
		return clean, false, fmt.Sprintf("رقم الشاصي يجب أن يتكون من 17 رمزاً بالضبط (الحالي: %d)", len(clean))
	}

	// 3. Check for internationally forbidden characters (I, O, Q)
	if strings.ContainsAny(clean, "IOQ") {
		return clean, false, "رقم الشاصي يحتوي على أحرف غير صالحة (I, O, Q)"
	}

	// 4. Ensure alphanumeric without special characters
	if !vinRegex.MatchString(clean) {
		return clean, false, "رقم الشاصي يجب أن يحتوي على أرقام وحروف إنجليزية صالحة فقط"
	}

	// 5. Reject repeating / junk test patterns (e.g. 11111111111111111)
	charCounts := make(map[rune]int)
	for _, r := range clean {
		charCounts[r]++
		if charCounts[r] >= 9 {
			return clean, false, "رقم الشاصي غير صحيح (نمط أرقام أو حروف مكررة غير صالح)"
		}
	}

	// 6. Position 10 (index 9) is the Model Year code: 0, U, Z are illegal in all VIN standards
	yearChar := clean[9]
	if yearChar == '0' || yearChar == 'U' || yearChar == 'Z' {
		return clean, false, "رقم الشاصي غير صحيح (رمز سنة الصنع في الخانة العاشرة غير صالح)"
	}

	// 7. Positions 14-17 (last 4 characters of VIS serial number) must be numeric digits
	if !lastFourDigitsRegex.MatchString(clean[13:17]) {
		return clean, false, "رقم الشاصي غير صحيح (الأرقام التسلسلية الأخيرة يجب أن تكون أرقاماً)"
	}

	// 8. Strict ISO 3779 / NHTSA Check Digit Verification (Position 9, index 8)
	runes := []rune(clean)
	sum := 0
	for i := 0; i < 17; i++ {
		val, exists := vinCharValues[runes[i]]
		if !exists {
			return clean, false, "رقم الشاصي يحتوي على رموز غير صالحة"
		}
		sum += val * vinWeights[i]
	}

	remainder := sum % 11
	var expectedCheckDigit rune
	if remainder == 10 {
		expectedCheckDigit = 'X'
	} else {
		expectedCheckDigit = rune('0' + remainder)
	}

	if runes[8] != expectedCheckDigit {
		return clean, false, "رقم الشاصي غير صحيح: يوجد خطأ في كتابة أحد الأرقام أو الحروف، يرجى مراجعة الرقم بدقة"
	}

	// 9. Verify real manufacturer & model registration via NHTSA vPIC (cached in memory)
	if ok, nhtsaErr := s.verifyRealManufacturer(clean); !ok {
		return clean, false, nhtsaErr
	}

	return clean, true, ""
}

// verifyRealManufacturer checks that the VIN decodes to a real vehicle Make & ModelYear (not a phantom/unregistered VIN)
func (s *VinValidatorService) verifyRealManufacturer(cleanVin string) (bool, string) {
	if cached, ok := nhtsaValidCache.Load(cleanVin); ok {
		res := cached.(nhtsaCheckResult)
		if time.Since(res.CachedAt) < 12*time.Hour {
			return res.IsValid, res.ErrorMsg
		}
	}

	url := fmt.Sprintf("https://vpic.nhtsa.dot.gov/api/vehicles/DecodeVinValues/%s?format=json", cleanVin)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		// If NHTSA API is temporarily unreachable, rely on the strict mathematical check digit passed in step 8
		return true, ""
	}
	defer resp.Body.Close()

	var payload struct {
		Results []struct {
			Make        string `json:"Make"`
			Model       string `json:"Model"`
			ModelYear   string `json:"ModelYear"`
			VehicleType string `json:"VehicleType"`
			ErrorCode   string `json:"ErrorCode"`
		} `json:"Results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || len(payload.Results) == 0 {
		return true, ""
	}

	item := payload.Results[0]
	makeName := strings.TrimSpace(item.Make)
	modelYear := strings.TrimSpace(item.ModelYear)
	errorCodes := strings.Split(strings.TrimSpace(item.ErrorCode), ",")

	// Check if NHTSA explicitly flags Check Digit error (1), Invalid Manufacturer (7), or No Detailed Data (8) with no Make
	hasCheckDigitOrMakerError := false
	for _, code := range errorCodes {
		c := strings.TrimSpace(code)
		if c == "1" || c == "6" || c == "7" {
			hasCheckDigitOrMakerError = true
			break
		}
	}

	if makeName == "" || modelYear == "" || hasCheckDigitOrMakerError {
		errMsg := "رقم الشاصي هذا غير صحيح أو غير مسجل لأي مركبة حقيقية، يرجى التأكد من صحة الرقم"
		nhtsaValidCache.Store(cleanVin, nhtsaCheckResult{
			IsValid:  false,
			ErrorMsg: errMsg,
			CachedAt: time.Now(),
		})
		return false, errMsg
	}

	nhtsaValidCache.Store(cleanVin, nhtsaCheckResult{
		IsValid:  true,
		ErrorMsg: "",
		CachedAt: time.Now(),
	})
	return true, ""
}

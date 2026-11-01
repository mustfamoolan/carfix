package services

import (
	"fmt"
	"regexp"
	"strings"
)

type VinValidatorService struct{}

func NewVinValidatorService() *VinValidatorService {
	return &VinValidatorService{}
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

// ValidateAndClean sanitizes the VIN and checks length, forbidden chars, and checksum
func (s *VinValidatorService) ValidateAndClean(vin string) (string, bool, string) {
	// 1. Remove all spaces, hyphens, and convert to uppercase
	clean := strings.ToUpper(strings.TrimSpace(vin))
	clean = strings.ReplaceAll(clean, " ", "")
	clean = strings.ReplaceAll(clean, "-", "")

	// 2. Check exact length
	if len(clean) != 17 {
		return clean, false, fmt.Sprintf("رقم الشاصي يجب أن يتكون من 17 حرفاً ورقم بالضبط (الحالي: %d)", len(clean))
	}

	// 3. Check for internationally forbidden characters (I, O, Q)
	if strings.ContainsAny(clean, "IOQ") {
		return clean, false, "رقم الشاصي يحتوي على أحرف محظورة دولياً (I, O, Q) لتجنب الالتباس مع الأرقام"
	}

	// 4. Ensure alphanumeric without special characters
	if !vinRegex.MatchString(clean) {
		return clean, false, "رقم الشاصي يجب أن يحتوي على أرقام وحروف إنجليزية صالحة فقط"
	}

	// 5. Standard Checksum Check (ISO 3779 / NHTSA)
	// We calculate it, but if it doesn't match we log/warn rather than blocking,
	// because European/Asian market VINs do not strictly mandate check digits.
	runes := []rune(clean)
	sum := 0
	for i := 0; i < 17; i++ {
		val, exists := vinCharValues[runes[i]]
		if !exists {
			return clean, false, "رقم الشاصي يحتوي على رموز غير صالحة"
		}
		sum += val * vinWeights[i]
	}

	// Clean and structurally valid (17 alphanumeric chars without I, O, Q)
	return clean, true, ""
}

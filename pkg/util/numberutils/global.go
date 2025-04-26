package numberutils

import "unicode"

// IsDigits checks if the given string contains only digits (0-9).
// It returns true if all characters in the string are digits, false otherwise.
func IsDigits(str string) bool {
	for _, r := range str {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

package numberutils

import (
	"strconv"
)

// IsInt64 checks if the given string can be converted to a valid int64.
// It returns true if the string can be converted to an int64, false otherwise.
func IsInt64(str string) bool {
	_, err := strconv.ParseInt(str, 10, 64)
	return err == nil
}

// ToInt64 converts the given string to an int64.
// If the string cannot be converted, it returns 0.
func ToInt64(s string) int64 {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	return 0
}

// ToInt64WithDefault converts the given string to an int64.
// If the string cannot be converted, it returns the provided default value.
func ToInt64WithDefault(s string, defaultVal int64) int64 {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	return defaultVal
}

// ToInt64WithError converts the given string to an int64 and returns any error that occurred during conversion.
// It returns the int64 value if successful, or an error if the string cannot be converted.
func ToInt64WithError(str string) (int64, error) {
	return strconv.ParseInt(str, 10, 64)
}

// MaxInt64 returns the maximum value from a list of int64 values.
// It accepts a variadic number of int64 values and returns the largest one.
func MaxInt64(nums ...int64) int64 {
	if len(nums) == 0 {
		return 0
	}
	maxVal := nums[0]
	for _, num := range nums[1:] {
		if num > maxVal {
			maxVal = num
		}
	}
	return maxVal
}

// MinInt64 returns the minimum value from a list of int64 values.
// It accepts a variadic number of int64 values and returns the smallest one.
func MinInt64(nums ...int64) int64 {
	if len(nums) == 0 {
		return 0
	}
	minVal := nums[0]
	for _, num := range nums[1:] {
		if num < minVal {
			minVal = num
		}
	}
	return minVal
}

// IsInt64InRange checks if the given int64 is within the specified range (inclusive).
// It returns true if the number is greater than or equal to the minimum and less than or equal to the maximum.
func IsInt64InRange(num, min, max int64) bool {
	return num >= min && num <= max
}

// IsInt64Positive checks if the given int64 is positive.
// It returns true if the number is greater than zero.
func IsInt64Positive(number int64) bool {
	return number > 0
}

// IsInt64Negative checks if the given int64 is negative.
// It returns true if the number is less than zero.
func IsInt64Negative(number int64) bool {
	return number < 0
}

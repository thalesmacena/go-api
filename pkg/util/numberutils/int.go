package numberutils

import (
	"math"
	"strconv"
)

// IsInt checks if the given string can be converted to a valid integer.
// It returns true if the string can be converted to an integer, false otherwise.
func IsInt(str string) bool {
	_, err := strconv.Atoi(str)
	return err == nil
}

// ToInt converts the given string to an integer.
// If the string cannot be converted, it returns 0.
func ToInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

// ToIntWithDefault converts the given string to an integer.
// If the string cannot be converted, it returns the provided default value.
func ToIntWithDefault(s string, defaultVal int) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return defaultVal
}

// ToIntWithError converts the given string to an integer and returns any error that occurred during conversion.
// It returns the integer value if successful, or an error if the string cannot be converted.
func ToIntWithError(str string) (int, error) {
	return strconv.Atoi(str)
}

// MaxInt returns the maximum value from a list of integers.
// It accepts a variadic number of integers and returns the largest one.
func MaxInt(nums ...int) int {
	maxVal := math.MinInt
	for _, num := range nums {
		if num > maxVal {
			maxVal = num
		}
	}
	return maxVal
}

// MinInt returns the minimum value from a list of integers.
// It accepts a variadic number of integers and returns the smallest one.
func MinInt(nums ...int) int {
	minVal := math.MaxInt
	for _, num := range nums {
		if num < minVal {
			minVal = num
		}
	}
	return minVal
}

// IsIntInRange checks if the given number is within the specified range (inclusive).
// It returns true if the number is greater than or equal to the minimum and less than or equal to the maximum.
func IsIntInRange(num, min, max int) bool {
	return num >= min && num <= max
}

// IsIntPositive checks if the given number is positive.
// It returns true if the number is greater than zero.
func IsIntPositive(number int) bool {
	return number > 0
}

// IsIntNegative checks if the given number is negative.
// It returns true if the number is less than zero.
func IsIntNegative(number int) bool {
	return number < 0
}

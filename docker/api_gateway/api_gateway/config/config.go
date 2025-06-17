package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func parseByteSize(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Split into number and suffix
	var multiplier int64 = 1
	n := len(s)
	unit := s[n-1]
	value := s

	switch unit {
	case 'K', 'k':
		multiplier = 1024
		value = s[:n-1]
	case 'M', 'm':
		multiplier = 1024 * 1024
		value = s[:n-1]
	case 'G', 'g':
		multiplier = 1024 * 1024 * 1024
		value = s[:n-1]
	case 'T', 't':
		multiplier = 1024 * 1024 * 1024 * 1024
		value = s[:n-1]
	default:
		// No suffix — try to parse whole string as number of bytes
	}

	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size number in %q: %w", s, err)
	}

	return uint64(num * float64(multiplier)), nil
}

// GetEnv fetches an environment variable or returns a default value
func GetEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

// GetEnvBool fetches an environment variable as a boolean value or returns a default value
func GetEnvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)

	// Convert the string value to a boolean
	switch value {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		// If the value is not recognized, return the default value
		return defaultValue
	}
}

// GetEnvInt fetches an environment variable as an integer or returns a default value
func GetEnvInt(key string, defaultValue int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	// Convert the string value to an integer
	intValue, err := strconv.Atoi(value)
	if err != nil {
		// If conversion fails, return the default value
		return defaultValue
	}
	return intValue
}

// GetEnvInt64 fetches an environment variable as an int64 or returns a default value
func GetEnvInt64(key string, defaultValue int64) int64 {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	// Convert the string value to an int64
	int64Value, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		// If conversion fails, return the default value
		return defaultValue
	}
	return int64Value
}

// GetEnvFloat fetches an environment variable as a float64 or returns a default value
func GetEnvFloat(key string, defaultValue float64) float64 {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	// Convert the string value to a float64
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		// If conversion fails, return the default value
		return defaultValue
	}
	return floatValue
}

// GetEnvDuration fetches an environment variable as a duration string or returns a default value
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	value = strings.TrimSpace(value)
	// Validate the duration format (optional, depending on your use case)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		// If parsing fails, return the default value
		return defaultValue
	}
	return duration
}

// GetEnvByteSize fetches an environment variable as a byte size or returns a default value
func GetEnvByteSize(key string, defaultValue uint64) uint64 {
	value, exists := os.LookupEnv(key)
	value = strings.TrimSpace(value)

	if !exists || value == "" {
		return defaultValue
	}

	byteSize, err := parseByteSize(value)
	if err != nil {
		// If parsing fails, return the default value
		return defaultValue
	}
	return byteSize
}

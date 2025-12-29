package config
package config

import (
	"os"
	"testing"
	"time"
)

// TestGetEnv tests string environment variable retrieval
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue string
		shouldSet    bool
		want         string
	}{
		{
			name:         "existing env variable",
			key:          "TEST_STRING_VAR",
			value:        "test_value",
			defaultValue: "default",
			shouldSet:    true,
			want:         "test_value",
		},
		{
			name:         "missing env variable",
			key:          "MISSING_VAR",
			defaultValue: "default_value",
			shouldSet:    false,
			want:         "default_value",
		},
		{
			name:         "empty env variable",
			key:          "EMPTY_VAR",
			value:        "",
			defaultValue: "default",
			shouldSet:    true,
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSet {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnv() = %s, want %s", got, tt.want)
			}
		})
	}
}

// TestGetEnvBool tests boolean environment variable retrieval
func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue bool
		shouldSet    bool
		want         bool
	}{
		{
			name:         "true value",
			key:          "TEST_BOOL_TRUE",
			value:        "true",
			defaultValue: false,
			shouldSet:    true,
			want:         true,
		},
		{
			name:         "1 value",
			key:          "TEST_BOOL_ONE",
			value:        "1",
			defaultValue: false,
			shouldSet:    true,
			want:         true,
		},
		{
			name:         "yes value",
			key:          "TEST_BOOL_YES",
			value:        "yes",
			defaultValue: false,
			shouldSet:    true,
			want:         true,
		},
		{
			name:         "on value",
			key:          "TEST_BOOL_ON",
			value:        "on",
			defaultValue: false,
			shouldSet:    true,
			want:         true,
		},
		{
			name:         "false value",
			key:          "TEST_BOOL_FALSE",
			value:        "false",
			defaultValue: true,
			shouldSet:    true,
			want:         false,
		},
		{
			name:         "0 value",
			key:          "TEST_BOOL_ZERO",
			value:        "0",
			defaultValue: true,
			shouldSet:    true,
			want:         false,
		},
		{
			name:         "no value",
			key:          "TEST_BOOL_NO",
			value:        "no",
			defaultValue: true,
			shouldSet:    true,
			want:         false,
		},
		{
			name:         "off value",
			key:          "TEST_BOOL_OFF",
			value:        "off",
			defaultValue: true,
			shouldSet:    true,
			want:         false,
		},
		{
			name:         "missing env variable",
			key:          "MISSING_BOOL",
			defaultValue: true,
			shouldSet:    false,
			want:         true,
		},
		{
			name:         "invalid value returns default",
			key:          "TEST_BOOL_INVALID",
			value:        "invalid",
			defaultValue: true,
			shouldSet:    true,
			want:         true,
		},
		{
			name:         "case insensitive TRUE",
			key:          "TEST_BOOL_CASE",
			value:        "TRUE",
			defaultValue: false,
			shouldSet:    true,
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSet {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnvBool(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnvBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetEnvInt tests integer environment variable retrieval
func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue int
		shouldSet    bool
		want         int
	}{
		{
			name:         "positive integer",
			key:          "TEST_INT_POS",
			value:        "42",
			defaultValue: 0,
			shouldSet:    true,
			want:         42,
		},
		{
			name:         "negative integer",
			key:          "TEST_INT_NEG",
			value:        "-10",
			defaultValue: 0,
			shouldSet:    true,
			want:         -10,
		},
		{
			name:         "zero value",
			key:          "TEST_INT_ZERO",
			value:        "0",
			defaultValue: 100,
			shouldSet:    true,
			want:         0,
		},
		{
			name:         "missing env variable",
			key:          "MISSING_INT",
			defaultValue: 99,
			shouldSet:    false,
			want:         99,
		},
		{
			name:         "invalid value returns default",
			key:          "TEST_INT_INVALID",
			value:        "not_a_number",
			defaultValue: 50,
			shouldSet:    true,
			want:         50,
		},
		{
			name:         "large integer",
			key:          "TEST_INT_LARGE",
			value:        "999999",
			defaultValue: 0,
			shouldSet:    true,
			want:         999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSet {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnvInt(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnvInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestGetEnvInt64 tests int64 environment variable retrieval
func TestGetEnvInt64(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue int64
		shouldSet    bool
		want         int64
	}{
		{
			name:         "large int64",
			key:          "TEST_INT64_LARGE",
			value:        "9223372036854775807",
			defaultValue: 0,
			shouldSet:    true,
			want:         9223372036854775807,
		},
		{
			name:         "negative int64",
			key:          "TEST_INT64_NEG",
			value:        "-9223372036854775808",
			defaultValue: 0,
			shouldSet:    true,
			want:         -9223372036854775808,
		},
		{
			name:         "missing env variable",
			key:          "MISSING_INT64",
			defaultValue: 123456789,
			shouldSet:    false,
			want:         123456789,
		},
		{
			name:         "invalid value returns default",
			key:          "TEST_INT64_INVALID",
			value:        "invalid",
			defaultValue: 999,
			shouldSet:    true,
			want:         999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSet {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnvInt64(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnvInt64() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestGetEnvFloat tests float64 environment variable retrieval
func TestGetEnvFloat(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue float64
		shouldSet    bool
		want         float64
	}{
		{
			name:         "float value",
			key:          "TEST_FLOAT",
			value:        "3.14159",
			defaultValue: 0.0,
			shouldSet:    true,
			want:         3.14159,
		},
		{
			name:         "scientific notation",
			key:          "TEST_FLOAT_SCI",
			value:        "1.23e-4",
			defaultValue: 0.0,
			shouldSet:    true,
			want:         0.000123,
		},
		{
			name:         "negative float",
			key:          "TEST_FLOAT_NEG",
			value:        "-99.99",
			defaultValue: 0.0,
			shouldSet:    true,
			want:         -99.99,
		},
		{
			name:         "missing env variable",
			key:          "MISSING_FLOAT",
			defaultValue: 1.5,
			shouldSet:    false,
			want:         1.5,
		},
		{
			name:         "invalid value returns default",
			key:          "TEST_FLOAT_INVALID",
			value:        "not_a_float",
			defaultValue: 2.5,
			shouldSet:    true,
			want:         2.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSet {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnvFloat(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnvFloat() = %f, want %f", got, tt.want)
			}
		})
	}
}

// TestGetEnvDuration tests duration environment variable retrieval
func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue time.Duration
		shouldSet    bool
		want         time.Duration
	}{
		{
			name:         "seconds duration",
			key:          "TEST_DURATION_SEC",
			value:        "30s",
			defaultValue: 0,
			shouldSet:    true,
			want:         30 * time.Second,
		},
		{
			name:         "minutes duration",
			key:          "TEST_DURATION_MIN",
			value:        "5m",
			defaultValue: 0,
			shouldSet:    true,
			want:         5 * time.Minute,
		},
		{
			name:         "hours duration",
			key:          "TEST_DURATION_HOUR",
			value:        "2h",
			defaultValue: 0,
			shouldSet:    true,
			want:         2 * time.Hour,
		},
		{
			name:         "complex duration",
			key:          "TEST_DURATION_COMPLEX",
			value:        "1h30m45s",
			defaultValue: 0,
			shouldSet:    true,
			want:         1*time.Hour + 30*time.Minute + 45*time.Second,
		},
		{
			name:         "missing env variable",
			key:          "MISSING_DURATION",
			defaultValue: 10 * time.Second,
			shouldSet:    false,
			want:         10 * time.Second,
		},
		{
			name:         "invalid value returns default",
			key:          "TEST_DURATION_INVALID",
			value:        "invalid",
			defaultValue: 5 * time.Minute,
			shouldSet:    true,
			want:         5 * time.Minute,
		},
		{
			name:         "empty value returns default",
			key:          "TEST_DURATION_EMPTY",
			value:        "",
			defaultValue: 15 * time.Second,
			shouldSet:    true,
			want:         15 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSet {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnvDuration(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnvDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetEnvByteSize tests byte size environment variable retrieval
func TestGetEnvByteSize(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue uint64
		shouldSet    bool
		want         uint64
	}{
		{
			name:         "kilobytes",
			key:          "TEST_SIZE_KB",
			value:        "10K",
			defaultValue: 0,
			shouldSet:    true,
			want:         10 * 1024,
		},
		{
			name:         "megabytes",
			key:          "TEST_SIZE_MB",
			value:        "5M",
			defaultValue: 0,
			shouldSet:    true,
			want:         5 * 1024 * 1024,
		},
		{
			name:         "gigabytes",
			key:          "TEST_SIZE_GB",
			value:        "2G",
			defaultValue: 0,
			shouldSet:    true,
			want:         2 * 1024 * 1024 * 1024,
		},
		{
			name:         "bytes without suffix",
			key:          "TEST_SIZE_BYTES",
			value:        "1024",
			defaultValue: 0,
			shouldSet:    true,
			want:         1024,
		},
		{
			name:         "decimal megabytes",
			key:          "TEST_SIZE_DECIMAL",
			value:        "1.5M",
			defaultValue: 0,
			shouldSet:    true,
			want:         1.5 * 1024 * 1024,
		},
		{
			name:         "lowercase suffix",
			key:          "TEST_SIZE_LOWER",
			value:        "10k",
			defaultValue: 0,
			shouldSet:    true,
			want:         10 * 1024,
		},
		{
			name:         "missing env variable",
			key:          "MISSING_SIZE",
			defaultValue: 1024,
			shouldSet:    false,
			want:         1024,
		},
		{
			name:         "invalid value returns default",
			key:          "TEST_SIZE_INVALID",
			value:        "invalid",
			defaultValue: 2048,
			shouldSet:    true,
			want:         2048,
		},
		{
			name:         "empty value returns default",
			key:          "TEST_SIZE_EMPTY",
			value:        "",
			defaultValue: 4096,
			shouldSet:    true,
			want:         4096,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSet {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnvByteSize(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnvByteSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestGetEnvPercentage tests percentage environment variable retrieval
func TestGetEnvPercentage(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue float64
		shouldSet    bool
		want         float64
	}{
		{
			name:         "percentage value",
			key:          "TEST_PERCENT",
			value:        "75",
			defaultValue: 0.5,
			shouldSet:    true,
			want:         0.75,
		},
		{
			name:         "decimal fraction",
			key:          "TEST_PERCENT_FRAC",
			value:        "0.85",
			defaultValue: 0.5,
			shouldSet:    true,
			want:         0.85,
		},
		{
			name:         "100 percent",
			key:          "TEST_PERCENT_100",
			value:        "100",
			defaultValue: 0.5,
			shouldSet:    true,
			want:         1.0,
		},
		{
			name:         "zero percent",
			key:          "TEST_PERCENT_ZERO",
			value:        "0",
			defaultValue: 0.5,
			shouldSet:    true,
			want:         0.0,
		},
		{
			name:         "missing env variable",
			key:          "MISSING_PERCENT",
			defaultValue: 0.8,
			shouldSet:    false,
			want:         0.8,
		},
		{
			name:         "out of range returns default",
			key:          "TEST_PERCENT_RANGE",
			value:        "150",
			defaultValue: 0.5,
			shouldSet:    true,
			want:         0.5,
		},
		{
			name:         "negative value returns default",
			key:          "TEST_PERCENT_NEG",
			value:        "-10",
			defaultValue: 0.5,
			shouldSet:    true,
			want:         0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSet {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			got := GetEnvPercentage(tt.key, tt.defaultValue)
			// Use epsilon for float comparison
			epsilon := 0.0001
			if got < tt.want-epsilon || got > tt.want+epsilon {
				t.Errorf("GetEnvPercentage() = %f, want %f", got, tt.want)
			}
		})
	}
}

// TestParseByteSize tests the internal parseByteSize function
func TestParseByteSize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint64
		wantErr bool
	}{
		{
			name:    "kilobytes uppercase",
			input:   "10K",
			want:    10 * 1024,
			wantErr: false,
		},
		{
			name:    "megabytes uppercase",
			input:   "5M",
			want:    5 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "gigabytes uppercase",
			input:   "2G",
			want:    2 * 1024 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "terabytes uppercase",
			input:   "1T",
			want:    1024 * 1024 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "bytes without suffix",
			input:   "1024",
			want:    1024,
			wantErr: false,
		},
		{
			name:    "decimal with suffix",
			input:   "1.5M",
			want:    1.5 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "ABC",
			want:    0,
			wantErr: true,
		},
		{
			name:    "whitespace trimmed",
			input:   "  10M  ",
			want:    10 * 1024 * 1024,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseByteSize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseByteSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseByteSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestConcurrentConfigAccess tests thread-safety of config functions
func TestConcurrentConfigAccess(t *testing.T) {
	// Set up test environment variables
	os.Setenv("CONCURRENT_TEST_STRING", "test")
	os.Setenv("CONCURRENT_TEST_INT", "42")
	os.Setenv("CONCURRENT_TEST_BOOL", "true")
	defer func() {
		os.Unsetenv("CONCURRENT_TEST_STRING")
		os.Unsetenv("CONCURRENT_TEST_INT")
		os.Unsetenv("CONCURRENT_TEST_BOOL")
	}()

	const numGoroutines = 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Call various config functions concurrently
			_ = GetEnv("CONCURRENT_TEST_STRING", "default")
			_ = GetEnvInt("CONCURRENT_TEST_INT", 0)
			_ = GetEnvBool("CONCURRENT_TEST_BOOL", false)
			_ = GetEnvDuration("CONCURRENT_TEST_DURATION", 5*time.Second)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

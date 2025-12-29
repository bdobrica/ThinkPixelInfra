package logger

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"
)

// captureLogOutput captures log output for testing
func captureLogOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	f()
	log.SetOutput(nil) // Reset to default
	return buf.String()
}

// TestSetLogLevel tests log level setting
func TestSetLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level int
	}{
		{
			name:  "debug level",
			level: Debug,
		},
		{
			name:  "info level",
			level: Info,
		},
		{
			name:  "warning level",
			level: Warning,
		},
		{
			name:  "error level",
			level: Error,
		},
		{
			name:  "critical level",
			level: Critical,
		},
		{
			name:  "not set level",
			level: NotSet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLogLevel(tt.level)
			if LogLevel != tt.level {
				t.Errorf("SetLogLevel() LogLevel = %d, want %d", LogLevel, tt.level)
			}
		})
	}
}

// TestLogLevelConstants tests log level constant values
func TestLogLevelConstants(t *testing.T) {
	tests := []struct {
		name  string
		level int
		want  int
	}{
		{
			name:  "NotSet",
			level: NotSet,
			want:  0,
		},
		{
			name:  "Debug",
			level: Debug,
			want:  10,
		},
		{
			name:  "Info",
			level: Info,
			want:  20,
		},
		{
			name:  "Warning",
			level: Warning,
			want:  30,
		},
		{
			name:  "Error",
			level: Error,
			want:  40,
		},
		{
			name:  "Critical",
			level: Critical,
			want:  50,
		},
		{
			name:  "Fatal equals Critical",
			level: Fatal,
			want:  Critical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.level != tt.want {
				t.Errorf("Log level %s = %d, want %d", tt.name, tt.level, tt.want)
			}
		})
	}
}

// TestDebugf tests debug logging
func TestDebugf(t *testing.T) {
	// Set log level to Debug
	SetLogLevel(Debug)

	output := captureLogOutput(func() {
		Debugf("Debug message: %s", "test")
	})

	if !strings.Contains(output, "DEBUG") {
		t.Error("Debug log should contain DEBUG prefix")
	}
	if !strings.Contains(output, "Debug message: test") {
		t.Error("Debug log should contain the message")
	}
}

// TestInfof tests info logging
func TestInfof(t *testing.T) {
	// Set log level to Info
	SetLogLevel(Info)

	output := captureLogOutput(func() {
		Infof("Info message: %s", "test")
	})

	if !strings.Contains(output, "INFO") {
		t.Error("Info log should contain INFO prefix")
	}
	if !strings.Contains(output, "Info message: test") {
		t.Error("Info log should contain the message")
	}
}

// TestWarningf tests warning logging
func TestWarningf(t *testing.T) {
	// Set log level to Warning
	SetLogLevel(Warning)

	output := captureLogOutput(func() {
		Warningf("Warning message: %s", "test")
	})

	if !strings.Contains(output, "WARN") {
		t.Error("Warning log should contain WARN prefix")
	}
	if !strings.Contains(output, "Warning message: test") {
		t.Error("Warning log should contain the message")
	}
}

// TestErrorf tests error logging
func TestErrorf(t *testing.T) {
	// Set log level to Error
	SetLogLevel(Error)

	output := captureLogOutput(func() {
		Errorf("Error message: %s", "test")
	})

	if !strings.Contains(output, "ERROR") {
		t.Error("Error log should contain ERROR prefix")
	}
	if !strings.Contains(output, "Error message: test") {
		t.Error("Error log should contain the message")
	}
}

// TestCriticalf tests critical logging
func TestCriticalf(t *testing.T) {
	// Set log level to Critical
	SetLogLevel(Critical)

	output := captureLogOutput(func() {
		Criticalf("Critical message: %s", "test")
	})

	if !strings.Contains(output, "CRITICAL") {
		t.Error("Critical log should contain CRITICAL prefix")
	}
	if !strings.Contains(output, "Critical message: test") {
		t.Error("Critical log should contain the message")
	}
}

// TestLogLevelFiltering tests that logs below the set level are not printed
func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name         string
		setLevel     int
		logFunc      func()
		shouldLog    bool
		expectedText string
	}{
		{
			name:     "debug filtered at info level",
			setLevel: Info,
			logFunc: func() {
				Debugf("Debug message")
			},
			shouldLog:    false,
			expectedText: "DEBUG",
		},
		{
			name:     "info logged at info level",
			setLevel: Info,
			logFunc: func() {
				Infof("Info message")
			},
			shouldLog:    true,
			expectedText: "INFO",
		},
		{
			name:     "warning logged at info level",
			setLevel: Info,
			logFunc: func() {
				Warningf("Warning message")
			},
			shouldLog:    true,
			expectedText: "WARN",
		},
		{
			name:     "info filtered at warning level",
			setLevel: Warning,
			logFunc: func() {
				Infof("Info message")
			},
			shouldLog:    false,
			expectedText: "INFO",
		},
		{
			name:     "error logged at warning level",
			setLevel: Warning,
			logFunc: func() {
				Errorf("Error message")
			},
			shouldLog:    true,
			expectedText: "ERROR",
		},
		{
			name:     "debug filtered at error level",
			setLevel: Error,
			logFunc: func() {
				Debugf("Debug message")
			},
			shouldLog:    false,
			expectedText: "DEBUG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLogLevel(tt.setLevel)

			output := captureLogOutput(tt.logFunc)

			if tt.shouldLog {
				if !strings.Contains(output, tt.expectedText) {
					t.Errorf("Expected log output to contain %s, but it didn't: %s", tt.expectedText, output)
				}
			} else {
				if strings.Contains(output, tt.expectedText) {
					t.Errorf("Expected log output to NOT contain %s, but it did: %s", tt.expectedText, output)
				}
			}
		})
	}
}

// TestFormattedLogging tests logging with format specifiers
func TestFormattedLogging(t *testing.T) {
	SetLogLevel(Debug)

	tests := []struct {
		name         string
		logFunc      func()
		expectedText string
	}{
		{
			name: "string formatting",
			logFunc: func() {
				Infof("User: %s", "john_doe")
			},
			expectedText: "User: john_doe",
		},
		{
			name: "integer formatting",
			logFunc: func() {
				Infof("Count: %d", 42)
			},
			expectedText: "Count: 42",
		},
		{
			name: "multiple arguments",
			logFunc: func() {
				Infof("User: %s, ID: %d, Active: %v", "alice", 123, true)
			},
			expectedText: "User: alice, ID: 123, Active: true",
		},
		{
			name: "float formatting",
			logFunc: func() {
				Infof("Value: %.2f", 3.14159)
			},
			expectedText: "Value: 3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureLogOutput(tt.logFunc)

			if !strings.Contains(output, tt.expectedText) {
				t.Errorf("Expected log to contain %s, got: %s", tt.expectedText, output)
			}
		})
	}
}

// TestConcurrentLogging tests thread-safety of logging functions
func TestConcurrentLogging(t *testing.T) {
	SetLogLevel(Debug)

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Capture output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			Infof("Concurrent log %d", id)
		}(i)
	}

	wg.Wait()

	output := buf.String()

	// Verify all logs were written
	logCount := strings.Count(output, "INFO")
	if logCount != numGoroutines {
		t.Errorf("Expected %d log lines, got %d", numGoroutines, logCount)
	}
}

// TestConcurrentSetLogLevel tests thread-safety of SetLogLevel
func TestConcurrentSetLogLevel(t *testing.T) {
	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	levels := []int{Debug, Info, Warning, Error, Critical}

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			level := levels[id%len(levels)]
			SetLogLevel(level)
		}(i)
	}

	wg.Wait()

	// Just verify no panics occurred
	// The final log level is non-deterministic due to concurrent writes
}

// TestEmptyMessage tests logging with empty message
func TestEmptyMessage(t *testing.T) {
	SetLogLevel(Debug)

	output := captureLogOutput(func() {
		Infof("")
	})

	// Should still contain INFO prefix even with empty message
	if !strings.Contains(output, "INFO") {
		t.Error("Empty message should still contain INFO prefix")
	}
}

// TestLongMessage tests logging with very long message
func TestLongMessage(t *testing.T) {
	SetLogLevel(Debug)

	longMessage := strings.Repeat("A", 10000)

	output := captureLogOutput(func() {
		Infof("Long message: %s", longMessage)
	})

	if !strings.Contains(output, "INFO") {
		t.Error("Long message should contain INFO prefix")
	}
	if !strings.Contains(output, longMessage) {
		t.Error("Long message should be fully logged")
	}
}

// TestSpecialCharacters tests logging with special characters
func TestSpecialCharacters(t *testing.T) {
	SetLogLevel(Debug)

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "newlines",
			message: "Line 1\nLine 2\nLine 3",
		},
		{
			name:    "unicode",
			message: "日本語 エラー メッセージ",
		},
		{
			name:    "quotes",
			message: `Message with "quotes" and 'apostrophes'`,
		},
		{
			name:    "backslashes",
			message: `Path: C:\Users\test\file.txt`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureLogOutput(func() {
				Infof("Test: %s", tt.message)
			})

			if !strings.Contains(output, "INFO") {
				t.Error("Log should contain INFO prefix")
			}
			// Note: Some special characters may be escaped in output
		})
	}
}

// TestLogLevelBoundaries tests boundary conditions for log levels
func TestLogLevelBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		setLevel  int
		logLevel  int
		shouldLog bool
	}{
		{
			name:      "equal levels should log",
			setLevel:  Info,
			logLevel:  Info,
			shouldLog: true,
		},
		{
			name:      "higher level should log",
			setLevel:  Info,
			logLevel:  Warning,
			shouldLog: true,
		},
		{
			name:      "lower level should not log",
			setLevel:  Warning,
			logLevel:  Info,
			shouldLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLogLevel(tt.setLevel)

			var output string
			switch tt.logLevel {
			case Debug:
				output = captureLogOutput(func() { Debugf("test") })
			case Info:
				output = captureLogOutput(func() { Infof("test") })
			case Warning:
				output = captureLogOutput(func() { Warningf("test") })
			case Error:
				output = captureLogOutput(func() { Errorf("test") })
			case Critical:
				output = captureLogOutput(func() { Criticalf("test") })
			}

			hasOutput := len(output) > 0
			if hasOutput != tt.shouldLog {
				t.Errorf("Expected shouldLog=%v, got output=%v (length=%d)", tt.shouldLog, hasOutput, len(output))
			}
		})
	}
}

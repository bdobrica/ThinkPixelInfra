package logger

import (
	"context"
	"fmt"
	"log"
	"sync"

	"api_gateway/config"
)

const (
	Critical = 50
	Fatal    = Critical
	Error    = 40
	Warning  = 30
	Info     = 20
	Debug    = 10
	NotSet   = 0
)

var (
	LogLevel      int = Warning
	logLevelMutex sync.Mutex
)

func init() {
	localEnv := config.GetEnvBool("LOCAL", false)
	if localEnv {
		SetLogLevel(Debug)
	}
}

func SetLogLevel(level int) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	LogLevel = level
}

func Debugf(format string, v ...interface{}) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	if LogLevel <= Debug {
		log.Printf("[DEBUG] "+format, v...)
	}
}

func Infof(format string, v ...interface{}) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	if LogLevel <= Info {
		log.Printf("[INFO] "+format, v...)
	}
}

func Warningf(format string, v ...interface{}) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	if LogLevel <= Warning {
		log.Printf("[WARN] "+format, v...)
	}
}

func Errorf(format string, v ...interface{}) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	if LogLevel <= Error {
		log.Printf("[ERROR] "+format, v...)
	}
}

func Criticalf(format string, v ...interface{}) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	if LogLevel <= Critical {
		log.Printf("[CRITICAL] "+format, v...)
	}
}

func Fatalf(format string, v ...interface{}) {
	log.Fatalf("[FATAL] "+format, v...)
}

// Context-aware logging functions that include request IDs

// ContextKey is a custom type for context keys
type ContextKey string

// RequestIDKey is the context key for request IDs
const RequestIDKey ContextKey = "request_id"

// formatWithRequestID adds request ID to the log format if available in context
func formatWithRequestID(ctx context.Context, format string) string {
	if ctx == nil {
		return format
	}

	requestID, ok := ctx.Value(RequestIDKey).(string)
	if !ok || requestID == "" {
		return format
	}

	return fmt.Sprintf("[request_id=%s] %s", requestID, format)
}

// DebugfCtx logs a debug message with request ID from context
func DebugfCtx(ctx context.Context, format string, v ...interface{}) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	if LogLevel <= Debug {
		log.Printf("[DEBUG] "+formatWithRequestID(ctx, format), v...)
	}
}

// InfofCtx logs an info message with request ID from context
func InfofCtx(ctx context.Context, format string, v ...interface{}) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	if LogLevel <= Info {
		log.Printf("[INFO] "+formatWithRequestID(ctx, format), v...)
	}
}

// WarningfCtx logs a warning message with request ID from context
func WarningfCtx(ctx context.Context, format string, v ...interface{}) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	if LogLevel <= Warning {
		log.Printf("[WARN] "+formatWithRequestID(ctx, format), v...)
	}
}

// ErrorfCtx logs an error message with request ID from context
func ErrorfCtx(ctx context.Context, format string, v ...interface{}) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	if LogLevel <= Error {
		log.Printf("[ERROR] "+formatWithRequestID(ctx, format), v...)
	}
}

// CriticalfCtx logs a critical message with request ID from context
func CriticalfCtx(ctx context.Context, format string, v ...interface{}) {
	logLevelMutex.Lock()
	defer logLevelMutex.Unlock()
	if LogLevel <= Critical {
		log.Printf("[CRITICAL] "+formatWithRequestID(ctx, format), v...)
	}
}

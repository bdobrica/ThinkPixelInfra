package logger

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"snap_store/config"
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
	localEnv := fmt.Sprintf("%v", config.GetEnv("LOCAL", "false"))
	lowerEnv := strings.ToLower(localEnv)
	if lowerEnv == "true" || lowerEnv == "1" || lowerEnv == "yes" || lowerEnv == "on" {
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

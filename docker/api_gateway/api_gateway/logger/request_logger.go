package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"api_gateway/config"
)

// RequestLog defines the JSON structure for a log entry.
type RequestLog struct {
	Timestamp  time.Time           `json:"timestamp"`
	Method     string              `json:"method"`
	URL        string              `json:"url"`
	Headers    map[string][]string `json:"headers"`
	RemoteAddr string              `json:"remote_addr"`
}

// RequestLogger implements asynchronous, buffered logging with rotation and periodic flush.
type RequestLogger struct {
	filePath      string        // full path to the active log file (e.g. /logs/requests.jsonl)
	maxSize       int64         // maximum size in bytes before rotation
	maxFiles      int           // maximum number of rotated files to keep
	flushInterval time.Duration // flush the buffer every flushInterval if not empty

	mu          sync.Mutex
	file        *os.File
	writer      *bufio.Writer
	currentSize int64

	logCh  chan RequestLog
	doneCh chan struct{}
	wg     sync.WaitGroup
}

// NewLogger creates a new RequestLogger.
// bufferSize determines how many log entries can be queued before writes block.
// flushInterval defines how often the logger should flush its buffer.
func NewLogger(filePath string, maxSize int64, maxFiles, bufferSize int, flushInterval time.Duration) (*RequestLogger, error) {
	logger := &RequestLogger{
		filePath:      filePath,
		maxSize:       maxSize,
		maxFiles:      maxFiles,
		flushInterval: flushInterval,
		logCh:         make(chan RequestLog, bufferSize),
		doneCh:        make(chan struct{}),
	}

	if err := logger.openFile(); err != nil {
		return nil, err
	}

	logger.wg.Add(1)
	go logger.run()

	return logger, nil
}

// openFile opens (or creates) the active log file and prepares the buffered writer.
func (logger *RequestLogger) openFile() error {
	file, err := os.OpenFile(logger.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	fi, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	logger.currentSize = fi.Size()
	logger.file = file
	logger.writer = bufio.NewWriter(file)
	return nil
}

// rotateIfNeeded checks if adding n bytes would exceed the max file size,
// and if so rotates the file.
func (logger *RequestLogger) rotateIfNeeded(n int) error {
	logger.mu.Lock()
	defer logger.mu.Unlock()

	// if we haven't reached the max size yet, nothing to do
	if logger.currentSize+int64(n) < logger.maxSize {
		return nil
	}

	// flush and close current file
	if err := logger.writer.Flush(); err != nil {
		return err
	}
	if err := logger.file.Close(); err != nil {
		return err
	}

	// Rename the current file by appending a timestamp.
	timestamp := time.Now().Format("20060102150405")
	ext := filepath.Ext(logger.filePath)                       // e.g. ".jsonl"
	base := logger.filePath[0 : len(logger.filePath)-len(ext)] // remove extension
	newName := fmt.Sprintf("%s-%s%s", base, timestamp, ext)
	if err := os.Rename(logger.filePath, newName); err != nil {
		return err
	}

	// Clean up older rotated files if needed.
	if err := logger.cleanupOldFiles(); err != nil {
		return err
	}

	// Open a new file
	if err := logger.openFile(); err != nil {
		return err
	}
	return nil
}

// cleanupOldFiles removes the oldest rotated files if more than maxFiles exist.
func (logger *RequestLogger) cleanupOldFiles() error {
	dir := filepath.Dir(logger.filePath)
	ext := filepath.Ext(logger.filePath)
	base := filepath.Base(logger.filePath)
	base = base[0 : len(base)-len(ext)]

	// Look for rotated files that match the pattern e.g. "requests-<timestamp>.jsonl"
	pattern := fmt.Sprintf("%s-%s%s", base, "*", ext)
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return err
	}

	// Sort files by modification time.
	sort.Slice(matches, func(i, j int) bool {
		fi, err1 := os.Stat(matches[i])
		fj, err2 := os.Stat(matches[j])
		if err1 != nil || err2 != nil {
			return false
		}
		return fi.ModTime().Before(fj.ModTime())
	})

	// Delete oldest files if there are more than maxFiles.
	excess := len(matches) - logger.maxFiles
	for i := 0; i < excess; i++ {
		_ = os.Remove(matches[i])
	}
	return nil
}

// run is the goroutine that listens for log entries and writes them to disk.
// It also uses a ticker to periodically flush the buffer.
func (logger *RequestLogger) run() {
	defer logger.wg.Done()
	ticker := time.NewTicker(logger.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case entry := <-logger.logCh:
			logger.writeEntry(entry)
		case <-ticker.C:
			// Flush periodically.
			logger.mu.Lock()
			_ = logger.writer.Flush()
			logger.mu.Unlock()
		case <-logger.doneCh:
			// Drain remaining log entries.
			for {
				select {
				case entry := <-logger.logCh:
					logger.writeEntry(entry)
				default:
					logger.mu.Lock()
					_ = logger.writer.Flush()
					_ = logger.file.Close()
					logger.mu.Unlock()
					return
				}
			}
		}
	}
}

// writeEntry serializes a RequestLog to JSON and writes it, rotating if needed.
func (logger *RequestLogger) writeEntry(entry RequestLog) {
	data, err := json.Marshal(entry)
	if err != nil {
		// If marshaling fails, skip the log entry.
		return
	}
	line := string(data) + "\n"
	n := len(line)
	// Check and perform rotation if needed.
	if err := logger.rotateIfNeeded(n); err != nil {
		// In a real system, you might want to log this error somewhere.
	}
	logger.mu.Lock()
	_, _ = logger.writer.WriteString(line)
	logger.currentSize += int64(n)
	logger.mu.Unlock()
}

// LogRequest queues a request for logging. If the queue is full, the log entry is dropped.
func (logger *RequestLogger) LogRequest(r *http.Request) {
	headers := make(map[string][]string, len(r.Header))
	for k, v := range r.Header {
		headers[k] = v
	}
	entry := RequestLog{
		Timestamp:  time.Now(),
		Method:     r.Method,
		URL:        r.URL.String(),
		Headers:    headers,
		RemoteAddr: r.RemoteAddr,
	}
	select {
	case logger.logCh <- entry:
	default:
		// Queue full; dropping log entry.
	}
}

// Shutdown signals the logger to flush its buffer and close the file.
// Call Shutdown() from your application’s graceful shutdown handler.
func (logger *RequestLogger) Shutdown() {
	close(logger.doneCh)
	logger.wg.Wait()
}

var RLogger *RequestLogger

func init() {
	logPath := config.GetEnv("API_GATEWAY_LOG_FILE_PATH", "/var/log/requests.jsonl")
	maxSizeStr := config.GetEnv("API_GATEWAY_LOG_MAX_SIZE", "10485760")       // default 10 MB
	maxFilesStr := config.GetEnv("API_GATEWAY_LOG_MAX_FILES", "5")            // default 5
	bufferSizeStr := config.GetEnv("API_GATEWAY_LOG_BUFFER_SIZE", "100")      // default 100
	flushIntervalStr := config.GetEnv("API_GATEWAY_LOG_FLUSH_INTERVAL", "60") // default 60 seconds

	var (
		maxSize    int64
		maxFiles   int
		bufferSize int
	)

	if v, err := strconv.ParseInt(maxSizeStr, 10, 64); err == nil {
		maxSize = v
	}
	if v, err := strconv.Atoi(maxFilesStr); err == nil {
		maxFiles = v
	}
	if v, err := strconv.Atoi(bufferSizeStr); err == nil {
		bufferSize = v
	}
	flushIntervalSec := 60 // default value in seconds
	if v, err := strconv.Atoi(flushIntervalStr); err == nil {
		flushIntervalSec = v
	}

	// Create the logger.
	var err error
	RLogger, err = NewLogger(logPath, maxSize, maxFiles, bufferSize, time.Duration(flushIntervalSec)*time.Second)
	if err != nil {
		// Errorf is assumed to be a helper that logs errors.
		Errorf("Failed to create request logger: %v", err)
		// If logger creation failed, we schedule shutdown (if it was partially created).
		if RLogger != nil {
			defer RLogger.Shutdown()
		}
	}
}

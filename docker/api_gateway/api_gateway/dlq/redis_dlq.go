package dlq

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"api_gateway/config"
	"api_gateway/logger"
	"api_gateway/metrics"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// Redis key patterns
	dlqKeyPrefix        = "dlq:site:"         // dlq:site:{site_id}
	dlqCounterKeyPrefix = "dlq:counter:site:" // dlq:counter:site:{site_id}
	dlqStatsKey         = "dlq:stats"         // Global DLQ stats (sorted set by count)

	// Configuration
	defaultDLQTTL          = 24 * time.Hour             // 24 hours for DLQ messages in Redis
	defaultThreshold       = 100                        // Alert after 100 failures per site
	defaultThresholdWindow = 1 * time.Hour              // Within 1 hour window
	defaultExportInterval  = 6 * time.Hour              // Export to S3 every 6 hours
	defaultExportPath      = "/var/log/api-gateway/dlq" // Path for DLQ exports
)

var (
	// Redis client for DLQ (shared with other components or dedicated)
	dlqRedisClient     *redis.Client
	dlqRedisClientLock sync.Mutex

	// Configuration from environment
	dlqTTL            = config.GetEnvDuration("API_GATEWAY_DLQ_TTL", defaultDLQTTL)
	dlqThreshold      = config.GetEnvInt("API_GATEWAY_DLQ_THRESHOLD", defaultThreshold)
	dlqThresholdWin   = config.GetEnvDuration("API_GATEWAY_DLQ_THRESHOLD_WINDOW", defaultThresholdWindow)
	dlqExportPath     = config.GetEnv("API_GATEWAY_DLQ_EXPORT_PATH", defaultExportPath)
	dlqExportEnabled  = config.GetEnvBool("API_GATEWAY_DLQ_EXPORT_ENABLED", true)
	dlqExportInterval = config.GetEnvDuration("API_GATEWAY_DLQ_EXPORT_INTERVAL", defaultExportInterval)

	// Metrics
	dlqThresholdExceeded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "api_gateway_dlq_threshold_exceeded_total",
		Help: "Total number of times DLQ threshold was exceeded per site",
	})
)

// DLQMessage represents a failed message in the dead letter queue
type DLQMessage struct {
	Timestamp    int64  `json:"timestamp"`
	SiteID       int32  `json:"site_id"`
	DocID        int32  `json:"doc_id"`
	Subject      string `json:"subject"`
	ErrorMessage string `json:"error_message"`
	RetryCount   int32  `json:"retry_count"`
	PayloadJSON  string `json:"payload_json"`
}

// Initialize DLQ system with a dedicated Redis client
func InitializeDLQ(redisAddr string, redisPassword string) error {
	dlqRedisClientLock.Lock()
	defer dlqRedisClientLock.Unlock()

	if dlqRedisClient != nil {
		return nil // Already initialized
	}

	// Create dedicated Redis client for DLQ
	dlqRedisClient = redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     redisPassword,
		DB:           1, // Use DB 1 for DLQ to separate from main data
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := dlqRedisClient.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("failed to connect to Redis for DLQ: %w", err)
	}

	logger.Infof("DLQ Redis client initialized (TTL=%s, Threshold=%d/%s)",
		dlqTTL, dlqThreshold, dlqThresholdWin)

	// Start background export worker if enabled
	if dlqExportEnabled {
		go startExportWorker()
	}

	return nil
}

// StoreDLQMessage stores a failed message in Redis with TTL
func StoreDLQMessage(siteID int32, docID int32, subject string, errorMsg string, payloadJSON string, retryCount int32) error {
	if dlqRedisClient == nil {
		return fmt.Errorf("DLQ Redis client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create DLQ message
	msg := DLQMessage{
		Timestamp:    time.Now().Unix(),
		SiteID:       siteID,
		DocID:        docID,
		Subject:      subject,
		ErrorMessage: errorMsg,
		RetryCount:   retryCount,
		PayloadJSON:  payloadJSON,
	}

	// Serialize to JSON
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ message: %w", err)
	}

	// Store in Redis List with TTL
	dlqKey := fmt.Sprintf("%s%d", dlqKeyPrefix, siteID)

	// Use pipeline for atomic operations
	pipe := dlqRedisClient.Pipeline()
	pipe.RPush(ctx, dlqKey, msgJSON)
	pipe.Expire(ctx, dlqKey, dlqTTL)

	// Increment counter for threshold checking
	counterKey := fmt.Sprintf("%s%d", dlqCounterKeyPrefix, siteID)
	pipe.Incr(ctx, counterKey)
	pipe.Expire(ctx, counterKey, dlqThresholdWin)

	// Update global stats (sorted set by count)
	pipe.ZIncrBy(ctx, dlqStatsKey, 1, fmt.Sprintf("%d", siteID))

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to store DLQ message in Redis: %w", err)
	}

	logger.Infof("Stored DLQ message for site %d, doc %d in Redis (key=%s)", siteID, docID, dlqKey)

	// Check threshold
	go checkThreshold(siteID)

	return nil
}

// checkThreshold checks if a site has exceeded the failure threshold
func checkThreshold(siteID int32) {
	if dlqRedisClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	counterKey := fmt.Sprintf("%s%d", dlqCounterKeyPrefix, siteID)
	count, err := dlqRedisClient.Get(ctx, counterKey).Int()
	if err != nil {
		if err != redis.Nil {
			logger.Warningf("Failed to get DLQ counter for site %d: %v", siteID, err)
		}
		return
	}

	if count >= dlqThreshold {
		logger.Errorf("⚠️  DLQ THRESHOLD EXCEEDED: Site %d has %d failures in %s window (threshold: %d)",
			siteID, count, dlqThresholdWin, dlqThreshold)

		dlqThresholdExceeded.Inc()
		metrics.NATSDLQMessages.Add(float64(count))

		// Could trigger alert here (email, Slack, PagerDuty, etc.)
		// For now, just log at ERROR level
	}
}

// GetDLQMessages retrieves all DLQ messages for a site
func GetDLQMessages(siteID int32, limit int) ([]DLQMessage, error) {
	if dlqRedisClient == nil {
		return nil, fmt.Errorf("DLQ Redis client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dlqKey := fmt.Sprintf("%s%d", dlqKeyPrefix, siteID)

	// Get messages from list (most recent first if using RPUSH)
	var results []string
	var err error

	if limit > 0 {
		results, err = dlqRedisClient.LRange(ctx, dlqKey, int64(-limit), -1).Result()
	} else {
		results, err = dlqRedisClient.LRange(ctx, dlqKey, 0, -1).Result()
	}

	if err != nil {
		if err == redis.Nil {
			return []DLQMessage{}, nil
		}
		return nil, fmt.Errorf("failed to retrieve DLQ messages: %w", err)
	}

	messages := make([]DLQMessage, 0, len(results))
	for _, msgJSON := range results {
		var msg DLQMessage
		if err := json.Unmarshal([]byte(msgJSON), &msg); err != nil {
			logger.Warningf("Failed to unmarshal DLQ message: %v", err)
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetDLQStats returns DLQ statistics across all sites
func GetDLQStats() (map[string]interface{}, error) {
	if dlqRedisClient == nil {
		return nil, fmt.Errorf("DLQ Redis client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get top sites by failure count from sorted set
	topSites, err := dlqRedisClient.ZRevRangeWithScores(ctx, dlqStatsKey, 0, 9).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get DLQ stats: %w", err)
	}

	stats := map[string]interface{}{
		"ttl":               dlqTTL.String(),
		"threshold":         dlqThreshold,
		"threshold_window":  dlqThresholdWin.String(),
		"export_enabled":    dlqExportEnabled,
		"export_interval":   dlqExportInterval.String(),
		"export_path":       dlqExportPath,
		"top_failing_sites": make([]map[string]interface{}, 0),
	}

	topFailingSites := make([]map[string]interface{}, 0)
	for _, z := range topSites {
		siteID := z.Member.(string)
		topFailingSites = append(topFailingSites, map[string]interface{}{
			"site_id": siteID,
			"count":   int(z.Score),
		})
	}
	stats["top_failing_sites"] = topFailingSites

	return stats, nil
}

// DeleteDLQMessages deletes all DLQ messages for a site (useful after reinjection)
func DeleteDLQMessages(siteID int32) error {
	if dlqRedisClient == nil {
		return fmt.Errorf("DLQ Redis client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dlqKey := fmt.Sprintf("%s%d", dlqKeyPrefix, siteID)
	counterKey := fmt.Sprintf("%s%d", dlqCounterKeyPrefix, siteID)

	pipe := dlqRedisClient.Pipeline()
	pipe.Del(ctx, dlqKey)
	pipe.Del(ctx, counterKey)
	pipe.ZRem(ctx, dlqStatsKey, fmt.Sprintf("%d", siteID))

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete DLQ messages: %w", err)
	}

	logger.Infof("Deleted all DLQ messages for site %d", siteID)
	return nil
}

// exportToFile exports DLQ messages to a JSONL file for S3 archival
func exportToFile() error {
	if dlqRedisClient == nil {
		return fmt.Errorf("DLQ Redis client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get all site IDs from the stats sorted set
	siteIDs, err := dlqRedisClient.ZRange(ctx, dlqStatsKey, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			logger.Infof("No DLQ messages to export")
			return nil
		}
		return fmt.Errorf("failed to get site IDs from DLQ stats: %w", err)
	}

	if len(siteIDs) == 0 {
		logger.Infof("No DLQ messages to export")
		return nil
	}

	// Create export directory with timestamp
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	exportDir := filepath.Join(dlqExportPath, timestamp)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	totalExported := 0

	for _, siteIDStr := range siteIDs {
		siteID, err := strconv.ParseInt(siteIDStr, 10, 32)
		if err != nil {
			logger.Warningf("Invalid site ID in DLQ stats: %s", siteIDStr)
			continue
		}

		messages, err := GetDLQMessages(int32(siteID), 0)
		if err != nil {
			logger.Warningf("Failed to get DLQ messages for site %d: %v", siteID, err)
			continue
		}

		if len(messages) == 0 {
			continue
		}

		// Write to JSONL file (one JSON per line)
		filename := filepath.Join(exportDir, fmt.Sprintf("site_%d.jsonl", siteID))
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			logger.Warningf("Failed to create export file for site %d: %v", siteID, err)
			continue
		}

		encoder := json.NewEncoder(f)
		for _, msg := range messages {
			if err := encoder.Encode(msg); err != nil {
				logger.Warningf("Failed to encode message for site %d: %v", siteID, err)
			} else {
				totalExported++
			}
		}
		f.Close()

		logger.Infof("Exported %d DLQ messages for site %d to %s", len(messages), siteID, filename)
	}

	logger.Infof("DLQ export completed: %d messages exported to %s", totalExported, exportDir)
	return nil
}

// startExportWorker runs periodic exports to files for S3 archival
func startExportWorker() {
	logger.Infof("Starting DLQ export worker (interval: %s)", dlqExportInterval)

	ticker := time.NewTicker(dlqExportInterval)
	defer ticker.Stop()

	for range ticker.C {
		logger.Infof("Running DLQ export job...")
		if err := exportToFile(); err != nil {
			logger.Errorf("DLQ export failed: %v", err)
		}
	}
}

// GetAllSiteIDs returns all site IDs that have DLQ messages
func GetAllSiteIDs() ([]int32, error) {
	if dlqRedisClient == nil {
		return nil, fmt.Errorf("DLQ Redis client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	siteIDStrs, err := dlqRedisClient.ZRange(ctx, dlqStatsKey, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return []int32{}, nil
		}
		return nil, fmt.Errorf("failed to get site IDs: %w", err)
	}

	siteIDs := make([]int32, 0, len(siteIDStrs))
	for _, siteIDStr := range siteIDStrs {
		siteID, err := strconv.ParseInt(siteIDStr, 10, 32)
		if err != nil {
			logger.Warningf("Invalid site ID in DLQ stats: %s", siteIDStr)
			continue
		}
		siteIDs = append(siteIDs, int32(siteID))
	}

	// Sort for consistent ordering
	sort.Slice(siteIDs, func(i, j int) bool {
		return siteIDs[i] < siteIDs[j]
	})

	return siteIDs, nil
}

package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
)

// DLQMessage represents a failed message (matches dlq/redis_dlq.go)
type DLQMessage struct {
	Timestamp    int64  `json:"timestamp"`
	SiteID       int32  `json:"site_id"`
	DocID        int32  `json:"doc_id"`
	Subject      string `json:"subject"`
	ErrorMessage string `json:"error_message"`
	RetryCount   int32  `json:"retry_count"`
	PayloadJSON  string `json:"payload_json"`  // Human-readable JSON for debugging
	PayloadProto string `json:"payload_proto"` // Base64-encoded protobuf for reinjection
}

const (
	dlqKeyPrefix = "dlq:site:"
)

// getEnv retrieves an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves an integer environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// extractSentinelInfo extracts host, port and master name from sentinel address string
// Format: <sentinel-address>:<port>/<master-node>
func extractSentinelInfo(sentinelAddr string) (host string, port string, masterName string, err error) {
	// Split the string into host:port and masterName
	parts := strings.Split(sentinelAddr, "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid sentinel address format: %s (expected format: host:port/master)", sentinelAddr)
	}

	hostPort := parts[0]
	masterName = parts[1]

	// Split host:port into host and port
	hostPortParts := strings.Split(hostPort, ":")
	if len(hostPortParts) != 2 {
		return "", "", "", fmt.Errorf("invalid sentinel address format: %s (expected format: host:port/master)", sentinelAddr)
	}

	host = hostPortParts[0]
	port = hostPortParts[1]

	return host, port, masterName, nil
}

var (
	// Flags (with environment variable fallbacks)
	source      = flag.String("source", "redis", "Source: 'redis' or 'file'")
	redisAddr   = flag.String("redis", getEnv("API_GATEWAY_DLQ_REDIS_ADDR", "localhost:26379/mymaster"), "Redis Sentinel address (format: host:port/master)")
	redisPass   = flag.String("redis-pass", getEnv("API_GATEWAY_DLQ_REDIS_PASSWORD", ""), "Redis password")
	redisDB     = flag.Int("redis-db", getEnvInt("API_GATEWAY_DLQ_REDIS_DB", 1), "Redis database number")
	natsURL     = flag.String("nats", getEnv("API_GATEWAY_NATS_URL", "nats://nats:4222"), "NATS URL")
	natsSubject = flag.String("subject", getEnv("API_GATEWAY_DOCUMENT_QUEUE_SUBJECT", "store.jobs"), "NATS subject to publish to")
	natsNKey    = flag.String("nats-nkey", getEnv("API_GATEWAY_NATS_NKEY_PATH", "/etc/nats/nkeys/nats.nk"), "NATS NKey file path")
	natsTLSCert = flag.String("nats-tls-cert", getEnv("API_GATEWAY_NATS_TLS_CERT", "/etc/nats/tls/tls.crt"), "NATS TLS certificate path")
	natsTLSKey  = flag.String("nats-tls-key", getEnv("API_GATEWAY_NATS_TLS_KEY", "/etc/nats/tls/tls.key"), "NATS TLS key path")
	natsTLSCA   = flag.String("nats-tls-ca", getEnv("API_GATEWAY_NATS_TLS_CA", "/etc/nats/tls/ca.crt"), "NATS TLS CA certificate path")
	siteID      = flag.Int("site-id", 0, "Site ID to reinject (0 = all sites)")
	filePattern = flag.String("file", "", "File or directory pattern for file source (e.g., /var/log/api-gateway/dlq/*/site_*.jsonl)")
	dryRun      = flag.Bool("dry-run", false, "Dry run: show messages without reinjecting")
	deleteAfter = flag.Bool("delete", false, "Delete messages from Redis after successful reinjection")
	limit       = flag.Int("limit", 0, "Limit number of messages to reinject (0 = no limit)")
	verbose     = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	if *source != "redis" && *source != "file" {
		fmt.Fprintf(os.Stderr, "Invalid source: %s (must be 'redis' or 'file')\n", *source)
		os.Exit(1)
	}

	if *source == "file" && *filePattern == "" {
		fmt.Fprintf(os.Stderr, "File pattern required for file source\n")
		flag.Usage()
		os.Exit(1)
	}

	var messages []DLQMessage
	var err error

	// Load messages from source
	if *source == "redis" {
		messages, err = loadFromRedis()
	} else {
		messages, err = loadFromFiles()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load messages: %v\n", err)
		os.Exit(1)
	}

	if len(messages) == 0 {
		fmt.Println("No messages to reinject")
		return
	}

	fmt.Printf("Loaded %d messages from %s\n", len(messages), *source)

	if *dryRun {
		fmt.Println("\n=== DRY RUN MODE ===")
		for i, msg := range messages {
			fmt.Printf("[%d] Site %d, Doc %d, Subject: %s, Error: %s\n",
				i+1, msg.SiteID, msg.DocID, msg.Subject, msg.ErrorMessage)
			if *verbose {
				fmt.Printf("    Timestamp: %s, Retries: %d\n",
					time.Unix(msg.Timestamp, 0).Format(time.RFC3339), msg.RetryCount)
			}
		}
		fmt.Printf("\nTotal: %d messages (use without --dry-run to reinject)\n", len(messages))
		return
	}

	// Connect to NATS
	nc, err := connectToNATS()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to NATS: %v\n", err)
		os.Exit(1)
	}
	defer nc.Close()

	fmt.Printf("Connected to NATS at %s\n", *natsURL)

	// Reinject messages
	reinjected := 0
	failed := 0

	for i, msg := range messages {
		if *limit > 0 && reinjected >= *limit {
			fmt.Printf("Reached limit of %d messages\n", *limit)
			break
		}

		// Check if we have protobuf data
		if msg.PayloadProto == "" {
			fmt.Fprintf(os.Stderr, "Warning: Message %d (site %d, doc %d) has no protobuf data, skipping\n",
				i+1, msg.SiteID, msg.DocID)
			failed++
			continue
		}

		// Decode base64 protobuf data
		protoBytes, err := base64.StdEncoding.DecodeString(msg.PayloadProto)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to decode protobuf for message %d (site %d, doc %d): %v\n",
				i+1, msg.SiteID, msg.DocID, err)
			failed++
			continue
		}

		// Republish to NATS using decoded protobuf bytes
		err = nc.Publish(*natsSubject, protoBytes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to publish message %d (site %d, doc %d): %v\n",
				i+1, msg.SiteID, msg.DocID, err)
			failed++
			continue
		}

		reinjected++
		if *verbose {
			fmt.Printf("[%d/%d] Reinjected: Site %d, Doc %d\n",
				reinjected, len(messages), msg.SiteID, msg.DocID)
		}
	}

	fmt.Printf("\nReinjection complete: %d succeeded, %d failed\n", reinjected, failed)

	// Delete from Redis if requested
	if *deleteAfter && *source == "redis" && reinjected > 0 {
		if err := deleteFromRedis(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to delete messages from Redis: %v\n", err)
		} else {
			fmt.Printf("Deleted messages from Redis\n")
		}
	}
}

func connectToNATS() (*nats.Conn, error) {
	var opts []nats.Option

	// NKey authentication
	if *natsNKey != "" {
		if _, err := os.Stat(*natsNKey); err == nil {
			opt, err := nats.NkeyOptionFromSeed(*natsNKey)
			if err != nil {
				return nil, fmt.Errorf("failed to create NKey option: %w", err)
			}
			opts = append(opts, opt)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: NKey file not found: %s. Skipping NKey authentication.\n", *natsNKey)
		}
	}

	// TLS configuration
	if *natsTLSCert != "" && *natsTLSKey != "" {
		_, certErr := os.Stat(*natsTLSCert)
		_, keyErr := os.Stat(*natsTLSKey)
		if certErr == nil && keyErr == nil {
			opts = append(opts, nats.ClientCert(*natsTLSCert, *natsTLSKey))
		} else {
			fmt.Fprintf(os.Stderr, "Warning: TLS certificate or key file not found. Skipping TLS authentication.\n")
		}
	}
	if *natsTLSCA != "" {
		if _, err := os.Stat(*natsTLSCA); err == nil {
			opts = append(opts, nats.RootCAs(*natsTLSCA))
		} else {
			fmt.Fprintf(os.Stderr, "Warning: TLS CA file not found: %s. Using system CA certificates.\n", *natsTLSCA)
		}
	}

	// Connect to NATS server
	return nats.Connect(*natsURL, opts...)
}

func createRedisClient() (*redis.Client, error) {
	// Parse sentinel address format
	host, port, masterName, err := extractSentinelInfo(*redisAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid sentinel address format: %w", err)
	}

	sentinelAddr := fmt.Sprintf("%s:%s", host, port)

	// Create Redis Failover Client
	client := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:       masterName,
		Password:         *redisPass,
		SentinelPassword: *redisPass,
		SentinelAddrs:    []string{sentinelAddr},
		DB:               *redisDB,
		DialTimeout:      5 * time.Second,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis Sentinel at %s / %s: %w", sentinelAddr, masterName, err)
	}

	return client, nil
}

func loadFromRedis() ([]DLQMessage, error) {
	client, err := createRedisClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var messages []DLQMessage

	if *siteID > 0 {
		// Load for specific site
		msgs, err := loadSiteFromRedis(client, ctx, int32(*siteID))
		if err != nil {
			return nil, err
		}
		messages = msgs
	} else {
		// Load for all sites
		// Get all site IDs from the stats sorted set
		siteIDs, err := client.ZRange(ctx, "dlq:stats", 0, -1).Result()
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("failed to get site IDs: %w", err)
		}

		for _, siteIDStr := range siteIDs {
			sid, err := strconv.ParseInt(siteIDStr, 10, 32)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid site ID: %s\n", siteIDStr)
				continue
			}

			msgs, err := loadSiteFromRedis(client, ctx, int32(sid))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load site %d: %v\n", sid, err)
				continue
			}
			messages = append(messages, msgs...)
		}
	}

	return messages, nil
}

func loadSiteFromRedis(client *redis.Client, ctx context.Context, siteID int32) ([]DLQMessage, error) {
	dlqKey := fmt.Sprintf("%s%d", dlqKeyPrefix, siteID)

	results, err := client.LRange(ctx, dlqKey, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return []DLQMessage{}, nil
		}
		return nil, fmt.Errorf("failed to get messages for site %d: %w", siteID, err)
	}

	messages := make([]DLQMessage, 0, len(results))
	for _, msgJSON := range results {
		var msg DLQMessage
		if err := json.Unmarshal([]byte(msgJSON), &msg); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to unmarshal message: %v\n", err)
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func loadFromFiles() ([]DLQMessage, error) {
	var files []string

	// Check if pattern is a directory or file
	info, err := os.Stat(*filePattern)
	if err == nil {
		if info.IsDir() {
			// Directory: find all .jsonl files
			pattern := filepath.Join(*filePattern, "**/*.jsonl")
			files, err = filepath.Glob(pattern)
			if err != nil {
				// Try without ** glob
				pattern = filepath.Join(*filePattern, "*.jsonl")
				files, err = filepath.Glob(pattern)
				if err != nil {
					return nil, fmt.Errorf("failed to glob files: %w", err)
				}
			}
		} else {
			// Single file
			files = []string{*filePattern}
		}
	} else {
		// Treat as glob pattern
		files, err = filepath.Glob(*filePattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob files: %w", err)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files found matching pattern: %s", *filePattern)
	}

	var messages []DLQMessage

	for _, file := range files {
		// Filter by site ID if specified
		if *siteID > 0 {
			expectedName := fmt.Sprintf("site_%d.jsonl", *siteID)
			if !strings.HasSuffix(file, expectedName) {
				continue
			}
		}

		msgs, err := loadFromFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load %s: %v\n", file, err)
			continue
		}
		messages = append(messages, msgs...)

		if *verbose {
			fmt.Printf("Loaded %d messages from %s\n", len(msgs), file)
		}
	}

	return messages, nil
}

func loadFromFile(filename string) ([]DLQMessage, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var messages []DLQMessage
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg DLQMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to unmarshal line: %v\n", err)
			continue
		}
		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func deleteFromRedis() error {
	client, err := createRedisClient()
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if *siteID > 0 {
		// Delete for specific site
		dlqKey := fmt.Sprintf("%s%d", dlqKeyPrefix, *siteID)
		counterKey := fmt.Sprintf("dlq:counter:site:%d", *siteID)

		pipe := client.Pipeline()
		pipe.Del(ctx, dlqKey)
		pipe.Del(ctx, counterKey)
		pipe.ZRem(ctx, "dlq:stats", fmt.Sprintf("%d", *siteID))

		_, err := pipe.Exec(ctx)
		return err
	} else {
		// Delete for all sites
		siteIDs, err := client.ZRange(ctx, "dlq:stats", 0, -1).Result()
		if err != nil && err != redis.Nil {
			return err
		}

		for _, siteIDStr := range siteIDs {
			sid, _ := strconv.ParseInt(siteIDStr, 10, 32)
			dlqKey := fmt.Sprintf("%s%d", dlqKeyPrefix, sid)
			counterKey := fmt.Sprintf("dlq:counter:site:%d", sid)

			pipe := client.Pipeline()
			pipe.Del(ctx, dlqKey)
			pipe.Del(ctx, counterKey)
			_, _ = pipe.Exec(ctx)
		}

		// Clear stats
		return client.Del(ctx, "dlq:stats").Err()
	}
}

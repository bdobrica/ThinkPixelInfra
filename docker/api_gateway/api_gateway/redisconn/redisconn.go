package redisconn

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"api_gateway/config"
	"api_gateway/logger"

	"github.com/redis/go-redis/v9"
)

type redisClientCacheEntry struct {
	client    *redis.Client
	expiresAt time.Time
}

var (
	clientCache      sync.Map // Cache of Redis clients
	sentinelPassword = config.GetEnv("API_GATEWAY_REDIS_PASSWORD", "")
	ttlFromEnv, _    = strconv.Atoi(config.GetEnv("API_GATEWAY_REDIS_CLIENT_TTL", "3600"))
	clientCacheTTL   = time.Duration(ttlFromEnv) * time.Second
	clientCacheLock  sync.Mutex
)

// extract redis host, port and master name from redis server string
func extractIndexingNodeInfo(indexingNode string) (host string, port string, masterName string, err error) {
	// Split the string into host:port and masterName
	parts := strings.Split(indexingNode, "/")
	if len(parts) != 2 {
		logger.Errorf("Invalid IndexingNode format: %s\n", indexingNode)
		return "", "", "", fmt.Errorf("invalid IndexingNode format: %s", indexingNode)
	}

	hostPort := parts[0]
	masterName = parts[1]

	// Split host:port into host and port
	hostPortParts := strings.Split(hostPort, ":")
	if len(hostPortParts) != 2 {
		logger.Errorf("Invalid IndexingNode format: %s\n", indexingNode)
		return "", "", "", fmt.Errorf("invalid IndexingNode format: %s", indexingNode)
	}

	host = hostPortParts[0]
	port = hostPortParts[1]

	return host, port, masterName, nil
}

// initializeRedisClient initializes a new Redis client based on IndexingNode string
func initializeRedisClient(indexingNode string) (*redis.Client, error) {
	// Parse IndexingNode string
	host, port, masterName, err := extractIndexingNodeInfo(indexingNode)
	if err != nil {
		return nil, fmt.Errorf("invalid IndexingNode format: %s", indexingNode)
	}

	sentinelAddr := fmt.Sprintf("%s:%s", host, port)

	// Create Redis Failover Client
	client := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:       masterName,
		Password:         sentinelPassword,
		SentinelPassword: sentinelPassword,
		SentinelAddrs:    []string{sentinelAddr},
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

// getRedisClient retrieves or creates a Redis client for a given IndexingNode string
func getRedisClient(indexingNode string) (*redis.Client, error) {
	// Check cache for existing client
	if entry, ok := clientCache.Load(indexingNode); ok {
		cacheEntry := entry.(redisClientCacheEntry)
		if time.Now().Before(cacheEntry.expiresAt) {
			// Valid client found in cache
			return cacheEntry.client, nil
		}
		// Expired client, remove it
		clientCache.Delete(indexingNode)
	}

	// Create a new client
	clientCacheLock.Lock()
	defer clientCacheLock.Unlock()

	// Double-check to avoid race condition
	if entry, ok := clientCache.Load(indexingNode); ok {
		cacheEntry := entry.(redisClientCacheEntry)
		if time.Now().Before(cacheEntry.expiresAt) {
			return cacheEntry.client, nil
		}
		clientCache.Delete(indexingNode)
	}

	client, err := initializeRedisClient(indexingNode)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client for %s: %w", indexingNode, err)
	}

	// Cache the new client with expiration
	clientCache.Store(indexingNode, redisClientCacheEntry{
		client:    client,
		expiresAt: time.Now().Add(clientCacheTTL),
	})

	return client, nil
}

// updateOffsetsInRedis updates the list of offsets in Redis
func updateOffsetsInRedis(client *redis.Client, ctx context.Context, key string, newOffsets []int) error {
	val, err := client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	var offsets []int
	if err == redis.Nil {
		offsets = newOffsets
	} else {
		err = json.Unmarshal([]byte(val), &offsets)
		if err != nil {
			return err
		}
		offsets = append(offsets, newOffsets...)
	}

	offsetsJSON, err := json.Marshal(offsets)
	if err != nil {
		return err
	}

	err = client.Set(ctx, key, offsetsJSON, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

// removeOffsetFromRedis removes a specific offset from the list in Redis
func removeOffsetFromRedis(client *redis.Client, ctx context.Context, key string, offset int) error {
	val, err := client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	if err == redis.Nil {
		return nil // No offsets to update
	}

	var offsets []int
	if err := json.Unmarshal([]byte(val), &offsets); err != nil {
		return err
	}

	// Remove the specific offset
	for i, off := range offsets {
		if off == offset {
			offsets = append(offsets[:i], offsets[i+1:]...)
			break
		}
	}

	offsetsJSON, err := json.Marshal(offsets)
	if err != nil {
		return err
	}

	if _, err := client.Set(ctx, key, offsetsJSON, 0).Result(); err != nil {
		return err
	}

	return nil
}

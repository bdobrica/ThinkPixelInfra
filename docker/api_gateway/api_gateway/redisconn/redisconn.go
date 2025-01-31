package redisconn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"api_gateway/config"
	"api_gateway/logger"
	"api_gateway/model"

	"github.com/go-redis/redis/v8"
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

// StoreEmbeddings stores embeddings and associated metadata in Redis
func StoreEmbeddings(siteID int, indexingNode string, embeddings []model.EmbeddingResponse) (int, error) {
	ctx := context.Background()
	storedCount := 0
	prefix := fmt.Sprintf("%d:", siteID)
	indexName := fmt.Sprintf("index:%d", siteID)
	offsetsMap := make(map[string][]int)

	logger.Debugf("Storing %d embeddings with prefix %s in index %s", len(embeddings), prefix, indexName)

	client, err := getRedisClient(indexingNode)
	if err != nil {
		return storedCount, fmt.Errorf("failed to get Redis client: %w", err)
	}

	var maxDim int = 0
	for _, emb := range embeddings {
		key := fmt.Sprintf("%s%d:%d", prefix, emb.ID, emb.Offset)
		embeddingBytes, err := base64.StdEncoding.DecodeString(emb.Embedding)
		if err != nil {
			logger.Errorf("Error decoding embedding for ID %d: %v", emb.ID, err)
			return storedCount, fmt.Errorf("failed to decode embedding: %w", err)
		}

		// Check if the embedding has the maximum dimension
		if len(embeddingBytes)/4 > maxDim {
			maxDim = len(embeddingBytes) / 4
		}

		fields := map[string]interface{}{
			"id":        emb.ID,
			"text":      emb.Text,
			"offset":    emb.Offset,
			"embedding": embeddingBytes,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		logger.Debugf("Storing document with key %s", key)

		if _, err := client.HSet(ctx, key, fields).Result(); err != nil {
			logger.Errorf("Error storing document with key %s: %v", key, err)
			return storedCount, fmt.Errorf("failed to store document: %w", err)
		}
		storedCount++

		// Update offsets map
		offsetMapKey := fmt.Sprintf("~%s:%d", prefix, emb.ID)
		if _, ok := offsetsMap[offsetMapKey]; !ok {
			offsetsMap[offsetMapKey] = []int{}
		}
		offsetsMap[offsetMapKey] = append(offsetsMap[offsetMapKey], emb.Offset)
	}

	logger.Debugf("Successfully stored %d documents", storedCount)

	// Check if the FT index exists
	if _, err := client.Do(ctx, "FT.INFO", indexName).Result(); err != nil {
		logger.Debugf("FT index %s does not exist, creating...", indexName)
		if _, err := client.Do(ctx, "FT.CREATE", indexName,
			"ON", "HASH", // Indicate indexing on Redis hashes
			"PREFIX", "1", prefix, // Use prefix for document keys
			"SCHEMA",              // Define schema for the index
			"embedding", "VECTOR", // Define the vector field
			"FLAT", "6", // Use FLAT index for vector search
			"TYPE", "FLOAT32", // Type of vector elements
			"DIM", strconv.Itoa(maxDim), // Dimension of the vector
			"DISTANCE_METRIC", "COSINE", // Cosine distance for similarity
		).Result(); err != nil {
			logger.Errorf("Error creating FT index: %v", err)
			return storedCount, fmt.Errorf("failed to create FT index: %w", err)
		}
	}

	// Update offsets in Redis
	for key, offsets := range offsetsMap {
		err := updateOffsetsInRedis(client, ctx, key, offsets)
		if err != nil {
			logger.Errorf("Error updating offsets in Redis: %v", err)
			return storedCount, fmt.Errorf("failed to update offsets in Redis: %w", err)
		}
	}

	return storedCount, nil
}

// SearchEmbeddings performs ANN search for a set of embeddings and returns top results
func SearchEmbeddings(siteId int, indexingNode string, embeddings []model.EmbeddingResponse, limit int) ([]map[string]interface{}, error) {
	ctx := context.Background()
	results := []map[string]interface{}{}
	indexName := fmt.Sprintf("index:%d", siteId)
	logger.Debugf("Searching embeddings in index %s", indexName)

	client, err := getRedisClient(indexingNode)
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis client: %w", err)
	}

	for _, emb := range embeddings {
		embeddingBytes, err := base64.StdEncoding.DecodeString(emb.Embedding)
		if err != nil {
			logger.Errorf("Error decoding embedding: %v", err)
			return nil, fmt.Errorf("failed to decode embedding: %w", err)
		}

		logger.Debugf("Performing search for embedding with size %d bytes", len(embeddingBytes))

		cmd := client.Do(ctx, "FT.SEARCH", indexName,
			fmt.Sprintf("*=>[KNN %d @embedding $query_vec AS score]", limit),
			"PARAMS", "2", "query_vec", embeddingBytes,
			"SORTBY", "score",
			"DIALECT", "2")

		searchResults, err := cmd.Result()
		if err != nil {
			logger.Errorf("Error executing search query: %v", err)
			return nil, fmt.Errorf("failed to execute search query: %w", err)
		}

		resultsArray, ok := searchResults.([]interface{})
		if !ok {
			logger.Errorf("Unexpected search result format: %v", searchResults)
			return nil, fmt.Errorf("unexpected search result format")
		}

		// Process results
		for i := 1; i < len(resultsArray); i++ {
			logger.Debugf("Processing search result %d", i)
			doc, ok := resultsArray[i].([]interface{})
			if !ok {
				continue
			}

			// Convert sub-array to a map
			docMap := make(map[string]interface{})
			for j := 0; j < len(doc)-1; j += 2 { // Keys are at even indices, values are at odd indices
				key, keyOk := doc[j].(string)
				if !keyOk {
					continue
				}
				value := doc[j+1]

				// Add to map, allowing for different types of values
				docMap[key] = value
			}

			// Convert id to int
			var id int
			if rawID, ok := docMap["id"].(string); ok {
				id, err = strconv.Atoi(rawID)
				if err != nil {
					logger.Errorf("Failed to parse ID as int: %v", err)
					continue
				}
			}

			// Convert score to float64
			var score float64
			if rawScore, ok := docMap["score"].(string); ok {
				score, err = strconv.ParseFloat(rawScore, 64)
				if err != nil {
					logger.Errorf("Failed to parse score as float64: %v", err)
					continue
				}
			}
			if score < 0 {
				score = 0 - score // Negative scores are not allowed
			}
			// Convert score to a percentage
			score = 100 * (1 - score)

			// Add the processed document to results
			results = append(results, map[string]interface{}{
				"id":    id,
				"text":  docMap["text"],
				"score": score,
			})
		}

	}

	// Sort results by score and return top N
	sort.Slice(results, func(i, j int) bool {
		return results[i]["score"].(float64) > results[j]["score"].(float64)
	})

	if len(results) > limit {
		results = results[:limit]
	}

	logger.Debugf("Search completed. Returning %d results", len(results))

	return results, nil
}

// RemoveAllEmbeddings removes all embeddings for a given siteId and ID
func RemoveAllEmbeddings(siteID int, indexingNode, id string) error {
	ctx := context.Background()
	prefix := fmt.Sprintf("%d:", siteID)
	client, err := getRedisClient(indexingNode)
	if err != nil {
		return fmt.Errorf("failed to get Redis client: %w", err)
	}

	// Retrieve the list of offsets
	offsetKey := fmt.Sprintf("~%s%s", prefix, id)
	val, err := client.Get(ctx, offsetKey).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get offsets: %w", err)
	}

	if err == redis.Nil {
		return nil // No offsets to delete
	}

	var offsets []int
	if err := json.Unmarshal([]byte(val), &offsets); err != nil {
		return fmt.Errorf("failed to unmarshal offsets: %w", err)
	}

	// Delete all keys corresponding to the offsets
	for _, offset := range offsets {
		key := fmt.Sprintf("%s%s:%d", prefix, id, offset)
		if _, err := client.Del(ctx, key).Result(); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", key, err)
		}
	}

	// Delete the offset record
	if _, err := client.Del(ctx, offsetKey).Result(); err != nil {
		return fmt.Errorf("failed to delete offset record: %w", err)
	}

	return nil
}

// RemoveEmbeddingByOffset removes a specific embedding by siteId, ID, and offset
func RemoveEmbeddingByOffset(siteID int, indexingNode, id string, offset int) error {
	ctx := context.Background()
	prefix := fmt.Sprintf("%d:", siteID)
	client, err := getRedisClient(indexingNode)
	if err != nil {
		return fmt.Errorf("failed to get Redis client: %w", err)
	}

	// Delete the specific embedding record
	key := fmt.Sprintf("%s%s:%d", prefix, id, offset)
	if _, err := client.Del(ctx, key).Result(); err != nil {
		return fmt.Errorf("failed to delete embedding record: %w", err)
	}

	// Update the list of offsets
	offsetKey := fmt.Sprintf("~%s%s", prefix, id)
	err = removeOffsetFromRedis(client, ctx, offsetKey, offset)
	if err != nil {
		return fmt.Errorf("failed to update offset record: %w", err)
	}

	return nil
}

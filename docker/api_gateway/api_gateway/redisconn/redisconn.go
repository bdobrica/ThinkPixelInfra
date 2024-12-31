package redisconn

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"sync"
	"time"
	"strconv"

	"github.com/go-redis/redis/v8"
	"api_gateway/model"
	"api_gateway/config"
	"api_gateway/logger"
)

type redisClientCacheEntry struct {
	client     *redis.Client
	expiresAt  time.Time
}

var (
	clientCache     sync.Map // Cache of Redis clients
	defaultTTL      = time.Hour
	ttlFromEnv, _   = strconv.Atoi(config.GetEnv("API_GATEWAY_REDIS_CLIENT_TTL", "3600"))
	clientCacheTTL  = time.Duration(ttlFromEnv) * time.Second
	clientCacheLock sync.Mutex
)

// initializeRedisClient initializes a new Redis client based on RedisServer string
func initializeRedisClient(redisServer string) (*redis.Client, error) {
	// Parse RedisServer string
	var host, port, masterName string
	_, err := fmt.Sscanf(redisServer, "%s:%s/%s", &host, &port, &masterName)
	if err != nil {
		return nil, fmt.Errorf("invalid RedisServer format: %s", redisServer)
	}

	sentinelAddr := fmt.Sprintf("%s:%s", host, port)

	// Create Redis Failover Client
	client := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    masterName,
		SentinelAddrs: []string{sentinelAddr},
		DialTimeout:   5 * time.Second,
		ReadTimeout:   5 * time.Second,
		WriteTimeout:  5 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis Sentinel at %s: %w", sentinelAddr, err)
	}

	return client, nil
}

// getRedisClient retrieves or creates a Redis client for a given RedisServer string
func getRedisClient(redisServer string) (*redis.Client, error) {
	// Check cache for existing client
	if entry, ok := clientCache.Load(redisServer); ok {
		cacheEntry := entry.(redisClientCacheEntry)
		if time.Now().Before(cacheEntry.expiresAt) {
			// Valid client found in cache
			return cacheEntry.client, nil
		}
		// Expired client, remove it
		clientCache.Delete(redisServer)
	}

	// Create a new client
	clientCacheLock.Lock()
	defer clientCacheLock.Unlock()

	// Double-check to avoid race condition
	if entry, ok := clientCache.Load(redisServer); ok {
		cacheEntry := entry.(redisClientCacheEntry)
		if time.Now().Before(cacheEntry.expiresAt) {
			return cacheEntry.client, nil
		}
		clientCache.Delete(redisServer)
	}

	client, err := initializeRedisClient(redisServer)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client for %s: %w", redisServer, err)
	}

	// Cache the new client with expiration
	clientCache.Store(redisServer, redisClientCacheEntry{
		client:    client,
		expiresAt: time.Now().Add(clientCacheTTL),
	})

	return client, nil
}

// StoreEmbeddings stores embeddings and associated metadata in Redis
func StoreEmbeddings(siteID int, redisServer string, embeddings []model.EmbeddingResponse) (int, error) {
	ctx := context.Background()
	storedCount := 0
	prefix := fmt.Sprintf("%d:", siteID)
	indexName := fmt.Sprintf("index:%d", siteID)
	logger.Debugf("Storing %d embeddings with prefix %s in index %s", len(embeddings), prefix, indexName)
	
	client, err := getRedisClient(redisServer)
	if err != nil {
		return storedCount, fmt.Errorf("failed to get Redis client: %w", err)
	}

	// Check if the FT index exists
	if _, err := client.Do(ctx, "FT.INFO", indexName).Result(); err != nil {
		logger.Debugf("FT index %s does not exist, creating...", indexName)
		if _, err := client.Do(ctx, "FT.CREATE", indexName,
			"ON", "HASH",           // Indicate indexing on Redis hashes
			"PREFIX", "1", prefix,  // Use prefix for document keys
			"SCHEMA",               // Define schema for the index
			"embedding", "VECTOR",  // Define the vector field
			"FLAT", "6",            // Use FLAT index for vector search
			"TYPE", "FLOAT32",      // Type of vector elements
			"DIM", "768",           // Dimension of the vector
			"DISTANCE_METRIC", "COSINE", // Cosine distance for similarity
		).Result(); err != nil {
			logger.Errorf("Error creating FT index: %v", err)
			return storedCount, fmt.Errorf("failed to create FT index: %w", err)
		}
	}

	for _, emb := range embeddings {
		key := fmt.Sprintf("%s%d:%d", prefix, emb.ID, emb.Offset)
		embeddingBytes, err := base64.StdEncoding.DecodeString(emb.Embedding)
		if err != nil {
			logger.Errorf("Error decoding embedding for ID %d: %v", emb.ID, err)
			return storedCount, fmt.Errorf("failed to decode embedding: %w", err)
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
	}

	logger.Debugf("Successfully stored %d documents", storedCount)

	return storedCount, nil
}

// SearchEmbeddings performs ANN search for a set of embeddings and returns top results
func SearchEmbeddings(siteId int, redisServer string, embeddings []model.EmbeddingResponse, limit int) ([]map[string]interface{}, error) {
	ctx := context.Background()
	results := []map[string]interface{}{}
	indexName := fmt.Sprintf("index:%d", siteId)
	logger.Debugf("Searching embeddings in index %s", indexName)
	
	client, err := getRedisClient(redisServer)
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

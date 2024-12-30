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

var (
	redisClient *redis.Client
	redisMaster string
	redisMu     sync.Mutex
)

// initializeRedisClient connects to the Redis Sentinel service to determine the current master
func initializeRedisClient() error {
    logger.Debugf("Initializing Redis client via Sentinel")

    // Sentinel address and options
    sentinelAddr := config.GetEnv("REDIS_SENTINEL_ADDR", "redis-sentinel:26379")
    sentinelClient := redis.NewFailoverClient(&redis.FailoverOptions{
        MasterName:    "mymaster",
        SentinelAddrs: []string{sentinelAddr},
        DialTimeout:   5 * time.Second,
        ReadTimeout:   5 * time.Second,
        WriteTimeout:  5 * time.Second,
    })

	logger.Debugf("Connecting to Redis Sentinel at %s", sentinelAddr)

    // Test connection to Sentinel
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if _, err := sentinelClient.Ping(ctx).Result(); err != nil {
        logger.Errorf("Failed to connect to Redis Sentinel: %v", err)
        return fmt.Errorf("failed to connect to Redis Sentinel: %w", err)
    }

    // Update the redisClient to point to the master
    redisClient = sentinelClient
    redisMaster = sentinelAddr

	logger.Debugf("Redis client initialized with master: %s", redisMaster)

    return nil
}

// getRedisClient ensures the Redis client is valid or reconnects if necessary
func getRedisClient() (*redis.Client, error) {
    redisMu.Lock()
    defer redisMu.Unlock()

    if redisClient == nil {
		logger.Debugf("Redis client not initialized, initializing...")
        // Call initializeRedisClient without acquiring the lock again
        if err := initializeRedisClient(); err != nil {
            return nil, err
        }
    } else {
        if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
            logger.Errorf("Error pinging Redis master: %v", err)
			logger.Debugf("Reinitializing Redis client...")
            // Call initializeRedisClient without acquiring the lock again
            if err := initializeRedisClient(); err != nil {
                return nil, fmt.Errorf("failed to reconnect to Redis master: %w", err)
            }
        }
    }

	logger.Debugf("Redis client is healthy")

    return redisClient, nil
}

// StoreEmbeddings stores embeddings and associated metadata in Redis
func StoreEmbeddings(embeddings []model.EmbeddingResponse) (int, error) {
	ctx := context.Background()
	storedCount := 0
	prefix := "document:"

	client, err := getRedisClient()
	if err != nil {
		return storedCount, fmt.Errorf("failed to get Redis client: %w", err)
	}

	// Check if the FT index exists
	if _, err := client.Do(ctx, "FT.INFO", "index_name").Result(); err != nil {
		logger.Debugf("FT index does not exist, creating...")
		if _, err := client.Do(ctx, "FT.CREATE", "index_name",
			"ON", "HASH",           // Indicate indexing on Redis hashes
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
func SearchEmbeddings(embeddings []model.EmbeddingResponse, limit int) ([]map[string]interface{}, error) {
	ctx := context.Background()
	results := []map[string]interface{}{}

	client, err := getRedisClient()
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

		cmd := client.Do(ctx, "FT.SEARCH", "index_name",
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

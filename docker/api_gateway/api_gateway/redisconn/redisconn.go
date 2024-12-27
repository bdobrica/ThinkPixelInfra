package redisconn

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"time"

	"github.com/go-redis/redis/v8"
	"api_gateway/model"
	"api_gateway/config"
)

var redisClient = redis.NewClient(&redis.Options{
	Addr: config.GetEnv("API_GATEWAY_REDIS_ADDR", "localhost:6379"),
})

// StoreEmbeddings stores embeddings and associated metadata in Redis
func StoreEmbeddings(embeddings []model.EmbeddingResponse) (int, error) {
	ctx := context.Background()
	storedCount := 0
	prefix := "document:"

	for _, emb := range embeddings {
		key := fmt.Sprintf("%s%d:%d", prefix, emb.ID, emb.Offset)
		embeddingBytes, err := base64.StdEncoding.DecodeString(emb.Embedding)
		if err != nil {
			return storedCount, err
		}

		fields := map[string]interface{}{
			"id":        emb.ID,
			"text":      emb.Text,
			"offset":    emb.Offset,
			"embedding": embeddingBytes,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		_, err = redisClient.HSet(ctx, key, fields).Result()
		if err != nil {
			return storedCount, err
		}
		storedCount++
	}

	return storedCount, nil
}

// SearchEmbeddings performs ANN search for a set of embeddings and returns top results
func SearchEmbeddings(embeddings []model.EmbeddingResponse, limit int) ([]map[string]interface{}, error) {
	ctx := context.Background()
	results := []map[string]interface{}{}

	for _, emb := range embeddings {
		embeddingBytes, err := base64.StdEncoding.DecodeString(emb.Embedding)
		if err != nil {
			return nil, err
		}

		query := redis.NewSearchQuery().WithVectorQuery("embedding", embeddingBytes, limit)
		result, err := redisClient.Do(ctx, query).Result()
		if err != nil {
			return nil, err
		}

		for _, doc := range result.Documents {
			results = append(results, map[string]interface{}{
				"id":    doc.ID,
				"text":  doc.Fields["text"],
				"score": doc.Fields["score"],
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

	return results, nil
}

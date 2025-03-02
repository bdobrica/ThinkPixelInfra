package redisconn

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

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

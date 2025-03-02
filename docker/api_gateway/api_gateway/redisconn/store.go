package redisconn

import (
	"api_gateway/logger"
	"api_gateway/model"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// StoreEmbeddings stores embeddings and associated metadata in Redis using a JSON schema
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

		embedding := Embedding{
			ID:        emb.ID,
			Text:      emb.Text,
			Offset:    emb.Offset,
			Embedding: embeddingBytes,
			Timestamp: time.Now(),
		}

		embeddingJSON, err := json.Marshal(embedding)
		if err != nil {
			logger.Errorf("Error marshaling embedding for ID %d: %v", emb.ID, err)
			return storedCount, fmt.Errorf("failed to marshal embedding: %w", err)
		}

		logger.Debugf("Storing embedding with key %s", key)

		if _, err := client.JSONSet(ctx, key, ".", embeddingJSON).Result(); err != nil {
			logger.Errorf("Error storing embedding with key %s: %v", key, err)
			return storedCount, fmt.Errorf("failed to store embedding: %w", err)
		}
		storedCount++

		// Update offsets map
		offsetMapKey := fmt.Sprintf("~%s:%d", prefix, emb.ID)
		if _, ok := offsetsMap[offsetMapKey]; !ok {
			offsetsMap[offsetMapKey] = []int{}
		}
		offsetsMap[offsetMapKey] = append(offsetsMap[offsetMapKey], emb.Offset)
	}

	logger.Debugf("Successfully stored %d embeddings", storedCount)

	// Check if the FT index exists
	if _, err := client.Do(ctx, "FT.INFO", indexName).Result(); err != nil {
		logger.Debugf("FT index %s does not exist, creating...", indexName)
		if _, err := client.Do(ctx, "FT.CREATE", indexName,
			"ON", "JSON", // Indicate indexing on Redis JSON
			"PREFIX", "1", prefix, // Use prefix for document keys
			"SCHEMA",                                   // Define schema for the index
			"$.embedding", "AS", "embedding", "VECTOR", // Define the vector field
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

// StoreDocument stores a full document (with metadata and chunks offsets)
// in Redis under the "~<site_id>:<document_id>" key. It also ensures that
// a BM25 index exists for full document search.
func StoreDocument(siteID int, indexingNode string, documentID int, fullText string, title string, cat []string, tag []string, chunkOffsets []int) error {
	ctx := context.Background()
	client, err := getRedisClient(indexingNode)
	if err != nil {
		return fmt.Errorf("failed to get Redis client: %w", err)
	}

	// Build the Redis key for full documents using the tilde prefix.
	key := fmt.Sprintf("~%d:%d", siteID, documentID)

	// Create the document using the defined type.
	document := Document{
		ID:        documentID,
		Text:      fullText,
		Title:     title,
		Cat:       cat,
		Tag:       tag,
		Chunks:    chunkOffsets,
		Timestamp: time.Now(),
	}

	documentJSON, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal full document: %w", err)
	}

	// Store the full document as a JSON document in Redis.
	if _, err := client.JSONSet(ctx, key, ".", documentJSON).Result(); err != nil {
		return fmt.Errorf("failed to store full document: %w", err)
	}
	logger.Debugf("Stored full document with key %s", key)

	// Ensure that the BM25 index for full documents exists.
	bm25Index := fmt.Sprintf("bm25:%d", siteID)
	// Check if the BM25 index exists.
	if _, err := client.Do(ctx, "FT.INFO", bm25Index).Result(); err != nil {
		logger.Debugf("BM25 index %s does not exist, creating...", bm25Index)
		// We use the same prefix pattern as for full documents.
		prefix := fmt.Sprintf("~%d:", siteID)
		// Create the BM25 index on JSON documents with the specified schema.
		// We index the "text" and "title" fields as TEXT, and "cat" and "tag"
		// as TAG fields (which are well suited for filtering exact matches).
		_, err := client.Do(ctx, "FT.CREATE", bm25Index,
			"ON", "JSON",
			"PREFIX", "1", prefix,
			"SCHEMA",
			"$.text", "AS", "text", "TEXT",
			"$.title", "AS", "title", "TEXT",
			"$.cat[*]", "AS", "cat", "TAG",
			"$.tag[*]", "AS", "tag", "TAG",
		).Result()
		if err != nil {
			logger.Errorf("Error creating BM25 index %s: %v", bm25Index, err)
			return fmt.Errorf("failed to create BM25 index: %w", err)
		}
		logger.Debugf("Created BM25 index %s", bm25Index)
	}

	return nil
}

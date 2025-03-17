package qdrant

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"time"

	"api_gateway/logger"
	"api_gateway/model"

	"github.com/qdrant/go-client/qdrant"
)

// bytesToFloat32Slice converts a byte slice (in little-endian order) into a slice of float32.
func bytesToFloat32Slice(b []byte) ([]float32, error) {
	if len(b)%4 != 0 {
		return nil, fmt.Errorf("invalid embedding length, must be a multiple of 4")
	}
	count := len(b) / 4
	vec := make([]float32, count)
	for i := 0; i < count; i++ {
		bits := binary.LittleEndian.Uint32(b[i*4 : (i+1)*4])
		vec[i] = math.Float32frombits(bits)
	}
	return vec, nil
}

// StoreEmbeddings converts and upserts embedding points into Qdrant.
func StoreEmbeddings(siteID int, indexingNode string, embeddings []model.EmbeddingResponse) (int, error) {
	ctx := context.Background()
	storedCount := 0

	client := defaultPool.Get()
	defer defaultPool.Put(client)

	points := make([]*qdrant.PointStruct, 0, len(embeddings))
	var dim int
	for i, emb := range embeddings {
		decoded, err := base64.StdEncoding.DecodeString(emb.Embedding)
		if err != nil {
			logger.Errorf("Error decoding embedding for ID %d: %v", emb.ID, err)
			return storedCount, fmt.Errorf("failed to decode embedding: %w", err)
		}
		vec, err := bytesToFloat32Slice(decoded)
		if err != nil {
			logger.Errorf("Error converting embedding for ID %d: %v", emb.ID, err)
			return storedCount, fmt.Errorf("failed to convert embedding: %w", err)
		}
		if i == 0 {
			dim = len(vec)
		} else if len(vec) != dim {
			return storedCount, fmt.Errorf("embedding dimension mismatch: expected %d, got %d", dim, len(vec))
		}

		// Create a composite point ID (e.g., "siteID:productID:offset").
		pointID := qdrant.NewID(fmt.Sprintf("%d:%d:%d", siteID, emb.ID, emb.Offset))
		payload := map[string]interface{}{
			"id":        fmt.Sprintf("%d", emb.ID),
			"text":      emb.Text,
			"offset":    emb.Offset,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		pt := qdrant.PointStruct{
			Id:      pointID,
			Vectors: qdrant.NewVectorsDense(vec),
			Payload: qdrant.NewValueMap(payload),
		}
		points = append(points, &pt)
		storedCount++
	}

	// Ensure the collection exists.
	if err := ensureCollectionExists(ctx, siteID, dim, client); err != nil {
		return storedCount, fmt.Errorf("failed to ensure collection exists: %w", err)
	}

	collectionName := fmt.Sprintf("site_%d", siteID)
	upsertReq := qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         points,
	}

	if _, err := client.client.Upsert(ctx, &upsertReq); err != nil {
		return storedCount, fmt.Errorf("failed to upsert points: %w", err)
	}
	logger.Debugf("Stored %d embeddings in collection %s", storedCount, collectionName)
	return storedCount, nil
}

// SearchEmbeddings performs a vector search using the official client's SearchPoints method.
func SearchEmbeddings(siteID int, indexingNode string, embeddings []model.EmbeddingResponse, limit int) ([]map[string]interface{}, error) {
	ctx := context.Background()
	results := []map[string]interface{}{}

	client := defaultPool.Get()
	defer defaultPool.Put(client)

	collectionName := fmt.Sprintf("site_%d", siteID)
	for _, emb := range embeddings {
		decoded, err := base64.StdEncoding.DecodeString(emb.Embedding)
		if err != nil {
			logger.Errorf("Error decoding embedding: %v", err)
			return nil, fmt.Errorf("failed to decode embedding: %w", err)
		}
		vec, err := bytesToFloat32Slice(decoded)
		if err != nil {
			logger.Errorf("Error converting embedding: %v", err)
			return nil, fmt.Errorf("failed to convert embedding: %w", err)
		}

		uintLimit := uint64(limit)
		searchReq := qdrant.QueryPoints{
			CollectionName: collectionName,
			Query:          qdrant.NewQueryDense(vec),
			Limit:          &uintLimit,
			WithPayload: &qdrant.WithPayloadSelector{
				SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true},
			},
			WithVectors: &qdrant.WithVectorsSelector{
				SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: false},
			},
		}
		resp, err := client.client.Query(ctx, &searchReq)
		if err != nil {
			logger.Errorf("Error executing search query: %v", err)
			return nil, fmt.Errorf("failed to execute search query: %w", err)
		}

		for _, point := range resp {
			// Convert the cosine distance to a similarity percentage.
			score := 100 * (1 - point.Score)
			itemIDValue, ok := point.Payload["id"]
			if !ok {
				continue
			}
			itemID, err := strconv.Atoi(itemIDValue.GetStringValue())
			if err != nil {
				continue
			}
			res := map[string]interface{}{
				"id":    itemID,
				"text":  point.Payload["text"],
				"score": score,
			}
			results = append(results, res)
		}
	}
	if len(results) > limit {
		results = results[:limit]
	}
	logger.Debugf("Search completed in collection %s. Returning %d results", collectionName, len(results))
	return results, nil
}

// RemoveAllEmbeddings deletes points matching a filter (by product_id).
func RemoveAllEmbeddings(siteID int, indexingNode, id string) error {
	ctx := context.Background()

	client := defaultPool.Get()
	defer defaultPool.Put(client)

	collectionName := fmt.Sprintf("site_%d", siteID)
	deleteReq := qdrant.DeletePoints{
		CollectionName: collectionName,
		Points: qdrant.NewPointsSelectorFilter(&qdrant.Filter{
			Must: []*qdrant.Condition{
				qdrant.NewMatch("id", id),
			},
		}),
	}
	res, err := client.client.Delete(ctx, &deleteReq)
	if err != nil {
		return fmt.Errorf("failed to delete points by filter: %w", err)
	}
	logger.Debugf("Removed %d embeddings for item id %s in collection %s", res.Status.Number(), id, collectionName)
	return nil
}

// RemoveEmbeddingByOffset deletes a specific point by its composite ID.
func RemoveEmbeddingByOffset(siteID int, indexingNode, id string, offset int) error {
	ctx := context.Background()

	client := defaultPool.Get()
	defer defaultPool.Put(client)

	collectionName := fmt.Sprintf("site_%d", siteID)
	deleteReq := qdrant.DeletePoints{
		CollectionName: collectionName,
		Points: qdrant.NewPointsSelectorFilter(&qdrant.Filter{
			Must: []*qdrant.Condition{
				qdrant.NewMatch("id", id),
				qdrant.NewMatchInt("offset", int64(offset)),
			},
		}),
	}
	res, err := client.client.Delete(ctx, &deleteReq)
	if err != nil {
		return fmt.Errorf("failed to delete points by filter: %w", err)
	}
	logger.Debugf("Removed %d embeddings for item id %s in collection %s", res.Status.Number(), id, collectionName)
	return nil
}

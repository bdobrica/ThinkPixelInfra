package qdrant

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"api_gateway/logger"
	"api_gateway/model"

	"github.com/qdrant/go-client/qdrant"
)

// StoreEmbeddings converts and upserts embedding points into Qdrant.
func StoreHybridEmbeddings(siteID int, indexingNode string, embeddings []model.EmbeddingResponse, fullText string) (int, error) {
	ctx := context.Background()
	collectionName := getCollectionName(siteID)
	storedCount := 0

	client := defaultPool.Get()
	defer defaultPool.Put(client)

	var (
		dim     int
		vectors []*qdrant.Vector
		itemID  int
	)
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
			itemID = emb.ID
		} else if len(vec) != dim {
			return storedCount, fmt.Errorf("embedding dimension mismatch: expected %d, got %d", dim, len(vec))
		}
		vectors = append(vectors, qdrant.NewVectorDense(vec))
		storedCount++
	}

	pointID := qdrant.NewId(fmt.Sprintf("%d:%d", siteID, itemID))
	payload := map[string]interface{}{
		"id":        fmt.Sprintf("%d", itemID),
		"text":      fullText,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	pt := qdrant.PointStruct{
		Id:      pointID,
		Vectors: qdrant.NewVectorsDense(vec),
		Payload: qdrant.NewValueMap(payload),
	}
	upsertReq := qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         [&pt],
	}

	if _, err := client.client.Upsert(ctx, &upsertReq); err != nil {
		return storedCount, fmt.Errorf("failed to upsert points: %w", err)
	}
	logger.Debugf("Stored %d embeddings in collection %s", storedCount, collectionName)
	return storedCount, nil
}

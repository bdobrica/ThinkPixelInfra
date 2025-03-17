package qdrant

import (
	"api_gateway/logger"
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

func getCollectionName(siteID int) string {
	return fmt.Sprintf("site:%d", siteID)
}

// ensureCollectionsExist checks whether both the vector and text collections exist,
// and creates them if they do not.
// - The vector collection ("site:{siteID}:vec") is created with HNSW configuration.
// - The text collection ("site:{siteID}:text") is created to support BM25 search on the "text" payload.
func ensureCollectionExists(ctx context.Context, siteID, dim int, client *QdrantClient) error {
	// Define collection names.
	collectionName := getCollectionName(siteID)

	collectionExists, err := client.client.CollectionExists(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}
	if collectionExists {
		return nil
	}

	createCollectionReq := qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: qdrant.NewVectorsConfigMap(
			map[string]*qdrant.VectorParams{
				"dense": {
					Size:     uint64(dim),
					Distance: qdrant.Distance_Cosine,
					MultivectorConfig: &qdrant.MultiVectorConfig{
						Comparator: qdrant.MultiVectorComparator_MaxSim,
					},
				},
			}),
		SparseVectorsConfig: qdrant.NewSparseVectorsConfig(
			map[string]*qdrant.SparseVectorParams{
				"sparse": {
					Index: &qdrant.SparseIndexConfig{
						OnDisk: qdrant.PtrOf(false),
					},
					Modifier: qdrant.PtrOf(qdrant.Modifier_Idf),
				},
			},
		),
	}

	err = client.client.CreateCollection(ctx, &createCollectionReq)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	logger.Debugf("Collection %s created with dimension %d", collectionName, dim)
	return nil
}

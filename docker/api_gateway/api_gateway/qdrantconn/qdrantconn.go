package qdrantconn

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"api_gateway/config"
	"api_gateway/logger"
	"api_gateway/model"

	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type qdrantClientCacheEntry struct {
	client    *qdrant.Client
	expiresAt time.Time
}

var (
	clientCache     sync.Map
	qdrantApiKey    = config.GetEnv("API_GATEWAY_QDRANT_API_KEY", "")
	clientCacheTTL  = config.GetEnvDuration("API_GATEWAY_QDRANT_CLIENT_TTL", 3600*time.Second)
	clientCacheLock sync.Mutex
)

// extract qdrant host and port from qdrant server string
func extractIndexingNodeInfo(indexingNode string) (host string, port int, err error) {
	// Split host:port into host and port
	hostPortParts := strings.Split(indexingNode, ":")
	if len(hostPortParts) != 2 {
		logger.Errorf("Invalid IndexingNode format: %s\n", indexingNode)
		return "", 0, fmt.Errorf("invalid IndexingNode format: %s", indexingNode)
	}

	host = hostPortParts[0]
	portRaw := hostPortParts[1]
	port, err = strconv.Atoi(portRaw)
	if err != nil {
		logger.Errorf("Invalid port number: %s\n", portRaw)
		return "", 0, fmt.Errorf("invalid port number: %s", portRaw)
	}

	return host, port, nil
}

func initializeQdrantClient(indexingNode string) (*qdrant.Client, error) {
	host, port, err := extractIndexingNodeInfo(indexingNode)
	if err != nil {
		return nil, fmt.Errorf("invalid IndexingNode format: %s (%w)", indexingNode, err)
	}

	cfg := &qdrant.Config{
		Host:   host,
		Port:   port,
		APIKey: qdrantApiKey,
		UseTLS: false,
	}
	client, err := qdrant.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func getQdrantClient(indexingNode string) (*qdrant.Client, error) {
	// Check cache for existing client
	if entry, ok := clientCache.Load(indexingNode); ok {
		cacheEntry := entry.(qdrantClientCacheEntry)
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
		cacheEntry := entry.(qdrantClientCacheEntry)
		if time.Now().Before(cacheEntry.expiresAt) {
			return cacheEntry.client, nil
		}
		clientCache.Delete(indexingNode)
	}

	client, err := initializeQdrantClient(indexingNode)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client for %s: %w", indexingNode, err)
	}

	// Cache the new client with expiration
	clientCache.Store(indexingNode, qdrantClientCacheEntry{
		client:    client,
		expiresAt: time.Now().Add(clientCacheTTL),
	})

	return client, nil
}

// StoreEmbeddings upserts a batch of embeddings into a per-site collection.
// If the collection doesn’t exist yet, it tries to create it.
func StoreEmbeddings(siteID int, indexingNode string, embeddings []model.EmbeddingResponse) (int, error) {
	ctx := context.Background()
	indexName := fmt.Sprintf("site_%d", siteID)

	logger.Debugf("Storing %d embeddings in index %s", len(embeddings), indexName)

	// Get Qdrant client
	client, err := getQdrantClient(indexingNode)
	if err != nil {
		return 0, fmt.Errorf("failed to get Qdrant client: %w", err)
	}

	// Try to create the collection (ignore "AlreadyExists")
	// adjust VectorsConfig and SparseVectorsConfig to your names/dims
	err = client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: indexName,
		VectorsConfig: qdrant.NewVectorsConfigMap(
			map[string]*qdrant.VectorParams{
				"dense": {Size: uint64(len(embeddings[0].DenseVector)), Distance: qdrant.Distance_Cosine},
			},
		),
		SparseVectorsConfig: qdrant.NewSparseVectorsConfig(
			map[string]*qdrant.SparseVectorParams{
				"sparse": {Modifier: qdrant.Modifier_Idf.Enum()},
			},
		),
	})
	if err != nil {
		if se, ok := status.FromError(err); !ok || se.Code() != codes.AlreadyExists {
			return 0, err
		}
	}

	// Build Qdrant points
	points := make([]*qdrant.PointStruct, len(embeddings))
	for index, embedding := range embeddings {
		payload := map[string]interface{}{
			"site_id": siteID,
			"post_id": embedding.ID,
			"offset":  embedding.Offset,
			"text":    embedding.Text,
		}
		// NamedVectors: "dense" -> DenseVector, "sparse" -> SparseVector
		vectors := qdrant.NewVectorsMap(map[string]*qdrant.Vector{
			"dense": qdrant.NewVector(embedding.DenseVector...),
			"sparse": qdrant.NewVectorSparse(
				keysFromMap(embedding.SparseVector),
				valuesFromMap(embedding.SparseVector),
			),
		})

		pointId := qdrant.NewIDNum(uint64(embedding.ID)<<32 | uint64(embedding.Offset))
		points[index] = &qdrant.PointStruct{
			Id:      pointId,
			Vectors: vectors,
			Payload: qdrant.NewValueMap(payload),
		}
	}

	_, err = client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: indexName,
		Points:         points,
	}) // :contentReference[oaicite:2]{index=2}
	if err != nil {
		return 0, err
	}
	return len(embeddings), nil
}

// SearchEmbeddings runs dense+ sparse kNN for each query chunk and
// then applies Reciprocal Rank Fusion (RRF) to fuse all lists into one.
func SearchEmbeddings(siteID int, indexingNode string, embeddings []model.EmbeddingResponse, limit int) ([]map[string]interface{}, error) {
	ctx := context.Background()
	indexName := fmt.Sprintf("site_%d", siteID)

	// Get Qdrant client
	client, err := getQdrantClient(indexingNode)
	if err != nil {
		return nil, fmt.Errorf("failed to get Qdrant client: %w", err)
	}

	// collect all ranked lists
	results := []map[string]interface{}{}
	for _, e := range embeddings {
		searchResults, err := client.Query(ctx, &qdrant.QueryPoints{
			CollectionName: indexName,
			Prefetch: []*qdrant.PrefetchQuery{
				{
					Query: qdrant.NewQuerySparse(
						keysFromMap(e.SparseVector),
						valuesFromMap(e.SparseVector),
					),
					Using: qdrant.PtrOf("sparse"),
					Limit: qdrant.PtrOf(uint64(limit)),
				},
				{
					Query: qdrant.NewQueryDense(e.DenseVector),
					Using: qdrant.PtrOf("dense"),
					Limit: qdrant.PtrOf(uint64(limit)),
				},
			},
			Query:       qdrant.NewQueryFusion(qdrant.Fusion_RRF),
			Limit:       qdrant.PtrOf(uint64(limit)),
			WithPayload: qdrant.NewWithPayloadInclude("post_id", "text"),
		})

		if err != nil {
			if se, ok := status.FromError(err); ok && se.Code() == codes.NotFound {
				logger.Errorf("Collection %s not found: %v", indexName, err)
				return nil, fmt.Errorf("collection %s not found: %w", indexName, err)
			}
			return nil, fmt.Errorf("failed to query Qdrant: %w", err)
		}

		for _, searchResult := range searchResults {
			if searchResult.Id == nil {
				continue
			}
			id := searchResult.Payload["post_id"].GetIntegerValue()
			score := searchResult.Score
			text := searchResult.Payload["text"].GetStringValue()
			results = append(results, map[string]interface{}{
				"id":    id,
				"text":  text,
				"score": score,
			})
		}
	}

	// fuse with RRF (K=60 is common) :contentReference[oaicite:4]{index=4}

	// Sort results by score and return top N
	sort.Slice(results, func(i, j int) bool {
		return results[i]["score"].(float32) > results[j]["score"].(float32)
	})

	if len(results) > limit {
		results = results[:limit]
	}

	logger.Debugf("Search completed. Returning %d results", len(results))

	return results, nil
}

// RemoveAllEmbeddings deletes every point for that postID under this site.
func RemoveAllEmbeddings(siteID int, indexingNode, id string) error {
	ctx := context.Background()
	indexName := fmt.Sprintf("site_%d", siteID)

	// Get Qdrant client
	client, err := getQdrantClient(indexingNode)
	if err != nil {
		return fmt.Errorf("failed to get Qdrant client: %w", err)
	}

	postID, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	// delete by filter :contentReference[oaicite:5]{index=5}
	_, err = client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: indexName,
		Points: qdrant.NewPointsSelectorFilter(
			&qdrant.Filter{
				Must: []*qdrant.Condition{
					qdrant.NewMatchInt("site_id", int64(siteID)),
					qdrant.NewMatchInt("post_id", int64(postID)),
				},
			},
		),
	})

	return err
}

// RemoveEmbeddingByOffset deletes just the one chunk (offset) for that post.
func RemoveEmbeddingByOffset(siteID int, indexingNode, id string, offset int) error {
	ctx := context.Background()
	indexName := fmt.Sprintf("site_%d", siteID)

	// Get Qdrant client
	client, err := getQdrantClient(indexingNode)
	if err != nil {
		return fmt.Errorf("failed to get Qdrant client: %w", err)
	}

	postID, err := strconv.Atoi(id)
	if err != nil {
		return err
	}

	// delete by filter :contentReference[oaicite:5]{index=5}
	_, err = client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: indexName,
		Points: qdrant.NewPointsSelectorFilter(
			&qdrant.Filter{
				Must: []*qdrant.Condition{
					qdrant.NewMatchInt("site_id", int64(siteID)),
					qdrant.NewMatchInt("post_id", int64(postID)),
					qdrant.NewMatchInt("offset", int64(offset)),
				},
			},
		),
	})

	return err
}

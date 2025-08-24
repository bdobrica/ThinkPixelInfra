package document_queue

import (
	"fmt"
	"sync"
	"time"

	"api_gateway/db"
	"api_gateway/logger"
	"api_gateway/model"
	"api_gateway/qdrantconn"
	"api_gateway/redisconn"
)

// siteCacheEntry holds metadata and configuration for a site's API key, used for caching site-specific model and indexing info.
type siteCacheEntry struct {
	ID               int       // Unique identifier for the site/API key
	IndexingNodeType string    // Type of indexing backend (e.g., "qdrant", "redis")
	IndexingNode     string    // Identifier or address of the indexing node
	ExpiresAt        time.Time // Expiration time for the cache entry
	Model            string    // Model name used for embeddings
	ChunkSize        int       // Chunk size for text splitting
	ChunkOverlap     int       // Overlap size for text chunks
}

// siteCache is an in-memory thread-safe cache for site API key data.
// It maps site IDs to siteCacheEntry structs and uses a RWMutex for concurrency safety.
var siteCache = struct {
	Data map[int32]siteCacheEntry // Cached site data by site ID
	Lock sync.RWMutex             // Read-write mutex for safe concurrent access
}{
	Data: make(map[int32]siteCacheEntry),
}

// getCachedSiteData returns the siteCacheEntry for a given siteId.
// If the entry is missing or expired, it queries the database and repopulates the cache.
// Removes expired entries before querying the database.
func getCachedSiteData(siteId int32) (siteCacheEntry, error) {
	// Check the in-memory cache first
	siteCache.Lock.RLock()
	entry, exists := siteCache.Data[siteId]
	siteCache.Lock.RUnlock()

	// Return the cached result if it exists and is not expired
	if exists {
		if entry.ExpiresAt.After(time.Now()) {
			return entry, nil
		}
		// Remove the expired entry from the cache
		siteCache.Lock.Lock()
		delete(siteCache.Data, siteId)
		siteCache.Lock.Unlock()
	}

	// Query the database if not in cache
	keyDetails, err := db.GetAPIKeyDetailsByID(int(siteId))
	logger.Debugf("Queried API key details for site ID %d: %+v", siteId, keyDetails)
	if err != nil {
		logger.Errorf("Failed to get API key details for site ID %d: %v", siteId, err)
		return siteCacheEntry{}, fmt.Errorf("failed to get API key details: %w", err)
	}

	// Add the valid key to the cache
	cacheEntry := siteCacheEntry{
		ID:               keyDetails.ID,
		IndexingNodeType: keyDetails.IndexingNodeType,
		IndexingNode:     keyDetails.IndexingNode,
		ExpiresAt:        keyDetails.ExpiresAt,
		Model:            keyDetails.Model,
		ChunkSize:        keyDetails.ChunkSize,
		ChunkOverlap:     keyDetails.ChunkOverlap,
	}
	siteCache.Lock.Lock()
	siteCache.Data[siteId] = cacheEntry
	siteCache.Lock.Unlock()

	return cacheEntry, nil
}

// PeriodicCacheCleanup runs every hour to remove expired cache entries from siteCache.
// This prevents memory leaks and ensures only valid API keys remain cached.
func PeriodicCacheCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		siteCache.Lock.Lock()
		for key, entry := range siteCache.Data {
			if entry.ExpiresAt.Before(time.Now()) {
				delete(siteCache.Data, key)
			}
		}
		siteCache.Lock.Unlock()
	}
}

// getStoreCallback returns a StoreCallback function based on the siteCacheEntry's IndexingNodeType.
// It selects the appropriate storage backend (qdrant or redis) for embeddings.
func getStoreCallback(siteCacheEntry siteCacheEntry) (StoreCallback, error) {
	switch siteCacheEntry.IndexingNodeType {
	case "qdrant":
		return func(embeddings []model.EmbeddingResponse) (int, error) {
			return qdrantconn.StoreEmbeddings(siteCacheEntry.ID, siteCacheEntry.IndexingNode, embeddings)
		}, nil
	case "redis":
		return func(embeddings []model.EmbeddingResponse) (int, error) {
			return redisconn.StoreEmbeddings(siteCacheEntry.ID, siteCacheEntry.IndexingNode, embeddings)
		}, nil
	default:
		return nil, fmt.Errorf("unsupported IndexingNode: %s", siteCacheEntry.IndexingNode)
	}
}

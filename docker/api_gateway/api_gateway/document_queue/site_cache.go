package document_queue

import (
	"sync"
	"time"

	"api_gateway/db"
)

// cacheEntry holds metadata and configuration for a site's API key, used for caching site-specific model and indexing info.
type cacheEntry struct {
	ID               int       // Unique identifier for the site/API key
	IndexingNodeType string    // Type of indexing backend (e.g., "qdrant", "redis")
	IndexingNode     string    // Identifier or address of the indexing node
	ExpiresAt        time.Time // Expiration time for the cache entry
	Model            string    // Model name used for embeddings
	ChunkSize        int       // Chunk size for text splitting
	ChunkOverlap     int       // Overlap size for text chunks
}

// siteCache is an in-memory thread-safe cache for site API key data.
// It maps site IDs to cacheEntry structs and uses a RWMutex for concurrency safety.
var siteCache = struct {
	Data map[int32]cacheEntry // Cached site data by site ID
	Lock sync.RWMutex         // Read-write mutex for safe concurrent access
}{
	Data: make(map[int32]cacheEntry),
}

// getCachedSiteData returns the cacheEntry for a given siteId.
// If the entry is missing or expired, it queries the database and repopulates the cache.
// Removes expired entries before querying the database.
func getCachedSiteData(siteId int32) (cacheEntry, error) {
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
	if err != nil {
		return cacheEntry{}, err
	}

	// Add the valid key to the cache
	cacheEntry := cacheEntry{
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

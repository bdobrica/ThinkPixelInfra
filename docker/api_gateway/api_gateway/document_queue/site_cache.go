package document_queue

import (
	"sync"
	"time"

	"api_gateway/db"
)

// cacheEntry represents an in-memory cache entry for an API key
type cacheEntry struct {
	ID               int
	IndexingNodeType string
	IndexingNode     string
	ExpiresAt        time.Time
	Model            string
	ChunkSize        int
	ChunkOverlap     int
}

// siteCache stores valid API keys in-memory
var siteCache = struct {
	Data map[int32]cacheEntry
	Lock sync.RWMutex
}{
	Data: make(map[int32]cacheEntry),
}

// getCachedSiteData retrieves cached API Key data by API Key ID, repopulating the cache if necessary
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

// PeriodicCacheCleanup removes expired entries from the cache
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

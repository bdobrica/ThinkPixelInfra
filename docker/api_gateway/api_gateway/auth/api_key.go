package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"api_gateway/db"
)

// CacheEntry represents an in-memory cache entry for an API key
type CacheEntry struct {
	ID               int
	IndexingNodeType string
	IndexingNode     string
	ExpiresAt        time.Time
	MaxSearchResults int
	Model            string
	ChunkSize        int
	ChunkOverlap     int
}

// KeyCache stores valid API keys in-memory
var KeyCache = struct {
	Data map[string]CacheEntry
	Lock sync.RWMutex
}{
	Data: make(map[string]CacheEntry),
}

// GetCachedAPIKeyData retrieves cached API Key data by API Key ID, repopulating the cache if necessary
func GetCachedAPIKeyData(hashedKey string) (CacheEntry, error) {
	// Check the in-memory cache first
	KeyCache.Lock.RLock()
	entry, exists := KeyCache.Data[hashedKey]
	KeyCache.Lock.RUnlock()

	// Return the cached result if it exists and is not expired
	if exists {
		if entry.ExpiresAt.After(time.Now()) {
			return entry, nil
		}
		// Remove the expired entry from the cache
		KeyCache.Lock.Lock()
		delete(KeyCache.Data, hashedKey)
		KeyCache.Lock.Unlock()
	}

	// Query the database if not in cache
	keyDetails, err := db.GetAPIKeyDetails(hashedKey)
	if err != nil {
		return CacheEntry{}, err
	}

	// Add the valid key to the cache
	cacheEntry := CacheEntry{
		ID:               keyDetails.ID,
		IndexingNodeType: keyDetails.IndexingNodeType,
		IndexingNode:     keyDetails.IndexingNode,
		ExpiresAt:        keyDetails.ExpiresAt,
		MaxSearchResults: keyDetails.MaxSearchResults,
		Model:            keyDetails.Model,
		ChunkSize:        keyDetails.ChunkSize,
		ChunkOverlap:     keyDetails.ChunkOverlap,
	}
	KeyCache.Lock.Lock()
	KeyCache.Data[hashedKey] = cacheEntry
	KeyCache.Lock.Unlock()

	return cacheEntry, nil
}

// ValidateAPIKey validates the provided API key and retrieves its hashed value
func ValidateAPIKey(apiKey string) (bool, string, error) {
	// Hash the provided API key
	hasher := sha256.New()
	hasher.Write([]byte(apiKey))
	hashedKey := hex.EncodeToString(hasher.Sum(nil))

	_, err := GetCachedAPIKeyData(hashedKey)
	if err != nil {
		return false, "", err
	}

	return true, hashedKey, nil
}

// PeriodicCacheCleanup removes expired entries from the cache
func PeriodicCacheCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		KeyCache.Lock.Lock()
		for key, entry := range KeyCache.Data {
			if entry.ExpiresAt.Before(time.Now()) {
				delete(KeyCache.Data, key)
			}
		}
		KeyCache.Lock.Unlock()
	}
}

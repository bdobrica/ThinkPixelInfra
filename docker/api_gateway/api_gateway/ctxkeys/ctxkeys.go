package ctxkeys

// ContextKey defines the type for context keys to avoid conflicts
type ContextKey string

const (
	// CacheEntryKey is used to store the cache entry in the request context
	CacheEntryKey ContextKey = "cacheEntry"

	// DocumentQueueKey is used to store the document queue in the request context
	DocumentQueueKey ContextKey = "documentQueue"

	// RequestIDKey is the context key for request IDs
	RequestIDKey ContextKey = "requestID"
)

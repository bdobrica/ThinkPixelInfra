package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// RequestIDKey is the context key for request IDs
const RequestIDKey ContextKey = "request_id"

// RequestIDHeader is the HTTP header name for request IDs
const RequestIDHeader = "X-Request-ID"

// RequestIDMiddleware generates or extracts a request ID and adds it to the context
// If the request already has an X-Request-ID header, it uses that value
// Otherwise, it generates a new UUID v4
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get request ID from header
		requestID := r.Header.Get(RequestIDHeader)

		// If not present, generate a new one
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Add request ID to response header
		w.Header().Set(RequestIDHeader, requestID)

		// Call the next handler with the new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from the context
// Returns empty string if not found
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	requestID, ok := ctx.Value(RequestIDKey).(string)
	if !ok {
		return ""
	}

	return requestID
}

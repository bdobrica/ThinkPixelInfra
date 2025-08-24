package middleware

import (
	"context"
	"net/http"

	"api_gateway/document_queue"
)

const DocumentQueueKey ContextKey = "documentQueue"

// DocumentQueueMiddleware is a middleware that injects the DocumentQueue into the request context.
func DocumentQueueMiddleware(next http.Handler, documentQueue *document_queue.DocumentQueue) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), DocumentQueueKey, documentQueue)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

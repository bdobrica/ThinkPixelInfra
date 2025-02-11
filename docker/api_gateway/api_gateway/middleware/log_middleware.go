package middleware

import (
	"net/http"

	"api_gateway/logger"
)

// LoggingMiddleware returns a middleware that logs incoming HTTP requests.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the request asynchronously using the global logger.RLogger.
		if logger.RLogger != nil {
			logger.RLogger.LogRequest(r)
		}
		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}

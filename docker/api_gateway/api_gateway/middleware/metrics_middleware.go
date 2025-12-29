package middleware

import (
	"api_gateway/metrics"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// MetricsMiddleware instruments HTTP requests with Prometheus metrics
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture status code
		wrapped := newResponseWriter(w)

		// Process the request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()

		// Get the matched route pattern (e.g., "/store", "/search")
		route := mux.CurrentRoute(r)
		var endpoint string
		if route != nil {
			if pathTemplate, err := route.GetPathTemplate(); err == nil {
				endpoint = pathTemplate
			} else {
				endpoint = r.URL.Path
			}
		} else {
			endpoint = r.URL.Path
		}

		// Record request counter
		metrics.RequestsTotal.WithLabelValues(
			endpoint,
			r.Method,
			strconv.Itoa(wrapped.statusCode),
		).Inc()

		// Record request duration
		metrics.RequestDuration.WithLabelValues(endpoint).Observe(duration)
	})
}

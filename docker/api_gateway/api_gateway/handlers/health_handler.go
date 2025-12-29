package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"api_gateway/db"
	"api_gateway/logger"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// ReadyHandler checks if the service is ready to accept requests
// Returns 200 if all dependencies are healthy, 503 otherwise
func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    make(map[string]string),
	}

	allHealthy := true

	// Check database connection
	dbConn, err := db.GetDBConnection()
	if err != nil {
		response.Checks["database"] = "unavailable: " + err.Error()
		allHealthy = false
	} else {
		// Ping with timeout
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := dbConn.PingContext(ctx); err != nil {
			response.Checks["database"] = "unhealthy: " + err.Error()
			allHealthy = false
		} else {
			response.Checks["database"] = "healthy"
		}
	}

	// Set response status
	if allHealthy {
		response.Status = "ready"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	} else {
		response.Status = "not_ready"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("Failed to encode health response: %v", err)
	}
}

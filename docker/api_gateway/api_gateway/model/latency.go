package model

import (
	"api_gateway/config"
	"sync"
)

// modelLatencyMs stores the current smoothed model latency in milliseconds.
// It is updated using exponential decay to smooth out spikes.
var (
	modelLatencyMs float64    = 0.0 // Current model latency (ms)
	modelMutex     sync.Mutex       // Mutex for safe concurrent access
)

// GetModelLatencyMs returns the current smoothed model latency in milliseconds.
// Thread-safe read using a mutex.
func GetModelLatencyMs() float64 {
	modelMutex.Lock()
	defer modelMutex.Unlock()
	return modelLatencyMs
}

// UpdateModelLatencyMs updates the smoothed model latency value using exponential decay.
// The decay factor is read from the environment variable API_GATEWAY_MODEL_LATENCY_DECAY.
// Thread-safe write using a mutex.
func UpdateModelLatencyMs(latencyMs float64) {
	modelMutex.Lock()
	defer modelMutex.Unlock()
	decay := config.GetEnvFloat("API_GATEWAY_MODEL_LATENCY_DECAY", 0.1) // Default decay factor
	modelLatencyMs = modelLatencyMs*(1.0-decay) + latencyMs*decay
}

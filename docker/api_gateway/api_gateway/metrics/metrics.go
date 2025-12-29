package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_gateway_requests_total",
			Help: "Total number of HTTP requests processed, partitioned by endpoint, method, and status code",
		},
		[]string{"endpoint", "method", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_gateway_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// NATS message metrics
	NATSMessagesPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_gateway_nats_messages_published_total",
			Help: "Total number of messages published to NATS",
		},
		[]string{"status"}, // success or failure
	)

	NATSMessagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_gateway_nats_messages_processed_total",
			Help: "Total number of messages processed from NATS subscriptions",
		},
		[]string{"status"}, // success or failure
	)

	NATSDLQMessages = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "api_gateway_nats_dlq_messages_total",
			Help: "Total number of messages sent to dead letter queue",
		},
	)

	NATSUnmarshalErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "api_gateway_nats_unmarshal_errors_total",
			Help: "Total number of NATS message unmarshal errors",
		},
	)

	// Model API metrics
	ModelLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_gateway_model_latency_seconds",
			Help:    "Model API request latency in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0},
		},
		[]string{"model", "operation"}, // operation: embeddings, inference, etc.
	)

	// Database metrics
	DBQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_gateway_db_queries_total",
			Help: "Total number of database queries executed",
		},
		[]string{"operation", "status"},
	)

	DBConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api_gateway_db_connections",
			Help: "Number of database connections by state",
		},
		[]string{"state"}, // open, idle, in_use
	)
)

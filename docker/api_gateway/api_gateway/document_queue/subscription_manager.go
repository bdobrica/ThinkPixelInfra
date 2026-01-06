package document_queue

import (
	"api_gateway/config"
	"api_gateway/dlq"
	"api_gateway/logger"
	"api_gateway/metrics"
	"api_gateway/model"
	"context"
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// StoreCallback defines a function type for storing model embeddings.
// It takes a slice of EmbeddingResponse and returns the number of stored items and an error.
type StoreCallback func([]model.EmbeddingResponse) (int, error)

// SubscriptionManager manages the document queue subscription and related state.
type SubscriptionManager struct {
	documentQueue   *DocumentQueue
	docSubscription *nats.Subscription
	dlqSubscription *nats.Subscription
	docSubMutex     sync.Mutex
}

// deadLetterHandler processes messages that have exceeded their retry limit
// This stores the failed message in Redis with TTL for ephemeral storage
func deadLetterHandler(payload *DocumentQueuePayload, subject string, protoData []byte) error {
	logger.Errorf("Dead letter message received for site %d, document ID %d after %d retries. Last error: %s",
		payload.SiteId, payload.Id, payload.RetryCount, payload.ErrorMessage)

	// Serialize payload to JSON for debugging
	payloadJSON, err := marshalPayloadToJSON(payload)
	if err != nil {
		logger.Warningf("Failed to marshal DLQ payload to JSON: %v", err)
		payloadJSON = "{}" // Use empty JSON if marshaling fails
	}

	// Store in Redis DLQ with TTL (both JSON and protobuf)
	err = dlq.StoreDLQMessage(
		payload.SiteId,
		payload.Id,
		subject,
		payload.ErrorMessage,
		payloadJSON,
		protoData, // Store original protobuf bytes for reinjection
		payload.RetryCount,
	)
	if err != nil {
		logger.Errorf("Failed to store dead letter in Redis: %v", err)
		return err
	}

	// Increment DLQ metrics
	metrics.NATSDLQMessages.Inc()

	return nil
}

// marshalPayloadToJSON converts a DocumentQueuePayload to JSON string
func marshalPayloadToJSON(payload *DocumentQueuePayload) (string, error) {
	// Create a simplified structure for JSON serialization
	data := map[string]interface{}{
		"site_id":              payload.SiteId,
		"id":                   payload.Id,
		"text":                 payload.Text,
		"extra":                payload.Extra,
		"retry_count":          payload.RetryCount,
		"max_retries":          payload.MaxRetries,
		"status":               payload.Status.String(),
		"error_message":        payload.ErrorMessage,
		"timestamp_millis":     payload.TimestampMillis,
		"last_retry_timestamp": payload.LastRetryTimestamp,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// subscriberHandler processes a queued document payload:
// 1. Prepares the text for model inference
// 2. Retrieves site-specific cache data
// 3. Calls the model API to get embeddings
// 4. Stores the embeddings using the appropriate backend
// Returns an error if processing fails, triggering retry logic
func subscriberHandler(payload *DocumentQueuePayload) error {
	logger.Debugf("Received document for site ID %d (retry %d/%d): %+v",
		payload.SiteId, payload.RetryCount, payload.MaxRetries, payload)

	// Prepare TextItem for model inference
	textItem := model.TextItem{
		Text: payload.Text,
		Metadata: model.Metadata{
			ID:    int(payload.Id),
			Extra: payload.Extra,
		},
	}

	// Retrieve CacheEntry from context
	cacheEntry, err := getCachedSiteData(payload.SiteId)
	if err != nil {
		logger.Errorf("Error retrieving cached site data for site ID %d: %v", payload.SiteId, err)
		return err
	}
	logger.Debugf("CacheEntry for site ID %d: %+v", payload.SiteId, cacheEntry)

	// Call model API to get embeddings (with metrics)
	start := time.Now()
	response, err := model.GetEmbeddings([]model.TextItem{textItem}, cacheEntry.Model, cacheEntry.ChunkSize, cacheEntry.ChunkOverlap)
	metrics.ModelLatency.WithLabelValues(cacheEntry.Model, "embeddings").Observe(time.Since(start).Seconds())

	if err != nil {
		logger.Errorf("Error retrieving embeddings: %v", err)
		return err
	}

	// Store embeddings using getStoreCallback
	storeCallback, err := getStoreCallback(cacheEntry)
	if err != nil {
		logger.Errorf("Error getting store callback: %v", err)
		return err
	}

	storedCount, err := storeCallback(response)
	if err != nil {
		logger.Errorf("Error storing documents: %v", err)
		return err
	}

	logger.Infof("Successfully stored %d documents for site ID %d", storedCount, cacheEntry.ID)
	return nil
}

// startSubscription initializes and starts the NATS subscription for document queue messages.
// It ensures only one subscriber is running at a time.

func (sm *SubscriptionManager) startSubscription() error {
	// Check if the subscriber is already running
	sm.docSubMutex.Lock()
	defer sm.docSubMutex.Unlock()

	if sm.docSubscription != nil {
		logger.Warningf("Subscriber is already running, skipping start")
		return nil
	}

	// Create DocumentQueue object
	if sm.documentQueue == nil {
		logger.Errorf("Failed to initialize document queue")
		return nil
	}

	// Start the subscriber to listen for messages on the document queue
	subject := config.GetEnv("API_GATEWAY_DOCUMENT_QUEUE_SUBJECT", "store.jobs")
	mainSub, dlqSub, err := sm.documentQueue.SubscribeWithDLQ(subscriberHandler, deadLetterHandler)
	if err != nil {
		logger.Errorf("Failed to start subscriber: %v", err)
		return err
	}
	sm.docSubscription = mainSub
	sm.dlqSubscription = dlqSub
	logger.Infof("Subscriber started, listening for messages on subject: %s and DLQ: %s.dlq", subject, subject)
	return nil
}

// stopSubscription stops and cleans up the NATS subscriptions for document queue messages.
// It ensures safe shutdown and prevents duplicate stops.

func (sm *SubscriptionManager) stopSubscription() error {
	// Check if the subscriber is running
	sm.docSubMutex.Lock()
	defer sm.docSubMutex.Unlock()

	var lastErr error

	if sm.docSubscription != nil {
		// Stop the main subscriber
		err := sm.docSubscription.Unsubscribe()
		if err != nil {
			logger.Errorf("Error stopping main subscriber: %v", err)
			lastErr = err
		}
		sm.docSubscription = nil
	}

	if sm.dlqSubscription != nil {
		// Stop the DLQ subscriber
		err := sm.dlqSubscription.Unsubscribe()
		if err != nil {
			logger.Errorf("Error stopping DLQ subscriber: %v", err)
			lastErr = err
		}
		sm.dlqSubscription = nil
	}

	if sm.docSubscription == nil && sm.dlqSubscription == nil {
		logger.Warningf("Subscriber is not running, skipping stop")
		return nil
	}

	logger.Infof("Subscribers stopped")
	return lastErr
}

func NewSubscriptionManager(dq *DocumentQueue) *SubscriptionManager {
	return &SubscriptionManager{
		documentQueue:   dq,
		docSubscription: nil,
		dlqSubscription: nil,
	}
}

// Monitor monitors model latency and manages the document queue subscription accordingly.
// - If latency is below target, ensures the subscriber is running.
// - If latency is above target, probabilistically stops or starts the subscriber based on tunnelingProbability.
// - Runs in a loop, checking latency at intervals.
// - Context cancellation will stop the monitor gracefully.
func (sm *SubscriptionManager) Monitor(ctx context.Context) {
	targetLatency := config.GetEnvDuration("API_GATEWAY_MODEL_TARGET_LATENCY", 100*time.Millisecond)         // Default target latency in ms
	tunnelingProbability := config.GetEnvPercentage("API_GATEWAY_MODEL_TUNNELING_PROBABILITY", 0.1)          // Default tunneling probability
	latencyCheckInterval := config.GetEnvDuration("API_GATEWAY_MODEL_LATENCY_CHECK_INTERVAL", 5*time.Second) // Default latency check interval

	targetLatencyMs := float64(targetLatency.Milliseconds())

	if sm.documentQueue == nil {
		logger.Errorf("DocumentQueue is not initialized, cannot start SubscriptionManager")
		return
	}

	logger.Infof("Starting SubscriptionManager with target latency %.2f ms and tunneling probability %.2f", targetLatencyMs, tunnelingProbability)

	ticker := time.NewTicker(latencyCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Infof("Monitor context cancelled, stopping subscription manager")
			if err := sm.stopSubscription(); err != nil {
				logger.Errorf("Failed to stop subscription during shutdown: %v", err)
			}
			return
		case <-ticker.C:
			latencyMs := model.GetModelLatencyMs()
			sm.docSubMutex.Lock()
			running := sm.docSubscription != nil && sm.dlqSubscription != nil
			sm.docSubMutex.Unlock()

			if latencyMs < targetLatencyMs {
				if !running {
					if err := sm.startSubscription(); err != nil {
						logger.Errorf("Failed to start subscription: %v", err)
					}
				}
			} else {
				if running {
					if rand.Float64() < 1-tunnelingProbability {
						if err := sm.stopSubscription(); err != nil {
							logger.Errorf("Failed to stop subscription: %v", err)
						}
					}
				} else {
					if rand.Float64() < tunnelingProbability {
						if err := sm.startSubscription(); err != nil {
							logger.Errorf("Failed to start subscription: %v", err)
						}
					}
				}
			}
		}
	}
}

// Stop gracefully stops the subscription manager
func (sm *SubscriptionManager) Stop() error {
	return sm.stopSubscription()
}

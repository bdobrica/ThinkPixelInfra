package document_queue

import (
	"api_gateway/config"
	"api_gateway/logger"
	"api_gateway/model"
	"api_gateway/qdrantconn"
	"api_gateway/redisconn"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// StoreCallback defines a function type for storing model embeddings.
// It takes a slice of EmbeddingResponse and returns the number of stored items and an error.
type StoreCallback func([]model.EmbeddingResponse) (int, error)

// docSubscription holds the current NATS subscription for document queue processing.
var docSubscription *nats.Subscription

// docSubMutex protects access to docSubscription for safe concurrent operations.
var docSubMutex sync.Mutex

// getStoreCallback returns a StoreCallback function based on the cacheEntry's IndexingNodeType.
// It selects the appropriate storage backend (qdrant or redis) for embeddings.
func getStoreCallback(cacheEntry cacheEntry) (StoreCallback, error) {
	switch cacheEntry.IndexingNodeType {
	case "qdrant":
		return func(embeddings []model.EmbeddingResponse) (int, error) {
			return qdrantconn.StoreEmbeddings(cacheEntry.ID, cacheEntry.IndexingNode, embeddings)
		}, nil
	case "redis":
		return func(embeddings []model.EmbeddingResponse) (int, error) {
			return redisconn.StoreEmbeddings(cacheEntry.ID, cacheEntry.IndexingNode, embeddings)
		}, nil
	default:
		return nil, fmt.Errorf("unsupported IndexingNode: %s", cacheEntry.IndexingNode)
	}
}

// subscriberHandler processes a queued document payload:
// 1. Prepares the text for model inference
// 2. Retrieves site-specific cache data
// 3. Calls the model API to get embeddings
// 4. Stores the embeddings using the appropriate backend
func subscriberHandler(payload *DocumentQueuePayload) {
	logger.Debugf("Received document for site ID %d: %+v", payload.SiteId, payload)

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
		return
	}
	logger.Debugf("CacheEntry for site ID %d: %+v", payload.SiteId, cacheEntry)

	// Call model API to get embeddings
	response, err := model.GetEmbeddings([]model.TextItem{textItem}, cacheEntry.Model, cacheEntry.ChunkSize, cacheEntry.ChunkOverlap)
	if err != nil {
		logger.Errorf("Error retrieving embeddings: %v", err)
		return
	}

	// Store embeddings using getStoreCallback
	storeCallback, err := getStoreCallback(cacheEntry)
	if err != nil {
		logger.Errorf("Error getting store callback: %v", err)
		return
	}

	storedCount, err := storeCallback(response)
	if err != nil {
		logger.Errorf("Error storing documents: %v", err)
		return
	}

	logger.Infof("Stored %d documents for site ID %d", storedCount, cacheEntry.ID)
}

// startSubscription initializes and starts the NATS subscription for document queue messages.
// It ensures only one subscriber is running at a time.
func startSubscription() error {
	// Check if the subscriber is already running
	docSubMutex.Lock()
	defer docSubMutex.Unlock()

	if docSubscription != nil {
		logger.Warningf("Subscriber is already running, skipping start")
		return nil
	}

	// Start the subscriber to listen for messages on the document queue
	subject := config.GetEnv("API_GATEWAY_DOCUMENT_QUEUE_SUBJECT", "store.jobs")
	sub, err := Subscribe(subject, subscriberHandler)
	if err != nil {
		logger.Errorf("Failed to start subscriber: %v", err)
		return err
	}
	docSubscription = sub
	logger.Infof("Subscriber started, listening for messages on subject: %s", subject)
	return nil
}

// stopSubscription stops and cleans up the NATS subscription for document queue messages.
// It ensures safe shutdown and prevents duplicate stops.
func stopSubscription() error {
	// Check if the subscriber is running
	docSubMutex.Lock()
	defer docSubMutex.Unlock()

	if docSubscription == nil {
		logger.Warningf("Subscriber is not running, skipping stop")
		return nil
	}

	// Stop the subscriber
	err := docSubscription.Unsubscribe()
	if err != nil {
		logger.Errorf("Error stopping subscriber: %v", err)
		return err
	}
	docSubscription = nil
	logger.Infof("Subscriber stopped")
	return nil
}

// SubscriptionManager monitors model latency and manages the document queue subscription accordingly.
// - If latency is below target, ensures the subscriber is running.
// - If latency is above target, probabilistically stops or starts the subscriber based on tunnelingProbability.
// - Runs in a loop, checking latency at intervals.
func SubscriptionManager() {
	targetLatencyMs := config.GetEnvFloat("API_GATEWAY_MODEL_TARGET_LATENCY_MS", 100)                        // Default target latency in ms
	tunnelingProbability := config.GetEnvPercentage("API_GATEWAY_MODEL_TUNNELING_PROBABILITY", 0.1)          // Default tunneling probability
	latencyCheckInterval := config.GetEnvDuration("API_GATEWAY_MODEL_LATENCY_CHECK_INTERVAL", 5*time.Second) // Default latency check interval

	logger.Infof("Starting SubscriptionManager with target latency %.2f ms and tunneling probability %.2f", targetLatencyMs, tunnelingProbability)

	for {
		latencyMs := model.GetModelLatencyMs()
		docSubMutex.Lock()
		running := docSubscription != nil
		docSubMutex.Unlock()

		if latencyMs < targetLatencyMs {
			if !running {
				if err := startSubscription(); err != nil {
					logger.Errorf("Failed to start subscription: %v", err)
				}
			}
		} else {
			if running {
				if rand.Float64() < 1-tunnelingProbability {
					if err := stopSubscription(); err != nil {
						logger.Errorf("Failed to stop subscription: %v", err)
					}
				}
			} else {
				if rand.Float64() < tunnelingProbability {
					if err := startSubscription(); err != nil {
						logger.Errorf("Failed to start subscription: %v", err)
					}
				}
			}
		}
		time.Sleep(latencyCheckInterval)
	}
}

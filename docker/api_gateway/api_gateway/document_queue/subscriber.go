package document_queue

import (
	"api_gateway/config"
	"api_gateway/logger"
	"api_gateway/model"
	"api_gateway/qdrantconn"
	"api_gateway/redisconn"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
)

type StoreCallback func([]model.EmbeddingResponse) (int, error)

var (
	docSubscription *nats.Subscription
	docSubMutex     sync.Mutex
)

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

// subscriberHandler processes queued documents: sends to model, stores results
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

func StartSubscription() {
	// Check if the subscriber is already running
	docSubMutex.Lock()
	defer docSubMutex.Unlock()

	if docSubscription != nil {
		logger.Warningf("Subscriber is already running, skipping start")
		return
	}

	// Start the subscriber to listen for messages on the document queue
	subject := config.GetEnv("API_GATEWAY_DOCUMENT_QUEUE_SUBJECT", "store.jobs")
	sub, err := Subscribe(subject, subscriberHandler)
	if err != nil {
		logger.Fatalf("Failed to start subscriber: %v", err)
	}
	docSubscription = sub
	logger.Infof("Subscriber started, listening for messages on subject: %s", subject)
}

func StopSubscription() {
	// Check if the subscriber is running
	docSubMutex.Lock()
	defer docSubMutex.Unlock()

	if docSubscription == nil {
		logger.Warningf("Subscriber is not running, skipping stop")
		return
	}

	// Stop the subscriber
	err := docSubscription.Unsubscribe()
	if err != nil {
		logger.Errorf("Error stopping subscriber: %v", err)
		return
	}
	docSubscription = nil
	logger.Infof("Subscriber stopped")
}

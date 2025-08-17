package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"api_gateway/auth"
	"api_gateway/config"
	"api_gateway/document_queue"
	"api_gateway/logger"
	"api_gateway/middleware"
	"api_gateway/utils"
)

type StoreRequest []struct {
	ID    int               `json:"id"`
	Text  string            `json:"text"`
	Extra map[string]string `json:"extra,omitempty"`
}

type StoreResponse struct {
	ReceivedTexts    int    `json:"received_texts"`
	PendingDocuments int    `json:"pending_documents"`
	PendingIDs       []int  `json:"pending_ids,omitempty"`
	Timestamp        string `json:"timestamp"`
}

// StoreHandler handles storing webpage data
func StoreHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Remember to soft-fail, i.e. if at least one item is stored, return success even if some fail, but log the errors and return them in the response
	// Retrieve CacheEntry from context
	cacheEntry, ok := r.Context().Value(middleware.CacheEntryKey).(auth.CacheEntry)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "CacheEntry not found in context")
		return
	}
	logger.Debugf("CacheEntry %+v", cacheEntry)

	// Parse input
	var req StoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(req) == 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "Request must contain at least one item")
		return
	}

	// Publish each document as a protobuf payload to the document queue
	var failed int
	var publishedIDs []int
	subject := config.GetEnv("API_GATEWAY_DOCUMENT_QUEUE_SUBJECT", "store.jobs")
	for _, item := range req {
		payload := &document_queue.DocumentQueuePayload{
			SiteId:          int32(cacheEntry.ID),
			Id:              int32(item.ID),
			Text:            item.Text,
			Extra:           item.Extra,
			TimestampMillis: time.Now().UnixMilli(),
		}
		err := document_queue.Publish(subject, payload)
		if err != nil {
			logger.Errorf("Failed to publish document to queue: %v", err)
			failed++
		} else {
			publishedIDs = append(publishedIDs, item.ID)
		}
	}

	logger.Infof("Queued %d documents (failed: %d) for site ID %d", len(publishedIDs), failed, cacheEntry.ID)
	_ = utils.RespondWithJSON(w, http.StatusAccepted, StoreResponse{
		ReceivedTexts:    len(req),
		PendingDocuments: len(publishedIDs),
		PendingIDs:       publishedIDs,
		Timestamp:        time.Now().Format(time.RFC3339),
	})
}

package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"api_gateway/auth"
	"api_gateway/ctxkeys"
	"api_gateway/document_queue"
	"api_gateway/logger"
	"api_gateway/utils"
)

type StoreRequest []struct {
	ID       int               `json:"id"`
	Text     string            `json:"text"`
	Language string            `json:"language,omitempty"`
	Extra    map[string]string `json:"extra,omitempty"`
}

type StoreResponse struct {
	ReceivedTexts    int          `json:"received_texts"`
	PendingDocuments int          `json:"pending_documents"`
	FailedDocuments  int          `json:"failed_documents"`
	PendingIDs       []int        `json:"pending_ids,omitempty"`
	FailedItems      []FailedItem `json:"failed_items,omitempty"`
	Timestamp        string       `json:"timestamp"`
}

type FailedItem struct {
	ID    int    `json:"id"`
	Error string `json:"error"`
}

// StoreHandler handles storing webpage data with soft-fail behavior.
// If at least one item is successfully queued, returns 202 Accepted with details.
// If all items fail, returns 500 Internal Server Error.
func StoreHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Retrieve CacheEntry from context
	cacheEntry, ok := ctx.Value(ctxkeys.CacheEntryKey).(auth.CacheEntry)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "CacheEntry not found in context")
		return
	}
	logger.DebugfCtx(ctx, "CacheEntry %+v", cacheEntry)

	// Parse input
	var req StoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.ErrorfCtx(ctx, "Failed to decode request body: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(req) == 0 {
		logger.WarningfCtx(ctx, "Empty request received")
		utils.RespondWithError(w, http.StatusBadRequest, "Request must contain at least one item")
		return
	}

	// Create DocumentQueue object
	dq := ctx.Value(ctxkeys.DocumentQueueKey).(*document_queue.DocumentQueue)
	if dq == nil {
		logger.ErrorfCtx(ctx, "DocumentQueue not found in context")
		utils.RespondWithError(w, http.StatusServiceUnavailable, "DocumentQueue not found in context")
		return
	}

	// Publish each document as a protobuf payload to the document queue
	var publishedIDs []int
	var failedItems []FailedItem

	for _, item := range req {
		// Use proper payload creation with retry logic initialization
		language := "auto"
		if item.Language != "" {
			language = item.Language
		}
		payload := document_queue.NewDocumentQueuePayload(
			int32(cacheEntry.ID),
			int32(item.ID),
			item.Text,
			language,
			item.Extra,
			0, // use default max_retries from config
		)

		err := dq.Publish(payload)
		if err != nil {
			logger.ErrorfCtx(ctx, "Failed to publish document ID %d to queue: %v", item.ID, err)
			failedItems = append(failedItems, FailedItem{
				ID:    item.ID,
				Error: err.Error(),
			})
		} else {
			publishedIDs = append(publishedIDs, item.ID)
		}
	}

	// Soft-fail: Return success if at least one item was queued
	statusCode := http.StatusAccepted
	if len(publishedIDs) == 0 {
		// All items failed
		statusCode = http.StatusInternalServerError
		logger.ErrorfCtx(ctx, "Failed to queue all %d documents for site ID %d", len(req), cacheEntry.ID)
	} else if len(failedItems) > 0 {
		// Partial success
		logger.WarningfCtx(ctx, "Queued %d/%d documents for site ID %d (%d failed)",
			len(publishedIDs), len(req), cacheEntry.ID, len(failedItems))
	} else {
		// Complete success
		logger.InfofCtx(ctx, "Queued all %d documents for site ID %d", len(publishedIDs), cacheEntry.ID)
	}

	_ = utils.RespondWithJSON(w, statusCode, StoreResponse{
		ReceivedTexts:    len(req),
		PendingDocuments: len(publishedIDs),
		FailedDocuments:  len(failedItems),
		PendingIDs:       publishedIDs,
		FailedItems:      failedItems,
		Timestamp:        time.Now().Format(time.RFC3339),
	})
}

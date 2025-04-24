package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"api_gateway/auth"
	"api_gateway/logger"
	"api_gateway/middleware"
	"api_gateway/model"
	"api_gateway/qdrantconn"
	"api_gateway/redisconn"
	"api_gateway/utils"
)

type StoreRequest []struct {
	ID    int               `json:"id"`
	Text  string            `json:"text"`
	Extra map[string]string `json:"extra,omitempty"`
}

type StoreCallback func([]model.EmbeddingResponse) (int, error)

func getStoreCallback(cacheEntry auth.CacheEntry) (StoreCallback, error) {
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

// StoreHandler handles storing webpage data
func StoreHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve CacheEntry from context
	cacheEntry, ok := r.Context().Value(middleware.CacheEntryKey).(auth.CacheEntry)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "CacheEntry not found in context")
		return
	}
	logger.Debugf("CacheEntry %+v", cacheEntry)

	// Parse input
	var input StoreRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(input) == 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "Input must contain at least one item")
		return
	}

	// Prepare TextItems for model inference
	textItems := make([]model.TextItem, len(input))
	for i, item := range input {
		textItems[i] = model.TextItem{
			Text: item.Text,
			Metadata: model.Metadata{
				ID:    item.ID,
				Extra: item.Extra,
			},
		}
	}

	// Make async request to the model API to get embeddings
	responseChan := make(chan []model.EmbeddingResponse, 1)
	errChan := make(chan error, 1)
	go func() {
		response, err := model.GetEmbeddings(textItems, cacheEntry.Model, cacheEntry.ChunkSize, cacheEntry.ChunkOverlap)
		if err != nil {
			errChan <- err
			return
		}
		responseChan <- response
	}()

	select {
	case embeddings := <-responseChan:
		// Get the store callback function based on the indexing node type
		storeCallback, err := getStoreCallback(cacheEntry)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error getting store callback: "+err.Error())
			return
		}

		// Store embeddings in the database
		storedCount, err := storeCallback(embeddings)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error storing documents: "+err.Error())
			return
		}

		// Send summary back as JSON
		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"received_texts":   len(input),
			"stored_documents": storedCount,
			"timestamp":        time.Now().Format(time.RFC3339),
		})
	case err := <-errChan:
		utils.RespondWithError(w, http.StatusInternalServerError, "Error retrieving embeddings: "+err.Error())
	case <-time.After(15 * time.Second):
		utils.RespondWithError(w, http.StatusRequestTimeout, "Request timed out")
	}
}

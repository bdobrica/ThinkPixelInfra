package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"api_gateway/auth"
	"api_gateway/logger"
	"api_gateway/middleware"
	"api_gateway/model"
	"api_gateway/redisconn"
	"api_gateway/utils"
)

type StoreRequest []struct {
    ID    int               `json:"id"`
    Text  string            `json:"text"`
    Extra map[string]string `json:"extra,omitempty"`
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
		response, err := model.GetEmbeddings(textItems)
		if err != nil {
			errChan <- err
			return
		}
		responseChan <- response
	}()

	select {
	case embeddings := <-responseChan:
		storedCount, err := redisconn.StoreEmbeddings(cacheEntry.ID, cacheEntry.RedisServer, embeddings)
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

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

type SearchRequest struct {
	Text string `json:"text"`
	ID   int    `json:"id"`
}

// SearchHandler handles embedding creation and ANN search
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve CacheEntry from context
	cacheEntry, ok := r.Context().Value(middleware.CacheEntryKey).(auth.CacheEntry)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "CacheEntry not found in context")
		return
	}
	logger.Debugf("CacheEntry %+v", cacheEntry)

	// Parse input
	var input SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if input.Text == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Text field is required")
		return
	}

	// Prepare TextItem for model inference
	textItem := model.TextItem{
		Text: input.Text,
		Metadata: model.Metadata{
			ID: input.ID,
		},
	}

	// Make async request to the model API to get the embedding
	embeddingChan := make(chan []model.EmbeddingResponse)
	errChan := make(chan error)
	go func() {
		embeddings, err := model.GetEmbeddings([]model.TextItem{textItem})
		if err != nil {
			errChan <- err
			return
		}
		embeddingChan <- embeddings
	}()

	select {
	case embeddings := <-embeddingChan:
		// Perform ANN search on Redis for all embeddings
		results, err := redisconn.SearchEmbeddings(cacheEntry.ID, cacheEntry.IndexingNode, embeddings, cacheEntry.MaxSearchResults)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error performing ANN search")
			return
		}

		// Send results back as JSON
		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"results": results})
	case err := <-errChan:
		utils.RespondWithError(w, http.StatusInternalServerError, "Error retrieving embedding: "+err.Error())
	case <-time.After(10 * time.Second):
		utils.RespondWithError(w, http.StatusRequestTimeout, "Request timed out")
	}
}

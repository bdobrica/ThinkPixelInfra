package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"api_gateway/auth"
	"api_gateway/ctxkeys"
	"api_gateway/logger"
	"api_gateway/model"
	"api_gateway/qdrantconn"
	"api_gateway/redisconn"
	"api_gateway/utils"
)

type SearchRequest struct {
	Text string `json:"text"`
}

type SearchResponse struct {
	Results []map[string]interface{} `json:"results"`
}

type SearchCallback func([]model.EmbeddingResponse) ([]map[string]interface{}, error)

func getSearchCallback(cacheEntry auth.CacheEntry) (SearchCallback, error) {
	switch cacheEntry.IndexingNodeType {
	case "qdrant":
		return func(embeddings []model.EmbeddingResponse) ([]map[string]interface{}, error) {
			return qdrantconn.SearchEmbeddings(cacheEntry.ID, cacheEntry.IndexingNode, embeddings, cacheEntry.MaxSearchResults)
		}, nil
	case "redis":
		return func(embeddings []model.EmbeddingResponse) ([]map[string]interface{}, error) {
			return redisconn.SearchEmbeddings(cacheEntry.ID, cacheEntry.IndexingNode, embeddings, cacheEntry.MaxSearchResults)
		}, nil
	default:
		return nil, fmt.Errorf("unsupported IndexingNode: %s", cacheEntry.IndexingNode)
	}
}

// SearchHandler handles embedding creation and ANN search
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve CacheEntry from context
	cacheEntry, ok := r.Context().Value(ctxkeys.CacheEntryKey).(auth.CacheEntry)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "CacheEntry not found in context")
		return
	}
	logger.Debugf("CacheEntry %+v", cacheEntry)

	// Parse input
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Text == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Text field is required")
		return
	}

	// Prepare TextItem for model inference
	textItem := model.TextItem{
		Text: req.Text,
		Metadata: model.Metadata{
			ID: 0, // ID is not used for search, but can be set if needed
		},
	}

	// Make async request to the model API to get the embedding
	embeddingChan := make(chan []model.EmbeddingResponse)
	errChan := make(chan error)
	go func() {
		embeddings, err := model.GetEmbeddings([]model.TextItem{textItem}, cacheEntry.Model, len(req.Text), 0)
		if err != nil {
			errChan <- err
			return
		}
		embeddingChan <- embeddings
	}()

	select {
	case embeddings := <-embeddingChan:
		// Get the search callback based on the indexing node type
		searchCallback, err := getSearchCallback(cacheEntry)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error getting search callback: "+err.Error())
			return
		}

		// Perform ANN search
		results, err := searchCallback(embeddings)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error performing ANN search")
			return
		}

		// Send results back as JSON
		_ = utils.RespondWithJSON(w, http.StatusOK, SearchResponse{
			Results: results,
		})
	case err := <-errChan:
		utils.RespondWithError(w, http.StatusInternalServerError, "Error retrieving embedding: "+err.Error())
	case <-time.After(10 * time.Second):
		utils.RespondWithError(w, http.StatusRequestTimeout, "Request timed out")
	}
}

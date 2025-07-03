package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"api_gateway/auth"
	"api_gateway/logger"
	"api_gateway/middleware"
	"api_gateway/qdrantconn"
	"api_gateway/redisconn"
	"api_gateway/utils"
)

type RemoveAllEmbeddingsRequest struct {
	ID string `json:"id"`
}

type RemoveAllEmbeddingsResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
}

type RemoveMultipleEmbeddingsRequest struct {
	IDs []string `json:"ids"`
}

type RemoveMultipleEmbeddingsResponse struct {
	Message string   `json:"message"`
	IDs     []string `json:"ids"`
}

type RemoveAllEmbeddingsCallback func(string) error

func getRemoveAllEmbeddingsCallback(cacheEntry auth.CacheEntry) (RemoveAllEmbeddingsCallback, error) {
	switch cacheEntry.IndexingNodeType {
	case "qdrant":
		return func(id string) error {
			return qdrantconn.RemoveAllEmbeddings(cacheEntry.ID, cacheEntry.IndexingNode, id)
		}, nil
	case "redis":
		return func(id string) error {
			return redisconn.RemoveAllEmbeddings(cacheEntry.ID, cacheEntry.IndexingNode, id)
		}, nil
	default:
		return nil, fmt.Errorf("unsupported IndexingNode: %s", cacheEntry.IndexingNode)
	}
}

func RemoveEmbeddingsHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve CacheEntry from context
	cacheEntry, ok := r.Context().Value(middleware.CacheEntryKey).(auth.CacheEntry)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "CacheEntry not found in context")
		return
	}
	logger.Debugf("CacheEntry %+v", cacheEntry)

	// Parse input
	var req RemoveAllEmbeddingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	removeAllEmbeddingsCallback, err := getRemoveAllEmbeddingsCallback(cacheEntry)
	if err != nil {
		logger.Errorf("Error getting remove all embeddings callback: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get remove all embeddings callback")
		return
	}

	err = removeAllEmbeddingsCallback(req.ID)
	if err != nil {
		logger.Errorf("Error removing all embeddings: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to remove embeddings")
		return
	}

	// Respond with success message
	_ = utils.RespondWithJSON(w, http.StatusOK, RemoveAllEmbeddingsResponse{
		Message: "Embeddings removed successfully",
		ID:      req.ID,
	})
}

func RemoveMultipleEmbeddingsHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve CacheEntry from context
	cacheEntry, ok := r.Context().Value(middleware.CacheEntryKey).(auth.CacheEntry)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "CacheEntry not found in context")
		return
	}
	logger.Debugf("CacheEntry %+v", cacheEntry)

	// Parse input
	var req RemoveMultipleEmbeddingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Get the remove all embeddings callback for the current indexing node type
	removeAllEmbeddingsCallback, err := getRemoveAllEmbeddingsCallback(cacheEntry)
	if err != nil {
		logger.Errorf("Error getting remove all embeddings callback: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get remove all embeddings callback")
		return
	}

	for _, id := range req.IDs {
		// Call the callback function to remove all embeddings for the given ID
		err := removeAllEmbeddingsCallback(id)
		if err != nil {
			logger.Errorf("Error removing all embeddings for ID %s: %v", id, err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to remove embeddings")
			return
		}
	}

	// Respond with success message
	_ = utils.RespondWithJSON(w, http.StatusOK, RemoveMultipleEmbeddingsResponse{
		Message: "Embeddings removed successfully",
		IDs:     req.IDs,
	})
}

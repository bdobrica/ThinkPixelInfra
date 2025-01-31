package handlers

import (
	"encoding/json"
	"net/http"

	"api_gateway/auth"
	"api_gateway/logger"
	"api_gateway/middleware"
	"api_gateway/redisconn"
	"api_gateway/utils"
)

type RemoveAllEmbeddingsRequest struct {
	ID string `json:"id"`
}

type RemoveMultipleEmbeddingsRequest struct {
	IDs []string `json:"ids"`
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

	err := redisconn.RemoveAllEmbeddings(cacheEntry.ID, cacheEntry.IndexingNode, req.ID)
	if err != nil {
		logger.Errorf("Error removing all embeddings: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to remove embeddings")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Embeddings removed successfully"))
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

	for _, id := range req.IDs {
		err := redisconn.RemoveAllEmbeddings(cacheEntry.ID, cacheEntry.IndexingNode, id)
		if err != nil {
			logger.Errorf("Error removing embeddings for ID %s: %v", id, err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to remove embeddings")
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Embeddings removed successfully"))
}

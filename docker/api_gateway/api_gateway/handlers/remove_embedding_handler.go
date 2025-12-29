package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"api_gateway/auth"
	"api_gateway/ctxkeys"
	"api_gateway/logger"
	"api_gateway/qdrantconn"
	"api_gateway/redisconn"
	"api_gateway/utils"
)

type RemoveEmbeddingRequest struct {
	ID     string `json:"id"`
	Offset int    `json:"offset"`
}

type RemoveEmbeddingResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
	Offset  int    `json:"offset"`
}

type RemoveMultipleEmbeddingsByOffsetRequest struct {
	ID      string `json:"id"`
	Offsets []int  `json:"offsets"`
}

type RemoveMultipleEmbeddingsByOffsetResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
	Offsets []int  `json:"offsets"`
}

type RemoveEmbeddingByOffsetCallback func(string, int) error

func getRemoveEmbeddingByOffsetCallback(cacheEntry auth.CacheEntry) (RemoveEmbeddingByOffsetCallback, error) {
	switch cacheEntry.IndexingNodeType {
	case "qdrant":
		return func(id string, offset int) error {
			return qdrantconn.RemoveEmbeddingByOffset(cacheEntry.ID, cacheEntry.IndexingNode, id, offset)
		}, nil
	case "redis":
		return func(id string, offset int) error {
			return redisconn.RemoveEmbeddingByOffset(cacheEntry.ID, cacheEntry.IndexingNode, id, offset)
		}, nil
	default:
		return nil, fmt.Errorf("unsupported IndexingNode: %s", cacheEntry.IndexingNode)
	}
}

func RemoveEmbeddingByOffsetHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve CacheEntry from context
	cacheEntry, ok := r.Context().Value(ctxkeys.CacheEntryKey).(auth.CacheEntry)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "CacheEntry not found in context")
		return
	}
	logger.Debugf("CacheEntry %+v", cacheEntry)

	// Parse input
	var req RemoveEmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	removeEmbeddingByOffsetCallback, err := getRemoveEmbeddingByOffsetCallback(cacheEntry)
	if err != nil {
		logger.Errorf("Error getting remove embedding callback: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get remove embedding callback")
		return
	}

	err = removeEmbeddingByOffsetCallback(req.ID, req.Offset)
	if err != nil {
		logger.Errorf("Error removing embedding: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to remove embedding")
		return
	}

	_ = utils.RespondWithJSON(w, http.StatusOK, RemoveEmbeddingResponse{
		Message: "Embedding removed successfully",
		ID:      req.ID,
		Offset:  req.Offset,
	})
}

func RemoveMultipleEmbeddingsByOffsetHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve CacheEntry from context
	cacheEntry, ok := r.Context().Value(ctxkeys.CacheEntryKey).(auth.CacheEntry)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "CacheEntry not found in context")
		return
	}
	logger.Debugf("CacheEntry %+v", cacheEntry)

	// Parse input
	var req RemoveMultipleEmbeddingsByOffsetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Get the callback function for removing embeddings by offset for current indexing node type
	removeEmbeddingByOffsetCallback, err := getRemoveEmbeddingByOffsetCallback(cacheEntry)
	if err != nil {
		logger.Errorf("Error getting remove embedding callback: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get remove embedding callback")
		return
	}

	for _, offset := range req.Offsets {
		err := removeEmbeddingByOffsetCallback(req.ID, offset)
		if err != nil {
			logger.Errorf("Error removing embedding for ID %s and offset %d: %v", req.ID, offset, err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to remove embedding")
			return
		}
	}

	_ = utils.RespondWithJSON(w, http.StatusOK, RemoveMultipleEmbeddingsByOffsetResponse{
		Message: "Embeddings removed successfully",
		ID:      req.ID,
		Offsets: req.Offsets,
	})
}

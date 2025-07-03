package handlers

import (
	"encoding/json"
	"net/http"

	"api_gateway/db"
	"api_gateway/middleware"
	"api_gateway/utils"
)

type MaxBatchTextSizeRequest struct {
	Timeout int `json:"timeout"`
}

type MaxBatchTextSizeResponse struct {
	Size int `json:"size"`
}

// MaxBatchTextSizeHandler handles the request to get the maximum batch text size
func MaxBatchTextSizeHandler(w http.ResponseWriter, r *http.Request) {
	cacheEntry, ok := r.Context().Value(middleware.CacheEntryKey).(db.APIKeyDetails)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve cache entry from context")
		return
	}

	// Parse input
	var req MaxBatchTextSizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	_ = utils.RespondWithJSON(w, http.StatusOK, MaxBatchTextSizeResponse{
		Size: cacheEntry.ChunkSize,
	})
}

package handlers

import (
	"net/http"

	"api_gateway/db"
	"api_gateway/middleware"
	"api_gateway/register"
	"api_gateway/utils"
)

// RefreshAPIKeyResponse represents the response structure for the RefreshAPIKeyHandler
type RefreshAPIKeyResponse struct {
	APIKey string `json:"api_key"`
}

// RefreshAPIKeyHandler handles the API key refresh request
func RefreshAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	cacheEntry, ok := r.Context().Value(middleware.CacheEntryKey).(db.APIKeyDetails)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve cache entry from context")
		return
	}

	newApiKey := register.GenerateAPIKey()
	if err := db.UpdateAPIKeyBySiteID(cacheEntry.ID, newApiKey); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update API key: "+err.Error())
		return
	}

	_ = utils.RespondWithJSON(w, 200, RefreshAPIKeyResponse{
		APIKey: newApiKey,
	})
}

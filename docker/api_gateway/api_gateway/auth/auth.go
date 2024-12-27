package auth

import (
	"net/http"
	"api_gateway/utils"
	"api_gateway/config"
)

// AuthHandler exchanges API key for a JWT
func AuthHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" || !validateAPIKey(apiKey) {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid API Key")
		return
	}

	jwt, err := GenerateJWT(apiKey)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error generating token")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"token": jwt})
}

// Simulated API Key validation
func validateAPIKey(apiKey string) bool {
	if config.GetEnv("LOCAL", "false") == "true" {
		return apiKey == "valid-api-key"
	}
	// In production, query your database or cache to validate
	return apiKey == "valid-api-key"
}

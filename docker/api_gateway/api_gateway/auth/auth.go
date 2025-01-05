package auth

import (
	"api_gateway/utils"
	"net/http"
)

// AuthHandler exchanges API key for a JWT
func AuthHandler(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "API Key is required")
		return
	}

	valid, hashedKey, err := ValidateAPIKey(apiKey)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error validating API Key: "+err.Error())
		return
	}
	if !valid {
		utils.RespondWithError(w, http.StatusUnauthorized, "Invalid API Key")
		return
	}

	jwt, exp, err := GenerateJWT(apiKey, hashedKey)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Error generating token: "+err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"token": jwt,
		"exp":   exp,
	})
}

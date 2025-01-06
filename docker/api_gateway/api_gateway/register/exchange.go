package register

import (
	"encoding/json"
	"fmt"
	"net/http"

	"api_gateway/db"
	"api_gateway/logger"
	"api_gateway/utils"
)

type ExchangeRequest struct {
	Domain      string `json:"domain"`
	Path        string `json:"path"`
	SaltedToken string `json:"salted_token"`
}

type APIKeyResponse struct {
	APIKey       string `json:"api_key"`
	SaltedAPIKey string `json:"salted_api_key"`
	Message      string `json:"message"`
}

func exchangeTokenForAPIKey(domain, path, saltedToken string) (string, string, error) {
	// Stub for database logic
	logger.Infof("Exchanging token for API key for domain %s and path %s", domain, path)

	validationToken, requestSalt, err := db.GetVerifiedTokenDetails(domain, path)
	if err != nil {
		return "", "", fmt.Errorf("Failed to get validation token details: %v", err)
	}

	if saltedToken != saltMessage(validationToken, requestSalt) {
		return "", "", fmt.Errorf("Invalid salted token %s", saltedToken)
	}

	// Generate and return a new API key
	apiKey := generateAPIKey()
	logger.Infof("Generated API key for domain %s and path %s", domain, path)

	if err := db.ActivateAPIKey(validationToken, apiKey); err != nil {
		return "", "", fmt.Errorf("Failed to activate API key for domain %s and path %s: %v", domain, path, err)
	}
	saltedAPIKey := saltMessage(apiKey, requestSalt)

	return apiKey, saltedAPIKey, nil
}

// Handle token exchange for API key
func ExchangeTokenHandler(w http.ResponseWriter, r *http.Request) {
	var request ExchangeRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Verify token and get API key
	apiKey, saltedApiKey, err := exchangeTokenForAPIKey(request.Domain, request.Path, request.SaltedToken)
	if err != nil {
		utils.RespondWithError(w, http.StatusForbidden, err.Error())
		return
	}

	resp := APIKeyResponse{
		APIKey:       apiKey,
		SaltedAPIKey: saltedApiKey,
		Message:      "Validation successful. Use the API key to access protected routes.",
	}
	utils.RespondWithJSON(w, http.StatusOK, resp)
}

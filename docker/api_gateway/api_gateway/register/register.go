package register

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"api_gateway/config"
	"api_gateway/db"
	"api_gateway/logger"
	"api_gateway/utils"
)

type RegisterRequest struct {
	Domain          string `json:"domain"`
	Path            string `json:"path"`
	EstimatedPages  int    `json:"estimated_pages"`
	AveragePageSize int    `json:"average_page_size"`
	StDevPageSize   int    `json:"st_dev_page_size"`
}

type RegisterResponse struct {
	ValidationToken          string    `json:"validation_token"`
	ValidationTokenExpiresAt time.Time `json:"validation_token_expires_at"`
	Message                  string    `json:"message"`
}

func getValidationTokenExpiry() (time.Time, error) {
	expiryDuration := config.GetEnvDuration("API_GATEWAY_VERIFICATION_TOKEN_EXPIRY", 5*time.Minute)
	return time.Now().Add(expiryDuration), nil
}

// Store the registration data in the database
func storeRegistrationData(req RegisterRequest, token string, tokenExpiresAt time.Time) error {
	logger.Infof("Storing registration data for domain %s and path %s", req.Domain, req.Path)
	err := db.StoreRegistrationData(req.Domain, req.Path, token, tokenExpiresAt, req.EstimatedPages, req.AveragePageSize, req.StDevPageSize)
	if err != nil {
		return fmt.Errorf("failed to store registration data: %s", err.Error())
	}
	return nil
}

// Handle the initial registration process
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Check if a record already exists with the same parameters
	_, _, err := db.GetFailedTokenDetails(req.Domain, req.Path)
	if err == nil {
		// If the validation has failed, reset it to pending
		refreshedToken := generateValidationToken()
		refreshedTokenExpiresAt, err := getValidationTokenExpiry()
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if err := db.ResetValidationStatus(req.Domain, req.Path, refreshedToken, refreshedTokenExpiresAt); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		resp := RegisterResponse{
			ValidationToken:          refreshedToken,
			ValidationTokenExpiresAt: refreshedTokenExpiresAt,
			Message:                  "Validation has been reset to pending. Please respond to the challenge.",
		}
		utils.RespondWithJSON(w, http.StatusOK, resp)

		// Perform asynchronous validation
		go verifyDomain(req.Domain, req.Path, refreshedToken)
		return
	} else {
		logger.Infof("No failed token found for domain %s and path %s (%v)", req.Domain, req.Path, err)
	}

	// Generate a validation token
	validationToken := generateValidationToken()
	validationTokenExpiresAt, err := getValidationTokenExpiry()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Store the data and token in the database
	err = storeRegistrationData(req, validationToken, validationTokenExpiresAt)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Respond to the client with the validation token
	resp := RegisterResponse{
		ValidationToken:          validationToken,
		ValidationTokenExpiresAt: validationTokenExpiresAt,
		Message:                  "Validation initiated. Please respond to the challenge.",
	}
	utils.RespondWithJSON(w, http.StatusOK, resp)

	// Perform asynchronous validation
	go verifyDomain(req.Domain, req.Path, validationToken)
}

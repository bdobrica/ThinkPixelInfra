package register

import (
	"encoding/json"
	"fmt"
	"net/http"

	"api_gateway/db"
	"api_gateway/logger"
    "api_gateway/utils"
)

type RegisterRequest struct {
	Domain          string `json:"domain"`
	Path            string `json:"path"`
	Salt            string `json:"salt"`
	EstimatedPages  int    `json:"estimated_pages"`
	AveragePageSize int    `json:"average_page_size"`
	StDevPageSize   int    `json:"st_dev_page_size"`
}

type RegisterResponse struct {
	VerificationToken string `json:"verification_token"`
	SaltedToken       string `json:"salted_token"`
	Message           string `json:"message"`
}

// Store the registration data in the database
func storeRegistrationData(req RegisterRequest, token string) error {
	logger.Infof("Storing registration data for domain %s and path %s", req.Domain, req.Path)
	err := db.StoreRegistrationData(req.Domain, req.Path, req.Salt, token, req.EstimatedPages, req.AveragePageSize, req.StDevPageSize)
	if err != nil {
		return fmt.Errorf("Failed to store registration data: %s", err.Error())
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
	existingToken, _, err := db.GetFailedTokenDetails(req.Domain, req.Path)
	if err == nil {
		// If the verification has failed, reset it to pending
		if err := db.ResetVerificationStatus(req.Domain, req.Path, req.Salt); err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		resp := RegisterResponse{
			VerificationToken: existingToken,
            SaltedToken:       saltMessage(existingToken, req.Salt),
			Message:           "Verification has been reset to pending. Please respond to the challenge.",
		}
		utils.RespondWithJSON(w, http.StatusOK, resp)

        // Perform asynchronous verification
        go verifyDomain(req.Domain, req.Path, req.Salt, existingToken)
		return
	}

	// Generate a verification token
	verificationToken := generateVerificationToken()

	// Store the data and token in the database
	err = storeRegistrationData(req, verificationToken)
	if err != nil {
        utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Respond to the client with the verification token
	resp := RegisterResponse{
		VerificationToken: verificationToken,
		SaltedToken:       saltMessage(verificationToken, req.Salt),
		Message:           "Verification initiated. Please respond to the challenge.",
	}
    utils.RespondWithJSON(w, http.StatusOK, resp)

	// Perform asynchronous verification
	go verifyDomain(req.Domain, req.Path, req.Salt, verificationToken)
}

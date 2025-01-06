package register

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"api_gateway/config"
	"api_gateway/db"
	"api_gateway/logger"
	"strconv"
)

type ValidationResponse struct {
	Domain      string `json:"domain"`
	Path        string `json:"path"`
	SaltedToken string `json:"salted_token"`
}

func verifyDomain(domain string, path string, salt string, token string) {
	validationSuffix := config.GetEnv("API_GATEWAY_VALIDATION_SUFFIX", "wp-content/plugins/thinkpixel/rpc/validate/")
	timeoutStr := config.GetEnv("API_GATEWAY_VALIDATION_TIMEOUT", "5s")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		logger.Fatalf("Failed to parse validation timeout: %v", err)
		return
	}

	maxAttemptsStr := config.GetEnv("API_GATEWAY_VALIDATION_MAX_ATTEMPTS", "3")
	maxAttemptsRaw, err := strconv.Atoi(maxAttemptsStr)
	if err != nil || maxAttemptsRaw < 1 {
		maxAttemptsRaw = 3 // Default to 3 attempts if invalid
	}
	maxAttempts := int32(maxAttemptsRaw)

	url := fmt.Sprintf("https://%s%s/%s", domain, path, validationSuffix)
	client := http.Client{Timeout: timeout}

	logger.Infof("Validate domain %s%s with token %s (attempts: %d)", domain, path, token, maxAttempts)

	// Atomic counter to track attempts
	var attemptCounter int32

	// Immediate first attempt
	executeValidation(client, url, domain, path, salt, token, &attemptCounter, maxAttempts)

	// Schedule retries asynchronously with exponential backoff
	for i := int32(1); i < maxAttempts; i++ {
		delay := time.Duration(1<<i) * time.Second // Exponential backoff: 2^i seconds
		time.AfterFunc(delay, func() {
			executeValidation(client, url, domain, path, salt, token, &attemptCounter, maxAttempts)
		})
	}
}

func executeValidation(client http.Client, url string, domain string, path string, salt string, token string, attemptCounter *int32, maxAttempts int32) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Errorf("Failed to create validation request: %v", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("Validation request failed for %s: %v", url, err)
		checkFinalAttempt(domain, path, token, attemptCounter, maxAttempts)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Infof("Validation failed for %s with status code %d", url, resp.StatusCode)
		checkFinalAttempt(domain, path, token, attemptCounter, maxAttempts)
		return
	}

	var validationResponse ValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&validationResponse); err != nil {
		logger.Errorf("Failed to decode validation response: %v", err)
		checkFinalAttempt(domain, path, token, attemptCounter, maxAttempts)
		return
	}

	validationToken, _, err := db.GetPendingTokenDetails(validationResponse.Domain, validationResponse.Path)
	if err != nil {
		logger.Errorf("Failed to retrieve validation token details for domain: %s, path: %s, salt: %s: %v", validationResponse.Domain, validationResponse.Path, salt, err)
		checkFinalAttempt(domain, path, token, attemptCounter, maxAttempts)
		return
	}

	// Validate the response: proof that the requester knows the salted token and the salt
	saltedToken := saltMessage(saltMessage(validationToken, salt), salt)
	if saltedToken == validationResponse.SaltedToken {
		if err := db.VerifyToken(token, domain, path); err != nil {
			logger.Errorf("Failed to mark token as verified: %v", err)
		} else {
			logger.Infof("Validation succeeded for %s", url)
		}
	} else {
		checkFinalAttempt(domain, path, token, attemptCounter, maxAttempts)
	}
}

func checkFinalAttempt(domain, path, token string, attemptCounter *int32, maxAttempts int32) {
	if atomic.AddInt32(attemptCounter, 1) == maxAttempts {
		if err := db.FailToken(token, domain, path); err != nil {
			logger.Errorf("Failed to mark token as failed: %v", err)
		} else {
			logger.Infof("Final validation attempt failed. Token marked as failed for %s%s", domain, path)
		}
	}
}

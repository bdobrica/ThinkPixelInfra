package register

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"api_gateway/config"
	"api_gateway/db"
	"api_gateway/logger"
	"strconv"
)

type ValidationResponse struct {
	Domain          string `json:"domain"`
	Path            string `json:"path"`
	ValidationToken string `json:"validation_token"`
	Nonce           string `json:"nonce"`
}

func verifyDomain(domain, path, token string) {
	validationSuffix := config.GetEnv("API_GATEWAY_VALIDATION_SUFFIX", "?rest_route=/thinkpixel/v1/validate/")
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

	protocol := getUrlProtocol()
	url := fmt.Sprintf("%s://%s%s%s", protocol, domain, path, validationSuffix)
	client := http.Client{Timeout: timeout}

	logger.Infof("Validate domain %s%s with token %s (attempts: %d)", domain, path, token, maxAttempts)

	// Atomic counter to track attempts
	var (
		attemptCounter int32
		timers         []*time.Timer
		once           sync.Once
	)

	successChan := make(chan struct{})
	doneChan := make(chan struct{}) // To signal all attempts are finished

	// Schedule retries asynchronously with exponential backoff
	for i := int32(0); i < maxAttempts; i++ {
		var delay time.Duration
		if i > 0 {
			delay = (1<<(i-1))*timeout + time.Second
		} else {
			delay = time.Duration(0)
		}
		timer := time.AfterFunc(delay, func() {
			select {
			case <-successChan:
				return
			default:
				err := executeValidation(client, url, domain, path, token, &attemptCounter, maxAttempts)
				if err != nil {
					logger.Errorf("Validation failed for %s%s (attempt %d): %v", domain, path, attemptCounter, err)
				} else {
					logger.Infof("Validation succeeded for %s%s", domain, path)
					once.Do(func() {
						close(successChan) // Notify all listeners and ensure resources are cleaned up
					})
				}
			}
		})
		timers = append(timers, timer)
	}

	// Clean up timers when success occurs
	go func() {
		select {
		case <-successChan: // Success case
			logger.Infof("Validation succeeded for %s%s. Stopping running timers", domain, path)
			for _, timer := range timers {
				if timer != nil {
					timer.Stop() // Stop all timers
				}
			}
		case <-doneChan: // All attempts finished without success
			logger.Errorf("Validation failed for %s%s after %d attempts", domain, path, maxAttempts)
			return
		}
	}()

	// Wait for all attempts to finish
	go func() {
		logger.Infof("Waiting for all attempts to finish for %s%s", domain, path)
		delay := time.Duration(0)
		for i := int32(1); i < maxAttempts; i++ {
			delay += (1<<(i-1))*timeout + time.Second
		}
		time.Sleep(delay) // Wait until the last timer should fire
		logger.Infof("All attempts finished for %s%s", domain, path)
		close(doneChan) // Signal that all retries are done
	}()
}

func executeValidation(client http.Client, url string, domain string, path string, token string, attemptCounter *int32, maxAttempts int32) error {
	logger.Infof("Executing validation for %s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("validation request failed for %s: %v", url, err)
		return checkFinalAttempt(domain, path, token, attemptCounter, maxAttempts, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("validation failed for %s with status code %d", url, resp.StatusCode)
		return checkFinalAttempt(domain, path, token, attemptCounter, maxAttempts, err)
	}

	var validationResponse ValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&validationResponse); err != nil {
		err = fmt.Errorf("failed to decode validation response: %v", err)
		checkFinalAttempt(domain, path, token, attemptCounter, maxAttempts, err)
	}

	validationToken, validationTokenExpiresAt, err := db.GetPendingTokenDetails(validationResponse.Domain, validationResponse.Path)
	if err != nil {
		err = fmt.Errorf("failed to retrieve validation token details for domain: %s and path: %s: %v", validationResponse.Domain, validationResponse.Path, err)
		return checkFinalAttempt(domain, path, token, attemptCounter, maxAttempts, err)
	}

	if time.Now().After(validationTokenExpiresAt) {
		err = fmt.Errorf("validation token has expired for %s%s", domain, path)
		return checkFinalAttempt(domain, path, token, attemptCounter, maxAttempts, err)
	}

	// Validate the token
	if validationToken == validationResponse.ValidationToken {
		if err := db.VerifyToken(token, domain, path); err != nil {
			logger.Errorf("failed to mark token as verified: %v", err)
		} else {
			logger.Infof("Validation succeeded for %s", url)
		}
	} else {
		err = fmt.Errorf("validation token mismatch for %s%s", domain, path)
		return checkFinalAttempt(domain, path, token, attemptCounter, maxAttempts, err)
	}

	// Trigger key exchange asynchronously and handle any errors
	go func() {
		if err := triggerKeyExchange(domain, path, validationResponse.Nonce); err != nil {
			logger.Errorf("Key exchange failed for %s%s: %v", domain, path, err)
		}
	}()
	return nil
}

func checkFinalAttempt(domain, path, token string, attemptCounter *int32, maxAttempts int32, passedError error) error {
	if atomic.AddInt32(attemptCounter, 1) == maxAttempts {
		if err := db.FailToken(token, domain, path); err != nil {
			return fmt.Errorf("final attempt failed: %v, failed to mark token as failed: %v", passedError, err)
		} else {
			return fmt.Errorf("final attempt failed: %v, token marked as failed", passedError)
		}
	}
	return passedError
}

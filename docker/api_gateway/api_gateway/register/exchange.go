package register

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"api_gateway/config"
	"api_gateway/db"
	"api_gateway/logger"
)

type APIKeyRequest struct {
	APIKey  string `json:"api_key"`
	Nonce   string `json:"nonce"`
	Message string `json:"message"`
}

type APIKeyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func triggerKeyExchange(domain, path, nonce string) error {
	keyExchangeSuffix := config.GetEnv("API_GATEWAY_KEY_EXCHANGE_SUFFIX", "?rest_route=/thinkpixel/v1/exchange/")
	url := fmt.Sprintf("https://%s%s%s", domain, path, keyExchangeSuffix)
	apiKey := GenerateAPIKey()
	logger.Infof("Generated API key for domain %s and path %s", domain, path)

	// Send the API key to the client
	apiKeyRequest := APIKeyRequest{
		APIKey:  apiKey,
		Nonce:   nonce,
		Message: "Validation successful. Use the API key to access protected routes.",
	}

	payloadBytes, err := json.Marshal(apiKeyRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal key exchange payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create key exchange request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	timeOutStr := config.GetEnv("API_GATEWAY_KEY_EXCHANGE_TIMEOUT", "10s")
	timeOut, err := time.ParseDuration(timeOutStr)
	if err != nil {
		return fmt.Errorf("failed to parse key exchange timeout: %v", err)
	}
	client := &http.Client{
		Timeout: timeOut,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("key exchange request failed for %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("key exchange failed for %s with status code %d", url, resp.StatusCode)
	}

	var apiKeyResponse APIKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiKeyResponse); err != nil {
		return fmt.Errorf("failed to decode key exchange response: %v", err)
	}

	if !apiKeyResponse.Success {
		return fmt.Errorf("key exchange failed for %s: %s", url, apiKeyResponse.Message)
	}

	logger.Infof("Key exchange succeeded for %s", url)
	if err := db.ActivateAPIKey(domain, path, apiKey); err != nil {
		return fmt.Errorf("failed to activate API key for domain %s and path %s: %v", domain, path, err)
	}

	return nil
}

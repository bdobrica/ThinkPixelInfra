package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"api_gateway/config"
	"api_gateway/logger"
	"api_gateway/utils"
)

// initHTTPClient creates and returns an HTTP client with a timeout configured from environment variables.
// Used for making requests to the model API.
func initHTTPClient() (*http.Client, error) {
	defaultTimeout := 10 * time.Second
	timeout := config.GetEnvDuration("API_GATEWAY_MODEL_TIMEOUT", defaultTimeout)
	return &http.Client{Timeout: timeout}, nil
}

// truncateForLogging returns a string representation of an object, truncating it for logging if not in debug mode.
// Useful for logging large payloads without flooding logs.
func truncateForLogging(o any) string {
	var oString string

	oBytes, err := json.Marshal(o)
	if err != nil {
		oString = fmt.Sprintf("%v", o) // Fallback to default string representation
	} else {
		oString = string(oBytes)
	}

	enableDebugging := config.GetEnvBool("LOCAL", true)
	if enableDebugging || len(oString) <= 256 {
		return oString // No truncation in debug mode
	}
	return oString[:250] + "..." + oString[len(oString)-3:]
}

// callModelAPI sends a batch of TextItems to the model API and decodes the response.
// Handles request marshalling, HTTP POST, error logging, and response decoding.
// Returns the decoded inference response or an error.
func callModelAPI(client *http.Client, textItems []TextItem, model, language, mode string, chunkSize, chunkOverlap int) (*InferenceResponse, error) {
	requestPayload := InferenceRequest{
		TextItems:    textItems,
		Language:     language,
		Mode:         mode,
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		logger.Errorf("Error marshalling request payload: %v", err)
		return nil, err
	}

	ModelURL := fmt.Sprintf("http://%s:8000/infer", model)
	resp, err := client.Post(ModelURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		// Log request details and error
		logger.Errorf("Calling model %s with payload (truncated) %s resulted in error: %v", model, truncateForLogging(requestBody), err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Log request details and status code
		logger.Errorf("Model %s API returned status %d for request (truncated) %s", model, resp.StatusCode, truncateForLogging(requestBody))
		return nil, fmt.Errorf("model API returned status %d", resp.StatusCode)
	}

	var inferenceResponse InferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&inferenceResponse); err != nil {
		logger.Errorf("Error decoding response for request %s to model %s: %v", requestBody, model, err)
		return nil, err
	}

	return &inferenceResponse, nil
}

// callModelAPIWithRetry calls the model API with retry logic and exponential backoff for recoverable errors.
// Returns the inference response or an error after exhausting retries.
func callModelAPIWithRetry(client *http.Client, textItems []TextItem, model, language, mode string, chunkSize, chunkOverlap int) (*InferenceResponse, error) {
	// Retry logic for model API call
	modelMaxRetries := config.GetEnvInt("API_GATEWAY_MODEL_MAX_RETRIES", 3)
	modelRetryDelay := config.GetEnvDuration("API_GATEWAY_MODEL_RETRY_DELAY", 500*time.Millisecond)

	for attempt := 0; attempt < modelMaxRetries; attempt++ {
		inferenceResponse, err := callModelAPI(client, textItems, model, language, mode, chunkSize, chunkOverlap)
		if err == nil {
			return inferenceResponse, nil // Success
		}

		if !utils.IsRecoverableError(err) {
			logger.Errorf("Non-recoverable error calling model %s: %v", model, err)
			return nil, err // Non-recoverable error, return immediately
		}

		logger.Errorf("Attempt %d/%d failed for model %s: %v", attempt+1, modelMaxRetries, model, err)
		if attempt < modelMaxRetries-1 {
			logger.Infof("Retrying in %s...", modelRetryDelay)
			time.Sleep(modelRetryDelay * (1 << attempt)) // Exponential backoff
		}
	}

	return nil, fmt.Errorf("failed to call model %s after %d attempts", model, modelMaxRetries)
}

// decodeInferenceResponse parses the inference response and appends results to the provided embeddingsResponses slice.
// Expects Metadata.Extra to contain an "Offset" key as a string integer.
// Returns an error if decoding fails or required metadata is missing.
func decodeInferenceResponse(inferenceResponse InferenceResponse, embeddingsResponses *[]EmbeddingResponse) error {
	for i, result := range inferenceResponse.Results {
		decodedDenseVector, err := decodeFloatArray(result.DenseVector)
		if err != nil {
			logger.Errorf("Error decoding dense vector for item %d: %v", i, err)
			return fmt.Errorf("error decoding dense vector for item %d: %v", i, err)
		}
		decodedSparseVector, err := decodeFloatMap(result.SparseVector)
		if err != nil {
			logger.Errorf("Error decoding sparse vector for item %d: %v", i, err)
			return fmt.Errorf("error decoding sparse vector for item %d: %v", i, err)
		}

		*embeddingsResponses = append(*embeddingsResponses, EmbeddingResponse{
			ID:           result.Metadata.ID,
			Text:         result.Text,
			Offset:       result.Offset,
			DenseVector:  decodedDenseVector,
			SparseVector: decodedSparseVector,
		})
	}
	return nil
}

// GetEmbeddings retrieves embeddings for an array of text items using the specified model.
// Handles HTTP client initialization, model API calls, latency logging, and response decoding.
// Returns a slice of EmbeddingResponse or an error.
func getEmbeddings(textItems []TextItem, model, language, mode string, chunkSize, chunkOverlap int) ([]EmbeddingResponse, error) {
	// Initialize HTTP client
	client, err := initHTTPClient()
	if err != nil {
		logger.Errorf("Failed to initialize HTTP client: %v", err)
		return nil, err
	}

	logger.Debugf("Using model %s", model)

	// Process the queue and call the model API
	embeddings := make([]EmbeddingResponse, 0)

	inferenceResponse, err := callModelAPIWithRetry(client, textItems, model, language, mode, chunkSize, chunkOverlap)
	if err != nil {
		logger.Errorf("Error calling model API: %v", err)
		return nil, err
	}
	// Log the latency of the inference response
	logger.Debugf("Model API response latency: %.2f ms", inferenceResponse.Latency*1e-3)
	// Update the model latency metric
	UpdateModelLatencyMs(inferenceResponse.Latency * 1e-3)
	// Decode the inference response and append to embeddings
	if err := decodeInferenceResponse(*inferenceResponse, &embeddings); err != nil {
		logger.Errorf("Error decoding inference response: %v", err)
		return nil, err
	}

	logger.Debugf("Retrieved %d embeddings for model %s", len(embeddings), model)
	return embeddings, nil
}

func GetSearchEmbeddings(textItems []TextItem, model, language string, chunkSize, chunkOverlap int) ([]EmbeddingResponse, error) {
	return getEmbeddings(textItems, model, language, "search", chunkSize, chunkOverlap)
}

func GetStoreEmbeddings(textItems []TextItem, model, language string, chunkSize, chunkOverlap int) ([]EmbeddingResponse, error) {
	return getEmbeddings(textItems, model, language, "store", chunkSize, chunkOverlap)
}

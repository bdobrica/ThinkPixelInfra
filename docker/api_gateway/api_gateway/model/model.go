package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"api_gateway/config"
	"api_gateway/inference_queue"
	"api_gateway/logger"
	"api_gateway/utils"
)

// initHTTPClient initializes an HTTP client with a configurable timeout.
func initHTTPClient() (*http.Client, error) {
	defaultTimeout := 10 * time.Second
	timeout := config.GetEnvDuration("API_GATEWAY_MODEL_TIMEOUT", defaultTimeout)
	return &http.Client{Timeout: timeout}, nil
}

// truncateForLogging truncates an object to a string representation suitable for logging.
func truncateForLogging(o any) string {
	objectStr := fmt.Sprintf("%v", o)
	enableDebugging := config.GetEnvBool("LOCAL", true)
	if enableDebugging || len(objectStr) <= 256 {
		return objectStr // No truncation in debug mode
	}
	return objectStr[:250] + "..." + objectStr[len(objectStr)-3:]
}

// callModelAPI sends a batch of TextItems to the model API and decodes the response.
// Note: If the request body is large, consider truncating or summarizing it in logs.
func callModelAPI(client *http.Client, model string, batch []TextItem) (*InferenceResponse, error) {
	requestPayload := InferenceRequest{
		TextItems: batch,
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

// callModelAPIWithRetry retries the model API call with exponential backoff on recoverable errors.
func callModelAPIWithRetry(client *http.Client, model string, batch []TextItem) (*InferenceResponse, error) {
	// Retry logic for model API call
	modelMaxRetries := config.GetEnvInt("API_GATEWAY_MODEL_MAX_RETRIES", 3)
	modelRetryDelay := config.GetEnvDuration("API_GATEWAY_MODEL_RETRY_DELAY", 500*time.Millisecond)

	for attempt := 0; attempt < modelMaxRetries; attempt++ {
		inferenceResponse, err := callModelAPI(client, model, batch)
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

// decodeInferenceResponse decodes the inference response and appends results to embeddingsResponses.
// Expects Metadata.Extra to contain an "Offset" key as a string integer.
func decodeInferenceResponse(inferenceResponse InferenceResponse, embeddingsResponses *[]EmbeddingResponse) error {
	for i, result := range inferenceResponse.Results {
		offsetStr, ok := result.Metadata.Extra["Offset"]
		if !ok {
			logger.Errorf("Missing Offset in Metadata.Extra for item %d", i)
			return fmt.Errorf("missing Offset in Metadata.Extra for item %d", i)
		}
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			logger.Errorf("Invalid Offset value in Metadata.Extra for item %d: %v", i, err)
			return fmt.Errorf("invalid Offset value in Metadata.Extra for item %d: %v", i, err)
		}

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
			Offset:       offset,
			DenseVector:  decodedDenseVector,
			SparseVector: decodedSparseVector,
		})
	}
	return nil
}

// GetEmbeddings retrieves embeddings for an array of text items using the specified model.
func GetEmbeddings(textItems []TextItem, model string, chunkSize, chunkOverlap int) ([]EmbeddingResponse, error) {
	// Initialize HTTP client
	client, err := initHTTPClient()
	if err != nil {
		logger.Errorf("Failed to initialize HTTP client: %v", err)
		return nil, err
	}

	// Initialize inference queue
	queueAny := queuePool.Get()
	if queueAny == nil {
		return nil, fmt.Errorf("failed to obtain inference queue from pool")
	}
	queue := queueAny.(*inference_queue.InferenceQueue[TextItem])
	defer func() {
		err = queue.Reset()
		if err != nil {
			logger.Errorf("Error resetting inference queue: %v", err)
		}
		queuePool.Put(queue)
	}()

	logger.Debugf("Using model %s", model)
	logger.Debugf("Splitting text items into chunks of size %d with overlap %d", chunkSize, chunkOverlap)

	splitItems := splitTextItems(textItems, chunkSize, chunkOverlap)
	queue.PushBatch(splitItems)

	// Process the queue and call the model API
	embeddings := make([]EmbeddingResponse, 0)

	for {
		batch, err := queue.GetBatch()
		if err != nil {
			if err == io.EOF {
				break // No more batches to process
			}
			logger.Errorf("Error popping batch from queue: %v", err)
			return nil, err
		}
		if len(batch) == 0 {
			logger.Debugf("No items in batch, skipping")
			continue
		}
		logger.Debugf("Processing batch of %d items", len(batch))
		inferenceResponse, err := callModelAPIWithRetry(client, model, batch)
		if err != nil {
			logger.Errorf("Error calling model API: %v", err)
			return nil, err
		}
		// Log the latency of the inference response
		logger.Debugf("Model API response latency: %.2f ms", inferenceResponse.Latency*1000)
		// Decode the inference response and append to embeddings
		if err := decodeInferenceResponse(*inferenceResponse, &embeddings); err != nil {
			logger.Errorf("Error decoding inference response: %v", err)
			return nil, err
		}
	}

	return embeddings, nil
}

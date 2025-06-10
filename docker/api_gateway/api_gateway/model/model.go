package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"api_gateway/config"
	"api_gateway/logger"
)

type Metadata struct {
	ID    int               `json:"id"`
	Extra map[string]string `json:"extra,omitempty"`
}

type TextItem struct {
	Text     string   `json:"text"`
	Metadata Metadata `json:"metadata"`
}

type InferenceRequest struct {
	TextItems []TextItem `json:"text_items"`
}

type EmbeddingsItem struct {
	Text         string            `json:"text"`
	DenseVector  string            `json:"dense_vector"`
	SparseVector map[string]string `json:"sparse_vector"`
	Metadata     Metadata          `json:"metadata"`
}

type InferenceResponse struct {
	Results []EmbeddingsItem `json:"results"`
	Latency float64          `json:"latency"`
}

type EmbeddingResponse struct {
	ID           int             `json:"id"`
	Text         string          `json:"text"`
	Offset       int             `json:"offset"`
	DenseVector  []float32       `json:"dense_vector"`
	SparseVector map[int]float32 `json:"sparse_vector"`
}

// GetEmbeddings retrieves embeddings for an array of text items
func GetEmbeddings(textItems []TextItem, model string, chunkSize, chunkOverlap int) ([]EmbeddingResponse, error) {
	timeoutStr := config.GetEnv("API_GATEWAY_MODEL_TIMEOUT", "10s")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		logger.Errorf("Failed to parse model timeout: %v", err)
		return nil, err
	}

	client := &http.Client{Timeout: timeout}

	logger.Debugf("Using model %s", model)
	logger.Debugf("Splitting text items into chunks of size %d with overlap %d", chunkSize, chunkOverlap)

	splitTextItems := splitTextItems(textItems, chunkSize, chunkOverlap)

	requestPayload := InferenceRequest{
		TextItems: splitTextItems,
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
		logger.Errorf("Calling model %s with payload %s resulted in error: %v", model, requestBody, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Log request details and status code
		logger.Errorf("Model %s API returned status %d for request %s", model, resp.StatusCode, requestBody)
		return nil, fmt.Errorf("model API returned status %d", resp.StatusCode)
	}

	var inferenceResponse InferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&inferenceResponse); err != nil {
		logger.Errorf("Error decoding response for request %s to model %s: %v", requestBody, model, err)
		return nil, err
	}

	// Convert to EmbeddingResponse
	embeddings := make([]EmbeddingResponse, len(inferenceResponse.Results))
	for i, item := range inferenceResponse.Results {
		offsetStr, ok := item.Metadata.Extra["Offset"]
		if !ok {
			logger.Errorf("Missing Offset in Metadata.Extra for item %d", i)
			return nil, fmt.Errorf("missing Offset in Metadata.Extra for item %d", i)
		}
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			logger.Errorf("Invalid Offset value in Metadata.Extra for item %d: %v", i, err)
			return nil, fmt.Errorf("invalid Offset value in Metadata.Extra for item %d: %v", i, err)
		}

		decodedDenseVector, err := decodeFloatArray(item.DenseVector)
		if err != nil {
			logger.Errorf("Error decoding dense vector for item %d: %v", i, err)
			return nil, fmt.Errorf("error decoding dense vector for item %d: %v", i, err)
		}
		decodedSparseVector, err := decodeFloatMap(item.SparseVector)
		if err != nil {
			logger.Errorf("Error decoding sparse vector for item %d: %v", i, err)
			return nil, fmt.Errorf("error decoding sparse vector for item %d: %v", i, err)
		}

		embeddings[i] = EmbeddingResponse{
			ID:           item.Metadata.ID,
			Text:         item.Text,
			Offset:       offset,
			DenseVector:  decodedDenseVector,
			SparseVector: decodedSparseVector,
		}
	}

	return embeddings, nil
}

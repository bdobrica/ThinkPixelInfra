package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
	Text     string   `json:"text"`
	Vector   string   `json:"vector"`
	Metadata Metadata `json:"metadata"`
}

type InferenceResponse struct {
	Results []EmbeddingsItem `json:"results"`
	Latency float64          `json:"latency"`
}

type EmbeddingResponse struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	Offset    int    `json:"offset"`
	Embedding string `json:"embedding"`
}

// GetEmbeddings retrieves embeddings for an array of text items
func GetEmbeddings(textItems []TextItem, model string, chunkSize, chunkOverlap int) ([]EmbeddingResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	logger.Debugf("Using model %s", model)
	logger.Debugf("Splitting text items into chunks of size %d with overlap %d", chunkSize, chunkOverlap)

	splitTextItems := splitTextItems(textItems, chunkSize, chunkOverlap)

	requestPayload := InferenceRequest{
		TextItems: splitTextItems,
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, err
	}

	ModelURL := fmt.Sprintf("http://%s:8000/infer", model)
	resp, err := client.Post(ModelURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model API returned status %d", resp.StatusCode)
	}

	var inferenceResponse InferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&inferenceResponse); err != nil {
		return nil, err
	}

	// Convert to EmbeddingResponse
	embeddings := make([]EmbeddingResponse, len(inferenceResponse.Results))
	for i, item := range inferenceResponse.Results {
		offsetStr, ok := item.Metadata.Extra["Offset"]
		if !ok {
			return nil, fmt.Errorf("missing Offset in Metadata.Extra for item %d", i)
		}
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, fmt.Errorf("invalid Offset value in Metadata.Extra for item %d: %v", i, err)
		}
		embeddings[i] = EmbeddingResponse{
			ID:        item.Metadata.ID,
			Text:      item.Text,
			Offset:    offset,
			Embedding: item.Vector,
		}
	}

	return embeddings, nil
}

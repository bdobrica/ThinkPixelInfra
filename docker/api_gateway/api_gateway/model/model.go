package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"api_gateway/config"
)

var ModelURL = config.GetEnv("API_GATEWAY_MODEL_URL", "http://model:8000/infer")

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
func GetEmbeddings(textItems []TextItem) ([]EmbeddingResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	requestPayload := InferenceRequest{
		TextItems: textItems,
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, err
	}

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
		embeddings[i] = EmbeddingResponse{
			ID:        item.Metadata.ID,
			Text:      item.Text,
			Offset:    0, // Offset can be set based on logic if needed
			Embedding: item.Vector,
		}
	}

	return embeddings, nil
}

// GetEmbeddingsFromTexts retrieves embeddings for a list of text objects
func GetEmbeddingsFromTexts(input []struct {
	ID    int               `json:"id"`
	Text  string            `json:"text"`
	Extra map[string]string `json:"extra,omitempty"`
}) ([]EmbeddingResponse, error) {
	textItems := make([]TextItem, len(input))
	for i, item := range input {
		textItems[i] = TextItem{
			Text: item.Text,
			Metadata: Metadata{
				ID:    item.ID,
				Extra: item.Extra,
			},
		}
	}

	return GetEmbeddings(textItems)
}

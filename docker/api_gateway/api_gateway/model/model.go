package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"api_gateway/config"
)

var ModelURL = config.GetEnv("API_GATEWAY_MODEL_URL", "http://model:8080/infer")

type EmbeddingResponse struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	Offset    int    `json:"offset"`
	Embedding string `json:"embedding"`
}

// GetEmbeddings retrieves embeddings for an array of texts
func GetEmbeddings(texts []string) ([]EmbeddingResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	requestBody, err := json.Marshal(map[string]interface{}{"texts": texts})
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

	var embeddings []EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
		return nil, err
	}

	return embeddings, nil
}

// GetEmbeddingsFromTexts retrieves embeddings for a list of text objects
func GetEmbeddingsFromTexts(input []struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}) ([]EmbeddingResponse, error) {
	texts := make([]string, len(input))
	for i, item := range input {
		texts[i] = item.Text
	}

	return GetEmbeddings(texts)
}

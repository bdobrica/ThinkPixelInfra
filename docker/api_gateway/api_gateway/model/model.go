package model

import (
	"api_gateway/logger"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Metadata struct {
	ID    int               `json:"id"`
	Extra map[string]string `json:"extra,omitempty"`
}

type TextItem struct {
	Text     string   `json:"text"`
	Metadata Metadata `json:"metadata"`
}

type EmbeddingRequest struct {
	TextItems []TextItem `json:"text_items"`
}

type DenseItem struct {
	Text     string   `json:"text"`
	Vector   string   `json:"vector"`
	Metadata Metadata `json:"metadata"`
}

type DenseResponse struct {
	Results []DenseItem `json:"results"`
	Latency float64     `json:"latency"`
}

type SparseItem struct {
	Text     string   `json:"text"`
	Values   string   `json:"values"`
	Indices  string   `json:"indices"`
	Metadata Metadata `json:"metadata"`
}

type SparseResponse struct {
	Results []SparseItem `json:"results"`
	Latency float64      `json:"latency"`
}

type DenseEmbedding struct {
	Values []float32 `json:"values"`
}

type SparseEmbedding struct {
	Values  []float32 `json:"values"`
	Indices []int     `json:"indices"`
}

type EmbeddingResponse struct {
	ID              int             `json:"id"`
	Text            string          `json:"text"`
	Offset          int             `json:"offset"`
	DenseEmbedding  DenseEmbedding  `json:"dense_embedding"`
	SparseEmbedding SparseEmbedding `json:"sparse_embedding"`
}

// GetEmbeddings retrieves embeddings for an array of text items
func callEmbeddingsModel[TResponse interface{ DenseResponse | SparseResponse }](textItems []TextItem, model string) (*TResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	logger.Debugf("Using model %s", model)

	requestPayload := EmbeddingRequest{
		TextItems: textItems,
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

	var inferenceResponse TResponse
	if err := json.NewDecoder(resp.Body).Decode(&inferenceResponse); err != nil {
		logger.Errorf("Error decoding response for request %s to model %s: %v", requestBody, model, err)
		return nil, err
	}

	return &inferenceResponse, nil
}

func fetchEmbeddings[T interface{ DenseResponse | SparseResponse }](
	batch []TextItem,
	model string,
	callFn func([]TextItem, string) (*T, error),
	ch chan<- T,
	errChan chan<- error,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	resp, err := callFn(batch, model)
	if err != nil {
		errChan <- err
		return
	}

	ch <- *resp
}

// GetEmbeddings retrieves embeddings for an array of text items
func GetEmbeddings(textItems []TextItem, denseModel, sparseModel string, chunkSize, chunkOverlap, batchSize int) ([]EmbeddingResponse, error) {
	logger.Debugf("Splitting text items into chunks of size %d with overlap %d", chunkSize, chunkOverlap)

	splitTextItems := splitTextItems(textItems, chunkSize, chunkOverlap)
	batchedItems := batchTextItems(splitTextItems, batchSize)

	denseChan := make(chan DenseResponse, 1)
	sparseChan := make(chan SparseResponse, 1)
	errChan := make(chan error, 2)

	var wg sync.WaitGroup

	for _, batch := range batchedItems {
		wg.Add(2)

		go fetchEmbeddings(batch, denseModel, callEmbeddingsModel[DenseResponse], denseChan, errChan, &wg)
		go fetchEmbeddings(batch, sparseModel, callEmbeddingsModel[SparseResponse], sparseChan, errChan, &wg)
	}

	wg.Wait()
	close(denseChan)
	close(sparseChan)
	close(errChan)

	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	denseResponse := <-denseChan
	sparseResponse := <-sparseChan

	return mergeEmbeddings(denseResponse, sparseResponse)
}

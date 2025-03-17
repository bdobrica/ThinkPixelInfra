package model

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Helper function to parse Dense vector
func parseDenseVector(vector string) DenseEmbedding {
	var values []float32
	_ = json.Unmarshal([]byte(vector), &values)
	return DenseEmbedding{Values: values}
}

// Helper function to parse Sparse vector
func parseSparseVector(indicesStr, valuesStr string) SparseEmbedding {
	var indices []int
	var values []float32

	_ = json.Unmarshal([]byte(indicesStr), &indices)
	_ = json.Unmarshal([]byte(valuesStr), &values)

	return SparseEmbedding{Indices: indices, Values: values}
}

func mergeEmbeddings(denseResponse DenseResponse, sparseResponse SparseResponse) ([]EmbeddingResponse, error) {
	embeddingMap := make(map[string]EmbeddingResponse)

	// Process Dense embeddings
	for _, item := range denseResponse.Results {
		offset, _ := strconv.Atoi(item.Metadata.Extra["offset"]) // Convert offset to int
		key := fmt.Sprintf("%d-%d", item.Metadata.ID, offset)

		embeddingMap[key] = EmbeddingResponse{
			ID:             item.Metadata.ID,
			Text:           item.Text,
			Offset:         offset,
			DenseEmbedding: parseDenseVector(item.Vector),
		}
	}

	// Process Sparse embeddings
	for _, item := range sparseResponse.Results {
		offset, _ := strconv.Atoi(item.Metadata.Extra["offset"])
		key := fmt.Sprintf("%d-%d", item.Metadata.ID, offset)

		if embedding, exists := embeddingMap[key]; exists {
			embedding.SparseEmbedding = parseSparseVector(item.Indices, item.Values)
			embeddingMap[key] = embedding
		} else {
			embeddingMap[key] = EmbeddingResponse{
				ID:              item.Metadata.ID,
				Text:            item.Text,
				Offset:          offset,
				SparseEmbedding: parseSparseVector(item.Indices, item.Values),
			}
		}
	}

	// Convert map to slice
	var mergedEmbeddings []EmbeddingResponse
	for _, embedding := range embeddingMap {
		mergedEmbeddings = append(mergedEmbeddings, embedding)
	}

	return mergedEmbeddings, nil
}

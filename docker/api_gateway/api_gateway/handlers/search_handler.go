package handlers

import (
	"context"
	"net/http"
	"time"

	"api_gateway/redisconn"
	"api_gateway/utils"
	"api_gateway/model"
)

// SearchHandler handles embedding creation and ANN search
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	// Parse input
	var input struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if input.Text == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Text field is required")
		return
	}

	// Make async request to the model API to get the embedding
	embeddingChan := make(chan []model.EmbeddingResponse)
	errChan := make(chan error)
	go func() {
		embeddings, err := model.GetEmbeddings([]string{input.Text})
		if err != nil {
			errChan <- err
			return
		}
		embeddingChan <- embeddings
	}()

	select {
	case embeddings := <-embeddingChan:
		// Perform ANN search on Redis for all embeddings
		results, err := redisconn.SearchEmbeddings(embeddings, 10)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error performing ANN search")
			return
		}

		// Send results back as JSON
		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"results": results})
	case err := <-errChan:
		utils.RespondWithError(w, http.StatusInternalServerError, "Error retrieving embedding: "+err.Error())
	case <-time.After(10 * time.Second):
		utils.RespondWithError(w, http.StatusRequestTimeout, "Request timed out")
	}
}

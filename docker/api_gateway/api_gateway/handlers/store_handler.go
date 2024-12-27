package handlers

import (
	"context"
	"net/http"
	"time"

	"api_gateway/redisconn"
	"api_gateway/utils"
	"api_gateway/model"
)

// StoreHandler handles storing webpage data
func StoreHandler(w http.ResponseWriter, r *http.Request) {
	// Parse input
	var input []struct {
		ID   int    `json:"id"`
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(input) == 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "Input must contain at least one item")
		return
	}

	// Make async request to the model API to get embeddings
	responseChan := make(chan []model.EmbeddingResponse, 1)
	errChan := make(chan error, 1)
	go func() {
		response, err := model.GetEmbeddingsFromTexts(input)
		if err != nil {
			errChan <- err
			return
		}
		responseChan <- response
	}()

	select {
	case embeddings := <-responseChan:
		storedCount, err := redisconn.StoreEmbeddings(embeddings)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error storing documents: "+err.Error())
			return
		}

		// Send summary back as JSON
		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"received_texts": len(input),
			"stored_documents": storedCount,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	case err := <-errChan:
		utils.RespondWithError(w, http.StatusInternalServerError, "Error retrieving embeddings: "+err.Error())
	case <-time.After(15 * time.Second):
		utils.RespondWithError(w, http.StatusRequestTimeout, "Request timed out")
	}
}

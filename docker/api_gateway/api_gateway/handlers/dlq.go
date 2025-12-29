package handlers

import (
	"api_gateway/dlq"
	"api_gateway/logger"
	"encoding/json"
	"net/http"
	"strconv"
)

// DLQStatsHandler returns statistics about the dead letter queue
func DLQStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := dlq.GetDLQStats()
	if err != nil {
		logger.Errorf("Failed to get DLQ stats: %v", err)
		http.Error(w, "Failed to get DLQ statistics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// DLQMessagesHandler returns DLQ messages for a specific site
func DLQMessagesHandler(w http.ResponseWriter, r *http.Request) {
	// Get site_id from query parameter
	siteIDStr := r.URL.Query().Get("site_id")
	if siteIDStr == "" {
		http.Error(w, "Missing site_id parameter", http.StatusBadRequest)
		return
	}

	siteID, err := strconv.ParseInt(siteIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid site_id parameter", http.StatusBadRequest)
		return
	}

	// Get limit from query parameter (default 100)
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	messages, err := dlq.GetDLQMessages(int32(siteID), limit)
	if err != nil {
		logger.Errorf("Failed to get DLQ messages for site %d: %v", siteID, err)
		http.Error(w, "Failed to get DLQ messages", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"site_id":  siteID,
		"count":    len(messages),
		"messages": messages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

package utils

import (
	"encoding/json"
	"net/http"
	"strings"
)

// RespondWithError sends an error response
func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, map[string]string{"error": message})
}

// RespondWithJSON sends a JSON response
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func IsRecoverableError(err error) bool {
	// Define recoverable errors based on your application's logic
	recoverableErrors := []string{
		"model API returned status",
	}

	for _, recoverable := range recoverableErrors {
		if strings.HasPrefix(err.Error(), recoverable) {
			return true
		}
	}
	return false
}

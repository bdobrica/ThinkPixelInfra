package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRespondWithError tests error response generation
func TestRespondWithError(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		message        string
		wantStatusCode int
		wantError      string
	}{
		{
			name:           "bad request error",
			code:           http.StatusBadRequest,
			message:        "Invalid request payload",
			wantStatusCode: http.StatusBadRequest,
			wantError:      "Invalid request payload",
		},
		{
			name:           "unauthorized error",
			code:           http.StatusUnauthorized,
			message:        "Invalid API Key",
			wantStatusCode: http.StatusUnauthorized,
			wantError:      "Invalid API Key",
		},
		{
			name:           "internal server error",
			code:           http.StatusInternalServerError,
			message:        "Database connection failed",
			wantStatusCode: http.StatusInternalServerError,
			wantError:      "Database connection failed",
		},
		{
			name:           "not found error",
			code:           http.StatusNotFound,
			message:        "Resource not found",
			wantStatusCode: http.StatusNotFound,
			wantError:      "Resource not found",
		},
		{
			name:           "empty error message",
			code:           http.StatusBadRequest,
			message:        "",
			wantStatusCode: http.StatusBadRequest,
			wantError:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			RespondWithError(w, tt.code, tt.message)

			// Check status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatusCode)
			}

			// Check Content-Type header
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %s, want application/json", contentType)
			}

			// Decode response body
			var resp ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check error message
			if resp.Error != tt.wantError {
				t.Errorf("Error message = %s, want %s", resp.Error, tt.wantError)
			}
		})
	}
}

// TestRespondWithJSON tests JSON response generation
func TestRespondWithJSON(t *testing.T) {
	type testPayload struct {
		Message string `json:"message"`
		Count   int    `json:"count"`
	}

	tests := []struct {
		name           string
		code           int
		payload        interface{}
		wantStatusCode int
	}{
		{
			name: "success response",
			code: http.StatusOK,
			payload: testPayload{
				Message: "Operation successful",
				Count:   42,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "accepted response",
			code:           http.StatusAccepted,
			payload:        map[string]string{"status": "queued"},
			wantStatusCode: http.StatusAccepted,
		},
		{
			name:           "created response",
			code:           http.StatusCreated,
			payload:        map[string]int{"id": 123},
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "empty payload",
			code:           http.StatusOK,
			payload:        map[string]string{},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "nil map payload",
			code:           http.StatusOK,
			payload:        map[string]string(nil),
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			err := RespondWithJSON(w, tt.code, tt.payload)
			if err != nil {
				t.Errorf("RespondWithJSON() unexpected error = %v", err)
			}

			// Check status code
			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatusCode)
			}

			// Check Content-Type header
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %s, want application/json", contentType)
			}

			// Verify response can be decoded as JSON
			var decoded map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
				t.Fatalf("Failed to decode JSON response: %v", err)
			}
		})
	}
}

// TestRespondWithJSONComplexTypes tests various payload types
func TestRespondWithJSONComplexTypes(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
	}{
		{
			name:    "string slice",
			payload: []string{"one", "two", "three"},
		},
		{
			name:    "int slice",
			payload: []int{1, 2, 3, 4, 5},
		},
		{
			name: "nested struct",
			payload: struct {
				User struct {
					Name  string `json:"name"`
					Email string `json:"email"`
				} `json:"user"`
				Count int `json:"count"`
			}{
				User: struct {
					Name  string `json:"name"`
					Email string `json:"email"`
				}{
					Name:  "John Doe",
					Email: "john@example.com",
				},
				Count: 10,
			},
		},
		{
			name:    "map with mixed values",
			payload: map[string]interface{}{"name": "test", "count": 42, "active": true},
		},
		{
			name:    "empty slice",
			payload: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			err := RespondWithJSON(w, http.StatusOK, tt.payload)
			if err != nil {
				t.Errorf("RespondWithJSON() unexpected error = %v", err)
			}

			// Verify response is valid JSON
			if !json.Valid(w.Body.Bytes()) {
				t.Error("Response body is not valid JSON")
			}
		})
	}
}

// TestErrorResponseStructure tests the ErrorResponse struct
func TestErrorResponseStructure(t *testing.T) {
	tests := []struct {
		name    string
		errResp ErrorResponse
		want    string
	}{
		{
			name:    "simple error",
			errResp: ErrorResponse{Error: "test error"},
			want:    "test error",
		},
		{
			name:    "empty error",
			errResp: ErrorResponse{Error: ""},
			want:    "",
		},
		{
			name:    "long error message",
			errResp: ErrorResponse{Error: "This is a very long error message with lots of details about what went wrong"},
			want:    "This is a very long error message with lots of details about what went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.errResp.Error != tt.want {
				t.Errorf("ErrorResponse.Error = %s, want %s", tt.errResp.Error, tt.want)
			}

			// Verify JSON marshaling
			data, err := json.Marshal(tt.errResp)
			if err != nil {
				t.Fatalf("Failed to marshal ErrorResponse: %v", err)
			}

			// Verify unmarshaling
			var decoded ErrorResponse
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal ErrorResponse: %v", err)
			}

			if decoded.Error != tt.want {
				t.Errorf("Decoded error = %s, want %s", decoded.Error, tt.want)
			}
		})
	}
}

// TestResponseHeaders tests that proper headers are set
func TestResponseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		function func(w http.ResponseWriter)
	}{
		{
			name: "RespondWithError sets headers",
			function: func(w http.ResponseWriter) {
				RespondWithError(w, http.StatusBadRequest, "test error")
			},
		},
		{
			name: "RespondWithJSON sets headers",
			function: func(w http.ResponseWriter) {
				RespondWithJSON(w, http.StatusOK, map[string]string{"test": "value"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.function(w)

			// Check Content-Type header is set
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type header = %s, want application/json", contentType)
			}

			// Verify status code was written
			if w.Code == 0 {
				t.Error("Status code was not set")
			}
		})
	}
}

// TestConcurrentResponses tests thread-safety of response functions
func TestConcurrentResponses(t *testing.T) {
	const numGoroutines = 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			w := httptest.NewRecorder()
			if id%2 == 0 {
				RespondWithError(w, http.StatusBadRequest, "concurrent error")
			} else {
				RespondWithJSON(w, http.StatusOK, map[string]int{"id": id})
			}

			// Verify response is valid
			if w.Code == 0 {
				t.Errorf("Goroutine %d: status code not set", id)
			}
			if w.Body.Len() == 0 {
				t.Errorf("Goroutine %d: empty response body", id)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// TestSpecialCharactersInError tests error messages with special characters
func TestSpecialCharactersInError(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "message with quotes",
			message: `Error: "invalid" parameter`,
		},
		{
			name:    "message with newlines",
			message: "Error on line 1\nand line 2",
		},
		{
			name:    "message with unicode",
			message: "Error: 日本語 エラー",
		},
		{
			name:    "message with backslashes",
			message: `Path C:\Users\test\file.txt not found`,
		},
		{
			name:    "message with HTML",
			message: "<script>alert('xss')</script>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			RespondWithError(w, http.StatusBadRequest, tt.message)

			// Verify response is valid JSON
			if !json.Valid(w.Body.Bytes()) {
				t.Error("Response with special characters is not valid JSON")
			}

			// Decode and verify message is preserved
			var resp ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp.Error != tt.message {
				t.Errorf("Error message not preserved: got %s, want %s", resp.Error, tt.message)
			}
		})
	}
}

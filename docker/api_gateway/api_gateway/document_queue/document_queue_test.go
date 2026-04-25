package document_queue

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

// TestNewDocumentQueuePayload tests payload creation with different parameters
func TestNewDocumentQueuePayload(t *testing.T) {
	tests := []struct {
		name        string
		siteID      int32
		id          int32
		text        string
		language    string
		extra       map[string]string
		maxRetries  int32
		wantRetries int32
	}{
		{
			name:        "default max retries (0 input)",
			siteID:      1,
			id:          100,
			text:        "Test document",
			language:    "en",
			extra:       map[string]string{"key": "value"},
			maxRetries:  0, // Should use default from config (3)
			wantRetries: 3,
		},
		{
			name:        "custom max retries",
			siteID:      2,
			id:          200,
			text:        "Another document",
			language:    "ro",
			extra:       map[string]string{"url": "https://example.com"},
			maxRetries:  5,
			wantRetries: 5,
		},
		{
			name:        "empty text",
			siteID:      3,
			id:          300,
			text:        "",
			language:    "auto",
			extra:       map[string]string{},
			maxRetries:  3,
			wantRetries: 3,
		},
		{
			name:        "empty language is preserved",
			siteID:      4,
			id:          400,
			text:        "Short query",
			language:    "",
			extra:       map[string]string{},
			maxRetries:  3,
			wantRetries: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := NewDocumentQueuePayload(tt.siteID, tt.id, tt.text, tt.language, tt.extra, tt.maxRetries)

			if payload.SiteId != tt.siteID {
				t.Errorf("SiteId = %d, want %d", payload.SiteId, tt.siteID)
			}
			if payload.Id != tt.id {
				t.Errorf("Id = %d, want %d", payload.Id, tt.id)
			}
			if payload.Text != tt.text {
				t.Errorf("Text = %s, want %s", payload.Text, tt.text)
			}
			if payload.Language != tt.language {
				t.Errorf("Language = %s, want %s", payload.Language, tt.language)
			}
			if len(payload.Extra) != len(tt.extra) {
				t.Errorf("Extra length = %d, want %d", len(payload.Extra), len(tt.extra))
			}
			if payload.MaxRetries != tt.wantRetries {
				t.Errorf("MaxRetries = %d, want %d", payload.MaxRetries, tt.wantRetries)
			}
			if payload.RetryCount != 0 {
				t.Errorf("RetryCount = %d, want 0", payload.RetryCount)
			}
			if payload.Status != MessageStatus_PENDING {
				t.Errorf("Status = %v, want PENDING", payload.Status)
			}
			if payload.TimestampMillis == 0 {
				t.Error("TimestampMillis should be set to current timestamp")
			}
		})
	}
}

// TestProtobufMarshaling tests marshal/unmarshal round-trip
func TestProtobufMarshaling(t *testing.T) {
	original := NewDocumentQueuePayload(
		42,
		1337,
		"Test document for marshaling",
		"es",
		map[string]string{"url": "https://example.com/page", "title": "Test Page"},
		5,
	)

	// Marshal to bytes
	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	// Unmarshal back to object
	unmarshaled := &DocumentQueuePayload{}
	if err := proto.Unmarshal(data, unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	// Verify all fields match
	if unmarshaled.SiteId != original.SiteId {
		t.Errorf("SiteId = %d, want %d", unmarshaled.SiteId, original.SiteId)
	}
	if unmarshaled.Id != original.Id {
		t.Errorf("Id = %d, want %d", unmarshaled.Id, original.Id)
	}
	if unmarshaled.Text != original.Text {
		t.Errorf("Text = %s, want %s", unmarshaled.Text, original.Text)
	}
	if unmarshaled.Language != original.Language {
		t.Errorf("Language = %s, want %s", unmarshaled.Language, original.Language)
	}
	if len(unmarshaled.Extra) != len(original.Extra) {
		t.Errorf("Extra map size = %d, want %d", len(unmarshaled.Extra), len(original.Extra))
	}
	for k, v := range original.Extra {
		if unmarshaled.Extra[k] != v {
			t.Errorf("Extra[%s] = %s, want %s", k, unmarshaled.Extra[k], v)
		}
	}
	if unmarshaled.TimestampMillis != original.TimestampMillis {
		t.Errorf("TimestampMillis = %d, want %d", unmarshaled.TimestampMillis, original.TimestampMillis)
	}
	if unmarshaled.RetryCount != original.RetryCount {
		t.Errorf("RetryCount = %d, want %d", unmarshaled.RetryCount, original.RetryCount)
	}
	if unmarshaled.MaxRetries != original.MaxRetries {
		t.Errorf("MaxRetries = %d, want %d", unmarshaled.MaxRetries, original.MaxRetries)
	}
	if unmarshaled.Status != original.Status {
		t.Errorf("Status = %v, want %v", unmarshaled.Status, original.Status)
	}
}

// TestMessageStatusTransitions tests status transitions
func TestMessageStatusTransitions(t *testing.T) {
	payload := NewDocumentQueuePayload(1, 100, "Test", "auto", map[string]string{}, 3)

	// Initial status should be PENDING
	if payload.Status != MessageStatus_PENDING {
		t.Errorf("Initial status = %v, want PENDING", payload.Status)
	}

	// Transition to PROCESSING
	payload.Status = MessageStatus_PROCESSING
	if payload.Status != MessageStatus_PROCESSING {
		t.Errorf("After transition: status = %v, want PROCESSING", payload.Status)
	}

	// Transition to COMPLETED (success path)
	payload.Status = MessageStatus_COMPLETED
	if payload.Status != MessageStatus_COMPLETED {
		t.Errorf("After completion: status = %v, want COMPLETED", payload.Status)
	}

	// Test failure path
	payload2 := NewDocumentQueuePayload(2, 200, "Test2", "fr", map[string]string{}, 3)
	payload2.Status = MessageStatus_PROCESSING
	payload2.Status = MessageStatus_FAILED
	if payload2.Status != MessageStatus_FAILED {
		t.Errorf("After failure: status = %v, want FAILED", payload2.Status)
	}

	// Test dead letter path
	payload2.Status = MessageStatus_DEAD_LETTER
	if payload2.Status != MessageStatus_DEAD_LETTER {
		t.Errorf("After DLQ: status = %v, want DEAD_LETTER", payload2.Status)
	}
}

// TestRetryLogic tests retry count and max retries behavior
func TestRetryLogic(t *testing.T) {
	maxRetries := int32(3)
	payload := NewDocumentQueuePayload(1, 100, "Test", "auto", map[string]string{}, maxRetries)

	// Initial retry count should be 0
	if payload.RetryCount != 0 {
		t.Errorf("Initial RetryCount = %d, want 0", payload.RetryCount)
	}

	// Simulate retries
	for i := int32(1); i <= maxRetries; i++ {
		payload.RetryCount++
		if payload.RetryCount > maxRetries {
			t.Errorf("RetryCount %d exceeded MaxRetries %d", payload.RetryCount, maxRetries)
		}
	}

	// Verify we can't retry more than max
	if payload.RetryCount >= maxRetries {
		// This is correct - should not retry anymore
		if payload.RetryCount != maxRetries {
			t.Errorf("Final RetryCount = %d, want %d", payload.RetryCount, maxRetries)
		}
	}
}

// TestTimestampValidation tests that timestamps are set correctly
func TestTimestampValidation(t *testing.T) {
	beforeTime := time.Now().UnixMilli()

	// Small sleep to ensure timestamp difference
	time.Sleep(2 * time.Millisecond)

	payload := NewDocumentQueuePayload(1, 100, "Test", "de", map[string]string{}, 3)

	time.Sleep(2 * time.Millisecond)
	afterTime := time.Now().UnixMilli()

	// TimestampMillis should be between before and after
	if payload.TimestampMillis < beforeTime || payload.TimestampMillis > afterTime {
		t.Errorf("TimestampMillis %d not in expected range [%d, %d]",
			payload.TimestampMillis, beforeTime, afterTime)
	}

	// LastRetryTimestamp should be 0 initially
	if payload.LastRetryTimestamp != 0 {
		t.Errorf("LastRetryTimestamp = %d, want 0", payload.LastRetryTimestamp)
	}
}

// TestLargePayload tests handling of large text payloads
func TestLargePayload(t *testing.T) {
	// Create 1MB text
	largeText := make([]byte, 1024*1024)
	for i := range largeText {
		largeText[i] = 'A'
	}

	payload := NewDocumentQueuePayload(
		1,
		100,
		string(largeText),
		"auto",
		map[string]string{"size": "1MB"},
		3,
	)

	// Test marshaling large payload
	data, err := proto.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal large payload: %v", err)
	}

	// Verify it can be unmarshaled
	unmarshaled := &DocumentQueuePayload{}
	if err := proto.Unmarshal(data, unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal large payload: %v", err)
	}

	if len(unmarshaled.Text) != len(payload.Text) {
		t.Errorf("Text length after unmarshal = %d, want %d",
			len(unmarshaled.Text), len(payload.Text))
	}
}

// TestErrorMessageField tests error message persistence
func TestErrorMessageField(t *testing.T) {
	payload := NewDocumentQueuePayload(1, 100, "Test", "it", map[string]string{}, 3)

	// Initially empty
	if payload.ErrorMessage != "" {
		t.Errorf("Initial ErrorMessage = %q, want empty", payload.ErrorMessage)
	}

	// Set error message
	errorMsg := "Connection timeout after 30s"
	payload.ErrorMessage = errorMsg
	payload.Status = MessageStatus_FAILED

	// Marshal and unmarshal to verify persistence
	data, err := proto.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	unmarshaled := &DocumentQueuePayload{}
	if err := proto.Unmarshal(data, unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.ErrorMessage != errorMsg {
		t.Errorf("ErrorMessage = %q, want %q", unmarshaled.ErrorMessage, errorMsg)
	}
	if unmarshaled.Status != MessageStatus_FAILED {
		t.Errorf("Status = %v, want FAILED", unmarshaled.Status)
	}
}

// TestExtraMapHandling tests edge cases with Extra map
func TestExtraMapHandling(t *testing.T) {
	tests := []struct {
		name  string
		extra map[string]string
	}{
		{
			name:  "nil map",
			extra: nil,
		},
		{
			name:  "empty map",
			extra: map[string]string{},
		},
		{
			name:  "single entry",
			extra: map[string]string{"key": "value"},
		},
		{
			name: "multiple entries",
			extra: map[string]string{
				"url":    "https://example.com",
				"title":  "Test Page",
				"author": "John Doe",
			},
		},
		{
			name: "empty values",
			extra: map[string]string{
				"key1": "",
				"key2": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := NewDocumentQueuePayload(1, 100, "Test", "auto", tt.extra, 3)

			// Marshal and unmarshal
			data, err := proto.Marshal(payload)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			unmarshaled := &DocumentQueuePayload{}
			if err := proto.Unmarshal(data, unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify map size
			if len(unmarshaled.Extra) != len(payload.Extra) {
				t.Errorf("Extra map size = %d, want %d",
					len(unmarshaled.Extra), len(payload.Extra))
			}

			// Verify all entries
			for k, v := range payload.Extra {
				if unmarshaled.Extra[k] != v {
					t.Errorf("Extra[%s] = %s, want %s", k, unmarshaled.Extra[k], v)
				}
			}
		})
	}
}

func TestLanguageFieldRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		language string
	}{
		{name: "auto language", language: "auto"},
		{name: "explicit language", language: "en"},
		{name: "empty language", language: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := NewDocumentQueuePayload(10, 20, "hello", tt.language, map[string]string{"source": "test"}, 2)

			data, err := proto.Marshal(payload)
			if err != nil {
				t.Fatalf("Failed to marshal payload: %v", err)
			}

			unmarshaled := &DocumentQueuePayload{}
			if err := proto.Unmarshal(data, unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal payload: %v", err)
			}

			if unmarshaled.Language != tt.language {
				t.Errorf("Language = %q, want %q", unmarshaled.Language, tt.language)
			}
		})
	}
}

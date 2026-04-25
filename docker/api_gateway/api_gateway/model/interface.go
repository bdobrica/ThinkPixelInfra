package model

// Metadata holds metadata for a text item, including an integer ID and optional extra fields.
// The Extra map may contain keys such as "Offset" (string representation of an integer offset).
type Metadata struct {
	ID    int               `json:"id"`              // Unique identifier for the item
	Extra map[string]string `json:"extra,omitempty"` // Additional metadata (e.g., "Offset")
}

// TextItem represents a text chunk and its associated metadata.
type TextItem struct {
	Text     string   `json:"text"`     // The text content
	Metadata Metadata `json:"metadata"` // Associated metadata
}

// InferenceRequest is the request payload for the model API.
type InferenceRequest struct {
	TextItems    []TextItem `json:"text_items"`    // List of text items to process
	Language     string     `json:"language"`      // Language of the text (e.g., "auto", "en", "es")
	Mode         string     `json:"mode"`          // Mode of operation (e.g., "store", "search")
	ChunkSize    int        `json:"chunk_size"`    // Size of text chunks
	ChunkOverlap int        `json:"chunk_overlap"` // Overlap between chunks
}

// EmbeddingsItem represents a single result from the model API, with encoded vectors and metadata.
type EmbeddingsItem struct {
	Text         string            `json:"text"`          // The text content
	Offset       int               `json:"offset"`        // Offset of the chunk in the original text
	DenseVector  string            `json:"dense_vector"`  // Base64-encoded dense vector
	SparseVector map[string]string `json:"sparse_vector"` // Base64-encoded sparse vector (key: base64 uint32, value: base64 float32)
	Metadata     Metadata          `json:"metadata"`      // Associated metadata
}

// InferenceResponse is the response from the model API.
type InferenceResponse struct {
	Results []EmbeddingsItem `json:"results"` // List of embedding results
	Latency float64          `json:"latency"` // Latency in seconds
}

// EmbeddingResponse is the decoded embedding result, with numeric vectors and offset.
type EmbeddingResponse struct {
	ID           int             `json:"id"`            // Unique identifier for the item
	Text         string          `json:"text"`          // The text content
	Offset       int             `json:"offset"`        // Offset of the chunk in the original text
	DenseVector  []float32       `json:"dense_vector"`  // Decoded dense vector
	SparseVector map[int]float32 `json:"sparse_vector"` // Decoded sparse vector (key: int, value: float32)
}

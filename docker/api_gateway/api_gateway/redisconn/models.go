package redisconn

import "time"

// Document represents the structure of a document stored in Redis.
type Document struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Title     string    `json:"title"`
	Cat       []string  `json:"cat"`
	Tag       []string  `json:"tag"`
	Chunks    []int     `json:"chunks"`
	Timestamp time.Time `json:"timestamp"`
}

// Embedding represents the structure of an embedding stored in Redis.
type Embedding struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Offset    int       `json:"offset"`
	Embedding []byte    `json:"embedding"`
	Timestamp time.Time `json:"timestamp"`
}

package storage

import (
	"sync"

	"snap_store/parser"
)

// Storage represents the in-memory storage for parsed documents.
type Storage struct {
	mu        sync.Mutex
	documents map[int]parser.ParsedDocument
}

// NewStorage initializes a new Storage instance.
func NewStorage() *Storage {
	return &Storage{
		documents: make(map[int]parser.ParsedDocument),
	}
}

// UpsertDocuments inserts or updates parsed documents.
func (s *Storage) UpsertDocuments(docs []parser.ParsedDocument) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, doc := range docs {
		s.documents[doc.ID] = doc
	}
}

// GetAllDocuments returns all parsed documents.
func (s *Storage) GetAllDocuments() []parser.ParsedDocument {
	s.mu.Lock()
	defer s.mu.Unlock()
	allDocs := make([]parser.ParsedDocument, 0, len(s.documents))
	for _, doc := range s.documents {
		allDocs = append(allDocs, doc)
	}
	return allDocs
}

// MergeDocuments merges incoming documents.
func (s *Storage) MergeDocuments(newDocs []parser.ParsedDocument) {
	s.UpsertDocuments(newDocs)
}

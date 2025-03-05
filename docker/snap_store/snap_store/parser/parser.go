package parser

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"snap_store/bucket"
	"snap_store/logger"
)

// LogEntry represents a single JSONL log entry.
type LogEntry struct {
	Date       string              `json:"date"`
	Timestamp  string              `json:"timestamp"`
	Method     string              `json:"method"`
	URL        string              `json:"url"`
	Headers    map[string][]string `json:"headers"`
	RemoteAddr string              `json:"remote_addr"`
	Body       string              `json:"body"`
}

// Document represents the JSON object embedded in LogEntry.Body.
type Document struct {
	ID    int    `json:"id"`
	Text  string `json:"text"`
	Extra struct {
		Title string `json:"title"`
		Type  string `json:"type"`
		// other fields ignored
	} `json:"extra"`
}

// ParsedDocument is the flattened record for Parquet storage.
type ParsedDocument struct {
	ID        int    `json:"id" parquet:"name=id, type=INT64"`
	Text      string `json:"text" parquet:"name=text, type=BYTE_ARRAY, convertedtype=UTF8"`
	Title     string `json:"title" parquet:"name=title, type=BYTE_ARRAY, convertedtype=UTF8"`
	Type      string `json:"type" parquet:"name=type, type=BYTE_ARRAY, convertedtype=UTF8"`
	Timestamp string `json:"timestamp" parquet:"name=timestamp, type=BYTE_ARRAY, convertedtype=UTF8"`
}

// DetectPodNames inspects the bucket structure (using a given prefix) to extract pod names.
func DetectPodNames(ctx context.Context, bc *bucket.BucketClient) ([]string, error) {
	// Assuming logs are stored as: logs/<cluster-id>/<pod-name>/...
	prefix := fmt.Sprintf("logs/%s/", bc.ClusterID)
	return bc.ListPodNames(ctx, prefix)
}

// ParseLogs fetches and parses logs from the bucket and returns flattened records.
func ParseLogs(ctx context.Context, bc *bucket.BucketClient, prefix string) ([]ParsedDocument, error) {
	var allDocs []ParsedDocument
	var wg sync.WaitGroup
	docCh := make(chan []ParsedDocument)

	// Collector goroutine
	go func() {
		for docs := range docCh {
			allDocs = append(allDocs, docs...)
		}
	}()

	objects, err := bc.ListObjects(prefix)
	if err != nil {
		logger.Errorf("Error listing objects: %v", err)
		return nil, err
	}

	for _, object := range objects {
		wg.Add(1)
		go func(objectInfo bucket.ObjectInfo) {
			defer wg.Done()

			obj, err := bc.GetObject(ctx, objectInfo.Key)
			if err != nil {
				logger.Errorf("Error fetching object %v: %v", objectInfo.Key, err)
				return
			}

			scanner := bufio.NewScanner(obj)
			var docsFromFile []ParsedDocument
			for scanner.Scan() {
				var entry LogEntry
				readBytes := scanner.Bytes()
				if len(readBytes) == 0 {
					logger.Infof("Empty line in log entry; skipping")
					continue
				}

				if err := json.Unmarshal(readBytes, &entry); err != nil {
					logger.Errorf("Error unmarshaling log entry from %s: %v", objectInfo.Key, err)
					continue
				}

				// Check if the body is empty
				if entry.Body == "" {
					logger.Debugf("Empty body in log entry from %s; skipping", objectInfo.Key)
					continue
				}

				var docs []Document
				if err := json.Unmarshal([]byte(entry.Body), &docs); err != nil {
					logger.Errorf("Error unmarshaling body field from %s: %v", objectInfo.Key, err)
					continue
				}

				// Transform each Document into a ParsedDocument that also holds the Timestamp.
				for _, doc := range docs {
					parsed := ParsedDocument{
						ID:        doc.ID,
						Text:      doc.Text,
						Title:     doc.Extra.Title,
						Type:      doc.Extra.Type,
						Timestamp: entry.Timestamp,
					}
					docsFromFile = append(docsFromFile, parsed)
				}
			}
			if err := scanner.Err(); err != nil {
				logger.Errorf("Scanner error for %s: %v", objectInfo.Key, err)
			}
			docCh <- docsFromFile
		}(object)
	}

	wg.Wait()
	close(docCh)

	logger.Infof("Parsed %d documents from logs.", len(allDocs))
	return allDocs, nil
}

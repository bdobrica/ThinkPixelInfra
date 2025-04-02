package model

import (
	"regexp"
	"strconv"
	"strings"
)

func addOffsetToMetadata(metadata Metadata, offset int) Metadata {
	newMetadata := Metadata{
		ID:    metadata.ID,
		Extra: make(map[string]string),
	}
	for k, v := range metadata.Extra {
		newMetadata.Extra[k] = v
	}
	newMetadata.Extra["Offset"] = strconv.Itoa(offset)
	return newMetadata
}

// splitText function splits a given text into chunks based on chunkSize and chunkOverlap
func splitText(textItem TextItem, chunkSize, chunkOverlap int) []TextItem {
	// Define sentence boundary characters
	sentenceDelimiters := regexp.MustCompile(`[.!?\n]`)
	wordsDelimiters := regexp.MustCompile(`\s+`)

	// Split text into sentences
	var validSentences []string
	var sentenceOffsets []int

	// Track offset of each sentence
	start := 0
	for _, sentence := range sentenceDelimiters.Split(textItem.Text, -1) {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) >= 3 {
			validSentences = append(validSentences, sentence)
			sentenceOffsets = append(sentenceOffsets, start)
		}
		start += len(sentence) + 1 // Account for delimiter
	}

	var chunks []TextItem

	metadata := textItem.Metadata
	if metadata.Extra == nil {
		metadata.Extra = make(map[string]string)
	}

	for i, sentence := range validSentences {
		sentenceStart := sentenceOffsets[i]
		if len(sentence) <= chunkSize {
			// If sentence is within the limit, add it as a chunk
			chunks = append(chunks, TextItem{sentence, addOffsetToMetadata(metadata, sentenceStart)})
		} else {
			// Break down large sentences into smaller chunks
			start := 0
			for start < len(sentence) {
				if start+chunkSize >= len(sentence) {
					chunks = append(chunks, TextItem{sentence[start:], addOffsetToMetadata(metadata, sentenceStart+start)})
					break
				}
				// Find first word boundary before chunkSize
				end := start + chunkSize
				if end < len(sentence) {
					// Move end back to the nearest space
					for end > start && !wordsDelimiters.MatchString(string(sentence[end])) {
						end--
					}
				}
				chunks = append(chunks, TextItem{sentence[start:end], addOffsetToMetadata(metadata, sentenceStart+start)})

				// Move back by chunkOverlap and find a word boundary
				start = end - chunkOverlap
				if start < 0 {
					start = 0
				}
				for start > 0 && !wordsDelimiters.MatchString(string(sentence[start])) {
					start--
				}
			}
		}
	}

	return chunks
}

func splitTextItems(textItems []TextItem, chunkSize, chunkOverlap int) []TextItem {
	var allChunks []TextItem

	for _, item := range textItems {
		// Split the current TextItem into chunks
		chunks := splitText(item, chunkSize, chunkOverlap)

		// Append the chunks to the result list
		allChunks = append(allChunks, chunks...)
	}

	return allChunks
}

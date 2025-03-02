package redisconn

import (
	"api_gateway/logger"
	"api_gateway/model"
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// SearchEmbeddings performs ANN search for a set of embeddings and returns top results
func SearchEmbeddings(siteId int, indexingNode string, embeddings []model.EmbeddingResponse, limit int) ([]map[string]interface{}, error) {
	ctx := context.Background()
	results := []map[string]interface{}{}
	indexName := fmt.Sprintf("index:%d", siteId)
	logger.Debugf("Searching embeddings in index %s for %d items", indexName, limit)

	client, err := getRedisClient(indexingNode)
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis client: %w", err)
	}

	for _, emb := range embeddings {
		embeddingBytes, err := base64.StdEncoding.DecodeString(emb.Embedding)
		if err != nil {
			logger.Errorf("Error decoding embedding: %v", err)
			return nil, fmt.Errorf("failed to decode embedding: %w", err)
		}

		logger.Debugf("Performing search for embedding with size %d bytes", len(embeddingBytes))

		cmd := client.Do(ctx, "FT.SEARCH", indexName,
			fmt.Sprintf("*=>[KNN %d @embedding $query_vec AS score]", limit),
			"PARAMS", "2", "query_vec", embeddingBytes,
			"SORTBY", "score",
			"DIALECT", "2",
			"LIMIT", "0", strconv.Itoa(limit)) // Add LIMIT clause

		searchResults, err := cmd.Result()
		if err != nil {
			logger.Errorf("Error executing search query: %v", err)
			return nil, fmt.Errorf("failed to execute search query: %w", err)
		}

		resultsArray, ok := searchResults.([]interface{})
		if !ok {
			logger.Errorf("Unexpected search result format: %v", searchResults)
			return nil, fmt.Errorf("unexpected search result format")
		}

		// Process results
		for i := 1; i < len(resultsArray); i++ {
			if i%10 == 0 { // Log every 10 results
				logger.Debugf("Processing search result %d", i)
			}
			doc, ok := resultsArray[i].([]interface{})
			if !ok {
				continue
			}

			// Convert sub-array to a map
			docMap := make(map[string]interface{})
			for j := 0; j < len(doc)-1; j += 2 { // Keys are at even indices, values are at odd indices
				key, keyOk := doc[j].(string)
				if !keyOk {
					continue
				}
				value := doc[j+1]

				// Add to map, allowing for different types of values
				docMap[key] = value
			}

			// Convert id to int
			var id int
			if rawID, ok := docMap["id"].(string); ok {
				id, err = strconv.Atoi(rawID)
				if err != nil {
					logger.Errorf("Failed to parse ID as int: %v", err)
					continue
				}
			}

			// Convert score to float64
			var score float64
			if rawScore, ok := docMap["score"].(string); ok {
				score, err = strconv.ParseFloat(rawScore, 64)
				if err != nil {
					logger.Errorf("Failed to parse score as float64: %v", err)
					continue
				}
			}
			if score < 0 {
				score = 0 - score // Negative scores are not allowed
			}
			// Convert score to a percentage
			score = 100 * (1 - score)

			// Add the processed document to results
			results = append(results, map[string]interface{}{
				"id":    id,
				"text":  docMap["text"],
				"score": score,
			})
		}

	}

	// Sort results by score and return top N
	sort.Slice(results, func(i, j int) bool {
		return results[i]["score"].(float64) > results[j]["score"].(float64)
	})

	if len(results) > limit {
		results = results[:limit]
	}

	logger.Debugf("Search completed. Returning %d results", len(results))

	return results, nil
}

// SearchDocuments performs a BM25 search on full documents using the BM25 index.
// It accepts a list of alternative query strings (to allow for multiple query formulations)
// and returns a list of document results (as maps) along with their BM25 score.
func SearchDocuments(siteID int, indexingNode string, queries []string, limit int) ([]map[string]interface{}, error) {
	ctx := context.Background()
	client, err := getRedisClient(indexingNode)
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis client: %w", err)
	}
	bm25Index := fmt.Sprintf("bm25:%d", siteID)

	// Combine the queries using the OR operator.
	// For example, if queries are ["foo", "bar"], the combined query becomes "foo | bar"
	combinedQuery := strings.Join(queries, " | ")

	// Execute the FT.SEARCH command with WITHSCORES so we get a score per document.
	// We also specify DIALECT 2 for RedisJSON.
	searchCmd, err := client.Do(ctx, "FT.SEARCH", bm25Index,
		combinedQuery,
		"WITHSCORES",
		"LIMIT", "0", strconv.Itoa(limit),
		"DIALECT", "2").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to execute BM25 search: %w", err)
	}

	resultsArray, ok := searchCmd.([]interface{})
	if !ok || len(resultsArray) < 1 {
		return nil, fmt.Errorf("unexpected BM25 search result format")
	}

	var results []map[string]interface{}
	// The first element is the total number of results; subsequent elements are documents.
	for i := 1; i < len(resultsArray); i++ {
		// Each document result is an array containing the key, the score, and then key-value pairs.
		docFields, ok := resultsArray[i].([]interface{})
		if !ok || len(docFields) < 2 {
			continue
		}

		// The second element is the score.
		score, err := strconv.ParseFloat(docFields[1].(string), 64)
		if err != nil {
			score = 0
		}

		docMap := make(map[string]interface{})
		docMap["score"] = score

		// Process remaining key-value pairs.
		// (Typically, the document JSON fields returned by RedisJSON.)
		for j := 2; j < len(docFields)-1; j += 2 {
			key, keyOk := docFields[j].(string)
			if !keyOk {
				continue
			}
			docMap[key] = docFields[j+1]
		}

		results = append(results, docMap)
	}

	return results, nil
}

// HybridSearch performs a hybrid search by combining BM25 full-document search
// and semantic embedding search. It accepts a mapping where each key is a query text
// (for the BM25 search) and the corresponding value is the semantic query (an EmbeddingResponse)
// used for the embedding search. The function aggregates results by document (post_id)
// and, for the embedding search, takes the maximum score among the chunks.
// Finally, it combines the two scores (here, by summing them) and returns the top results.
func HybridSearch(siteID int, indexingNode string, queryMapping map[string]model.EmbeddingResponse, limit int) ([]map[string]interface{}, error) {
	// For candidate retrieval, we request more than the final limit.
	candidateLimit := limit * 2

	// Structure to aggregate scores and document details per document.
	type docScores struct {
		bm25Score float64
		embScore  float64
		combined  float64
		doc       map[string]interface{}
	}
	aggregated := make(map[int]docScores)

	// For each query pair (text and corresponding embedding)
	for queryText, queryEmbedding := range queryMapping {
		// BM25 search on full documents using the query text.
		bm25Results, err := SearchDocuments(siteID, indexingNode, []string{queryText}, candidateLimit)
		if err != nil {
			logger.Errorf("BM25 search error for query '%s': %v", queryText, err)
			continue
		}

		// Semantic search on chunks using the embedding.
		embResults, err := SearchEmbeddings(siteID, indexingNode, []model.EmbeddingResponse{queryEmbedding}, candidateLimit)
		if err != nil {
			logger.Errorf("Embedding search error for query '%s': %v", queryText, err)
			continue
		}

		// Process BM25 results.
		for _, res := range bm25Results {
			var docID int
			switch id := res["id"].(type) {
			case float64:
				docID = int(id)
			case string:
				docID, _ = strconv.Atoi(id)
			}
			// Retrieve the BM25 score.
			bm25Score, _ := res["score"].(float64)

			entry, exists := aggregated[docID]
			if !exists || bm25Score > entry.bm25Score {
				entry.bm25Score = bm25Score
				// Save document details; preference is given to BM25 result.
				entry.doc = res
			}
			aggregated[docID] = entry
		}

		// Process embedding search results.
		// Since a document might have multiple chunks, we take the maximum score.
		for _, res := range embResults {
			var docID int
			switch id := res["id"].(type) {
			case float64:
				docID = int(id)
			case string:
				docID, _ = strconv.Atoi(id)
			}
			embScore, _ := res["score"].(float64)

			entry, exists := aggregated[docID]
			if !exists || embScore > entry.embScore {
				entry.embScore = embScore
				// If the document was not retrieved by BM25, you can store the embedding result details.
				if entry.doc == nil {
					entry.doc = res
				}
			}
			aggregated[docID] = entry
		}
	}

	// Combine the scores for each document.
	// Here, we simply add the BM25 score and the embedding score.
	// In production you might normalize or weight them differently.
	var combinedList []docScores
	for _, entry := range aggregated {
		entry.combined = entry.bm25Score + entry.embScore
		combinedList = append(combinedList, entry)
	}

	// Sort documents by the combined score in descending order.
	sort.Slice(combinedList, func(i, j int) bool {
		return combinedList[i].combined > combinedList[j].combined
	})

	// Select the top results.
	var finalResults []map[string]interface{}
	for i, entry := range combinedList {
		if i >= limit {
			break
		}
		// Optionally, you can add the combined score to the returned document.
		entry.doc["combined_score"] = entry.combined
		finalResults = append(finalResults, entry.doc)
	}

	return finalResults, nil
}

// Package milvus provides a k6 extension for load testing Milvus vector databases.
// This file contains search operations and recall calculation functionality.
package milvus

import (
	"fmt"
	"time"

	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// Search with flexible parameters
func (c *Client) Search(collectionName string, vectors [][]float32, topK int, params map[string]interface{}) ([]SearchResult, error) {
	start := time.Now()
	
	searchVectors := make([]entity.Vector, len(vectors))
	for i, v := range vectors {
		searchVectors[i] = entity.FloatVector(v)
	}

	option := milvusclient.NewSearchOption(collectionName, topK, searchVectors)

	// Set vector field name (default to "vector")
	vectorField := "vector"
	if field, ok := params["vectorField"].(string); ok {
		vectorField = field
	}
	option = option.WithANNSField(vectorField)

	// Set output fields
	if outputFields, ok := params["outputFields"].([]string); ok {
		option = option.WithOutputFields(outputFields...)
	} else {
		option = option.WithOutputFields("id")
	}

	// Set filter expression
	if expr, ok := params["expr"].(string); ok {
		option = option.WithFilter(expr)
	}

	// Set search parameters
	if searchParams, ok := params["searchParams"].(map[string]interface{}); ok {
		// Convert search params if needed
		_ = searchParams // placeholder for future search param handling
	}

	searchResult, err := c.client.Search(c.vu.Context(), option)
	
	// Calculate metrics
	duration := time.Since(start)
	tags := map[string]string{
		"operation":  "search",
		"collection": collectionName,
		"topk":       fmt.Sprintf("%d", topK),
	}
	
	if err != nil {
		tags["status"] = "error"
		c.mi.emitMetric(c.mi.metrics.MilvusErrors, 1, tags)
		c.mi.emitMetric(c.mi.metrics.MilvusDuration, float64(duration.Milliseconds()), tags)
		return nil, fmt.Errorf("failed to search: %v", err)
	}

	var results []SearchResult
	resultCount := 0
	for _, result := range searchResult {
		for i := 0; i < result.ResultCount; i++ {
			resultItem := SearchResult{
				Score:  result.Scores[i],
				Fields: make(map[string]interface{}),
			}

			// Get ID from the IDs column
			if idVal, err := result.IDs.Get(i); err == nil {
				resultItem.ID = idVal.(int64)
			}

			// Get other output fields
			if outputFields, ok := params["outputFields"].([]string); ok {
				for _, field := range outputFields {
					if field != "id" {
						if fieldColumn := result.GetColumn(field); fieldColumn != nil {
							if fieldVal, err := fieldColumn.Get(i); err == nil {
								resultItem.Fields[field] = fieldVal
							}
						}
					}
				}
			}

			results = append(results, resultItem)
			resultCount++
		}
	}

	// Emit success metrics
	tags["status"] = "success"
	c.mi.emitMetric(c.mi.metrics.MilvusReqs, 1, tags)
	c.mi.emitMetric(c.mi.metrics.MilvusDuration, float64(duration.Milliseconds()), tags)
	c.mi.emitMetric(c.mi.metrics.MilvusVectors, float64(len(vectors)), tags) // Query vectors
	c.mi.emitMetric(c.mi.metrics.MilvusErrors, 0, tags) // No error

	return results, nil
}

// SearchSimple provides backward compatibility for simple vector search
func (c *Client) SearchSimple(collectionName string, vectors [][]float32, topK int) ([]SearchResult, error) {
	params := map[string]interface{}{
		"vectorField":  "vector",
		"outputFields": []string{"id"},
	}
	return c.Search(collectionName, vectors, topK, params)
}

// SearchWithRecall performs a search and calculates recall if ground truth is provided
// groundTruth should contain the true relevant IDs for each query vector
func (c *Client) SearchWithRecall(collectionName string, vectors [][]float32, topK int, params map[string]interface{}, groundTruth [][]int64) ([]SearchResult, error) {
	// Perform the search
	results, err := c.Search(collectionName, vectors, topK, params)
	
	// Calculate and emit recall metric if ground truth is provided
	if err == nil && groundTruth != nil && len(groundTruth) > 0 {
		recall := calculateRecall(results, groundTruth, topK, len(vectors))
		
		// Emit recall metric
		tags := map[string]string{
			"operation":  "search_with_recall",
			"collection": collectionName,
			"topk":       fmt.Sprintf("%d", topK),
		}
		c.mi.emitMetric(c.mi.metrics.MilvusRecall, recall, tags)
	}
	
	return results, err
}

// calculateRecall computes recall@K for search results
// recall@K = (number of relevant items retrieved in top-K) / (total number of relevant items)
func calculateRecall(results []SearchResult, groundTruth [][]int64, topK int, numQueries int) float64 {
	if len(results) == 0 || len(groundTruth) == 0 || numQueries == 0 {
		return 0.0
	}
	
	totalRecall := 0.0
	validQueries := 0
	
	// For each query, calculate recall
	resultIdx := 0
	for queryIdx := 0; queryIdx < numQueries && queryIdx < len(groundTruth); queryIdx++ {
		if len(groundTruth[queryIdx]) == 0 {
			continue
		}
		
		// Create set of ground truth IDs for this query
		truthSet := make(map[int64]bool)
		for _, id := range groundTruth[queryIdx] {
			truthSet[id] = true
		}
		
		// Count how many retrieved results are in ground truth
		retrieved := 0
		maxResults := topK
		if resultIdx+topK > len(results) {
			maxResults = len(results) - resultIdx
		}
		
		for i := 0; i < maxResults && resultIdx+i < len(results); i++ {
			if truthSet[results[resultIdx+i].ID] {
				retrieved++
			}
		}
		
		// Calculate recall for this query
		queryRecall := float64(retrieved) / float64(len(groundTruth[queryIdx]))
		totalRecall += queryRecall
		validQueries++
		
		resultIdx += topK
	}
	
	// Return average recall across all queries
	if validQueries == 0 {
		return 0.0
	}
	return totalRecall / float64(validQueries)
}
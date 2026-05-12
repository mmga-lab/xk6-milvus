package milvus

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// Search performs vector similarity search with Recall support.
// The vectorsInput parameter accepts dense vectors ([][]float32), text queries ([]string for BM25),
// or sparse vectors. Type detection is automatic.
func (c *Client) Search(vectorsInput interface{}, topK int, params map[string]interface{}, collectionName ...string) interface{} {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		})
	}

	// Convert input to entity.Vector — supports dense, sparse, and text (BM25)
	searchVectors, err := convertToSearchVectors(vectorsInput)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to convert search vectors: %v", err),
		})
	}

	// Get vector field name (default to "vector")
	vectorField := "vector"
	if field, ok := params["vectorField"].(string); ok {
		vectorField = field
	}

	// Get output fields
	var outputFields []string
	if fields, ok := params["outputFields"].([]interface{}); ok {
		outputFields = make([]string, len(fields))
		for i, field := range fields {
			if fieldStr, ok := field.(string); ok {
				outputFields[i] = fieldStr
			}
		}
	} else if fields, ok := params["outputFields"].([]string); ok {
		outputFields = fields
	}

	if len(outputFields) == 0 {
		outputFields = []string{"id"}
	}

	// Create search option
	searchOption := milvusclient.NewSearchOption(coll, topK, searchVectors).
		WithANNSField(vectorField).
		WithOutputFields(outputFields...)

	// Set filter expression
	if expr, ok := params["expr"].(string); ok && expr != "" {
		searchOption = searchOption.WithFilter(expr)
	}

	// Set metric type through search param
	if metricType, ok := params["metricType"].(string); ok {
		searchOption = searchOption.WithSearchParam("metric_type", metricType)
	}

	// Execute search
	resultSets, err := c.client.Search(c.context(), searchOption)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to search: %v", err),
		})
	}

	// Convert results with pre-allocated capacity
	var results []SearchResult
	var recall float32
	isEmpty := true

	// Pre-allocate with estimated capacity
	totalResults := 0
	for _, resultSet := range resultSets {
		totalResults += resultSet.ResultCount
	}
	if totalResults > 0 {
		results = make([]SearchResult, 0, totalResults)
	}

	for _, resultSet := range resultSets {
		if resultSet.ResultCount > 0 {
			isEmpty = false
		}
		recall = resultSet.Recall // Capture recall from SDK

		for i := 0; i < resultSet.ResultCount; i++ {
			result := SearchResult{
				Score:  resultSet.Scores[i],
				Fields: make(map[string]interface{}),
			}

			// Get ID
			if idVal, err := resultSet.IDs.Get(i); err == nil {
				if id, ok := idVal.(int64); ok {
					result.ID = id
				}
			}

			// Get other fields
			for _, field := range outputFields {
				if field != "id" && field != "" {
					if fieldColumn := resultSet.GetColumn(field); fieldColumn != nil {
						if fieldVal, err := fieldColumn.Get(i); err == nil {
							result.Fields[field] = fieldVal
						}
					}
				}
			}

			results = append(results, result)
		}
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       results,
		Empty:        isEmpty,
		Recall:       recall, // NEW: Expose recall metric
	})
}

// HybridSearch performs multi-vector hybrid search with reranking (NEW - from Locust)
func (c *Client) HybridSearch(requestsInput interface{}, rerankerInput interface{}, limit int, outputFields []interface{}, collectionName ...string) interface{} {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		})
	}

	// Convert interface{} to []HybridSearchRequest using JSON marshal/unmarshal
	var requests []HybridSearchRequest
	requestsBytes, err := json.Marshal(requestsInput)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to marshal requests: %v", err),
		})
	}
	err = json.Unmarshal(requestsBytes, &requests)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to unmarshal requests: %v", err),
		})
	}

	// Convert interface{} to Reranker using JSON marshal/unmarshal
	var reranker Reranker
	rerankerBytes, err := json.Marshal(rerankerInput)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to marshal reranker: %v", err),
		})
	}
	err = json.Unmarshal(rerankerBytes, &reranker)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to unmarshal reranker: %v", err),
		})
	}

	if len(requests) == 0 {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "at least one search request required",
		})
	}

	// Build ANN requests
	var annRequests []*milvusclient.AnnRequest
	for _, req := range requests {
		// Use the shared convertToSearchVectors for dense, sparse, and text (BM25)
		searchVectors, err := convertToSearchVectors(req.Vectors)
		if err != nil || len(searchVectors) == 0 {
			errMsg := "unknown format"
			if err != nil {
				errMsg = err.Error()
			}
			return toMap(&OperationResult{
				Success:      false,
				ResponseTime: float64(time.Since(start).Milliseconds()),
				Error:        fmt.Sprintf("failed to parse vectors for field %s: %s", req.VectorField, errMsg),
			})
		}

		annReq := milvusclient.NewAnnRequest(req.VectorField, req.Limit, searchVectors...)

		// Apply params if provided
		if req.Params != nil {
			if expr, ok := req.Params["expr"].(string); ok && expr != "" {
				annReq = annReq.WithFilter(expr)
			}
			if metricType, ok := req.Params["metricType"].(string); ok {
				annReq = annReq.WithSearchParam("metric_type", metricType)
			}
		}

		annRequests = append(annRequests, annReq)
	}

	// Convert output fields
	fields := make([]string, len(outputFields))
	for i, field := range outputFields {
		if fieldStr, ok := field.(string); ok {
			fields[i] = fieldStr
		}
	}

	if len(fields) == 0 {
		fields = []string{"id"}
	}

	// Create hybrid search option
	hybridOption := milvusclient.NewHybridSearchOption(coll, limit, annRequests...).
		WithOutputFields(fields...)

	// Set reranker
	switch reranker.Type {
	case "rrf":
		rrfReranker := milvusclient.NewRRFReranker()
		if k, ok := reranker.Params["k"].(float64); ok {
			rrfReranker = rrfReranker.WithK(k)
		}
		hybridOption = hybridOption.WithReranker(rrfReranker)
	case "weighted":
		var weights []float64
		if w, ok := reranker.Params["weights"].([]interface{}); ok {
			weights = make([]float64, len(w))
			for i, weight := range w {
				if wf, ok := weight.(float64); ok {
					weights[i] = wf
				}
			}
		}
		if len(weights) > 0 {
			hybridOption = hybridOption.WithReranker(milvusclient.NewWeightedReranker(weights))
		}
	default:
		// Default to RRF
		hybridOption = hybridOption.WithReranker(milvusclient.NewRRFReranker())
	}

	// Execute hybrid search
	resultSets, err := c.client.HybridSearch(c.context(), hybridOption)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to hybrid search: %v", err),
		})
	}

	// Convert results with pre-allocated capacity
	var results []SearchResult
	var recall float32
	isEmpty := true

	// Pre-allocate with estimated capacity
	totalResults := 0
	for _, resultSet := range resultSets {
		totalResults += resultSet.ResultCount
	}
	if totalResults > 0 {
		results = make([]SearchResult, 0, totalResults)
	}

	for _, resultSet := range resultSets {
		if resultSet.ResultCount > 0 {
			isEmpty = false
		}
		recall = resultSet.Recall

		for i := 0; i < resultSet.ResultCount; i++ {
			result := SearchResult{
				Score:  resultSet.Scores[i],
				Fields: make(map[string]interface{}),
			}

			// Get ID
			if idVal, err := resultSet.IDs.Get(i); err == nil {
				if id, ok := idVal.(int64); ok {
					result.ID = id
				}
			}

			// Get other fields
			for _, field := range fields {
				if field != "id" && field != "" {
					if fieldColumn := resultSet.GetColumn(field); fieldColumn != nil {
						if fieldVal, err := fieldColumn.Get(i); err == nil {
							result.Fields[field] = fieldVal
						}
					}
				}
			}

			results = append(results, result)
		}
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       results,
		Empty:        isEmpty,
		Recall:       recall,
	})
}

// Query performs scalar query without vectors (NEW - from Locust)
func (c *Client) Query(filter string, outputFields []interface{}, collectionName ...string) interface{} {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		})
	}

	// Convert outputFields
	fields := make([]string, len(outputFields))
	for i, field := range outputFields {
		if fieldStr, ok := field.(string); ok {
			fields[i] = fieldStr
		}
	}

	if len(fields) == 0 {
		fields = []string{"id"}
	}

	option := milvusclient.NewQueryOption(coll).
		WithFilter(filter).
		WithOutputFields(fields...)

	resultSet, err := c.client.Query(c.context(), option)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to query: %v", err),
		})
	}

	// Convert results with pre-allocated capacity
	isEmpty := resultSet.ResultCount == 0
	results := make([]QueryResult, 0, resultSet.ResultCount)

	for i := 0; i < resultSet.ResultCount; i++ {
		result := QueryResult{
			Fields: make(map[string]interface{}),
		}

		for _, field := range fields {
			if fieldColumn := resultSet.GetColumn(field); fieldColumn != nil {
				if fieldVal, err := fieldColumn.Get(i); err == nil {
					result.Fields[field] = fieldVal
				}
			}
		}

		results = append(results, result)
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       results,
		Empty:        isEmpty,
	})
}

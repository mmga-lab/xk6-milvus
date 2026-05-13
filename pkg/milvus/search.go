package milvus

import (
	"encoding/json"
	"fmt"
	"strconv"
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
	if expr, ok := stringOption(params, "expr"); ok && expr != "" {
		searchOption = searchOption.WithFilter(expr)
	} else if expr, ok := stringOption(params, "filter"); ok && expr != "" {
		searchOption = searchOption.WithFilter(expr)
	}

	// Set metric type through search param
	if metricType, ok := stringOption(params, "metricType"); ok {
		searchOption = searchOption.WithSearchParam("metric_type", metricType)
	}
	if metricType, ok := stringOption(params, "metric_type"); ok {
		searchOption = searchOption.WithSearchParam("metric_type", metricType)
	}
	if offset, ok := intOption(params, "offset"); ok {
		searchOption = searchOption.WithOffset(offset)
	}
	if groupBy, ok := stringOption(params, "groupByField"); ok && groupBy != "" {
		searchOption = searchOption.WithGroupByField(groupBy)
	} else if groupBy, ok := stringOption(params, "groupingField"); ok && groupBy != "" {
		searchOption = searchOption.WithGroupByField(groupBy)
	}
	if groupSize, ok := intOption(params, "groupSize"); ok {
		searchOption = searchOption.WithGroupSize(groupSize)
	}
	if strict, ok := boolOption(params, "strictGroupSize"); ok {
		searchOption = searchOption.WithStrictGroupSize(strict)
	}
	if ignoreGrowing, ok := boolOption(params, "ignoreGrowing"); ok {
		searchOption = searchOption.WithIgnoreGrowing(ignoreGrowing)
	}
	for key, val := range searchParamMap(params) {
		searchOption = searchOption.WithSearchParam(key, searchParamValue(val))
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
			if resultSet.GroupByValue != nil {
				if groupByVal, err := resultSet.GroupByValue.Get(i); err == nil {
					result.GroupByValue = groupByVal
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
			if expr, ok := stringOption(req.Params, "expr"); ok && expr != "" {
				annReq = annReq.WithFilter(expr)
			} else if expr, ok := stringOption(req.Params, "filter"); ok && expr != "" {
				annReq = annReq.WithFilter(expr)
			}
			if metricType, ok := stringOption(req.Params, "metricType"); ok {
				annReq = annReq.WithSearchParam("metric_type", metricType)
			}
			if metricType, ok := stringOption(req.Params, "metric_type"); ok {
				annReq = annReq.WithSearchParam("metric_type", metricType)
			}
			if offset, ok := intOption(req.Params, "offset"); ok {
				annReq = annReq.WithOffset(offset)
			}
			if groupBy, ok := stringOption(req.Params, "groupByField"); ok && groupBy != "" {
				annReq = annReq.WithGroupByField(groupBy)
			} else if groupBy, ok := stringOption(req.Params, "groupingField"); ok && groupBy != "" {
				annReq = annReq.WithGroupByField(groupBy)
			}
			if groupSize, ok := intOption(req.Params, "groupSize"); ok {
				annReq = annReq.WithGroupSize(groupSize)
			}
			if strict, ok := boolOption(req.Params, "strictGroupSize"); ok {
				annReq = annReq.WithStrictGroupSize(strict)
			}
			if ignoreGrowing, ok := boolOption(req.Params, "ignoreGrowing"); ok {
				annReq = annReq.WithIgnoreGrowing(ignoreGrowing)
			}
			for key, val := range searchParamMap(req.Params) {
				annReq = annReq.WithSearchParam(key, searchParamValue(val))
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
			if resultSet.GroupByValue != nil {
				if groupByVal, err := resultSet.GroupByValue.Get(i); err == nil {
					result.GroupByValue = groupByVal
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
func (c *Client) Query(filter string, outputFields []interface{}, args ...interface{}) interface{} {
	start := time.Now()

	coll, options := c.parseQueryArgs(args...)
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
	if limit, ok := intOption(options, "limit"); ok {
		option = option.WithLimit(limit)
	}
	if offset, ok := intOption(options, "offset"); ok {
		option = option.WithOffset(offset)
	}

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

func (c *Client) parseQueryArgs(args ...interface{}) (string, map[string]interface{}) {
	coll := c.defaultCollection
	options := make(map[string]interface{})
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			if v != "" {
				coll = v
			}
		case map[string]interface{}:
			for key, val := range v {
				options[key] = val
			}
			if name, ok := stringOption(v, "collectionName"); ok && name != "" {
				coll = name
			}
		}
	}
	return coll, options
}

func searchParamMap(params map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	if nested, ok := params["params"].(map[string]interface{}); ok {
		for key, val := range nested {
			result[key] = val
		}
	}
	reserved := map[string]struct{}{
		"vectorField":      {},
		"outputFields":     {},
		"expr":             {},
		"filter":           {},
		"metricType":       {},
		"metric_type":      {},
		"params":           {},
		"offset":           {},
		"groupByField":     {},
		"groupingField":    {},
		"groupSize":        {},
		"strictGroupSize":  {},
		"ignoreGrowing":    {},
		"collectionName":   {},
		"partitionNames":   {},
		"consistencyLevel": {},
	}
	for key, val := range params {
		if _, ok := reserved[key]; ok {
			continue
		}
		result[key] = val
	}
	return result
}

func searchParamValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		if data, err := json.Marshal(v); err == nil {
			return string(data)
		}
		return fmt.Sprint(v)
	}
}

func stringOption(options map[string]interface{}, key string) (string, bool) {
	value, ok := options[key]
	if !ok || value == nil {
		return "", false
	}
	switch v := value.(type) {
	case string:
		return v, true
	default:
		return fmt.Sprint(v), true
	}
}

func intOption(options map[string]interface{}, key string) (int, bool) {
	value, ok := options[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		i, err := strconv.Atoi(v)
		return i, err == nil
	default:
		return 0, false
	}
}

func boolOption(options map[string]interface{}, key string) (bool, bool) {
	value, ok := options[key]
	if !ok || value == nil {
		return false, false
	}
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		b, err := strconv.ParseBool(v)
		return b, err == nil
	default:
		return false, false
	}
}

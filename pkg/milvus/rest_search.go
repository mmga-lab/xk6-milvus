package milvus

import (
	"encoding/json"
	"fmt"
)

// Search performs vector similarity search via REST API
func (rc *RestClient) Search(vectors [][]float32, topK int, params map[string]interface{}, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	body := rc.baseBody(coll)
	body["data"] = vectors
	body["limit"] = topK

	if params != nil {
		if field, ok := params["vectorField"].(string); ok {
			body["annsField"] = field
		}
		if mt, ok := params["metricType"].(string); ok {
			body["metricType"] = mt
		}
		if expr, ok := params["expr"].(string); ok && expr != "" {
			body["filter"] = expr
		}
		if fields, ok := params["outputFields"]; ok {
			switch f := fields.(type) {
			case []interface{}:
				strs := make([]string, len(f))
				for i, v := range f {
					if s, ok := v.(string); ok {
						strs[i] = s
					}
				}
				body["outputFields"] = strs
			case []string:
				body["outputFields"] = f
			}
		}
		if offset, ok := params["offset"]; ok {
			body["offset"] = offset
		}
		if gf, ok := params["groupingField"].(string); ok {
			body["groupingField"] = gf
		}
		if sp, ok := params["params"]; ok {
			body["searchParams"] = sp
		}
	}

	rawData, elapsed, err := rc.post("/entities/search", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var results []interface{}
	json.Unmarshal(rawData, &results)

	isEmpty := len(results) == 0

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: elapsed,
		Result:       results,
		Empty:        isEmpty,
		Recall:       0,
	})
}

// Query performs scalar query without vectors via REST API
func (rc *RestClient) Query(filter string, outputFields []interface{}, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	body := rc.baseBody(coll)
	body["filter"] = filter

	if len(outputFields) > 0 {
		fields := make([]string, len(outputFields))
		for i, f := range outputFields {
			if s, ok := f.(string); ok {
				fields[i] = s
			}
		}
		body["outputFields"] = fields
	}

	rawData, elapsed, err := rc.post("/entities/query", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var results []interface{}
	json.Unmarshal(rawData, &results)

	isEmpty := len(results) == 0

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: elapsed,
		Result:       results,
		Empty:        isEmpty,
	})
}

// HybridSearch performs multi-vector hybrid search via REST API
func (rc *RestClient) HybridSearch(requestsInput interface{}, rerankerInput interface{}, limit int, outputFields []interface{}, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	// Convert requestsInput to []HybridSearchRequest
	var requests []HybridSearchRequest
	requestsBytes, err := json.Marshal(requestsInput)
	if err != nil {
		return errorResult(0, fmt.Sprintf("failed to marshal requests: %v", err))
	}
	if err := json.Unmarshal(requestsBytes, &requests); err != nil {
		return errorResult(0, fmt.Sprintf("failed to unmarshal requests: %v", err))
	}

	// Convert rerankerInput to Reranker
	var reranker Reranker
	rerankerBytes, err := json.Marshal(rerankerInput)
	if err != nil {
		return errorResult(0, fmt.Sprintf("failed to marshal reranker: %v", err))
	}
	if err := json.Unmarshal(rerankerBytes, &reranker); err != nil {
		return errorResult(0, fmt.Sprintf("failed to unmarshal reranker: %v", err))
	}

	if len(requests) == 0 {
		return errorResult(0, ErrNoSearchRequests.Error())
	}

	body := rc.baseBody(coll)

	// Build search array for REST API
	searchArr := make([]map[string]interface{}, 0, len(requests))
	for _, req := range requests {
		s := map[string]interface{}{
			"data":      req.Vectors,
			"annsField": req.VectorField,
			"limit":     req.Limit,
		}
		if req.Params != nil {
			if mt, ok := req.Params["metricType"].(string); ok {
				s["metricType"] = mt
			}
			if expr, ok := req.Params["expr"].(string); ok && expr != "" {
				s["filter"] = expr
			}
		}
		searchArr = append(searchArr, s)
	}
	body["search"] = searchArr

	// Build rerank
	body["rerank"] = map[string]interface{}{
		"strategy": reranker.Type,
		"params":   reranker.Params,
	}

	body["limit"] = limit

	// Output fields
	if len(outputFields) > 0 {
		fields := make([]string, len(outputFields))
		for i, f := range outputFields {
			if s, ok := f.(string); ok {
				fields[i] = s
			}
		}
		body["outputFields"] = fields
	}

	rawData, elapsed, err := rc.post("/entities/hybrid_search", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var results []interface{}
	json.Unmarshal(rawData, &results)

	isEmpty := len(results) == 0

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: elapsed,
		Result:       results,
		Empty:        isEmpty,
		Recall:       0,
	})
}

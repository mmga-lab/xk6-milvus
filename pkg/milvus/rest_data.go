package milvus

import (
	"encoding/json"
	"fmt"
)

// columnsToRows converts column-based data to row-based data for REST API
// Input: {"field1": [v1, v2], "field2": [v1, v2]}
// Output: [{"field1": v1, "field2": v1}, {"field1": v2, "field2": v2}]
func columnsToRows(data map[string]interface{}) ([]map[string]interface{}, error) {
	// Find the length from the first array field
	length := 0
	for _, v := range data {
		if arr, ok := v.([]interface{}); ok {
			length = len(arr)
			break
		}
	}
	if length == 0 {
		return nil, fmt.Errorf("no valid array data found")
	}

	rows := make([]map[string]interface{}, length)
	for i := 0; i < length; i++ {
		rows[i] = make(map[string]interface{})
	}

	for key, val := range data {
		arr, ok := val.([]interface{})
		if !ok {
			return nil, fmt.Errorf("field %s is not an array", key)
		}
		if len(arr) != length {
			return nil, fmt.Errorf("field %s has length %d, expected %d", key, len(arr), length)
		}
		for i, v := range arr {
			rows[i][key] = v
		}
	}

	return rows, nil
}

// Insert inserts data into a collection via REST API
// Supports both column-based (map[string]interface{}) and row-based ([]interface{}) data
func (rc *RestClient) Insert(data map[string]interface{}, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	body := rc.baseBody(coll)

	// Convert column-based data to row-based
	rows, err := columnsToRows(data)
	if err != nil {
		return errorResult(0, fmt.Sprintf("failed to convert data: %v", err))
	}
	body["data"] = rows

	rawData, elapsed, err := rc.post("/entities/insert", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	// Parse insert result
	var insertResult map[string]interface{}
	json.Unmarshal(rawData, &insertResult)

	insertCount := 0
	if v, ok := insertResult["insertCount"]; ok {
		switch c := v.(type) {
		case float64:
			insertCount = int(c)
		case json.Number:
			if n, err := c.Int64(); err == nil {
				insertCount = int(n)
			}
		}
	}

	return successResult(elapsed, map[string]interface{}{
		"insert_count": insertCount,
	})
}

// Upsert upserts data into a collection via REST API
func (rc *RestClient) Upsert(data map[string]interface{}, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	body := rc.baseBody(coll)

	rows, err := columnsToRows(data)
	if err != nil {
		return errorResult(0, fmt.Sprintf("failed to convert data: %v", err))
	}
	body["data"] = rows

	rawData, elapsed, err := rc.post("/entities/upsert", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var upsertResult map[string]interface{}
	json.Unmarshal(rawData, &upsertResult)

	upsertCount := 0
	if v, ok := upsertResult["upsertCount"]; ok {
		switch c := v.(type) {
		case float64:
			upsertCount = int(c)
		case json.Number:
			if n, err := c.Int64(); err == nil {
				upsertCount = int(n)
			}
		}
	}

	return successResult(elapsed, map[string]interface{}{
		"upsert_count": upsertCount,
	})
}

// Delete deletes entities by filter expression via REST API
func (rc *RestClient) Delete(filter string, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	body := rc.baseBody(coll)
	body["filter"] = filter

	_, elapsed, err := rc.post("/entities/delete", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	return successResult(elapsed, map[string]interface{}{
		"delete_count": 0,
	})
}

// Get retrieves entities by IDs via REST API
func (rc *RestClient) Get(ids interface{}, outputFields []interface{}, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	body := rc.baseBody(coll)
	body["id"] = ids

	if len(outputFields) > 0 {
		fields := make([]string, len(outputFields))
		for i, f := range outputFields {
			if s, ok := f.(string); ok {
				fields[i] = s
			}
		}
		body["outputFields"] = fields
	}

	rawData, elapsed, err := rc.post("/entities/get", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var result interface{}
	json.Unmarshal(rawData, &result)
	return successResult(elapsed, result)
}

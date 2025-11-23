package milvus

import "encoding/json"

// getCollectionName returns collection name from params or default collection
func (c *Client) getCollectionName(collectionName ...string) string {
	if len(collectionName) > 0 && collectionName[0] != "" {
		return collectionName[0]
	}
	return c.defaultCollection
}

// toMap converts OperationResult to map[string]interface{} using JSON tags
// This ensures JavaScript code can access fields using camelCase names defined in JSON tags
func toMap(result *OperationResult) map[string]interface{} {
	data, _ := json.Marshal(result)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	return m
}

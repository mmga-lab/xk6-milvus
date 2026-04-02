package milvus

import (
	"encoding/json"
)

// CreateIndex creates an index on a field via REST API
func (rc *RestClient) CreateIndex(fieldName string, indexParams map[string]interface{}, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	body := rc.baseBody(coll)

	idx := map[string]interface{}{
		"fieldName": fieldName,
	}
	if it, ok := indexParams["indexType"].(string); ok {
		idx["indexType"] = it
	}
	if mt, ok := indexParams["metricType"].(string); ok {
		idx["metricType"] = mt
	}
	if name, ok := indexParams["indexName"].(string); ok {
		idx["indexName"] = name
	}
	if p, ok := indexParams["params"]; ok {
		idx["params"] = p
	}

	body["indexParams"] = []map[string]interface{}{idx}

	_, elapsed, err := rc.post("/indexes/create", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	indexType := "FLAT"
	if it, ok := indexParams["indexType"].(string); ok {
		indexType = it
	}

	return successResult(elapsed, map[string]interface{}{
		"field":      fieldName,
		"index_type": indexType,
	})
}

// DescribeIndex describes an index via REST API
func (rc *RestClient) DescribeIndex(indexName string, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	body := rc.baseBody(coll)
	body["indexName"] = indexName

	data, elapsed, err := rc.post("/indexes/describe", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var result interface{}
	json.Unmarshal(data, &result)
	return successResult(elapsed, result)
}

// DropIndex drops an index via REST API
func (rc *RestClient) DropIndex(indexName string, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	body := rc.baseBody(coll)
	body["indexName"] = indexName

	_, elapsed, err := rc.post("/indexes/drop", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	return successResult(elapsed, map[string]interface{}{"indexName": indexName})
}

// ListPartitions lists all partitions in a collection via REST API
func (rc *RestClient) ListPartitions(collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	data, elapsed, err := rc.post("/partitions/list", rc.baseBody(coll))
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var result interface{}
	json.Unmarshal(data, &result)
	return successResult(elapsed, result)
}

// CreatePartition creates a partition in a collection via REST API
func (rc *RestClient) CreatePartition(partitionName string, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	body := rc.baseBody(coll)
	body["partitionName"] = partitionName

	_, elapsed, err := rc.post("/partitions/create", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	return successResult(elapsed, map[string]interface{}{"partitionName": partitionName})
}

// DropPartition drops a partition from a collection via REST API
func (rc *RestClient) DropPartition(partitionName string, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	body := rc.baseBody(coll)
	body["partitionName"] = partitionName

	_, elapsed, err := rc.post("/partitions/drop", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	return successResult(elapsed, map[string]interface{}{"partitionName": partitionName})
}

// HasPartition checks if a partition exists via REST API
func (rc *RestClient) HasPartition(partitionName string, collectionName ...string) interface{} {
	coll := rc.getCollectionName(collectionName...)
	if coll == "" {
		return errorResult(0, ErrCollectionNameRequired.Error())
	}

	body := rc.baseBody(coll)
	body["partitionName"] = partitionName

	data, elapsed, err := rc.post("/partitions/has", body)
	if err != nil {
		return errorResult(elapsed, err.Error())
	}

	var result interface{}
	json.Unmarshal(data, &result)
	return successResult(elapsed, result)
}

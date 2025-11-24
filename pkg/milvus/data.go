package milvus

import (
	"fmt"
	"time"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// Insert inserts data into a collection
// Supports both collection-bound and explicit collection name
func (c *Client) Insert(data map[string]interface{}, collectionName ...string) interface{} {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		})
	}

	columns, err := c.convertDataToColumns(data)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to convert data: %v", err),
		})
	}

	option := milvusclient.NewColumnBasedInsertOption(coll, columns...)
	result, err := c.client.Insert(c.ctx, option)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to insert: %v", err),
		})
	}

	opResult := &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"insert_count": result.InsertCount,
		},
	}

	// Emit metrics with throughput calculation
	rowCount := 0
	// Get row count from first column
	if len(data) > 0 {
		for _, v := range data {
			if arr, ok := v.([]interface{}); ok {
				rowCount = len(arr)
				break
			}
		}
	}

	c.emitOperationMetrics(opResult, MetricMetadata{
		Operation:     "insert",
		Collection:    coll,
		RowCount:      rowCount,
		DataSizeBytes: calculateDataSize(data),
	})

	return toMap(opResult)
}

// Upsert upserts data into a collection (insert or update)
func (c *Client) Upsert(data map[string]interface{}, collectionName ...string) interface{} {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		})
	}

	columns, err := c.convertDataToColumns(data)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        wrapError("Upsert", err).Error(),
		})
	}

	option := milvusclient.NewColumnBasedInsertOption(coll, columns...)
	result, err := c.client.Upsert(c.ctx, option)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to upsert: %v", err),
		})
	}

	opResult := &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"upsert_count": result.UpsertCount,
		},
	}

	// Emit metrics with throughput calculation
	rowCount := 0
	if len(data) > 0 {
		for _, v := range data {
			if arr, ok := v.([]interface{}); ok {
				rowCount = len(arr)
				break
			}
		}
	}

	c.emitOperationMetrics(opResult, MetricMetadata{
		Operation:     "upsert",
		Collection:    coll,
		RowCount:      rowCount,
		DataSizeBytes: calculateDataSize(data),
	})

	return toMap(opResult)
}

// Delete deletes entities by filter expression (NEW - from Locust)
func (c *Client) Delete(filter string, collectionName ...string) interface{} {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        ErrCollectionNameRequired.Error(),
		})
	}

	option := milvusclient.NewDeleteOption(coll).WithExpr(filter)
	result, err := c.client.Delete(c.ctx, option)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to delete: %v", err),
		})
	}

	opResult := &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"delete_count": result.DeleteCount,
		},
	}

	// Emit metrics
	c.emitOperationMetrics(opResult, MetricMetadata{
		Operation:  "delete",
		Collection: coll,
		FilterUsed: filter != "",
	})

	return toMap(opResult)
}

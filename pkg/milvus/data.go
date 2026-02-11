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

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"insert_count": result.InsertCount,
		},
	})
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

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"upsert_count": result.UpsertCount,
		},
	})
}

// Flush flushes a collection to persist inserted data and seal growing segments
// This is a synchronous call that waits for flush to complete
func (c *Client) Flush(collectionName ...string) interface{} {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        ErrCollectionNameRequired.Error(),
		})
	}

	option := milvusclient.NewFlushOption(coll)
	task, err := c.client.Flush(c.ctx, option)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to flush: %v", err),
		})
	}

	// Wait for flush to complete
	err = task.Await(c.ctx)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to wait for flush: %v", err),
		})
	}

	segIDs, _, _, _ := task.GetFlushStats()
	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"segment_ids": segIDs,
		},
	})
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

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"delete_count": result.DeleteCount,
		},
	})
}

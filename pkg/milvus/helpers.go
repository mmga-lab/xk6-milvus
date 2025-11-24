package milvus

import (
	"encoding/json"
	"time"

	"go.k6.io/k6/metrics"
)

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
	data, err := json.Marshal(result)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}
	return m
}

// MetricMetadata holds additional metadata for metric emission
type MetricMetadata struct {
	Operation     string
	Collection    string
	RowCount      int     // Number of rows processed (for throughput calculation)
	DataSizeBytes int64   // Estimated data size in bytes (for MB/s calculation)
	TopK          int
	FilterUsed    bool
	OutputFields  int
	IsHybrid      bool
	NumRequests   int
	RerankerType  string
	IsSparse      bool
	IsIndexBuild  bool
	IndexDuration float64
}

// emitOperationMetrics emits k6 metrics for a Milvus operation
func (c *Client) emitOperationMetrics(result *OperationResult, metadata MetricMetadata) {
	// Skip if metrics not initialized or VU state not available
	if c.metrics == nil || c.vu.State() == nil {
		return
	}

	now := time.Now()
	state := c.vu.State()
	ctx := c.vu.Context()

	// Build base tags from VU state tags
	baseTags := state.Tags.GetCurrentValues().Tags.
		With("operation", metadata.Operation).
		With("success", boolToString(result.Success))
	if metadata.Collection != "" {
		baseTags = baseTags.With("collection", metadata.Collection)
	}

	// Helper function to emit a metric sample
	emitSample := func(metric *metrics.Metric, value float64, extraTags ...string) {
		// Start with base tags and add extra tags
		sampleTags := baseTags
		for i := 0; i < len(extraTags); i += 2 {
			if i+1 < len(extraTags) {
				sampleTags = sampleTags.With(extraTags[i], extraTags[i+1])
			}
		}

		metrics.PushIfNotDone(ctx, state.Samples, metrics.Sample{
			TimeSeries: metrics.TimeSeries{
				Metric: metric,
				Tags:   sampleTags,
			},
			Time:  now,
			Value: value,
		})
	}

	// Always emit: operation duration and operations counter
	emitSample(c.metrics.OperationDuration, result.ResponseTime)
	emitSample(c.metrics.OperationsTotal, 1)

	// Error rate (0 = success, 1 = error)
	if result.Success {
		emitSample(c.metrics.Errors, 0)
	} else {
		emitSample(c.metrics.Errors, 1)
	}

	// Empty results rate (search/query operations)
	if metadata.Operation == "search" || metadata.Operation == "query" || metadata.Operation == "hybrid_search" {
		if result.Empty {
			emitSample(c.metrics.EmptyResults, 1)
		} else {
			emitSample(c.metrics.EmptyResults, 0)
		}
	}

	// Search recall (search operations only)
	if metadata.Operation == "search" || metadata.Operation == "hybrid_search" {
		if result.Recall > 0 {
			emitSample(c.metrics.SearchRecall, float64(result.Recall))
		}
	}

	// Filter usage rate
	if metadata.FilterUsed {
		emitSample(c.metrics.FilterUsed, 1)
	} else if metadata.Operation == "search" || metadata.Operation == "query" {
		emitSample(c.metrics.FilterUsed, 0)
	}

	// Sparse vector operations rate
	if metadata.IsSparse {
		emitSample(c.metrics.SparseVectorOps, 1)
	} else if metadata.Operation == "search" || metadata.Operation == "insert" || metadata.Operation == "upsert" {
		emitSample(c.metrics.SparseVectorOps, 0)
	}

	// Search topK
	if metadata.TopK > 0 {
		emitSample(c.metrics.SearchTopK, float64(metadata.TopK))
	}

	// Output fields count
	if metadata.OutputFields > 0 {
		emitSample(c.metrics.OutputFieldsCount, float64(metadata.OutputFields))
	}

	// Result count (from operation result)
	if result.Result != nil {
		switch r := result.Result.(type) {
		case []SearchResult:
			emitSample(c.metrics.ResultCount, float64(len(r)))
		case []QueryResult:
			emitSample(c.metrics.ResultCount, float64(len(r)))
		case map[string]interface{}:
			if count, ok := r["insert_count"].(int64); ok {
				emitSample(c.metrics.RowsInserted, float64(count))
			}
			if count, ok := r["upsert_count"].(int64); ok {
				emitSample(c.metrics.RowsInserted, float64(count))
			}
			if count, ok := r["delete_count"].(int64); ok {
				emitSample(c.metrics.RowsDeleted, float64(count))
			}
		}
	}

	// Hybrid search specific metrics
	if metadata.IsHybrid {
		emitSample(c.metrics.HybridSearchRequests, float64(metadata.NumRequests))
		if metadata.RerankerType != "" {
			emitSample(c.metrics.RerankerOperations, 1, "reranker_type", metadata.RerankerType)
		}
	}

	// Throughput metrics (for insert/upsert operations)
	if result.ResponseTime > 0 {
		responseTimeSec := result.ResponseTime / 1000.0 // Convert ms to seconds

		// Rows per second throughput
		if metadata.RowCount > 0 {
			rowsPerSecond := float64(metadata.RowCount) / responseTimeSec
			emitSample(c.metrics.ThroughputRowsPS, rowsPerSecond)
		}

		// MB/s throughput
		if metadata.DataSizeBytes > 0 {
			mbps := (float64(metadata.DataSizeBytes) / (1024 * 1024)) / responseTimeSec
			emitSample(c.metrics.ThroughputMBPS, mbps)
		}
	}

	// Index build duration (separate from operation duration)
	if metadata.IsIndexBuild && metadata.IndexDuration > 0 {
		emitSample(c.metrics.IndexBuildDuration, metadata.IndexDuration)
	}

	// Collection operations
	if metadata.Operation == "create_collection" {
		emitSample(c.metrics.CollectionsCreated, 1)
	} else if metadata.Operation == "load_collection" {
		emitSample(c.metrics.CollectionLoaded, 1)
	} else if metadata.Operation == "release_collection" {
		emitSample(c.metrics.CollectionLoaded, 0)
	}
}

// boolToString converts bool to string for metric tags
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// calculateDataSize calculates the actual data size in bytes by serializing to JSON
// This gives a more accurate estimation than manual calculation
func calculateDataSize(data interface{}) int64 {
	if data == nil {
		return 0
	}

	// Serialize to JSON to get actual size
	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0
	}

	// Return the JSON byte size
	// Note: protobuf (used by gRPC) is typically 20-30% smaller than JSON,
	// but this gives us a reasonable upper-bound estimation
	return int64(len(jsonData))
}

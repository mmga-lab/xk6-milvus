package milvus

import (
	"context"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/metrics"
)

// OperationResult represents unified result structure for all operations
// Following Locust's design pattern for consistent metrics collection
type OperationResult struct {
	Success      bool        `json:"success"`
	ResponseTime float64     `json:"response_time_ms"`
	Result       interface{} `json:"result,omitempty"`
	Error        string      `json:"error,omitempty"`
	Empty        bool        `json:"empty"`
	Recall       float32     `json:"recall"`
}

// Metrics holds all k6 metrics for Milvus operations
type Metrics struct {
	// Trend metrics - statistical distribution
	OperationDuration    *metrics.Metric // Response time distribution
	SearchRecall         *metrics.Metric // Search quality metric
	IndexBuildDuration   *metrics.Metric // Index creation time

	// Counter metrics - cumulative sum
	OperationsTotal      *metrics.Metric // Total operations count
	RowsInserted         *metrics.Metric // Total rows inserted/upserted
	RowsDeleted          *metrics.Metric // Total rows deleted
	RerankerOperations   *metrics.Metric // Reranker usage by type
	CollectionsCreated   *metrics.Metric // Total collections created

	// Rate metrics - ratio of non-zero values
	Errors               *metrics.Metric // Error rate
	EmptyResults         *metrics.Metric // Empty result rate
	FilterUsed           *metrics.Metric // Filter expression usage rate
	SparseVectorOps      *metrics.Metric // Sparse vector usage rate

	// Gauge metrics - latest value
	ResultCount          *metrics.Metric // Actual results returned
	SearchTopK           *metrics.Metric // TopK parameter value
	OutputFieldsCount    *metrics.Metric // Number of output fields
	CollectionLoaded     *metrics.Metric // Collection load state (0/1)
	HybridSearchRequests *metrics.Metric // ANN requests in hybrid search

	// Throughput metrics
	ThroughputMBPS       *metrics.Metric // Data throughput in MB/s (Gauge)
	ThroughputRowsPS     *metrics.Metric // Row throughput in rows/s (Gauge)
}

// Client represents a Milvus client instance
type Client struct {
	client            *milvusclient.Client
	ctx               context.Context
	vu                modules.VU
	config            *ClientConfig
	defaultCollection string // Collection binding (Locust pattern) - deprecated, use config.DefaultCollection
	metrics           *Metrics // k6 metrics for tracking operations
}

// Field represents a field definition for schema
type Field struct {
	Name           string                 `json:"name"`
	DataType       string                 `json:"dataType"`
	IsPrimaryKey   bool                   `json:"isPrimaryKey,omitempty"`
	IsAutoID       bool                   `json:"isAutoID,omitempty"`
	Dimension      int64                  `json:"dimension,omitempty"`
	Description    string                 `json:"description,omitempty"`
	MaxLength      int64                  `json:"maxLength,omitempty"`
	EnableAnalyzer bool                   `json:"enableAnalyzer,omitempty"`
	EnableMatch    bool                   `json:"enableMatch,omitempty"`
	AnalyzerParams map[string]interface{} `json:"analyzerParams,omitempty"`
}

// Function represents a function definition for schema
type Function struct {
	Name             string            `json:"name"`
	FunctionType     string            `json:"functionType"`
	InputFieldNames  []string          `json:"inputFieldNames"`
	OutputFieldNames []string          `json:"outputFieldNames"`
	Params           map[string]string `json:"params,omitempty"`
}

// Schema represents a collection schema
type Schema struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Fields      []Field    `json:"fields"`
	Functions   []Function `json:"functions,omitempty"`
	NumShards   int32      `json:"numShards,omitempty"`
}

// SearchResult represents a single search result entry
type SearchResult struct {
	ID     int64                  `json:"id"`
	Score  float32                `json:"score"`
	Fields map[string]interface{} `json:"fields,omitempty"`
}

// QueryResult represents a single query result entry
type QueryResult struct {
	Fields map[string]interface{} `json:"fields"`
}

// HybridSearchRequest represents a single vector search request in hybrid search
type HybridSearchRequest struct {
	Vectors     interface{}            `json:"vectors"` // Can be [][]float32 for dense or []map[string]interface{} for sparse
	VectorField string                 `json:"vectorField"`
	Limit       int                    `json:"limit"`
	Params      map[string]interface{} `json:"params,omitempty"`
}

// Reranker represents the reranking strategy for hybrid search
type Reranker struct {
	Type   string                 `json:"type"`   // "rrf" or "weighted"
	Params map[string]interface{} `json:"params"` // parameters for reranker
}

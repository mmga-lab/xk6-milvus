package milvus

import (
	"context"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.k6.io/k6/js/modules"
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

// Client represents a Milvus client instance
type Client struct {
	client            *milvusclient.Client
	ctx               context.Context
	vu                modules.VU
	config            *ClientConfig
	defaultCollection string // Collection binding (Locust pattern) - deprecated, use config.DefaultCollection
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

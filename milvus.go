package milvus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/milvus", new(RootModule))
}

// Ensure the interfaces are implemented correctly
var (
	_ modules.Module   = &RootModule{}
	_ modules.Instance = &Milvus{}
)

// RootModule is the global module instance that creates module instances for each VU
type RootModule struct{}

// Milvus represents the JS module instance for each VU
type Milvus struct {
	vu modules.VU
}

// NewModuleInstance implements the modules.Module interface
// It creates a new instance of the Milvus module for each VU
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &Milvus{vu: vu}
}

// Exports implements the modules.Instance interface
// It returns the exports of the module for JavaScript
func (m *Milvus) Exports() modules.Exports {
	return modules.Exports{
		Default: m,
		Named: map[string]interface{}{
			"client":               m.Client,
			"clientWithCollection": m.ClientWithCollection,
		},
	}
}

// OperationResult represents unified result structure for all operations
// Following Locust's design pattern for consistent metrics collection
type OperationResult struct {
	Success      bool        `json:"success"`
	ResponseTime float64     `json:"response_time_ms"`
	Result       interface{} `json:"result,omitempty"`
	Error        string      `json:"error,omitempty"`
	Empty        bool        `json:"empty,omitempty"`
	Recall       float32     `json:"recall,omitempty"`
}

// Client represents a Milvus client instance
type Client struct {
	client            *milvusclient.Client
	ctx               context.Context
	vu                modules.VU
	defaultCollection string // Collection binding (Locust pattern)
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
	Vectors     [][]float32            `json:"vectors"`
	VectorField string                 `json:"vectorField"`
	Limit       int                    `json:"limit"`
	Params      map[string]interface{} `json:"params,omitempty"`
}

// Reranker represents the reranking strategy for hybrid search
type Reranker struct {
	Type    string                 `json:"type"`    // "rrf" or "weighted"
	Params  map[string]interface{} `json:"params"`  // parameters for reranker
}

// ==================== Client Creation ====================

// Client creates a new Milvus client (not bound to any collection)
func (m *Milvus) Client(address string, token ...string) (*Client, error) {
	return m.createClient(address, "", token...)
}

// ClientWithCollection creates a new Milvus client bound to a specific collection
// This follows Locust's pattern where client is tied to a collection
func (m *Milvus) ClientWithCollection(address, collectionName string, token ...string) (*Client, error) {
	return m.createClient(address, collectionName, token...)
}

func (m *Milvus) createClient(address, collectionName string, token ...string) (*Client, error) {
	ctx := m.vu.Context()

	config := &milvusclient.ClientConfig{
		Address: address,
	}

	// Parse token if provided (format: "username:password")
	if len(token) > 0 && token[0] != "" {
		parts := strings.Split(token[0], ":")
		if len(parts) == 2 {
			config.Username = parts[0]
			config.Password = parts[1]
		}
	}

	c, err := milvusclient.New(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus client: %v", err)
	}

	return &Client{
		client:            c,
		ctx:               ctx,
		vu:                m.vu,
		defaultCollection: collectionName,
	}, nil
}

func (c *Client) Close() error {
	return c.client.Close(c.ctx)
}

// ==================== Collection Operations ====================

func (c *Client) CreateCollectionFromJSON(schemaJSON string) *OperationResult {
	start := time.Now()

	var schema Schema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to parse schema JSON: %v", err),
		}
	}

	return c.CreateCollection(schema)
}

func (c *Client) CreateCollection(schema Schema) *OperationResult {
	start := time.Now()

	entitySchema := entity.NewSchema().
		WithName(schema.Name).
		WithDescription(schema.Description)

	for _, field := range schema.Fields {
		entityField := entity.NewField().
			WithName(field.Name).
			WithDescription(field.Description)

		// Set data type
		if field.DataType == "" {
			return &OperationResult{
				Success:      false,
				ResponseTime: float64(time.Since(start).Milliseconds()),
				Error:        fmt.Sprintf("field %s has empty dataType", field.Name),
			}
		}

		switch field.DataType {
		case "Int64":
			entityField = entityField.WithDataType(entity.FieldTypeInt64)
		case "Int32":
			entityField = entityField.WithDataType(entity.FieldTypeInt32)
		case "Int16":
			entityField = entityField.WithDataType(entity.FieldTypeInt16)
		case "Int8":
			entityField = entityField.WithDataType(entity.FieldTypeInt8)
		case "Bool":
			entityField = entityField.WithDataType(entity.FieldTypeBool)
		case "Float":
			entityField = entityField.WithDataType(entity.FieldTypeFloat)
		case "Double":
			entityField = entityField.WithDataType(entity.FieldTypeDouble)
		case "String":
			entityField = entityField.WithDataType(entity.FieldTypeString)
		case "VarChar":
			entityField = entityField.WithDataType(entity.FieldTypeVarChar)
		case "JSON":
			entityField = entityField.WithDataType(entity.FieldTypeJSON)
		case "FloatVector":
			entityField = entityField.WithDataType(entity.FieldTypeFloatVector).WithDim(field.Dimension)
		case "BinaryVector":
			entityField = entityField.WithDataType(entity.FieldTypeBinaryVector).WithDim(field.Dimension)
		case "Float16Vector":
			entityField = entityField.WithDataType(entity.FieldTypeFloat16Vector).WithDim(field.Dimension)
		case "BFloat16Vector":
			entityField = entityField.WithDataType(entity.FieldTypeBFloat16Vector).WithDim(field.Dimension)
		case "SparseFloatVector":
			entityField = entityField.WithDataType(entity.FieldTypeSparseVector)
		default:
			return &OperationResult{
				Success:      false,
				ResponseTime: float64(time.Since(start).Milliseconds()),
				Error:        fmt.Sprintf("unsupported data type: '%s' for field '%s'", field.DataType, field.Name),
			}
		}

		if field.IsPrimaryKey {
			entityField = entityField.WithIsPrimaryKey(true)
		}
		if field.IsAutoID {
			entityField = entityField.WithIsAutoID(true)
		}
		if field.MaxLength > 0 {
			entityField = entityField.WithMaxLength(field.MaxLength)
		}
		if field.EnableAnalyzer {
			entityField = entityField.WithEnableAnalyzer(true)
			if field.AnalyzerParams != nil {
				entityField = entityField.WithAnalyzerParams(field.AnalyzerParams)
			}
		}
		if field.EnableMatch {
			entityField = entityField.WithEnableMatch(true)
		}

		entitySchema = entitySchema.WithField(entityField)
	}

	// Add functions to schema
	for _, fn := range schema.Functions {
		entityFunc := entity.NewFunction().
			WithName(fn.Name).
			WithInputFields(fn.InputFieldNames...).
			WithOutputFields(fn.OutputFieldNames...)

		switch fn.FunctionType {
		case "BM25":
			entityFunc = entityFunc.WithType(entity.FunctionTypeBM25)
		case "TextEmbedding":
			entityFunc = entityFunc.WithType(entity.FunctionTypeTextEmbedding)
		default:
			return &OperationResult{
				Success:      false,
				ResponseTime: float64(time.Since(start).Milliseconds()),
				Error:        fmt.Sprintf("unsupported function type: %s", fn.FunctionType),
			}
		}

		for k, v := range fn.Params {
			entityFunc = entityFunc.WithParam(k, v)
		}

		entitySchema = entitySchema.WithFunction(entityFunc)
	}

	option := milvusclient.NewCreateCollectionOption(schema.Name, entitySchema)
	if schema.NumShards > 0 {
		option = option.WithShardNum(schema.NumShards)
	}

	err := c.client.CreateCollection(c.ctx, option)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to create collection: %v", err),
		}
	}

	return &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       map[string]interface{}{"collection": schema.Name},
	}
}

func (c *Client) DropCollection(collectionName string) *OperationResult {
	start := time.Now()

	if collectionName == "" {
		collectionName = c.defaultCollection
	}

	option := milvusclient.NewDropCollectionOption(collectionName)
	err := c.client.DropCollection(c.ctx, option)

	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to drop collection: %v", err),
		}
	}

	return &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       map[string]interface{}{"collection": collectionName},
	}
}

func (c *Client) HasCollection(collectionName string) *OperationResult {
	start := time.Now()

	if collectionName == "" {
		collectionName = c.defaultCollection
	}

	option := milvusclient.NewHasCollectionOption(collectionName)
	has, err := c.client.HasCollection(c.ctx, option)

	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to check collection: %v", err),
		}
	}

	return &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       map[string]interface{}{"exists": has},
	}
}

func (c *Client) LoadCollection(collectionName string) *OperationResult {
	start := time.Now()

	if collectionName == "" {
		collectionName = c.defaultCollection
	}

	option := milvusclient.NewLoadCollectionOption(collectionName)
	task, err := c.client.LoadCollection(c.ctx, option)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to load collection: %v", err),
		}
	}

	// Wait for collection to be loaded
	err = task.Await(c.ctx)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to wait for collection load: %v", err),
		}
	}

	return &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       map[string]interface{}{"collection": collectionName},
	}
}

func (c *Client) ReleaseCollection(collectionName string) *OperationResult {
	start := time.Now()

	if collectionName == "" {
		collectionName = c.defaultCollection
	}

	option := milvusclient.NewReleaseCollectionOption(collectionName)
	err := c.client.ReleaseCollection(c.ctx, option)

	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to release collection: %v", err),
		}
	}

	return &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       map[string]interface{}{"collection": collectionName},
	}
}

// ==================== Write Operations ====================

// Insert inserts data into a collection
// Supports both collection-bound and explicit collection name
func (c *Client) Insert(data map[string]interface{}, collectionName ...string) *OperationResult {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		}
	}

	columns, err := c.convertDataToColumns(data)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to convert data: %v", err),
		}
	}

	option := milvusclient.NewColumnBasedInsertOption(coll, columns...)
	result, err := c.client.Insert(c.ctx, option)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to insert: %v", err),
		}
	}

	return &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"insert_count": result.InsertCount,
		},
	}
}

// Upsert upserts data into a collection (insert or update)
func (c *Client) Upsert(data map[string]interface{}, collectionName ...string) *OperationResult {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		}
	}

	columns, err := c.convertDataToColumns(data)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to convert data: %v", err),
		}
	}

	option := milvusclient.NewColumnBasedInsertOption(coll, columns...)
	result, err := c.client.Upsert(c.ctx, option)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to upsert: %v", err),
		}
	}

	return &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"upsert_count": result.UpsertCount,
		},
	}
}

// Delete deletes entities by filter expression (NEW - from Locust)
func (c *Client) Delete(filter string, collectionName ...string) *OperationResult {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		}
	}

	option := milvusclient.NewDeleteOption(coll).WithExpr(filter)
	result, err := c.client.Delete(c.ctx, option)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to delete: %v", err),
		}
	}

	return &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result: map[string]interface{}{
			"delete_count": result.DeleteCount,
		},
	}
}

// ==================== Read Operations ====================

// Search performs vector similarity search with Recall support
func (c *Client) Search(vectors [][]float32, topK int, params map[string]interface{}, collectionName ...string) *OperationResult {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		}
	}

	// Convert vectors to entity.Vector
	searchVectors := make([]entity.Vector, len(vectors))
	for i, v := range vectors {
		searchVectors[i] = entity.FloatVector(v)
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
	if expr, ok := params["expr"].(string); ok && expr != "" {
		searchOption = searchOption.WithFilter(expr)
	}

	// Set metric type through search param
	if metricType, ok := params["metricType"].(string); ok {
		searchOption = searchOption.WithSearchParam("metric_type", metricType)
	}

	// Execute search
	resultSets, err := c.client.Search(c.ctx, searchOption)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to search: %v", err),
		}
	}

	// Convert results
	var results []SearchResult
	var recall float32
	isEmpty := true

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
				result.ID = idVal.(int64)
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

	return &OperationResult{
		Success:      !isEmpty,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       results,
		Empty:        isEmpty,
		Recall:       recall, // NEW: Expose recall metric
	}
}

// HybridSearch performs multi-vector hybrid search with reranking (NEW - from Locust)
func (c *Client) HybridSearch(requests []HybridSearchRequest, reranker Reranker, limit int, outputFields []interface{}, collectionName ...string) *OperationResult {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		}
	}

	if len(requests) == 0 {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "at least one search request required",
		}
	}

	// Build ANN requests
	var annRequests []*milvusclient.AnnRequest
	for _, req := range requests {
		// Convert vectors to entity.Vector
		searchVectors := make([]entity.Vector, len(req.Vectors))
		for i, v := range req.Vectors {
			searchVectors[i] = entity.FloatVector(v)
		}

		annReq := milvusclient.NewAnnRequest(req.VectorField, req.Limit, searchVectors...)

		// Apply params if provided
		if req.Params != nil {
			if expr, ok := req.Params["expr"].(string); ok && expr != "" {
				annReq = annReq.WithFilter(expr)
			}
			if metricType, ok := req.Params["metricType"].(string); ok {
				annReq = annReq.WithSearchParam("metric_type", metricType)
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
	resultSets, err := c.client.HybridSearch(c.ctx, hybridOption)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to hybrid search: %v", err),
		}
	}

	// Convert results
	var results []SearchResult
	var recall float32
	isEmpty := true

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
				result.ID = idVal.(int64)
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

	return &OperationResult{
		Success:      !isEmpty,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       results,
		Empty:        isEmpty,
		Recall:       recall,
	}
}

// Query performs scalar query without vectors (NEW - from Locust)
func (c *Client) Query(filter string, outputFields []interface{}, collectionName ...string) *OperationResult {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		}
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

	resultSet, err := c.client.Query(c.ctx, option)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to query: %v", err),
		}
	}

	// Convert results
	var results []QueryResult
	isEmpty := resultSet.ResultCount == 0

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

	return &OperationResult{
		Success:      !isEmpty,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       results,
		Empty:        isEmpty,
	}
}

// ==================== Index Operations ====================

func (c *Client) CreateIndex(fieldName string, indexParams map[string]interface{}, collectionName ...string) *OperationResult {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		}
	}

	var idx index.Index

	// Default to flat index if not specified
	indexType := "FLAT"
	metricType := entity.L2

	if iType, ok := indexParams["indexType"].(string); ok {
		indexType = iType
	}

	if mType, ok := indexParams["metricType"].(string); ok {
		switch mType {
		case "L2":
			metricType = entity.L2
		case "IP":
			metricType = entity.IP
		case "COSINE":
			metricType = entity.COSINE
		case "BM25":
			metricType = entity.BM25
		}
	}

	switch indexType {
	case "FLAT":
		idx = index.NewFlatIndex(metricType)
	case "IVF_FLAT":
		nlist := 1024
		if n, ok := indexParams["nlist"].(int); ok {
			nlist = n
		}
		idx = index.NewIvfFlatIndex(metricType, nlist)
	case "IVF_SQ8":
		nlist := 1024
		if n, ok := indexParams["nlist"].(int); ok {
			nlist = n
		}
		idx = index.NewIvfSQ8Index(metricType, nlist)
	case "IVF_PQ":
		nlist := 1024
		m := 4
		nbits := 8
		if n, ok := indexParams["nlist"].(int); ok {
			nlist = n
		}
		if mVal, ok := indexParams["m"].(int); ok {
			m = mVal
		}
		if nBits, ok := indexParams["nbits"].(int); ok {
			nbits = nBits
		}
		idx = index.NewIvfPQIndex(metricType, nlist, m, nbits)
	case "HNSW":
		M := 16
		efConstruction := 200
		if mVal, ok := indexParams["M"].(int); ok {
			M = mVal
		}
		if ef, ok := indexParams["efConstruction"].(int); ok {
			efConstruction = ef
		}
		idx = index.NewHNSWIndex(metricType, M, efConstruction)
	case "SPARSE_INVERTED_INDEX":
		dropRatio := 0.0
		if dr, ok := indexParams["dropRatio"].(float64); ok {
			dropRatio = dr
		}
		idx = index.NewSparseInvertedIndex(metricType, dropRatio)
	case "SPARSE_WAND":
		dropRatio := 0.0
		if dr, ok := indexParams["dropRatio"].(float64); ok {
			dropRatio = dr
		}
		idx = index.NewSparseWANDIndex(metricType, dropRatio)
	default:
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("unsupported index type: %s", indexType),
		}
	}

	option := milvusclient.NewCreateIndexOption(coll, fieldName, idx)
	task, err := c.client.CreateIndex(c.ctx, option)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to create index: %v", err),
		}
	}

	// Wait for index creation to complete
	err = task.Await(c.ctx)
	if err != nil {
		return &OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to wait for index creation: %v", err),
		}
	}

	return &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       map[string]interface{}{"field": fieldName, "index_type": indexType},
	}
}

// ==================== Helper Functions ====================

// getCollectionName returns collection name from params or default collection
func (c *Client) getCollectionName(collectionName ...string) string {
	if len(collectionName) > 0 && collectionName[0] != "" {
		return collectionName[0]
	}
	return c.defaultCollection
}

// convertDataToColumns converts map data to Milvus columns
func (c *Client) convertDataToColumns(data map[string]interface{}) ([]column.Column, error) {
	var columns []column.Column

	for fieldName, fieldData := range data {
		switch v := fieldData.(type) {
		case [][]float32:
			if len(v) > 0 {
				dim := len(v[0])
				columns = append(columns, column.NewColumnFloatVector(fieldName, dim, v))
			}
		case []int64:
			columns = append(columns, column.NewColumnInt64(fieldName, v))
		case []int32:
			columns = append(columns, column.NewColumnInt32(fieldName, v))
		case []float32:
			columns = append(columns, column.NewColumnFloat(fieldName, v))
		case []float64:
			columns = append(columns, column.NewColumnDouble(fieldName, v))
		case []string:
			columns = append(columns, column.NewColumnVarChar(fieldName, v))
		case []bool:
			columns = append(columns, column.NewColumnBool(fieldName, v))
		case []interface{}:
			if len(v) == 0 {
				continue
			}

			switch v[0].(type) {
			case int64:
				ids := make([]int64, len(v))
				for i, val := range v {
					ids[i] = val.(int64)
				}
				columns = append(columns, column.NewColumnInt64(fieldName, ids))
			case string:
				strs := make([]string, len(v))
				for i, val := range v {
					strs[i] = val.(string)
				}
				columns = append(columns, column.NewColumnVarChar(fieldName, strs))
			case float64:
				// Check if all values are integers
				isInteger := true
				for _, val := range v {
					f := val.(float64)
					if f != float64(int64(f)) {
						isInteger = false
						break
					}
				}

				if isInteger && fieldName == "id" {
					ids := make([]int64, len(v))
					for i, val := range v {
						ids[i] = int64(val.(float64))
					}
					columns = append(columns, column.NewColumnInt64(fieldName, ids))
				} else {
					floats := make([]float32, len(v))
					for i, val := range v {
						floats[i] = float32(val.(float64))
					}
					columns = append(columns, column.NewColumnFloat(fieldName, floats))
				}
			case bool:
				bools := make([]bool, len(v))
				for i, val := range v {
					bools[i] = val.(bool)
				}
				columns = append(columns, column.NewColumnBool(fieldName, bools))
			case []interface{}:
				// Vector field
				if len(v) > 0 {
					firstVec := v[0].([]interface{})
					dim := len(firstVec)
					vectors := make([][]float32, len(v))
					for i, vecInterface := range v {
						vec := vecInterface.([]interface{})
						floatVec := make([]float32, len(vec))
						for j, val := range vec {
							floatVec[j] = float32(val.(float64))
						}
						vectors[i] = floatVec
					}
					columns = append(columns, column.NewColumnFloatVector(fieldName, dim, vectors))
				}
			default:
				return nil, fmt.Errorf("unsupported interface{} element type for field %s: %T", fieldName, v[0])
			}
		default:
			return nil, fmt.Errorf("unsupported field type for field %s: %T", fieldName, fieldData)
		}
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("no valid columns provided")
	}

	return columns, nil
}

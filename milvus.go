package milvus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/column"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/milvus", new(Milvus))
}

type Milvus struct{}

type Client struct {
	client *milvusclient.Client
	ctx    context.Context
}

// Field represents a field definition for schema
type Field struct {
	Name         string `json:"name"`
	DataType     string `json:"dataType"`
	IsPrimaryKey bool   `json:"isPrimaryKey,omitempty"`
	IsAutoID     bool   `json:"isAutoID,omitempty"`
	Dimension    int64  `json:"dimension,omitempty"`
	Description  string `json:"description,omitempty"`
	MaxLength    int64  `json:"maxLength,omitempty"`
}

// Schema represents a collection schema
type Schema struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Fields      []Field `json:"fields"`
}

func (*Milvus) Client(address string) (*Client, error) {
	ctx := context.Background()
	c, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus client: %v", err)
	}

	return &Client{
		client: c,
		ctx:    ctx,
	}, nil
}

func (c *Client) Close() error {
	return c.client.Close(c.ctx)
}

// CreateCollectionFromJSON creates a collection from JSON schema string
func (c *Client) CreateCollectionFromJSON(schemaJSON string) error {
	var schema Schema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return fmt.Errorf("failed to parse schema JSON: %v", err)
	}
	return c.CreateCollection(schema)
}

// CreateCollection creates a collection with flexible schema
func (c *Client) CreateCollection(schema Schema) error {
	entitySchema := entity.NewSchema().
		WithName(schema.Name).
		WithDescription(schema.Description)

	for _, field := range schema.Fields {
		entityField := entity.NewField().
			WithName(field.Name).
			WithDescription(field.Description)

		// Set data type
		if field.DataType == "" {
			return fmt.Errorf("field %s has empty dataType", field.Name)
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
			return fmt.Errorf("unsupported data type: '%s' for field '%s'", field.DataType, field.Name)
		}

		// Set primary key
		if field.IsPrimaryKey {
			entityField = entityField.WithIsPrimaryKey(true)
		}

		// Set auto ID
		if field.IsAutoID {
			entityField = entityField.WithIsAutoID(true)
		}

		// Set max length for string/varchar fields
		if field.MaxLength > 0 {
			entityField = entityField.WithMaxLength(field.MaxLength)
		}

		entitySchema = entitySchema.WithField(entityField)
	}

	option := milvusclient.NewCreateCollectionOption(schema.Name, entitySchema)
	return c.client.CreateCollection(c.ctx, option)
}

// CreateCollectionSimple creates a collection with a simple vector field (backward compatibility)
func (c *Client) CreateCollectionSimple(collectionName string, dimension int64) error {
	schema := Schema{
		Name:        collectionName,
		Description: "Simple collection for k6 testing",
		Fields: []Field{
			{
				Name:         "id",
				DataType:     "Int64",
				IsPrimaryKey: true,
				IsAutoID:     true,
			},
			{
				Name:      "vector",
				DataType:  "FloatVector",
				Dimension: dimension,
			},
		},
	}
	return c.CreateCollection(schema)
}

func (c *Client) DropCollection(collectionName string) error {
	option := milvusclient.NewDropCollectionOption(collectionName)
	return c.client.DropCollection(c.ctx, option)
}

func (c *Client) HasCollection(collectionName string) (bool, error) {
	option := milvusclient.NewHasCollectionOption(collectionName)
	return c.client.HasCollection(c.ctx, option)
}

// Insert supports multiple field types with flexible data structure
func (c *Client) Insert(collectionName string, data map[string]interface{}) ([]int64, error) {
	var columns []column.Column

	for fieldName, fieldData := range data {
		switch v := fieldData.(type) {
		case [][]float32:
			// Float vector field
			if len(v) > 0 {
				dim := len(v[0])
				columns = append(columns, column.NewColumnFloatVector(fieldName, dim, v))
			}
		case []int64:
			// Int64 field
			columns = append(columns, column.NewColumnInt64(fieldName, v))
		case []int32:
			// Int32 field
			columns = append(columns, column.NewColumnInt32(fieldName, v))
		case []float32:
			// Float field
			columns = append(columns, column.NewColumnFloat(fieldName, v))
		case []float64:
			// Double field
			columns = append(columns, column.NewColumnDouble(fieldName, v))
		case []string:
			// String/VarChar field
			columns = append(columns, column.NewColumnVarChar(fieldName, v))
		case []bool:
			// Bool field
			columns = append(columns, column.NewColumnBool(fieldName, v))
		case []interface{}:
			// Handle JavaScript arrays converted to []interface{}
			if len(v) == 0 {
				continue
			}
			
			// Determine field type by examining the first element
			switch v[0].(type) {
			case string:
				// String/VarChar field
				strs := make([]string, len(v))
				for i, val := range v {
					strs[i] = val.(string)
				}
				columns = append(columns, column.NewColumnVarChar(fieldName, strs))
			case float64:
				// JavaScript numbers are float64, check if they should be treated as different types
				// Convert based on the schema field type
				if fieldName == "rating" {
					// rating is defined as Double in schema
					doubles := make([]float64, len(v))
					for i, val := range v {
						doubles[i] = val.(float64)
					}
					columns = append(columns, column.NewColumnDouble(fieldName, doubles))
				} else {
					// price and other numeric fields are Float, convert to float32
					floats := make([]float32, len(v))
					for i, val := range v {
						floats[i] = float32(val.(float64))
					}
					columns = append(columns, column.NewColumnFloat(fieldName, floats))
				}
			case bool:
				// Bool field
				bools := make([]bool, len(v))
				for i, val := range v {
					bools[i] = val.(bool)
				}
				columns = append(columns, column.NewColumnBool(fieldName, bools))
			case []interface{}:
				// Vector field (array of arrays)
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

	option := milvusclient.NewColumnBasedInsertOption(collectionName, columns...)
	result, err := c.client.Insert(c.ctx, option)
	if err != nil {
		return nil, fmt.Errorf("failed to insert: %v", err)
	}

	// Return placeholder IDs
	vectorCount := int64(0)
	for _, col := range columns {
		if col.Len() > int(vectorCount) {
			vectorCount = int64(col.Len())
		}
	}

	ids := make([]int64, vectorCount)
	for i := range ids {
		ids[i] = int64(i)
	}

	if result.InsertCount != vectorCount {
		return nil, fmt.Errorf("insert count mismatch: expected %d, got %d", vectorCount, result.InsertCount)
	}

	return ids, nil
}

// InsertVectors provides backward compatibility for simple vector insertion
func (c *Client) InsertVectors(collectionName string, vectors [][]float32) ([]int64, error) {
	data := map[string]interface{}{
		"vector": vectors,
	}
	return c.Insert(collectionName, data)
}

// Search with flexible parameters
func (c *Client) Search(collectionName string, vectors [][]float32, topK int, params map[string]interface{}) ([]SearchResult, error) {
	searchVectors := make([]entity.Vector, len(vectors))
	for i, v := range vectors {
		searchVectors[i] = entity.FloatVector(v)
	}

	option := milvusclient.NewSearchOption(collectionName, topK, searchVectors)

	// Set vector field name (default to "vector")
	vectorField := "vector"
	if field, ok := params["vectorField"].(string); ok {
		vectorField = field
	}
	option = option.WithANNSField(vectorField)

	// Set output fields
	if outputFields, ok := params["outputFields"].([]string); ok {
		option = option.WithOutputFields(outputFields...)
	} else {
		option = option.WithOutputFields("id")
	}

	// Set filter expression
	if expr, ok := params["expr"].(string); ok {
		option = option.WithFilter(expr)
	}

	// Set search parameters
	if searchParams, ok := params["searchParams"].(map[string]interface{}); ok {
		// Convert search params if needed
		_ = searchParams // placeholder for future search param handling
	}

	searchResult, err := c.client.Search(c.ctx, option)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %v", err)
	}

	var results []SearchResult
	for _, result := range searchResult {
		for i := 0; i < result.ResultCount; i++ {
			resultItem := SearchResult{
				Score: result.Scores[i],
				Fields: make(map[string]interface{}),
			}

			// Get ID from the IDs column
			if idVal, err := result.IDs.Get(i); err == nil {
				resultItem.ID = idVal.(int64)
			}

			// Get other output fields
			if outputFields, ok := params["outputFields"].([]string); ok {
				for _, field := range outputFields {
					if field != "id" {
						if fieldColumn := result.GetColumn(field); fieldColumn != nil {
							if fieldVal, err := fieldColumn.Get(i); err == nil {
								resultItem.Fields[field] = fieldVal
							}
						}
					}
				}
			}

			results = append(results, resultItem)
		}
	}

	return results, nil
}

// SearchSimple provides backward compatibility for simple vector search
func (c *Client) SearchSimple(collectionName string, vectors [][]float32, topK int) ([]SearchResult, error) {
	params := map[string]interface{}{
		"vectorField":  "vector",
		"outputFields": []string{"id"},
	}
	return c.Search(collectionName, vectors, topK, params)
}

func (c *Client) CreateIndex(collectionName string, fieldName string, indexParams map[string]interface{}) error {
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
	default:
		return fmt.Errorf("unsupported index type: %s", indexType)
	}

	option := milvusclient.NewCreateIndexOption(collectionName, fieldName, idx)
	task, err := c.client.CreateIndex(c.ctx, option)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}

	// Wait for index creation to complete
	err = task.Await(c.ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for index creation: %v", err)
	}

	return nil
}

// CreateIndexSimple provides backward compatibility for simple index creation
func (c *Client) CreateIndexSimple(collectionName string, fieldName string) error {
	params := map[string]interface{}{
		"indexType":  "FLAT",
		"metricType": "L2",
	}
	return c.CreateIndex(collectionName, fieldName, params)
}

func (c *Client) LoadCollection(collectionName string) error {
	option := milvusclient.NewLoadCollectionOption(collectionName)
	task, err := c.client.LoadCollection(c.ctx, option)
	if err != nil {
		return err
	}

	// Wait for collection to be loaded
	return task.Await(c.ctx)
}

func (c *Client) ReleaseCollection(collectionName string) error {
	option := milvusclient.NewReleaseCollectionOption(collectionName)
	return c.client.ReleaseCollection(c.ctx, option)
}

type SearchResult struct {
	ID     int64                  `json:"id"`
	Score  float32                `json:"score"`
	Fields map[string]interface{} `json:"fields,omitempty"`
}
// Package milvus provides a k6 extension for load testing Milvus vector databases.
// This file contains the Milvus client implementation and data operations.
package milvus

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.k6.io/k6/js/modules"
)

// Client represents a connection to a Milvus instance.
// It wraps the official Milvus client and provides methods for vector database operations.
type Client struct {
	client *milvusclient.Client
	vu     modules.VU
	mi     *ModuleInstance // Reference to module instance for metrics
}

// Close closes the Milvus client connection and releases associated resources.
// Should be called in k6 teardown function to ensure proper cleanup.
func (c *Client) Close() error {
	return c.client.Close(c.vu.Context())
}

// CreateCollectionFromJSON creates a collection from a JSON schema string.
// The JSON should contain a Schema object with collection name and field definitions.
// This method provides a convenient way to define complex schemas from JSON.
func (c *Client) CreateCollectionFromJSON(schemaJSON string) error {
	var schema Schema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return fmt.Errorf("failed to parse schema JSON: %v", err)
	}
	return c.CreateCollection(schema)
}

// CreateCollection creates a collection with a flexible schema.
// Supports various field types including vectors, scalars, and primary keys.
// The schema defines the structure and properties of the collection.
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
	return c.client.CreateCollection(c.vu.Context(), option)
}

// CreateCollectionSimple creates a collection with a simple auto-generated schema.
// Creates a collection with an auto-ID primary key and a vector field.
// This method provides backward compatibility with simpler usage patterns.
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
	return c.client.DropCollection(c.vu.Context(), option)
}

func (c *Client) HasCollection(collectionName string) (bool, error) {
	option := milvusclient.NewHasCollectionOption(collectionName)
	return c.client.HasCollection(c.vu.Context(), option)
}

func (c *Client) LoadCollection(collectionName string) error {
	option := milvusclient.NewLoadCollectionOption(collectionName)
	task, err := c.client.LoadCollection(c.vu.Context(), option)
	if err != nil {
		return err
	}

	// Wait for collection to be loaded
	return task.Await(c.vu.Context())
}

func (c *Client) ReleaseCollection(collectionName string) error {
	option := milvusclient.NewReleaseCollectionOption(collectionName)
	return c.client.ReleaseCollection(c.vu.Context(), option)
}

// Insert supports multiple field types with flexible data structure
func (c *Client) Insert(collectionName string, data map[string]interface{}) ([]int64, error) {
	start := time.Now()
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
	result, err := c.client.Insert(c.vu.Context(), option)

	// Calculate metrics
	duration := time.Since(start)
	vectorCount := int64(0)
	for _, col := range columns {
		if col.Len() > int(vectorCount) {
			vectorCount = int64(col.Len())
		}
	}

	// Emit metrics
	tags := map[string]string{
		"operation":  "insert",
		"collection": collectionName,
	}

	if err != nil {
		tags["status"] = "error"
		c.mi.emitMetric(c.mi.metrics.MilvusErrors, 1, tags)
		c.mi.emitMetric(c.mi.metrics.MilvusDuration, float64(duration.Milliseconds()), tags)
		return nil, fmt.Errorf("failed to insert: %v", err)
	}

	tags["status"] = "success"
	c.mi.emitMetric(c.mi.metrics.MilvusReqs, 1, tags)
	c.mi.emitMetric(c.mi.metrics.MilvusDuration, float64(duration.Milliseconds()), tags)
	c.mi.emitMetric(c.mi.metrics.MilvusVectors, float64(vectorCount), tags)
	c.mi.emitMetric(c.mi.metrics.MilvusErrors, 0, tags) // No error

	// Return placeholder IDs
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

// CreateIndex creates an index on a field with specified parameters.
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
	task, err := c.client.CreateIndex(c.vu.Context(), option)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}

	// Wait for index creation to complete
	err = task.Await(c.vu.Context())
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
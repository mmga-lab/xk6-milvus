package milvus

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// CreateCollectionFromJSON creates a collection from a JSON schema string
func (c *Client) CreateCollectionFromJSON(schemaJSON string) interface{} {
	start := time.Now()

	var schema Schema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to parse schema JSON: %v", err),
		})
	}

	return c.CreateCollection(schema)
}

// CreateCollection creates a collection with the given schema
func (c *Client) CreateCollection(schemaInput interface{}) interface{} {
	start := time.Now()

	// Convert interface{} to Schema using JSON marshal/unmarshal
	// This ensures proper handling of JSON tags from JavaScript objects
	var schema Schema
	schemaBytes, err := json.Marshal(schemaInput)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to marshal schema: %v", err),
		})
	}
	err = json.Unmarshal(schemaBytes, &schema)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to unmarshal schema: %v", err),
		})
	}

	entitySchema := entity.NewSchema().
		WithName(schema.Name).
		WithDescription(schema.Description)

	for _, field := range schema.Fields {
		entityField := entity.NewField().
			WithName(field.Name).
			WithDescription(field.Description)

		// Set data type
		if field.DataType == "" {
			return toMap(&OperationResult{
				Success:      false,
				ResponseTime: float64(time.Since(start).Milliseconds()),
				Error:        fmt.Sprintf("field %s has empty dataType", field.Name),
			})
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
			return toMap(&OperationResult{
				Success:      false,
				ResponseTime: float64(time.Since(start).Milliseconds()),
				Error:        fmt.Sprintf("unsupported data type: '%s' for field '%s'", field.DataType, field.Name),
			})
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
			return toMap(&OperationResult{
				Success:      false,
				ResponseTime: float64(time.Since(start).Milliseconds()),
				Error:        fmt.Sprintf("unsupported function type: %s", fn.FunctionType),
			})
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

	err = c.client.CreateCollection(c.ctx, option)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to create collection: %v", err),
		})
	}

	opResult := &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       map[string]interface{}{"collection": schema.Name},
	}

	// Emit metrics
	c.emitOperationMetrics(opResult, MetricMetadata{
		Operation:  "create_collection",
		Collection: schema.Name,
	})

	return toMap(opResult)
}

// DropCollection drops a collection
func (c *Client) DropCollection(collectionName ...string) interface{} {
	start := time.Now()

	name := c.defaultCollection
	if len(collectionName) > 0 && collectionName[0] != "" {
		name = collectionName[0]
	}

	option := milvusclient.NewDropCollectionOption(name)
	err := c.client.DropCollection(c.ctx, option)

	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to drop collection: %v", err),
		})
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       map[string]interface{}{"collection": name},
	})
}

// HasCollection checks if a collection exists
func (c *Client) HasCollection(collectionName ...string) interface{} {
	start := time.Now()

	name := c.defaultCollection
	if len(collectionName) > 0 && collectionName[0] != "" {
		name = collectionName[0]
	}

	if name == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        ErrCollectionNameRequired.Error(),
		})
	}

	option := milvusclient.NewHasCollectionOption(name)
	has, err := c.client.HasCollection(c.ctx, option)

	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to check collection: %v", err),
		})
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       has,
	})
}

// LoadCollection loads a collection into memory
func (c *Client) LoadCollection(collectionName ...string) interface{} {
	start := time.Now()

	name := c.defaultCollection
	if len(collectionName) > 0 && collectionName[0] != "" {
		name = collectionName[0]
	}

	if name == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        ErrCollectionNameRequired.Error(),
		})
	}

	option := milvusclient.NewLoadCollectionOption(name)
	task, err := c.client.LoadCollection(c.ctx, option)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to load collection: %v", err),
		})
	}

	// Wait for collection to be loaded
	err = task.Await(c.ctx)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to wait for collection load: %v", err),
		})
	}

	opResult := &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       map[string]interface{}{"collection": name},
	}

	// Emit metrics
	c.emitOperationMetrics(opResult, MetricMetadata{
		Operation:  "load_collection",
		Collection: name,
	})

	return toMap(opResult)
}

// ReleaseCollection releases a collection from memory
func (c *Client) ReleaseCollection(collectionName ...string) interface{} {
	start := time.Now()

	name := c.defaultCollection
	if len(collectionName) > 0 && collectionName[0] != "" {
		name = collectionName[0]
	}

	if name == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        ErrCollectionNameRequired.Error(),
		})
	}

	option := milvusclient.NewReleaseCollectionOption(name)
	err := c.client.ReleaseCollection(c.ctx, option)

	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to release collection: %v", err),
		})
	}

	opResult := &OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       map[string]interface{}{"collection": name},
	}

	// Emit metrics
	c.emitOperationMetrics(opResult, MetricMetadata{
		Operation:  "release_collection",
		Collection: name,
	})

	return toMap(opResult)
}

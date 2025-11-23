package milvus

import (
	"fmt"

	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
)

// convertDataToColumns converts map data to Milvus columns
func (c *Client) convertDataToColumns(data map[string]interface{}) ([]column.Column, error) {
	var columns []column.Column

	for fieldName, fieldData := range data {
		col, err := c.convertFieldToColumn(fieldName, fieldData)
		if err != nil {
			return nil, wrapError("convertDataToColumns", err)
		}
		if col != nil {
			columns = append(columns, col)
		}
	}

	if len(columns) == 0 {
		return nil, wrapError("convertDataToColumns", ErrEmptyData)
	}

	return columns, nil
}

// convertFieldToColumn converts a single field to a Milvus column
func (c *Client) convertFieldToColumn(fieldName string, fieldData interface{}) (column.Column, error) {
	switch v := fieldData.(type) {
	case [][]float32:
		if len(v) == 0 {
			return nil, nil // skip empty arrays
		}
		dim := len(v[0])
		return column.NewColumnFloatVector(fieldName, dim, v), nil

	case []int64:
		return column.NewColumnInt64(fieldName, v), nil

	case []int32:
		return column.NewColumnInt32(fieldName, v), nil

	case []float32:
		return column.NewColumnFloat(fieldName, v), nil

	case []float64:
		return column.NewColumnDouble(fieldName, v), nil

	case []string:
		return column.NewColumnVarChar(fieldName, v), nil

	case []bool:
		return column.NewColumnBool(fieldName, v), nil

	case []interface{}:
		return c.convertInterfaceSlice(fieldName, v)

	default:
		return nil, newError("convertFieldToColumn", ErrUnsupportedType,
			fmt.Sprintf("field %s has type %T", fieldName, fieldData))
	}
}

// convertInterfaceSlice converts []interface{} to appropriate column type
func (c *Client) convertInterfaceSlice(fieldName string, v []interface{}) (column.Column, error) {
	if len(v) == 0 {
		return nil, nil // skip empty arrays
	}

	switch v[0].(type) {
	case int64:
		ids := make([]int64, len(v))
		for i, val := range v {
			if id, ok := val.(int64); ok {
				ids[i] = id
			}
		}
		return column.NewColumnInt64(fieldName, ids), nil

	case string:
		strs := make([]string, len(v))
		for i, val := range v {
			if str, ok := val.(string); ok {
				strs[i] = str
			}
		}
		return column.NewColumnVarChar(fieldName, strs), nil

	case float64:
		return c.convertFloat64Slice(fieldName, v)

	case bool:
		bools := make([]bool, len(v))
		for i, val := range v {
			if b, ok := val.(bool); ok {
				bools[i] = b
			}
		}
		return column.NewColumnBool(fieldName, bools), nil

	case []interface{}:
		return c.convertNestedVectors(fieldName, v)

	case map[string]interface{}:
		// This is a sparse vector (array of objects)
		// Re-package as []map and call helper
		maps := make([]map[string]interface{}, len(v))
		for i, val := range v {
			if m, ok := val.(map[string]interface{}); ok {
				maps[i] = m
			}
		}
		return c.convertSparseVectors(fieldName, maps)

	default:
		return nil, newError("convertInterfaceSlice", ErrUnsupportedType,
			fmt.Sprintf("field %s has element type %T", fieldName, v[0]))
	}
}

// convertFloat64Slice converts []interface{} with float64 elements
func (c *Client) convertFloat64Slice(fieldName string, v []interface{}) (column.Column, error) {
	// Check if all values are integers
	isInteger := true
	for _, val := range v {
		f, ok := val.(float64)
		if !ok {
			return nil, wrapError("convertFloat64Slice", ErrInvalidDataType)
		}
		if f != float64(int64(f)) {
			isInteger = false
			break
		}
	}

	if isInteger && fieldName == "id" {
		ids := make([]int64, len(v))
		for i, val := range v {
			f, ok := val.(float64)
			if !ok {
				return nil, wrapError("convertFloat64Slice", ErrInvalidDataType)
			}
			ids[i] = int64(f)
		}
		return column.NewColumnInt64(fieldName, ids), nil
	}

	floats := make([]float32, len(v))
	for i, val := range v {
		f, ok := val.(float64)
		if !ok {
			return nil, wrapError("convertFloat64Slice", ErrInvalidDataType)
		}
		floats[i] = float32(f)
	}
	return column.NewColumnFloat(fieldName, floats), nil
}

// convertNestedVectors converts nested []interface{} to float vectors
func (c *Client) convertNestedVectors(fieldName string, v []interface{}) (column.Column, error) {
	if len(v) == 0 {
		return nil, wrapError("convertNestedVectors", ErrEmptyVectorArray)
	}

	firstVec, ok := v[0].([]interface{})
	if !ok {
		return nil, newError("convertNestedVectors", ErrInvalidDataType,
			fmt.Sprintf("field %s: expected []interface{}, got %T", fieldName, v[0]))
	}

	dim := len(firstVec)
	vectors := make([][]float32, len(v))

	for i, vecInterface := range v {
		vec, ok := vecInterface.([]interface{})
		if !ok {
			return nil, newError("convertNestedVectors", ErrInvalidDataType,
				fmt.Sprintf("field %s: vector %d is not []interface{}", fieldName, i))
		}

		floatVec := make([]float32, len(vec))
		for j, val := range vec {
			// Handle both float64 and int (JSON may encode integer floats as ints)
			switch v := val.(type) {
			case float64:
				floatVec[j] = float32(v)
			case int:
				floatVec[j] = float32(v)
			case int64:
				floatVec[j] = float32(v)
			default:
				return nil, newError("convertNestedVectors", ErrInvalidDataType,
					fmt.Sprintf("field %s: vector %d element %d is not float64: invalid data type", fieldName, i, j))
			}
		}
		vectors[i] = floatVec
	}

	return column.NewColumnFloatVector(fieldName, dim, vectors), nil
}

// convertSparseVectors converts array of sparse vector objects to SparseFloatVector column
func (c *Client) convertSparseVectors(fieldName string, v []map[string]interface{}) (column.Column, error) {
	if len(v) == 0 {
		return nil, nil // skip empty arrays
	}

	// Convert each sparse vector map to entity.SparseEmbedding
	sparseVectors := make([]entity.SparseEmbedding, len(v))
	for i, sparseMap := range v {
		var positions []uint32
		var values []float32
		
		for key, val := range sparseMap {
			// Convert string key to uint32
			var idx uint32
			fmt.Sscanf(key, "%d", &idx)
			if fval, ok := val.(float64); ok {
				positions = append(positions, idx)
				values = append(values, float32(fval))
			}
		}
		
		if sparse, err := entity.NewSliceSparseEmbedding(positions, values); err == nil {
			sparseVectors[i] = sparse
		} else {
			return nil, wrapError("convertSparseVectors", err)
		}
	}
	
	return column.NewColumnSparseVectors(fieldName, sparseVectors), nil
}

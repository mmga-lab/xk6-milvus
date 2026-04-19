package milvus

import (
	"encoding/json"
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
		return c.convertNestedArrays(fieldName, v)

	case map[string]interface{}:
		// Could be sparse vectors ({idx: val}) or JSON objects ({field: val})
		// Heuristic: if keys are numeric strings → sparse vector; otherwise → JSON
		firstMap := v[0].(map[string]interface{})
		isSparse := true
		for key := range firstMap {
			var idx uint32
			if _, err := fmt.Sscanf(key, "%d", &idx); err != nil {
				isSparse = false
				break
			}
		}
		if isSparse {
			maps := make([]map[string]interface{}, len(v))
			for i, val := range v {
				if m, ok := val.(map[string]interface{}); ok {
					maps[i] = m
				}
			}
			return c.convertSparseVectors(fieldName, maps)
		}
		// JSON objects: serialize each to []byte
		jsonBytes := make([][]byte, len(v))
		for i, val := range v {
			b, err := json.Marshal(val)
			if err != nil {
				return nil, newError("convertInterfaceSlice", ErrInvalidDataType,
					fmt.Sprintf("field %s: failed to marshal JSON at index %d", fieldName, i))
			}
			jsonBytes[i] = b
		}
		return column.NewColumnJSONBytes(fieldName, jsonBytes), nil

	default:
		return nil, newError("convertInterfaceSlice", ErrUnsupportedType,
			fmt.Sprintf("field %s has element type %T", fieldName, v[0]))
	}
}

// convertFloat64Slice converts []interface{} with float64 (or mixed int64/float64) elements.
// Goja JS runtime may encode integer values as int64 and fractional values as float64,
// causing mixed types within the same array. This function handles both.
func (c *Client) convertFloat64Slice(fieldName string, v []interface{}) (column.Column, error) {
	// Check if all values are integers (including int64 from Goja)
	isInteger := true
	for _, val := range v {
		switch f := val.(type) {
		case float64:
			if f != float64(int64(f)) {
				isInteger = false
			}
		case int64:
			// always integer
		default:
			return nil, wrapError("convertFloat64Slice", ErrInvalidDataType)
		}
		if !isInteger {
			break
		}
	}

	if isInteger && fieldName == "id" {
		ids := make([]int64, len(v))
		for i, val := range v {
			switch f := val.(type) {
			case float64:
				ids[i] = int64(f)
			case int64:
				ids[i] = f
			}
		}
		return column.NewColumnInt64(fieldName, ids), nil
	}

	floats := make([]float32, len(v))
	for i, val := range v {
		switch f := val.(type) {
		case float64:
			floats[i] = float32(f)
		case int64:
			floats[i] = float32(f)
		}
	}
	return column.NewColumnFloat(fieldName, floats), nil
}

// convertNestedArrays dispatches nested []interface{} to the correct column type
// by inspecting the first element's inner type: float64 → FloatVector, string → Array<VarChar>,
// bool → Array<Bool>, int64 → Array<Int64>, etc.
func (c *Client) convertNestedArrays(fieldName string, v []interface{}) (column.Column, error) {
	if len(v) == 0 {
		return nil, wrapError("convertNestedArrays", ErrEmptyVectorArray)
	}

	firstArr, ok := v[0].([]interface{})
	if !ok {
		return nil, newError("convertNestedArrays", ErrInvalidDataType,
			fmt.Sprintf("field %s: expected []interface{}, got %T", fieldName, v[0]))
	}

	if len(firstArr) == 0 {
		return nil, newError("convertNestedArrays", ErrEmptyVectorArray,
			fmt.Sprintf("field %s: first inner array is empty", fieldName))
	}

	// Dispatch based on first element's type
	switch firstArr[0].(type) {
	case float64, int64, int:
		return c.convertNestedNumericArrays(fieldName, v)
	case string:
		return c.convertNestedStringArrays(fieldName, v)
	case bool:
		return c.convertNestedBoolArrays(fieldName, v)
	case map[string]interface{}:
		// Array<Struct>: each row is an array of struct objects
		return c.convertNestedStructArrays(fieldName, v)
	default:
		return nil, newError("convertNestedArrays", ErrUnsupportedType,
			fmt.Sprintf("field %s: unsupported inner element type %T", fieldName, firstArr[0]))
	}
}

// convertNestedNumericArrays handles nested arrays of numbers.
// If all inner arrays have equal length, treats as FloatVector; otherwise as Array<Int64> or Array<Float>.
func (c *Client) convertNestedNumericArrays(fieldName string, v []interface{}) (column.Column, error) {
	// Check if this looks like a FloatVector (all same dimension) or an Array field
	isFloatVector := true
	firstLen := -1
	hasFloat := false
	hasNonNumeric := false

	for _, item := range v {
		arr, ok := item.([]interface{})
		if !ok {
			isFloatVector = false
			break
		}
		if firstLen == -1 {
			firstLen = len(arr)
		} else if len(arr) != firstLen {
			isFloatVector = false
		}
		for _, elem := range arr {
			switch f := elem.(type) {
			case float64:
				if f != float64(int64(f)) {
					hasFloat = true
				}
			case int64, int:
				// integer, continue
			default:
				hasNonNumeric = true
				isFloatVector = false
			}
		}
	}

	if hasNonNumeric {
		return nil, newError("convertNestedNumericArrays", ErrInvalidDataType,
			fmt.Sprintf("field %s: array contains non-numeric elements", fieldName))
	}

	// If all arrays have the same length and contain floats, treat as FloatVector
	if isFloatVector && (hasFloat || firstLen > 1) {
		dim := firstLen
		vectors := make([][]float32, len(v))
		for i, vecI := range v {
			vec := vecI.([]interface{})
			floatVec := make([]float32, len(vec))
			for j, val := range vec {
				switch f := val.(type) {
				case float64:
					floatVec[j] = float32(f)
				case int64:
					floatVec[j] = float32(f)
				case int:
					floatVec[j] = float32(f)
				default:
					return nil, newError("convertNestedNumericArrays", ErrInvalidDataType,
						fmt.Sprintf("field %s: vector %d element %d has type %T, expected numeric", fieldName, i, j, val))
				}
			}
			vectors[i] = floatVec
		}
		return column.NewColumnFloatVector(fieldName, dim, vectors), nil
	}

	// Variable-length or all-integer arrays → Array<Int64> or Array<Float>
	if hasFloat {
		// Array<Float>
		arrays := make([][]float32, len(v))
		for i, item := range v {
			arr, ok := item.([]interface{})
			if !ok {
				return nil, newError("convertNestedNumericArrays", ErrInvalidDataType,
					fmt.Sprintf("field %s: row %d is not array", fieldName, i))
			}
			floats := make([]float32, len(arr))
			for j, val := range arr {
				switch f := val.(type) {
				case float64:
					floats[j] = float32(f)
				case int64:
					floats[j] = float32(f)
				case int:
					floats[j] = float32(f)
				}
			}
			arrays[i] = floats
		}
		return column.NewColumnFloatArray(fieldName, arrays), nil
	}

	// Array<Int64>
	arrays := make([][]int64, len(v))
	for i, item := range v {
		arr, ok := item.([]interface{})
		if !ok {
			return nil, newError("convertNestedNumericArrays", ErrInvalidDataType,
				fmt.Sprintf("field %s: row %d is not array", fieldName, i))
		}
		ints := make([]int64, len(arr))
		for j, val := range arr {
			switch f := val.(type) {
			case float64:
				ints[j] = int64(f)
			case int64:
				ints[j] = f
			}
		}
		arrays[i] = ints
	}
	return column.NewColumnInt64Array(fieldName, arrays), nil
}

// convertNestedStringArrays converts nested string arrays to Array<VarChar>
func (c *Client) convertNestedStringArrays(fieldName string, v []interface{}) (column.Column, error) {
	arrays := make([][]string, len(v))
	for i, item := range v {
		arr, ok := item.([]interface{})
		if !ok {
			return nil, newError("convertNestedStringArrays", ErrInvalidDataType,
				fmt.Sprintf("field %s: row %d is not array", fieldName, i))
		}
		strs := make([]string, len(arr))
		for j, val := range arr {
			if s, ok := val.(string); ok {
				strs[j] = s
			}
		}
		arrays[i] = strs
	}
	return column.NewColumnVarCharArray(fieldName, arrays), nil
}

// convertNestedBoolArrays converts nested bool arrays to Array<Bool>
func (c *Client) convertNestedBoolArrays(fieldName string, v []interface{}) (column.Column, error) {
	arrays := make([][]bool, len(v))
	for i, item := range v {
		arr, ok := item.([]interface{})
		if !ok {
			return nil, newError("convertNestedBoolArrays", ErrInvalidDataType,
				fmt.Sprintf("field %s: row %d is not array", fieldName, i))
		}
		bools := make([]bool, len(arr))
		for j, val := range arr {
			if b, ok := val.(bool); ok {
				bools[j] = b
			}
		}
		arrays[i] = bools
	}
	return column.NewColumnBoolArray(fieldName, arrays), nil
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

// convertNestedStructArrays converts nested arrays of struct objects to a StructArray column.
// JS input (per row is an array of structs):
//
//	[[{name:"A",age:1,vec:[0.1,0.2]},{name:"B",age:2,vec:[0.3,0.4]}], [{name:"C",age:3,vec:[0.5,0.6]}]]
//
// Each sub-field becomes an Array column with numRows entries (one per row).
// Scalar sub-fields → Array<VarChar>, Array<Int64>, etc.
// Vector sub-fields → FloatVector with flattened data (Milvus VectorArray format).
func (c *Client) convertNestedStructArrays(fieldName string, v []interface{}) (column.Column, error) {
	numRows := len(v)
	firstRow, ok := v[0].([]interface{})
	if !ok || len(firstRow) == 0 {
		return nil, newError("convertNestedStructArrays", ErrInvalidDataType,
			fmt.Sprintf("field %s: first row is empty or not array", fieldName))
	}
	firstObj, ok := firstRow[0].(map[string]interface{})
	if !ok {
		return nil, newError("convertNestedStructArrays", ErrInvalidDataType,
			fmt.Sprintf("field %s: first struct element is not object", fieldName))
	}

	// Discover sub-field names and types from first struct object.
	// Process scalar fields first so columnStructArray.Len() returns numRows (not flattened vector count).
	type fieldInfo struct {
		name  string
		isVec bool
	}
	var scalarFields, vecFields []fieldInfo
	for key, val := range firstObj {
		fi := fieldInfo{name: key}
		if _, ok := val.([]interface{}); ok {
			fi.isVec = true
			vecFields = append(vecFields, fi)
		} else {
			scalarFields = append(scalarFields, fi)
		}
	}
	fields := append(scalarFields, vecFields...)

	// Build per-row arrays for each sub-field, then create Array columns
	var subColumns []column.Column
	for _, fi := range fields {
		if fi.isVec {
			// Vector sub-field: each row has multiple vectors → Array<FloatVector>
			// But SDK doesn't have ArrayOfVector column type, so we flatten into
			// a regular FloatVector column with total = sum of all struct counts
			var allVectors [][]float32
			dim := 0
			for _, rowI := range v {
				row, _ := rowI.([]interface{})
				for _, objI := range row {
					obj, _ := objI.(map[string]interface{})
					vecI, _ := obj[fi.name].([]interface{})
					if dim == 0 && len(vecI) > 0 {
						dim = len(vecI)
					}
					vec := make([]float32, len(vecI))
					for j, val := range vecI {
						switch f := val.(type) {
						case float64:
							vec[j] = float32(f)
						case int64:
							vec[j] = float32(f)
						}
					}
					allVectors = append(allVectors, vec)
				}
			}
			if dim > 0 {
				subColumns = append(subColumns, column.NewColumnFloatVector(fi.name, dim, allVectors))
			}
		} else {
			// Scalar sub-field: build Array<T> with numRows entries
			switch firstObj[fi.name].(type) {
			case string:
				arrays := make([][]string, numRows)
				for i, rowI := range v {
					row, _ := rowI.([]interface{})
					vals := make([]string, len(row))
					for j, objI := range row {
						obj, _ := objI.(map[string]interface{})
						if s, ok := obj[fi.name].(string); ok {
							vals[j] = s
						}
					}
					arrays[i] = vals
				}
				subColumns = append(subColumns, column.NewColumnVarCharArray(fi.name, arrays))
			case float64, int64:
				// Determine if all values are integer
				isInt := true
				for _, rowI := range v {
					row, _ := rowI.([]interface{})
					for _, objI := range row {
						obj, _ := objI.(map[string]interface{})
						if f, ok := obj[fi.name].(float64); ok && f != float64(int64(f)) {
							isInt = false
							break
						}
					}
					if !isInt {
						break
					}
				}
				if isInt {
					arrays := make([][]int64, numRows)
					for i, rowI := range v {
						row, _ := rowI.([]interface{})
						vals := make([]int64, len(row))
						for j, objI := range row {
							obj, _ := objI.(map[string]interface{})
							switch f := obj[fi.name].(type) {
							case float64:
								vals[j] = int64(f)
							case int64:
								vals[j] = f
							}
						}
						arrays[i] = vals
					}
					subColumns = append(subColumns, column.NewColumnInt64Array(fi.name, arrays))
				} else {
					arrays := make([][]float32, numRows)
					for i, rowI := range v {
						row, _ := rowI.([]interface{})
						vals := make([]float32, len(row))
						for j, objI := range row {
							obj, _ := objI.(map[string]interface{})
							if f, ok := obj[fi.name].(float64); ok {
								vals[j] = float32(f)
							}
						}
						arrays[i] = vals
					}
					subColumns = append(subColumns, column.NewColumnFloatArray(fi.name, arrays))
				}
			case bool:
				arrays := make([][]bool, numRows)
				for i, rowI := range v {
					row, _ := rowI.([]interface{})
					vals := make([]bool, len(row))
					for j, objI := range row {
						obj, _ := objI.(map[string]interface{})
						if b, ok := obj[fi.name].(bool); ok {
							vals[j] = b
						}
					}
					arrays[i] = vals
				}
				subColumns = append(subColumns, column.NewColumnBoolArray(fi.name, arrays))
			}
		}
	}

	return column.NewColumnStructArray(fieldName, subColumns), nil
}

// convertToSearchVectors converts various input types to []entity.Vector for search.
// Supports: [][]float32 (dense), []string (BM25 text), and mixed via JSON round-trip.
func convertToSearchVectors(input interface{}) ([]entity.Vector, error) {
	// Fast path: already [][]float32
	if vecs, ok := input.([][]float32); ok {
		result := make([]entity.Vector, len(vecs))
		for i, v := range vecs {
			result[i] = entity.FloatVector(v)
		}
		return result, nil
	}

	// JSON round-trip for Goja runtime values
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search input: %w", err)
	}

	// Try [][]float32 (dense vectors)
	var denseVecs [][]float32
	if err := json.Unmarshal(data, &denseVecs); err == nil && len(denseVecs) > 0 {
		// Verify it's actually numeric arrays, not strings parsed as arrays
		if len(denseVecs[0]) > 0 {
			result := make([]entity.Vector, len(denseVecs))
			for i, v := range denseVecs {
				result[i] = entity.FloatVector(v)
			}
			return result, nil
		}
	}

	// Try []string (BM25 text queries)
	var textQueries []string
	if err := json.Unmarshal(data, &textQueries); err == nil && len(textQueries) > 0 {
		result := make([]entity.Vector, len(textQueries))
		for i, text := range textQueries {
			result[i] = entity.Text(text)
		}
		return result, nil
	}

	// Try []map[string]interface{} (sparse vectors)
	var sparseMaps []map[string]interface{}
	if err := json.Unmarshal(data, &sparseMaps); err == nil && len(sparseMaps) > 0 {
		result := make([]entity.Vector, len(sparseMaps))
		for i, sparseMap := range sparseMaps {
			var positions []uint32
			var values []float32
			for key, val := range sparseMap {
				var idx uint32
				fmt.Sscanf(key, "%d", &idx)
				if fval, ok := val.(float64); ok {
					positions = append(positions, idx)
					values = append(values, float32(fval))
				}
			}
			sparse, err := entity.NewSliceSparseEmbedding(positions, values)
			if err != nil {
				return nil, fmt.Errorf("failed to create sparse embedding: %w", err)
			}
			result[i] = sparse
		}
		return result, nil
	}

	return nil, fmt.Errorf("unsupported search vector format")
}

package milvus

import (
	"testing"

	"github.com/milvus-io/milvus-proto/go-api/v3/schemapb"
	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.k6.io/k6/js/modules"
)

func TestConvertDataToColumns(t *testing.T) {
	// Create a minimal client for testing
	client := &Client{}

	tests := []struct {
		name        string
		data        map[string]any
		wantErr     bool
		errContains string
		validate    func(t *testing.T, cols []column.Column)
	}{
		{
			name: "valid float32 vector",
			data: map[string]any{
				"vector": [][]float32{{0.1, 0.2}, {0.3, 0.4}},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "vector", cols[0].Name())
			},
		},
		{
			name: "valid int64 field",
			data: map[string]any{
				"id": []int64{1, 2, 3},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "id", cols[0].Name())
			},
		},
		{
			name: "valid string field",
			data: map[string]any{
				"title": []string{"a", "b", "c"},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "title", cols[0].Name())
			},
		},
		{
			name: "valid float32 field",
			data: map[string]any{
				"price": []float32{1.1, 2.2, 3.3},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "price", cols[0].Name())
			},
		},
		{
			name: "valid bool field",
			data: map[string]any{
				"active": []bool{true, false, true},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "active", cols[0].Name())
			},
		},
		{
			name: "interface slice with int64",
			data: map[string]any{
				"id": []any{int64(1), int64(2), int64(3)},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "id", cols[0].Name())
			},
		},
		{
			name: "interface slice with strings",
			data: map[string]any{
				"title": []any{"a", "b", "c"},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "title", cols[0].Name())
			},
		},
		{
			name: "interface slice with float64 as integers (id field)",
			data: map[string]any{
				"id": []any{float64(1), float64(2), float64(3)},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "id", cols[0].Name())
			},
		},
		{
			name: "interface slice with float64 (non-id field)",
			data: map[string]any{
				"price": []any{float64(1.5), float64(2.5), float64(3.5)},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "price", cols[0].Name())
			},
		},
		{
			name: "interface slice with nested vectors",
			data: map[string]any{
				"vector": []any{
					[]any{float64(0.1), float64(0.2)},
					[]any{float64(0.3), float64(0.4)},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "vector", cols[0].Name())
			},
		},
		{
			name: "multiple fields",
			data: map[string]any{
				"id":     []int64{1, 2},
				"title":  []string{"a", "b"},
				"vector": [][]float32{{0.1, 0.2}, {0.3, 0.4}},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				assert.Len(t, cols, 3)
			},
		},
		{
			name: "struct array with vector subfield",
			data: map[string]any{
				"clips": []any{
					[]any{
						map[string]any{"tag": "a", "age": float64(10), "emb": []any{float64(0.1), float64(0.2)}},
						map[string]any{"tag": "b", "age": float64(11), "emb": []any{float64(0.3), float64(0.4)}},
					},
					[]any{
						map[string]any{"tag": "c", "age": float64(12), "emb": []any{float64(0.5), float64(0.6)}},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				require.Equal(t, "clips", cols[0].Name())

				fd := cols[0].FieldData()
				require.Equal(t, schemapb.DataType_ArrayOfStruct, fd.GetType())

				var embFieldFound bool
				for _, subField := range fd.GetStructArrays().GetFields() {
					if subField.GetFieldName() == "emb" {
						embFieldFound = true
						assert.Equal(t, schemapb.DataType_ArrayOfVector, subField.GetType())
						assert.EqualValues(t, 2, subField.GetVectors().GetVectorArray().GetDim())
						assert.Len(t, subField.GetVectors().GetVectorArray().GetData(), 2)
					}
				}
				assert.True(t, embFieldFound)
			},
		},
		{
			name:        "empty data",
			data:        map[string]any{},
			wantErr:     true,
			errContains: "no valid columns",
		},
		{
			name: "empty vector array",
			data: map[string]any{
				"vector": [][]float32{},
			},
			wantErr:     true,
			errContains: "no valid columns",
		},
		{
			name: "unsupported type",
			data: map[string]any{
				"invalid": map[string]string{"key": "value"},
			},
			wantErr:     true,
			errContains: "unsupported type",
		},
		{
			name: "valid int32 field",
			data: map[string]any{
				"count": []int32{1, 2, 3},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "count", cols[0].Name())
			},
		},
		{
			name: "valid float64 field",
			data: map[string]any{
				"score": []float64{1.5, 2.5, 3.5},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "score", cols[0].Name())
			},
		},
		{
			name: "interface slice with bool",
			data: map[string]any{
				"enabled": []any{true, false, true},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "enabled", cols[0].Name())
			},
		},
		{
			name: "interface slice with unsupported element type",
			data: map[string]any{
				"data": []any{map[string]string{"key": "value"}},
			},
			wantErr:     true,
			errContains: "unsupported type",
		},
		{
			name: "empty interface slice",
			data: map[string]any{
				"empty": []any{},
			},
			wantErr:     true,
			errContains: "no valid columns",
		},
		{
			name: "nested vectors with int elements",
			data: map[string]any{
				"vector": []any{
					[]any{1, 2, 3},
					[]any{4, 5, 6},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "vector", cols[0].Name())
			},
		},
		{
			name: "nested vectors with int64 elements",
			data: map[string]any{
				"vector": []any{
					[]any{int64(1), int64(2), int64(3)},
					[]any{int64(4), int64(5), int64(6)},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "vector", cols[0].Name())
			},
		},
		{
			name: "nested vectors with mixed valid types",
			data: map[string]any{
				"vector": []any{
					[]any{float64(1.0), 2, int64(3)},
					[]any{float64(4.0), 5, int64(6)},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
				assert.Equal(t, "vector", cols[0].Name())
			},
		},
		{
			name: "nested vectors with invalid element type",
			data: map[string]any{
				"vector": []any{
					[]any{float64(1.0), "invalid"},
				},
			},
			wantErr:     true,
			errContains: "non-numeric elements",
		},
		{
			name: "nested vectors with non-slice first element but nested second",
			data: map[string]any{
				"vector": []any{
					[]any{float64(1.0), float64(2.0)},
					"not a vector", // This will fail during conversion
				},
			},
			wantErr:     true,
			errContains: "invalid data type",
		},
		{
			name: "float64 slice with non-float64 element error path",
			data: map[string]any{
				"mixed": []any{float64(1.0)},
			},
			wantErr: false, // This should succeed with float conversion
			validate: func(t *testing.T, cols []column.Column) {
				require.Len(t, cols, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, err := client.convertDataToColumns(tt.data)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, cols)
				if tt.validate != nil {
					tt.validate(t, cols)
				}
			}
		})
	}
}

func TestGetCollectionName(t *testing.T) {
	tests := []struct {
		name              string
		defaultCollection string
		params            []string
		want              string
	}{
		{
			name:              "use default collection",
			defaultCollection: "test_collection",
			params:            nil,
			want:              "test_collection",
		},
		{
			name:              "use provided collection",
			defaultCollection: "test_collection",
			params:            []string{"other_collection"},
			want:              "other_collection",
		},
		{
			name:              "empty provided collection, use default",
			defaultCollection: "test_collection",
			params:            []string{""},
			want:              "test_collection",
		},
		{
			name:              "no default, no params",
			defaultCollection: "",
			params:            nil,
			want:              "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				defaultCollection: tt.defaultCollection,
			}

			got := client.getCollectionName(tt.params...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestModuleInterface(t *testing.T) {
	// Test that RootModule implements modules.Module
	var _ modules.Module = &RootModule{}

	// Test that Milvus implements modules.Instance
	var _ modules.Instance = &Milvus{}
}

func TestNewModuleInstance(t *testing.T) {
	root := &RootModule{}

	// Test with nil VU (we don't need a real VU for this test)
	instance := root.NewModuleInstance(nil)

	assert.NotNil(t, instance)

	// Verify it returns a Milvus instance
	milvus, ok := instance.(*Milvus)
	assert.True(t, ok, "instance should be of type *Milvus")
	assert.NotNil(t, milvus)
}

func TestExports(t *testing.T) {
	milvus := &Milvus{vu: nil}

	exports := milvus.Exports()

	// Verify default export
	assert.NotNil(t, exports.Default)
	assert.Equal(t, milvus, exports.Default)

	// Verify named exports
	assert.NotNil(t, exports.Named)
	assert.Contains(t, exports.Named, "client")
	assert.Contains(t, exports.Named, "clientWithCollection")

	// Verify the functions are not nil
	assert.NotNil(t, exports.Named["client"])
	assert.NotNil(t, exports.Named["clientWithCollection"])
}

func TestOperationResultStructure(t *testing.T) {
	// Test OperationResult structure
	result := &OperationResult{
		Success:      true,
		ResponseTime: 123.45,
		Result:       map[string]any{"count": 10},
		Error:        "",
		Empty:        false,
		Recall:       0.95,
	}

	assert.True(t, result.Success)
	assert.Equal(t, 123.45, result.ResponseTime)
	assert.NotNil(t, result.Result)
	assert.Empty(t, result.Error)
	assert.False(t, result.Empty)
	assert.Equal(t, float32(0.95), result.Recall)
}

func TestConvertToSearchVectorsFloatVectorArray(t *testing.T) {
	vectors, err := convertToSearchVectors([]any{
		[]any{
			[]any{float64(0.1), float64(0.2)},
			[]any{float64(0.3), float64(0.4)},
		},
	})

	require.NoError(t, err)
	require.Len(t, vectors, 1)

	vectorArray, ok := vectors[0].(entity.FloatVectorArray)
	require.True(t, ok)
	require.Len(t, vectorArray, 2)
	assert.Equal(t, entity.FloatVector{0.1, 0.2}, vectorArray[0])
	assert.Equal(t, entity.FloatVector{0.3, 0.4}, vectorArray[1])
}

func TestSchemaStructure(t *testing.T) {
	// Test Schema structure
	schema := Schema{
		Name: "test_collection",
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
				Dimension: 128,
			},
		},
		NumShards: 2,
	}

	assert.Equal(t, "test_collection", schema.Name)
	assert.Len(t, schema.Fields, 2)
	assert.Equal(t, int32(2), schema.NumShards)
}

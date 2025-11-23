package milvus

import (
	"testing"

	"github.com/milvus-io/milvus/client/v2/column"
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
			name: "empty data",
			data: map[string]any{},
			wantErr: true,
			errContains: "no valid columns",
		},
		{
			name: "empty vector array",
			data: map[string]any{
				"vector": [][]float32{},
			},
			wantErr: true,
			errContains: "no valid columns",
		},
		{
			name: "unsupported type",
			data: map[string]any{
				"invalid": map[string]string{"key": "value"},
			},
			wantErr: true,
			errContains: "unsupported type",
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

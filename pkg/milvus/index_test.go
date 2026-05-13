package milvus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildIndexScalarTypes(t *testing.T) {
	tests := []struct {
		name      string
		indexType string
		wantType  string
	}{
		{name: "inverted", indexType: "INVERTED", wantType: "INVERTED"},
		{name: "stl sort", indexType: "STL_SORT", wantType: "STL_SORT"},
		{name: "bitmap", indexType: "BITMAP", wantType: "BITMAP"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, indexType, _, err := buildIndex(map[string]interface{}{
				"indexType": tt.indexType,
			})

			require.NoError(t, err)
			require.NotNil(t, idx)
			assert.Equal(t, tt.indexType, indexType)
			assert.Equal(t, tt.wantType, idx.Params()["index_type"])
		})
	}
}

func TestBuildIndexNumericParamsFromJS(t *testing.T) {
	idx, indexType, indexName, err := buildIndex(map[string]interface{}{
		"index_type":  "HNSW",
		"metric_type": "MAX_SIM_COSINE",
		"indexName":   "struct_embedding_idx",
		"params": map[string]interface{}{
			"M":              float64(32),
			"efConstruction": float64(128),
		},
	})

	require.NoError(t, err)
	require.NotNil(t, idx)
	assert.Equal(t, "HNSW", indexType)
	assert.Equal(t, "struct_embedding_idx", indexName)
	assert.Equal(t, "MAX_SIM_COSINE", idx.Params()["metric_type"])
	assert.Equal(t, "32", idx.Params()["M"])
	assert.Equal(t, "128", idx.Params()["efConstruction"])
}

func TestBuildIndexUnsupportedType(t *testing.T) {
	idx, _, _, err := buildIndex(map[string]interface{}{
		"indexType": "UNSUPPORTED_INDEX_TYPE",
	})

	assert.Nil(t, idx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported index type")
}

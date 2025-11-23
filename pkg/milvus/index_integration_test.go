//go:build integration
// +build integration

package milvus

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateIndex_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestClientWithoutIndex(t)
	defer cleanup()

	// Insert some data first (can insert without index)
	vectors := make([][]float32, 10)
	ids := make([]int64, 10)
	titles := make([]string, 10)
	for i := 0; i < 10; i++ {
		vectors[i] = make([]float32, 128)
		for j := 0; j < 128; j++ {
			vectors[i][j] = float32(i*128 + j)
		}
		ids[i] = int64(i + 1)
		titles[i] = "Test " + string(rune('A'+i))
	}

	insertData := map[string]interface{}{
		"id":     ids,
		"title":  titles,
		"vector": vectors,
	}

	insertResult := client.Insert(insertData)
	insertMap := insertResult.(map[string]interface{})
	require.Equal(t, true, insertMap["success"])

	// Wait for data to be flushed
	time.Sleep(500 * time.Millisecond)

	// Test creating a FLAT index
	// Note: Milvus doesn't allow multiple indexes on the same field,
	// so we can only test one index creation per collection
	indexParams := map[string]interface{}{
		"indexType":  "FLAT",
		"metricType": "L2",
	}

	result := client.CreateIndex("vector", indexParams)
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, true, resultMap["success"], "flat index creation should succeed")
	assert.Greater(t, resultMap["response_time_ms"].(float64), 0.0)

	if resultData, ok := resultMap["result"].(map[string]interface{}); ok {
		assert.Equal(t, "vector", resultData["field"])
		assert.Equal(t, "FLAT", resultData["index_type"])
	}
}

func TestCreateIndex_MissingCollection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	milvusHost := os.Getenv("MILVUS_HOST")
	if milvusHost == "" {
		milvusHost = "localhost:19530"
	}

	milvusModule := &Milvus{
		vu: &mockVU{ctx: context.Background()},
	}

	// Create client without default collection
	client, err := milvusModule.Client(milvusHost)
	require.NoError(t, err)
	defer client.Close()

	t.Run("create_index_missing_collection_name", func(t *testing.T) {
		indexParams := map[string]interface{}{
			"indexType":  "FLAT",
			"metricType": "L2",
		}

		result := client.CreateIndex("vector", indexParams)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, false, resultMap["success"])
		assert.Contains(t, resultMap["error"], "collection name required")
	})
}

func TestCreateIndex_InvalidParams_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestClientWithoutIndex(t)
	defer cleanup()

	// Insert data (can insert without index)

	vectors := make([][]float32, 5)
	for i := 0; i < 5; i++ {
		vectors[i] = make([]float32, 128)
		for j := 0; j < 128; j++ {
			vectors[i][j] = float32(i*128 + j)
		}
	}

	insertData := map[string]interface{}{
		"id":     []int64{1, 2, 3, 4, 5},
		"title":  []string{"A", "B", "C", "D", "E"},
		"vector": vectors,
	}

	insertResult := client.Insert(insertData)
	require.Equal(t, true, insertResult.(map[string]interface{})["success"])
	time.Sleep(500 * time.Millisecond)

	t.Run("create_index_with_unsupported_type", func(t *testing.T) {
		indexParams := map[string]interface{}{
			"indexType":  "UNSUPPORTED_INDEX_TYPE",
			"metricType": "L2",
		}

		result := client.CreateIndex("vector", indexParams)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, false, resultMap["success"])
		assert.Contains(t, resultMap["error"], "unsupported index type")
	})
}

func TestCreateIndex_IVFVariants_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestClientWithoutIndex(t)
	defer cleanup()

	// Prepare and insert data (can insert without index)

	vectors := make([][]float32, 20)
	ids := make([]int64, 20)
	titles := make([]string, 20)
	for i := 0; i < 20; i++ {
		vectors[i] = make([]float32, 128)
		for j := 0; j < 128; j++ {
			vectors[i][j] = float32(i*128 + j)
		}
		ids[i] = int64(i + 1)
		titles[i] = "Data " + string(rune('A'+i))
	}

	insertData := map[string]interface{}{
		"id":     ids,
		"title":  titles,
		"vector": vectors,
	}

	insertResult := client.Insert(insertData)
	require.Equal(t, true, insertResult.(map[string]interface{})["success"])
	time.Sleep(500 * time.Millisecond)

	// Test creating an IVF_SQ8 index
	// Note: Testing only one index type per collection since Milvus doesn't allow
	// multiple indexes on the same field
	indexParams := map[string]interface{}{
		"indexType":  "IVF_SQ8",
		"metricType": "L2",
		"nlist":      64,
	}

	result := client.CreateIndex("vector", indexParams)
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, true, resultMap["success"], "IVF_SQ8 index creation should succeed")
}

func TestCreateIndex_ResponseTime_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestClientWithoutIndex(t)
	defer cleanup()

	// Prepare and insert minimal data (can insert without index)

	vectors := make([][]float32, 3)
	for i := 0; i < 3; i++ {
		vectors[i] = make([]float32, 128)
		for j := 0; j < 128; j++ {
			vectors[i][j] = float32(i*128 + j)
		}
	}

	insertData := map[string]interface{}{
		"id":     []int64{1, 2, 3},
		"title":  []string{"A", "B", "C"},
		"vector": vectors,
	}

	insertResult := client.Insert(insertData)
	require.Equal(t, true, insertResult.(map[string]interface{})["success"])
	time.Sleep(500 * time.Millisecond)

	t.Run("index_creation_has_response_time", func(t *testing.T) {
		indexParams := map[string]interface{}{
			"indexType":  "FLAT",
			"metricType": "L2",
		}

		result := client.CreateIndex("vector", indexParams)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.NotNil(t, resultMap["response_time_ms"])
		assert.Greater(t, resultMap["response_time_ms"].(float64), 0.0)
	})
}

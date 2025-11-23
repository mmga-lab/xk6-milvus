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

func TestSearch_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestClientWithoutIndex(t)
	defer cleanup()

	// Prepare data
	vectors := make([][]float32, 10)
	ids := make([]int64, 10)
	titles := make([]string, 10)
	for i := 0; i < 10; i++ {
		vectors[i] = make([]float32, 128)
		for j := 0; j < 128; j++ {
			vectors[i][j] = float32(i*128 + j)
		}
		ids[i] = int64(i + 1)
		titles[i] = "Product " + string(rune('A'+i))
	}

	insertData := map[string]interface{}{
		"id":     ids,
		"title":  titles,
		"vector": vectors,
	}

	// Insert data
	insertResult := client.Insert(insertData)
	require.Equal(t, true, insertResult.(map[string]interface{})["success"])

	// Wait for data to be flushed
	time.Sleep(500 * time.Millisecond)

	// Create index (required before loading)
	indexParams := map[string]interface{}{
		"indexType":  "FLAT",
		"metricType": "L2",
	}
	indexResult := client.CreateIndex("vector", indexParams)
	require.Equal(t, true, indexResult.(map[string]interface{})["success"])

	// Load collection after creating index
	loadResult := client.LoadCollection()
	require.Equal(t, true, loadResult.(map[string]interface{})["success"])

	time.Sleep(500 * time.Millisecond)

	t.Run("basic_search", func(t *testing.T) {
		searchVectors := [][]float32{{vectors[0][0], vectors[0][1]}}
		// Pad to 128 dimensions
		for len(searchVectors[0]) < 128 {
			searchVectors[0] = append(searchVectors[0], 0)
		}

		params := map[string]interface{}{
			"vectorField":  "vector",
			"outputFields": []string{"id", "title"},
		}

		result := client.Search(searchVectors, 5, params)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"], "search should succeed")
		assert.Greater(t, resultMap["response_time_ms"].(float64), 0.0)
		assert.Equal(t, false, resultMap["empty"], "search should return results")
	})

	t.Run("search_with_filter", func(t *testing.T) {
		searchVectors := [][]float32{vectors[0]}

		params := map[string]interface{}{
			"vectorField":  "vector",
			"outputFields": []string{"id", "title"},
			"expr":         "id > 5",
		}

		result := client.Search(searchVectors, 3, params)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
		assert.NotNil(t, resultMap["result"])
	})

	t.Run("search_with_metric_type", func(t *testing.T) {
		searchVectors := [][]float32{vectors[0]}

		params := map[string]interface{}{
			"vectorField":  "vector",
			"outputFields": []string{"id"},
			"metricType":   "L2",
		}

		result := client.Search(searchVectors, 5, params)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
	})

	t.Run("search_multiple_vectors", func(t *testing.T) {
		searchVectors := [][]float32{vectors[0], vectors[1], vectors[2]}

		params := map[string]interface{}{
			"vectorField":  "vector",
			"outputFields": []string{"id", "title"},
		}

		result := client.Search(searchVectors, 3, params)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
		assert.Equal(t, false, resultMap["empty"])
	})
}

func TestSearch_MissingCollection_Integration(t *testing.T) {
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

	client, err := milvusModule.Client(milvusHost)
	require.NoError(t, err)
	defer client.Close()

	t.Run("search_missing_collection_name", func(t *testing.T) {
		searchVector := make([]float32, 128)
		searchVectors := [][]float32{searchVector}

		params := map[string]interface{}{
			"vectorField": "vector",
		}

		result := client.Search(searchVectors, 5, params)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, false, resultMap["success"])
		assert.Contains(t, resultMap["error"], "collection name required")
	})
}

func TestQuery_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestClient(t)
	defer cleanup()

	// Prepare and insert data
	vectors := make([][]float32, 10)
	ids := make([]int64, 10)
	titles := make([]string, 10)
	for i := 0; i < 10; i++ {
		vectors[i] = make([]float32, 128)
		for j := 0; j < 128; j++ {
			vectors[i][j] = float32(i*128 + j)
		}
		ids[i] = int64(i + 1)
		titles[i] = "Item " + string(rune('A'+i))
	}

	insertData := map[string]interface{}{
		"id":     ids,
		"title":  titles,
		"vector": vectors,
	}

	insertResult := client.Insert(insertData)
	require.Equal(t, true, insertResult.(map[string]interface{})["success"])

	loadResult := client.LoadCollection()
	require.Equal(t, true, loadResult.(map[string]interface{})["success"])

	time.Sleep(500 * time.Millisecond)

	t.Run("query_with_filter", func(t *testing.T) {
		outputFields := []interface{}{"id", "title"}

		result := client.Query("id <= 5", outputFields)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"], "query should succeed")
		assert.Greater(t, resultMap["response_time_ms"].(float64), 0.0)
		assert.Equal(t, false, resultMap["empty"], "query should return results")

		// Check result structure
		if resultData, ok := resultMap["result"].([]interface{}); ok {
			assert.Greater(t, len(resultData), 0, "should have query results")
		}
	})

	t.Run("query_with_id_filter", func(t *testing.T) {
		outputFields := []interface{}{"id", "title"}

		result := client.Query("id in [1, 3, 5]", outputFields)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
		assert.Equal(t, false, resultMap["empty"])
	})

	t.Run("query_no_results", func(t *testing.T) {
		outputFields := []interface{}{"id"}

		result := client.Query("id > 1000", outputFields)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
		assert.Equal(t, true, resultMap["empty"], "should be empty when no results")
	})

	t.Run("query_with_explicit_collection", func(t *testing.T) {
		collectionName := client.defaultCollection
		outputFields := []interface{}{"id"}

		result := client.Query("id > 0", outputFields, collectionName)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
	})
}

func TestQuery_MissingCollection_Integration(t *testing.T) {
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

	client, err := milvusModule.Client(milvusHost)
	require.NoError(t, err)
	defer client.Close()

	t.Run("query_missing_collection_name", func(t *testing.T) {
		outputFields := []interface{}{"id"}

		result := client.Query("id > 0", outputFields)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, false, resultMap["success"])
		assert.Contains(t, resultMap["error"], "collection name required")
	})
}

func TestSearchOperations_ResponseTime_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestClient(t)
	defer cleanup()

	// Prepare minimal data
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

	loadResult := client.LoadCollection()
	require.Equal(t, true, loadResult.(map[string]interface{})["success"])

	time.Sleep(500 * time.Millisecond)

	t.Run("search_has_response_time", func(t *testing.T) {
		searchVectors := [][]float32{vectors[0]}
		params := map[string]interface{}{
			"vectorField": "vector",
		}

		result := client.Search(searchVectors, 2, params)
		resultMap := result.(map[string]interface{})

		assert.NotNil(t, resultMap["response_time_ms"])
		assert.Greater(t, resultMap["response_time_ms"].(float64), 0.0)
	})

	t.Run("query_has_response_time", func(t *testing.T) {
		outputFields := []interface{}{"id"}

		result := client.Query("id > 0", outputFields)
		resultMap := result.(map[string]interface{})

		assert.NotNil(t, resultMap["response_time_ms"])
		assert.Greater(t, resultMap["response_time_ms"].(float64), 0.0)
	})
}

func TestSearch_EmptyResults_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestClient(t)
	defer cleanup()

	// Insert only one record
	vectors := [][]float32{{}}
	for j := 0; j < 128; j++ {
		vectors[0] = append(vectors[0], float32(j))
	}

	insertData := map[string]interface{}{
		"id":     []int64{1},
		"title":  []string{"Single"},
		"vector": vectors,
	}

	insertResult := client.Insert(insertData)
	require.Equal(t, true, insertResult.(map[string]interface{})["success"])

	loadResult := client.LoadCollection()
	require.Equal(t, true, loadResult.(map[string]interface{})["success"])

	time.Sleep(500 * time.Millisecond)

	t.Run("search_with_impossible_filter", func(t *testing.T) {
		searchVectors := [][]float32{vectors[0]}

		params := map[string]interface{}{
			"vectorField":  "vector",
			"outputFields": []string{"id"},
			"expr":         "id > 1000", // No records match
		}

		result := client.Search(searchVectors, 5, params)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
		assert.Equal(t, true, resultMap["empty"], "should be empty when filter matches nothing")
	})
}

func TestSearch_Recall_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestClient(t)
	defer cleanup()

	// Insert enough data for recall testing
	numVectors := 50
	vectors := make([][]float32, numVectors)
	ids := make([]int64, numVectors)
	titles := make([]string, numVectors)

	for i := 0; i < numVectors; i++ {
		vectors[i] = make([]float32, 128)
		for j := 0; j < 128; j++ {
			vectors[i][j] = float32(i*128 + j)
		}
		ids[i] = int64(i + 1)
		titles[i] = "Doc " + string(rune('A'+(i%26)))
	}

	insertData := map[string]interface{}{
		"id":     ids,
		"title":  titles,
		"vector": vectors,
	}

	insertResult := client.Insert(insertData)
	require.Equal(t, true, insertResult.(map[string]interface{})["success"])

	loadResult := client.LoadCollection()
	require.Equal(t, true, loadResult.(map[string]interface{})["success"])

	time.Sleep(1 * time.Second)

	t.Run("search_includes_recall_metric", func(t *testing.T) {
		searchVectors := [][]float32{vectors[0]}

		params := map[string]interface{}{
			"vectorField":  "vector",
			"outputFields": []string{"id"},
		}

		result := client.Search(searchVectors, 10, params)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
		assert.NotNil(t, resultMap["recall"], "recall metric should be present")
		// Recall should be between 0 and 1
		recall := resultMap["recall"].(float64)
		assert.GreaterOrEqual(t, recall, 0.0)
		assert.LessOrEqual(t, recall, 1.0)
	})
}

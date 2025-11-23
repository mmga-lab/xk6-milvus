// go:build integration
//go:build integration
// +build integration

package milvus

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/grafana/sobek"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/lib"
)

// mockVU implements modules.VU interface for testing
type mockVU struct {
	ctx context.Context
}

func (m *mockVU) Context() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

func (m *mockVU) Events() common.Events {
	return common.Events{}
}

func (m *mockVU) InitEnv() *common.InitEnvironment {
	return nil
}

func (m *mockVU) State() *lib.State {
	return nil
}

func (m *mockVU) Runtime() *sobek.Runtime {
	return nil
}

func (m *mockVU) RegisterCallback() func(func() error) {
	return func(f func() error) {}
}

// setupTestClientWithoutIndex creates a test client and collection but no index
// Use this for tests that need to test index creation themselves
func setupTestClientWithoutIndex(t *testing.T) (*Client, string, func()) {
	t.Helper()

	milvusHost := os.Getenv("MILVUS_HOST")
	if milvusHost == "" {
		milvusHost = "localhost:19530"
	}

	collectionName := fmt.Sprintf("test_collection_%d", time.Now().UnixNano())

	milvusModule := &Milvus{
		vu: &mockVU{
			ctx: context.Background(),
		},
	}

	client, err := milvusModule.ClientWithCollection(milvusHost, collectionName)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create test collection
	schema := Schema{
		Name: collectionName,
		Fields: []Field{
			{
				Name:         "id",
				DataType:     "Int64",
				IsPrimaryKey: true,
				IsAutoID:     false,
			},
			{
				Name:     "title",
				DataType: "VarChar",
				MaxLength: 200,
			},
			{
				Name:      "vector",
				DataType:  "FloatVector",
				Dimension: 128,
			},
		},
		NumShards: 2,
	}

	result := client.CreateCollection(schema)
	resultMap, ok := result.(map[string]interface{})
	if !ok || !resultMap["success"].(bool) {
		t.Fatalf("Failed to create collection: %v", result)
	}

	cleanup := func() {
		// Drop collection
		client.DropCollection()
		client.Close()
	}

	return client, collectionName, cleanup
}

// setupTestClient creates a test client for integration tests with index and loaded collection
func setupTestClient(t *testing.T) (*Client, string, func()) {
	client, collectionName, cleanup := setupTestClientWithoutIndex(t)

	// Create index on vector field
	indexParams := map[string]interface{}{
		"indexType":  "IVF_FLAT",
		"metricType": "L2",
		"nlist":      128,
	}
	indexResult := client.CreateIndex("vector", indexParams)
	indexMap, ok := indexResult.(map[string]interface{})
	if !ok || !indexMap["success"].(bool) {
		cleanup()
		t.Fatalf("Failed to create index: %v", indexResult)
	}

	// Load collection
	loadResult := client.LoadCollection()
	loadMap, ok := loadResult.(map[string]interface{})
	if !ok || !loadMap["success"].(bool) {
		cleanup()
		t.Fatalf("Failed to load collection: %v", loadResult)
	}

	return client, collectionName, cleanup
}

func TestInsert_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, collectionName, cleanup := setupTestClient(t)
	defer cleanup()

	t.Run("insert_basic_data", func(t *testing.T) {
		// Generate test vectors
		vectors := make([][]float32, 3)
		for i := 0; i < 3; i++ {
			vectors[i] = make([]float32, 128)
			for j := 0; j < 128; j++ {
				vectors[i][j] = float32(i*128 + j)
			}
		}

		data := map[string]interface{}{
			"id":     []int64{1, 2, 3},
			"title":  []string{"Product A", "Product B", "Product C"},
			"vector": vectors,
		}

		result := client.Insert(data)

		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok, "result should be a map")

		assert.Equal(t, true, resultMap["success"], "insert should succeed")
		assert.NotNil(t, resultMap["response_time_ms"], "should have response time")

		if resultData, ok := resultMap["result"].(map[string]interface{}); ok {
			assert.Equal(t, float64(3), resultData["insert_count"], "should insert 3 records")
		}
	})

	t.Run("insert_with_explicit_collection", func(t *testing.T) {
		vectors := make([][]float32, 2)
		for i := 0; i < 2; i++ {
			vectors[i] = make([]float32, 128)
			for j := 0; j < 128; j++ {
				vectors[i][j] = float32(i*128 + j + 100)
			}
		}

		data := map[string]interface{}{
			"id":     []int64{10, 11},
			"title":  []string{"Item X", "Item Y"},
			"vector": vectors,
		}

		result := client.Insert(data, collectionName)

		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
	})
}

func TestUpsert_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestClient(t)
	defer cleanup()

	t.Run("upsert_new_records", func(t *testing.T) {
		vectors := make([][]float32, 2)
		for i := 0; i < 2; i++ {
			vectors[i] = make([]float32, 128)
			for j := 0; j < 128; j++ {
				vectors[i][j] = float32(i*128 + j)
			}
		}

		data := map[string]interface{}{
			"id":     []int64{100, 101},
			"title":  []string{"New A", "New B"},
			"vector": vectors,
		}

		result := client.Upsert(data)

		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"], "upsert should succeed")

		if resultData, ok := resultMap["result"].(map[string]interface{}); ok {
			assert.Equal(t, float64(2), resultData["upsert_count"])
		}
	})

	t.Run("upsert_existing_records", func(t *testing.T) {
		// First insert
		vectors := make([][]float32, 2)
		for i := 0; i < 2; i++ {
			vectors[i] = make([]float32, 128)
			for j := 0; j < 128; j++ {
				vectors[i][j] = float32(i*128 + j)
			}
		}

		data := map[string]interface{}{
			"id":     []int64{200, 201},
			"title":  []string{"Original A", "Original B"},
			"vector": vectors,
		}

		insertResult := client.Insert(data)
		insertMap := insertResult.(map[string]interface{})
		assert.Equal(t, true, insertMap["success"])

		// Wait a bit for consistency
		time.Sleep(100 * time.Millisecond)

		// Then upsert (update)
		data["title"] = []string{"Updated A", "Updated B"}

		upsertResult := client.Upsert(data)
		upsertMap := upsertResult.(map[string]interface{})

		assert.Equal(t, true, upsertMap["success"], "upsert should succeed")
	})
}

func TestDelete_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestClient(t)
	defer cleanup()

	t.Run("delete_by_id", func(t *testing.T) {
		// First insert some data
		vectors := make([][]float32, 5)
		for i := 0; i < 5; i++ {
			vectors[i] = make([]float32, 128)
			for j := 0; j < 128; j++ {
				vectors[i][j] = float32(i*128 + j)
			}
		}

		data := map[string]interface{}{
			"id":     []int64{1000, 1001, 1002, 1003, 1004},
			"title":  []string{"A", "B", "C", "D", "E"},
			"vector": vectors,
		}

		insertResult := client.Insert(data)
		insertMap := insertResult.(map[string]interface{})
		assert.Equal(t, true, insertMap["success"])

		// Wait for consistency
		time.Sleep(100 * time.Millisecond)

		// Delete some records
		deleteResult := client.Delete("id in [1001, 1002]")

		deleteMap, ok := deleteResult.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, deleteMap["success"], "delete should succeed")

		if resultData, ok := deleteMap["result"].(map[string]interface{}); ok {
			deleteCount := resultData["delete_count"]
			assert.Greater(t, deleteCount, float64(0), "should delete at least 1 record")
		}
	})

	t.Run("delete_by_filter", func(t *testing.T) {
		// Insert data with specific titles
		vectors := make([][]float32, 3)
		for i := 0; i < 3; i++ {
			vectors[i] = make([]float32, 128)
			for j := 0; j < 128; j++ {
				vectors[i][j] = float32(i*128 + j)
			}
		}

		data := map[string]interface{}{
			"id":     []int64{2000, 2001, 2002},
			"title":  []string{"Delete Me", "Keep Me", "Delete Me Too"},
			"vector": vectors,
		}

		insertResult := client.Insert(data)
		insertMap := insertResult.(map[string]interface{})
		assert.Equal(t, true, insertMap["success"])

		time.Sleep(100 * time.Millisecond)

		// Delete by id range
		deleteResult := client.Delete("id >= 2000 and id <= 2001")

		deleteMap, ok := deleteResult.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, deleteMap["success"])
	})

	t.Run("delete_missing_collection", func(t *testing.T) {
		// Create client without default collection
		milvusHost := os.Getenv("MILVUS_HOST")
		if milvusHost == "" {
			milvusHost = "localhost:19530"
		}

		milvusModule := &Milvus{
			vu: &mockVU{ctx: context.Background()},
		}

		tempClient, err := milvusModule.Client(milvusHost)
		require.NoError(t, err)
		defer tempClient.Close()

		result := tempClient.Delete("id > 0")

		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, false, resultMap["success"])
		assert.Equal(t, ErrCollectionNameRequired.Error(), resultMap["error"])
	})
}

func TestDataOperations_ResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, _, cleanup := setupTestClient(t)
	defer cleanup()

	t.Run("all_operations_have_response_time", func(t *testing.T) {
		vectors := make([][]float32, 1)
		vectors[0] = make([]float32, 128)
		for j := 0; j < 128; j++ {
			vectors[0][j] = float32(j)
		}

		data := map[string]interface{}{
			"id":     []int64{9999},
			"title":  []string{"Test"},
			"vector": vectors,
		}

		// Test Insert
		insertResult := client.Insert(data)
		insertMap := insertResult.(map[string]interface{})
		assert.NotNil(t, insertMap["response_time_ms"])
		assert.Greater(t, insertMap["response_time_ms"].(float64), 0.0)

		// Test Upsert
		upsertResult := client.Upsert(data)
		upsertMap := upsertResult.(map[string]interface{})
		assert.NotNil(t, upsertMap["response_time_ms"])
		assert.Greater(t, upsertMap["response_time_ms"].(float64), 0.0)

		// Test Delete
		deleteResult := client.Delete("id == 9999")
		deleteMap := deleteResult.(map[string]interface{})
		assert.NotNil(t, deleteMap["response_time_ms"])
		assert.Greater(t, deleteMap["response_time_ms"].(float64), 0.0)
	})
}

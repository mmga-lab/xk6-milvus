//go:build integration
// +build integration

package milvus

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCollection_Integration(t *testing.T) {
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

	collectionName := fmt.Sprintf("test_create_col_%d", time.Now().UnixNano())

	t.Run("create_collection_with_schema", func(t *testing.T) {
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
					Name:      "title",
					DataType:  "VarChar",
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
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"], "collection creation should succeed")
		assert.Greater(t, resultMap["response_time_ms"].(float64), 0.0)

		// Cleanup
		defer client.DropCollection(collectionName)
	})

	t.Run("create_collection_from_json", func(t *testing.T) {
		jsonCollectionName := fmt.Sprintf("test_json_col_%d", time.Now().UnixNano())
		schemaJSON := fmt.Sprintf(`{
			"name": "%s",
			"fields": [
				{
					"name": "id",
					"dataType": "Int64",
					"isPrimaryKey": true,
					"isAutoID": true
				},
				{
					"name": "text",
					"dataType": "VarChar",
					"maxLength": 512
				},
				{
					"name": "embedding",
					"dataType": "FloatVector",
					"dimension": 256
				}
			]
		}`, jsonCollectionName)

		result := client.CreateCollectionFromJSON(schemaJSON)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])

		// Cleanup
		defer client.DropCollection(jsonCollectionName)
	})

	t.Run("create_collection_with_invalid_json", func(t *testing.T) {
		invalidJSON := `{"name": "test", invalid json}`

		result := client.CreateCollectionFromJSON(invalidJSON)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, false, resultMap["success"])
		assert.Contains(t, resultMap["error"], "failed to parse schema JSON")
	})
}

func TestHasCollection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	milvusHost := os.Getenv("MILVUS_HOST")
	if milvusHost == "" {
		milvusHost = "localhost:19530"
	}

	collectionName := fmt.Sprintf("test_has_col_%d", time.Now().UnixNano())

	milvusModule := &Milvus{
		vu: &mockVU{ctx: context.Background()},
	}

	client, err := milvusModule.ClientWithCollection(milvusHost, collectionName)
	require.NoError(t, err)
	defer client.Close()

	// Create collection first
	schema := Schema{
		Name: collectionName,
		Fields: []Field{
			{Name: "id", DataType: "Int64", IsPrimaryKey: true, IsAutoID: true},
			{Name: "vector", DataType: "FloatVector", Dimension: 128},
		},
	}

	createResult := client.CreateCollection(schema)
	createMap := createResult.(map[string]interface{})
	require.Equal(t, true, createMap["success"])
	defer client.DropCollection()

	t.Run("has_existing_collection", func(t *testing.T) {
		result := client.HasCollection()
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
		assert.Equal(t, true, resultMap["result"])
	})

	t.Run("has_non_existing_collection", func(t *testing.T) {
		nonExistentName := "collection_that_does_not_exist_12345"
		result := client.HasCollection(nonExistentName)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
		assert.Equal(t, false, resultMap["result"])
	})
}

func TestLoadReleaseCollection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, collectionName, cleanup := setupTestClient(t)
	defer cleanup()

	t.Run("load_collection", func(t *testing.T) {
		result := client.LoadCollection()
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"], "load collection should succeed")
		assert.Greater(t, resultMap["response_time_ms"].(float64), 0.0)
	})

	t.Run("release_collection", func(t *testing.T) {
		result := client.ReleaseCollection()
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"], "release collection should succeed")
		assert.Greater(t, resultMap["response_time_ms"].(float64), 0.0)
	})

	t.Run("load_explicit_collection", func(t *testing.T) {
		result := client.LoadCollection(collectionName)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
	})
}

func TestDropCollection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	milvusHost := os.Getenv("MILVUS_HOST")
	if milvusHost == "" {
		milvusHost = "localhost:19530"
	}

	collectionName := fmt.Sprintf("test_drop_col_%d", time.Now().UnixNano())

	milvusModule := &Milvus{
		vu: &mockVU{ctx: context.Background()},
	}

	client, err := milvusModule.ClientWithCollection(milvusHost, collectionName)
	require.NoError(t, err)
	defer client.Close()

	// Create collection
	schema := Schema{
		Name: collectionName,
		Fields: []Field{
			{Name: "id", DataType: "Int64", IsPrimaryKey: true, IsAutoID: true},
			{Name: "vector", DataType: "FloatVector", Dimension: 64},
		},
	}

	createResult := client.CreateCollection(schema)
	createMap := createResult.(map[string]interface{})
	require.Equal(t, true, createMap["success"])

	t.Run("drop_collection", func(t *testing.T) {
		result := client.DropCollection()
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"], "drop collection should succeed")
		assert.Greater(t, resultMap["response_time_ms"].(float64), 0.0)
	})

	t.Run("verify_collection_dropped", func(t *testing.T) {
		// Wait a bit for the operation to propagate
		time.Sleep(100 * time.Millisecond)

		result := client.HasCollection()
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"])
		assert.Equal(t, false, resultMap["result"], "collection should not exist after drop")
	})
}

func TestCollectionOperations_MissingName(t *testing.T) {
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

	t.Run("has_collection_missing_name", func(t *testing.T) {
		result := client.HasCollection()
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, false, resultMap["success"])
		assert.Contains(t, resultMap["error"], "collection name required")
	})

	t.Run("load_collection_missing_name", func(t *testing.T) {
		result := client.LoadCollection()
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, false, resultMap["success"])
		assert.Contains(t, resultMap["error"], "collection name required")
	})

	t.Run("release_collection_missing_name", func(t *testing.T) {
		result := client.ReleaseCollection()
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, false, resultMap["success"])
		assert.Contains(t, resultMap["error"], "collection name required")
	})
}

func TestCreateCollectionWithComplexSchema_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	milvusHost := os.Getenv("MILVUS_HOST")
	if milvusHost == "" {
		milvusHost = "localhost:19530"
	}

	collectionName := fmt.Sprintf("test_complex_col_%d", time.Now().UnixNano())

	milvusModule := &Milvus{
		vu: &mockVU{ctx: context.Background()},
	}

	client, err := milvusModule.Client(milvusHost)
	require.NoError(t, err)
	defer client.Close()

	t.Run("create_collection_with_multiple_field_types", func(t *testing.T) {
		schema := Schema{
			Name:        collectionName,
			Description: "A complex test collection",
			Fields: []Field{
				{Name: "id", DataType: "Int64", IsPrimaryKey: true, IsAutoID: false},
				{Name: "int32_field", DataType: "Int32"},
				{Name: "float_field", DataType: "Float"},
				{Name: "double_field", DataType: "Double"},
				{Name: "bool_field", DataType: "Bool"},
				{Name: "varchar_field", DataType: "VarChar", MaxLength: 100},
				{Name: "vector_field", DataType: "FloatVector", Dimension: 128},
			},
			NumShards: 4,
		}

		result := client.CreateCollection(schema)
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, true, resultMap["success"], "complex collection creation should succeed")

		// Cleanup
		defer client.DropCollection(collectionName)
	})
}

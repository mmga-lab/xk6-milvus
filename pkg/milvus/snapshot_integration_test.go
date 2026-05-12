//go:build integration

package milvus

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestMilvusHost() string {
	host := os.Getenv("MILVUS_HOST")
	if host == "" {
		host = "localhost:19530"
	}
	return host
}

func createTestClient(t *testing.T) *milvusclient.Client {
	ctx := context.Background()
	client, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: getTestMilvusHost(),
	})
	require.NoError(t, err)
	return client
}

// skipIfSnapshotNotSupported checks if the Milvus server supports snapshot operations
// and skips the test if not supported (older Milvus versions don't have this feature)
func skipIfSnapshotNotSupported(t *testing.T, client *Client) {
	result := client.ListSnapshots(nil).(map[string]interface{})
	if !result["success"].(bool) {
		errMsg := result["error"].(string)
		if strings.Contains(errMsg, "Unimplemented") || strings.Contains(errMsg, "unknown method") {
			t.Skip("Skipping test: Milvus server does not support snapshot operations")
		}
	}
}

func TestSnapshotOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	sdkClient := createTestClient(t)
	defer sdkClient.Close(ctx)

	// Create a wrapper client for testing
	client := &Client{
		client:            sdkClient,
		ctx:               ctx,
		defaultCollection: "",
		config:            DefaultClientConfig(),
	}

	// Check if snapshot operations are supported
	skipIfSnapshotNotSupported(t, client)

	collectionName := fmt.Sprintf("test_snapshot_%d", time.Now().UnixNano())
	snapshotName := fmt.Sprintf("snap_%d", time.Now().UnixNano())

	// Cleanup at the end
	defer func() {
		client.DropSnapshot(snapshotName, map[string]interface{}{
			"collectionName": collectionName,
		})
		client.dropCollectionInternal(collectionName)
	}()

	// Create a test collection
	t.Run("Setup_CreateCollection", func(t *testing.T) {
		schema := map[string]interface{}{
			"name": collectionName,
			"fields": []interface{}{
				map[string]interface{}{
					"name":         "id",
					"dataType":     "Int64",
					"isPrimaryKey": true,
				},
				map[string]interface{}{
					"name":      "vector",
					"dataType":  "FloatVector",
					"dimension": int64(128),
				},
			},
		}

		result := client.CreateCollection(schema).(map[string]interface{})
		assert.True(t, result["success"].(bool), "Failed to create collection: %v", result["error"])

		// Insert some test data
		vectors := make([][]float32, 10)
		ids := make([]int64, 10)
		for i := 0; i < 10; i++ {
			ids[i] = int64(i)
			vectors[i] = make([]float32, 128)
			for j := 0; j < 128; j++ {
				vectors[i][j] = float32(i+j) / 100.0
			}
		}

		insertResult := client.Insert(map[string]interface{}{
			"id":     ids,
			"vector": vectors,
		}, collectionName).(map[string]interface{})
		assert.True(t, insertResult["success"].(bool), "Failed to insert data: %v", insertResult["error"])
	})

	// Test CreateSnapshot
	t.Run("CreateSnapshot", func(t *testing.T) {
		result := client.CreateSnapshot(snapshotName, collectionName, map[string]interface{}{
			"description": "Test snapshot for integration tests",
		}).(map[string]interface{})

		assert.True(t, result["success"].(bool), "Failed to create snapshot: %v", result["error"])
		assert.Greater(t, result["response_time_ms"].(float64), float64(0))

		if result["result"] != nil {
			resultMap := result["result"].(map[string]interface{})
			assert.Equal(t, snapshotName, resultMap["name"])
		}
	})

	// Test ListSnapshots
	t.Run("ListSnapshots", func(t *testing.T) {
		result := client.ListSnapshots(map[string]interface{}{
			"collectionName": collectionName,
		}).(map[string]interface{})

		assert.True(t, result["success"].(bool), "Failed to list snapshots: %v", result["error"])
		assert.Greater(t, result["response_time_ms"].(float64), float64(0))

		// Result can be []string or []interface{} depending on conversion
		var found bool
		switch snapshots := result["result"].(type) {
		case []string:
			for _, s := range snapshots {
				if s == snapshotName {
					found = true
					break
				}
			}
		case []interface{}:
			for _, s := range snapshots {
				if str, ok := s.(string); ok && str == snapshotName {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "Snapshot %s not found in list", snapshotName)
	})

	// Test DescribeSnapshot
	t.Run("DescribeSnapshot", func(t *testing.T) {
		result := client.DescribeSnapshot(snapshotName, map[string]interface{}{
			"collectionName": collectionName,
		}).(map[string]interface{})

		assert.True(t, result["success"].(bool), "Failed to describe snapshot: %v", result["error"])
		assert.Greater(t, result["response_time_ms"].(float64), float64(0))

		if result["result"] != nil {
			resultMap := result["result"].(map[string]interface{})
			assert.Equal(t, snapshotName, resultMap["name"])
			assert.Equal(t, collectionName, resultMap["collectionName"])
			assert.Equal(t, "Test snapshot for integration tests", resultMap["description"])
		}
	})

	// Test RestoreSnapshot
	restoredCollectionName := collectionName + "_restored"
	var jobID int64

	t.Run("RestoreSnapshot", func(t *testing.T) {
		// Cleanup restored collection at the end
		defer func() {
			client.dropCollectionInternal(restoredCollectionName)
		}()

		result := client.RestoreSnapshot(snapshotName, restoredCollectionName, map[string]interface{}{
			"collectionName": collectionName,
		}).(map[string]interface{})

		assert.True(t, result["success"].(bool), "Failed to restore snapshot: %v", result["error"])
		assert.Greater(t, result["response_time_ms"].(float64), float64(0))

		if result["result"] != nil {
			resultMap := result["result"].(map[string]interface{})
			jobID = int64(resultMap["jobId"].(float64))
			assert.Greater(t, jobID, int64(0))
		}
	})

	// Test GetRestoreSnapshotState
	t.Run("GetRestoreSnapshotState", func(t *testing.T) {
		if jobID == 0 {
			t.Skip("No job ID from restore operation")
		}

		// Wait a bit for the job to progress
		time.Sleep(time.Second)

		result := client.GetRestoreSnapshotState(jobID).(map[string]interface{})

		// Job may complete quickly and be removed, so we accept both success and "not found" errors
		assert.Greater(t, result["response_time_ms"].(float64), float64(0))

		if result["success"].(bool) && result["result"] != nil {
			resultMap := result["result"].(map[string]interface{})
			assert.Equal(t, jobID, int64(resultMap["jobId"].(float64)))
			assert.Equal(t, snapshotName, resultMap["snapshotName"])
			assert.Equal(t, restoredCollectionName, resultMap["collectionName"])
			// State should be one of the valid states
			state := resultMap["state"].(string)
			validStates := []string{
				"RestoreSnapshotNone",
				"RestoreSnapshotPending",
				"RestoreSnapshotExecuting",
				"RestoreSnapshotCompleted",
				"RestoreSnapshotFailed",
			}
			assert.Contains(t, validStates, state)
		} else {
			// Job may have completed and been cleaned up, which is acceptable
			t.Logf("GetRestoreSnapshotState returned: success=%v, error=%v", result["success"], result["error"])
		}
	})

	// Test ListRestoreSnapshotJobs
	t.Run("ListRestoreSnapshotJobs", func(t *testing.T) {
		result := client.ListRestoreSnapshotJobs().(map[string]interface{})

		assert.True(t, result["success"].(bool), "Failed to list restore jobs: %v", result["error"])
		assert.Greater(t, result["response_time_ms"].(float64), float64(0))

		// Result should be a list (may or may not contain our job)
		// Handle both []interface{} and []map[string]interface{} types
		if result["result"] != nil {
			switch jobs := result["result"].(type) {
			case []map[string]interface{}:
				assert.NotNil(t, jobs)
			case []interface{}:
				assert.NotNil(t, jobs)
			default:
				t.Logf("Unexpected result type: %T", result["result"])
			}
		}
	})

	// Test DropSnapshot
	t.Run("DropSnapshot", func(t *testing.T) {
		result := client.DropSnapshot(snapshotName, map[string]interface{}{
			"collectionName": collectionName,
		}).(map[string]interface{})

		assert.True(t, result["success"].(bool), "Failed to drop snapshot: %v", result["error"])
		assert.Greater(t, result["response_time_ms"].(float64), float64(0))
	})

	// Verify snapshot is dropped
	t.Run("VerifySnapshotDropped", func(t *testing.T) {
		result := client.DescribeSnapshot(snapshotName, map[string]interface{}{
			"collectionName": collectionName,
		}).(map[string]interface{})
		// Should fail because snapshot doesn't exist
		assert.False(t, result["success"].(bool))
	})
}

func TestSnapshotErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	sdkClient := createTestClient(t)
	defer sdkClient.Close(ctx)

	client := &Client{
		client:            sdkClient,
		ctx:               ctx,
		defaultCollection: "",
		config:            DefaultClientConfig(),
	}

	// Check if snapshot operations are supported
	skipIfSnapshotNotSupported(t, client)

	t.Run("CreateSnapshot_NoCollection", func(t *testing.T) {
		result := client.CreateSnapshot("test_snap", nil).(map[string]interface{})
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "collection name required")
	})

	t.Run("DescribeSnapshot_NotFound", func(t *testing.T) {
		result := client.DescribeSnapshot("nonexistent_snapshot_12345").(map[string]interface{})
		assert.False(t, result["success"].(bool))
	})

	t.Run("RestoreSnapshot_NoTargetCollection", func(t *testing.T) {
		result := client.RestoreSnapshot("some_snapshot", "").(map[string]interface{})
		assert.False(t, result["success"].(bool))
		assert.Contains(t, result["error"].(string), "target collection name required")
	})

	t.Run("GetRestoreSnapshotState_InvalidJobID", func(t *testing.T) {
		result := client.GetRestoreSnapshotState(-1).(map[string]interface{})
		// Should either fail or return empty state
		// This depends on Milvus server behavior
		assert.NotNil(t, result["response_time_ms"])
	})
}

func TestSnapshotWithBoundCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	sdkClient := createTestClient(t)
	defer sdkClient.Close(ctx)

	// First check if snapshot is supported using a temporary client
	tempClient := &Client{
		client:            sdkClient,
		ctx:               ctx,
		defaultCollection: "",
		config:            DefaultClientConfig(),
	}
	skipIfSnapshotNotSupported(t, tempClient)

	collectionName := fmt.Sprintf("test_bound_snapshot_%d", time.Now().UnixNano())
	snapshotName := fmt.Sprintf("bound_snap_%d", time.Now().UnixNano())

	// Create client with bound collection
	client := &Client{
		client:            sdkClient,
		ctx:               ctx,
		defaultCollection: collectionName,
		config:            DefaultClientConfig(),
	}

	// Cleanup
	defer func() {
		client.DropSnapshot(snapshotName)
		client.dropCollectionInternal(collectionName)
	}()

	// Create collection
	schema := map[string]interface{}{
		"name": collectionName,
		"fields": []interface{}{
			map[string]interface{}{
				"name":         "id",
				"dataType":     "Int64",
				"isPrimaryKey": true,
			},
			map[string]interface{}{
				"name":      "vector",
				"dataType":  "FloatVector",
				"dimension": int64(64),
			},
		},
	}
	result := client.CreateCollection(schema).(map[string]interface{})
	require.True(t, result["success"].(bool), "Failed to create collection")

	// Test CreateSnapshot with nil collectionName (should use bound collection)
	t.Run("CreateSnapshot_BoundCollection", func(t *testing.T) {
		result := client.CreateSnapshot(snapshotName, nil).(map[string]interface{})
		assert.True(t, result["success"].(bool), "Failed to create snapshot: %v", result["error"])
	})

	// Verify the snapshot was created for the bound collection
	t.Run("VerifySnapshot_BoundCollection", func(t *testing.T) {
		result := client.DescribeSnapshot(snapshotName).(map[string]interface{})
		assert.True(t, result["success"].(bool))

		if result["result"] != nil {
			resultMap := result["result"].(map[string]interface{})
			assert.Equal(t, collectionName, resultMap["collectionName"])
		}
	})
}

// dropCollectionInternal is a helper to drop collection using SDK directly
func (c *Client) dropCollectionInternal(name string) {
	opt := milvusclient.NewDropCollectionOption(name)
	_ = c.client.DropCollection(c.ctx, opt)
}

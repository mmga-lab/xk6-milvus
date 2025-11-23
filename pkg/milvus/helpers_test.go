package milvus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToMap(t *testing.T) {
	t.Run("successful operation result", func(t *testing.T) {
		result := &OperationResult{
			Success:      true,
			ResponseTime: 123.45,
			Result:       map[string]any{"count": 10, "ids": []int64{1, 2, 3}},
			Error:        "",
			Empty:        false,
			Recall:       0.95,
		}

		m := toMap(result)

		require.NotNil(t, m)
		assert.Equal(t, true, m["success"])
		assert.Equal(t, float64(123.45), m["response_time_ms"])
		assert.NotNil(t, m["result"])
		// Error field is omitted when empty due to omitempty tag
		_, hasError := m["error"]
		assert.False(t, hasError, "error field should be omitted when empty")
		assert.Equal(t, false, m["empty"])
		assert.InDelta(t, 0.95, m["recall"], 0.001)
	})

	t.Run("failed operation result", func(t *testing.T) {
		result := &OperationResult{
			Success:      false,
			ResponseTime: 50.0,
			Result:       nil,
			Error:        "connection timeout",
			Empty:        true,
			Recall:       0.0,
		}

		m := toMap(result)

		require.NotNil(t, m)
		assert.Equal(t, false, m["success"])
		assert.Equal(t, float64(50.0), m["response_time_ms"])
		assert.Nil(t, m["result"])
		assert.Equal(t, "connection timeout", m["error"])
		assert.Equal(t, true, m["empty"])
		assert.Equal(t, float64(0), m["recall"])
	})

	t.Run("empty result", func(t *testing.T) {
		result := &OperationResult{
			Success:      true,
			ResponseTime: 10.5,
			Result:       nil,
			Error:        "",
			Empty:        true,
			Recall:       0.0,
		}

		m := toMap(result)

		require.NotNil(t, m)
		assert.Equal(t, true, m["success"])
		assert.Equal(t, float64(10.5), m["response_time_ms"])
		// Result and Error fields are omitted when empty due to omitempty tag
		_, hasResult := m["result"]
		assert.False(t, hasResult, "result field should be omitted when nil")
		_, hasError := m["error"]
		assert.False(t, hasError, "error field should be omitted when empty")
		assert.Equal(t, true, m["empty"])
	})

	t.Run("result with complex nested data", func(t *testing.T) {
		result := &OperationResult{
			Success:      true,
			ResponseTime: 200.0,
			Result: map[string]any{
				"data": []map[string]any{
					{"id": int64(1), "score": 0.95},
					{"id": int64(2), "score": 0.85},
				},
				"metadata": map[string]string{
					"version": "2.0",
				},
			},
			Error:  "",
			Empty:  false,
			Recall: 0.98,
		}

		m := toMap(result)

		require.NotNil(t, m)
		assert.Equal(t, true, m["success"])
		assert.Equal(t, float64(200.0), m["response_time_ms"])

		// Verify nested data structure is preserved
		resultMap, ok := m["result"].(map[string]any)
		require.True(t, ok, "result should be a map")
		assert.Contains(t, resultMap, "data")
		assert.Contains(t, resultMap, "metadata")
	})

	t.Run("result with zero values", func(t *testing.T) {
		result := &OperationResult{
			Success:      false,
			ResponseTime: 0,
			Result:       nil,
			Error:        "",
			Empty:        false,
			Recall:       0,
		}

		m := toMap(result)

		require.NotNil(t, m)
		assert.Equal(t, false, m["success"])
		assert.Equal(t, float64(0), m["response_time_ms"])
		assert.Equal(t, float64(0), m["recall"])
	})

	t.Run("result with high recall", func(t *testing.T) {
		result := &OperationResult{
			Success:      true,
			ResponseTime: 150.0,
			Result: map[string]any{
				"hits": 100,
			},
			Error:  "",
			Empty:  false,
			Recall: 1.0,
		}

		m := toMap(result)

		require.NotNil(t, m)
		assert.Equal(t, float64(1.0), m["recall"])
	})

	t.Run("result with various result types", func(t *testing.T) {
		tests := []struct {
			name   string
			result any
		}{
			{
				name:   "string result",
				result: "operation completed",
			},
			{
				name:   "int result",
				result: 42,
			},
			{
				name:   "slice result",
				result: []string{"a", "b", "c"},
			},
			{
				name:   "bool result",
				result: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				opResult := &OperationResult{
					Success:      true,
					ResponseTime: 100.0,
					Result:       tt.result,
					Error:        "",
					Empty:        false,
					Recall:       0.0,
				}

				m := toMap(opResult)
				require.NotNil(t, m)
				assert.NotNil(t, m["result"])
			})
		}
	})

	t.Run("JSON field names are camelCase", func(t *testing.T) {
		result := &OperationResult{
			Success:      true,
			ResponseTime: 100.0,
			Result:       map[string]any{"test": "data"},
			Error:        "some error", // Non-empty to ensure it appears in map
			Empty:        false,
			Recall:       0.5,
		}

		m := toMap(result)

		// Verify JSON tag names (camelCase) are used, not Go field names
		assert.Contains(t, m, "success")
		assert.Contains(t, m, "response_time_ms")
		assert.Contains(t, m, "result")
		assert.Contains(t, m, "error") // Now present because it's not empty
		assert.Contains(t, m, "empty")
		assert.Contains(t, m, "recall")

		// Verify Go field names are NOT present
		assert.NotContains(t, m, "Success")
		assert.NotContains(t, m, "ResponseTime")
		assert.NotContains(t, m, "Result")
		assert.NotContains(t, m, "Error")
		assert.NotContains(t, m, "Empty")
		assert.NotContains(t, m, "Recall")
	})
}

func TestGetCollectionNameEdgeCases(t *testing.T) {
	// Additional edge cases for getCollectionName beyond the existing tests
	t.Run("multiple empty strings in params", func(t *testing.T) {
		client := &Client{
			defaultCollection: "default_col",
		}

		// Multiple empty strings should still return default
		// Note: function only checks first parameter
		got := client.getCollectionName("", "", "")
		assert.Equal(t, "default_col", got)
	})

	t.Run("first param takes precedence", func(t *testing.T) {
		client := &Client{
			defaultCollection: "default_col",
		}

		// Function only uses first parameter, others are ignored
		got := client.getCollectionName("first_col", "second_col")
		assert.Equal(t, "first_col", got)
	})

	t.Run("whitespace-only collection name", func(t *testing.T) {
		client := &Client{
			defaultCollection: "default_col",
		}

		// Whitespace-only string is not empty, so it should be returned
		got := client.getCollectionName("   ")
		assert.Equal(t, "   ", got)
	})
}

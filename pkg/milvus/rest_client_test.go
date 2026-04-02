package milvus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer creates a test HTTP server that returns the given response
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *RestClient) {
	t.Helper()
	server := httptest.NewServer(handler)
	rc := &RestClient{
		baseURL:           server.URL,
		defaultCollection: "test_collection",
		httpClient:        server.Client(),
	}
	return server, rc
}

// jsonHandler returns a handler that responds with a standard Milvus REST response
func jsonHandler(code int, data interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{"code": code}
		if data != nil {
			dataBytes, _ := json.Marshal(data)
			resp["data"] = json.RawMessage(dataBytes)
		}
		if code != 0 {
			resp["message"] = "test error"
		}
		json.NewEncoder(w).Encode(resp)
	}
}

func TestRestClientPost(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		server, rc := newTestServer(t, jsonHandler(0, map[string]interface{}{"ok": true}))
		defer server.Close()

		data, elapsed, err := rc.post("/collections/list", map[string]interface{}{})
		require.NoError(t, err)
		assert.NotNil(t, data)
		assert.GreaterOrEqual(t, elapsed, float64(0))
	})

	t.Run("error response", func(t *testing.T) {
		server, rc := newTestServer(t, jsonHandler(1800, nil))
		defer server.Close()

		_, _, err := rc.post("/collections/list", map[string]interface{}{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "test error")
	})

	t.Run("connection error", func(t *testing.T) {
		rc := &RestClient{
			baseURL:    "http://127.0.0.1:1", // unlikely to be listening
			httpClient: &http.Client{},
		}

		_, elapsed, err := rc.post("/collections/list", map[string]interface{}{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "request failed")
		assert.GreaterOrEqual(t, elapsed, float64(0))
	})
}

func TestRestClientGetCollectionName(t *testing.T) {
	rc := &RestClient{defaultCollection: "default_col"}

	assert.Equal(t, "default_col", rc.getCollectionName())
	assert.Equal(t, "explicit", rc.getCollectionName("explicit"))
	assert.Equal(t, "default_col", rc.getCollectionName(""))
}

func TestRestClientBaseBody(t *testing.T) {
	t.Run("without dbName", func(t *testing.T) {
		rc := &RestClient{}
		body := rc.baseBody("my_collection")
		assert.Equal(t, "my_collection", body["collectionName"])
		_, hasDB := body["dbName"]
		assert.False(t, hasDB)
	})

	t.Run("with dbName", func(t *testing.T) {
		rc := &RestClient{dbName: "my_db"}
		body := rc.baseBody("my_collection")
		assert.Equal(t, "my_collection", body["collectionName"])
		assert.Equal(t, "my_db", body["dbName"])
	})
}

func TestColumnsToRows(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		data := map[string]interface{}{
			"name":  []interface{}{"Alice", "Bob"},
			"score": []interface{}{95.0, 87.5},
		}
		rows, err := columnsToRows(data)
		require.NoError(t, err)
		require.Len(t, rows, 2)
		assert.Equal(t, "Alice", rows[0]["name"])
		assert.Equal(t, 87.5, rows[1]["score"])
	})

	t.Run("empty data", func(t *testing.T) {
		data := map[string]interface{}{}
		_, err := columnsToRows(data)
		require.Error(t, err)
	})

	t.Run("length mismatch", func(t *testing.T) {
		data := map[string]interface{}{
			"a": []interface{}{1, 2, 3},
			"b": []interface{}{1, 2},
		}
		_, err := columnsToRows(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "length")
	})

	t.Run("with vectors", func(t *testing.T) {
		data := map[string]interface{}{
			"id":     []interface{}{1, 2},
			"vector": []interface{}{[]float64{0.1, 0.2}, []float64{0.3, 0.4}},
		}
		rows, err := columnsToRows(data)
		require.NoError(t, err)
		require.Len(t, rows, 2)
		assert.Equal(t, 1, rows[0]["id"])
	})
}

func TestRestCollectionOperations(t *testing.T) {
	t.Run("hasCollection returns exists format", func(t *testing.T) {
		server, rc := newTestServer(t, jsonHandler(0, map[string]interface{}{"has": true}))
		defer server.Close()

		result := rc.HasCollection("test_collection")
		m := result.(map[string]interface{})
		assert.Equal(t, true, m["success"])
		r := m["result"].(map[string]interface{})
		assert.Equal(t, true, r["exists"])
	})

	t.Run("createCollection success", func(t *testing.T) {
		server, rc := newTestServer(t, jsonHandler(0, nil))
		defer server.Close()

		result := rc.CreateCollection(Schema{
			Name: "new_collection",
			Fields: []Field{
				{Name: "id", DataType: "Int64", IsPrimaryKey: true},
				{Name: "vec", DataType: "FloatVector", Dimension: 128},
			},
		})
		m := result.(map[string]interface{})
		assert.Equal(t, true, m["success"])
	})

	t.Run("collection name required", func(t *testing.T) {
		rc := &RestClient{httpClient: &http.Client{}}
		result := rc.DropCollection()
		m := result.(map[string]interface{})
		assert.Equal(t, false, m["success"])
		assert.Contains(t, m["error"], "collection name required")
	})
}

func TestRestSearchOperations(t *testing.T) {
	t.Run("search returns results with empty flag", func(t *testing.T) {
		searchData := []map[string]interface{}{
			{"id": 1, "distance": 0.5, "title": "A"},
			{"id": 2, "distance": 0.8, "title": "B"},
		}
		server, rc := newTestServer(t, jsonHandler(0, searchData))
		defer server.Close()

		result := rc.Search(
			[][]float32{{0.1, 0.2}}, 10,
			map[string]interface{}{"vectorField": "vec", "metricType": "L2"},
		)
		m := result.(map[string]interface{})
		assert.Equal(t, true, m["success"])
		assert.Equal(t, false, m["empty"])
	})

	t.Run("search empty results", func(t *testing.T) {
		server, rc := newTestServer(t, jsonHandler(0, []interface{}{}))
		defer server.Close()

		result := rc.Search([][]float32{{0.1}}, 10, map[string]interface{}{"vectorField": "vec"})
		m := result.(map[string]interface{})
		assert.Equal(t, true, m["success"])
		assert.Equal(t, true, m["empty"])
	})

	t.Run("query returns results", func(t *testing.T) {
		queryData := []map[string]interface{}{
			{"id": 1, "title": "A"},
		}
		server, rc := newTestServer(t, jsonHandler(0, queryData))
		defer server.Close()

		result := rc.Query("id > 0", []interface{}{"id", "title"})
		m := result.(map[string]interface{})
		assert.Equal(t, true, m["success"])
		assert.Equal(t, false, m["empty"])
	})
}

func TestRestInsertOperations(t *testing.T) {
	t.Run("insert with column data", func(t *testing.T) {
		server, rc := newTestServer(t, jsonHandler(0, map[string]interface{}{
			"insertCount": float64(3),
			"insertIds":   []int64{1, 2, 3},
		}))
		defer server.Close()

		result := rc.Insert(map[string]interface{}{
			"name":   []interface{}{"A", "B", "C"},
			"vector": []interface{}{[]float64{0.1}, []float64{0.2}, []float64{0.3}},
		})
		m := result.(map[string]interface{})
		assert.Equal(t, true, m["success"])
		r := m["result"].(map[string]interface{})
		assert.Equal(t, float64(3), r["insert_count"])
	})

	t.Run("delete success", func(t *testing.T) {
		server, rc := newTestServer(t, jsonHandler(0, nil))
		defer server.Close()

		result := rc.Delete("id > 0")
		m := result.(map[string]interface{})
		assert.Equal(t, true, m["success"])
	})
}

func TestRestIndexOperations(t *testing.T) {
	t.Run("createIndex success", func(t *testing.T) {
		server, rc := newTestServer(t, jsonHandler(0, nil))
		defer server.Close()

		result := rc.CreateIndex("vec", map[string]interface{}{
			"indexType": "HNSW", "metricType": "L2",
		})
		m := result.(map[string]interface{})
		assert.Equal(t, true, m["success"])
		r := m["result"].(map[string]interface{})
		assert.Equal(t, "vec", r["field"])
		assert.Equal(t, "HNSW", r["index_type"])
	})
}

func TestRestClientClose(t *testing.T) {
	rc := &RestClient{}
	result := rc.Close()
	m := result.(map[string]interface{})
	assert.Equal(t, true, m["success"])
}

func TestConvertSchemaToRest(t *testing.T) {
	schema := Schema{
		Name:      "test",
		NumShards: 4,
		Fields: []Field{
			{Name: "id", DataType: "Int64", IsPrimaryKey: true, IsAutoID: true},
			{Name: "text", DataType: "VarChar", MaxLength: 200, EnableAnalyzer: true, EnableMatch: true},
			{Name: "vec", DataType: "FloatVector", Dimension: 128},
		},
		Functions: []Function{
			{Name: "bm25", FunctionType: "BM25", InputFieldNames: []string{"text"}, OutputFieldNames: []string{"sparse"}},
		},
	}

	body := convertSchemaToRest(schema)

	assert.Equal(t, "test", body["collectionName"])
	assert.Equal(t, int32(4), body["numShards"])

	restSchema := body["schema"].(map[string]interface{})
	assert.Equal(t, true, restSchema["autoId"])

	fields := restSchema["fields"].([]map[string]interface{})
	require.Len(t, fields, 3)

	// Check ID field
	assert.Equal(t, "id", fields[0]["fieldName"])
	assert.Equal(t, true, fields[0]["isPrimary"])
	assert.Equal(t, true, fields[0]["autoId"])

	// Check VarChar field
	assert.Equal(t, "text", fields[1]["fieldName"])
	assert.Equal(t, true, fields[1]["enableAnalyzer"])
	assert.Equal(t, true, fields[1]["enableMatch"])
	tp := fields[1]["elementTypeParams"].(map[string]interface{})
	assert.Equal(t, "200", tp["max_length"])

	// Check vector field
	assert.Equal(t, "vec", fields[2]["fieldName"])
	tp2 := fields[2]["elementTypeParams"].(map[string]interface{})
	assert.Equal(t, "128", tp2["dim"])

	// Check functions
	fns := restSchema["functions"].([]map[string]interface{})
	require.Len(t, fns, 1)
	assert.Equal(t, "bm25", fns[0]["name"])
	assert.Equal(t, "BM25", fns[0]["type"])
}

func TestErrorResultAndSuccessResult(t *testing.T) {
	t.Run("errorResult", func(t *testing.T) {
		result := errorResult(42.5, "something failed")
		m := result.(map[string]interface{})
		assert.Equal(t, false, m["success"])
		assert.Equal(t, 42.5, m["response_time_ms"])
		assert.Equal(t, "something failed", m["error"])
	})

	t.Run("successResult", func(t *testing.T) {
		result := successResult(10.0, map[string]interface{}{"count": 5})
		m := result.(map[string]interface{})
		assert.Equal(t, true, m["success"])
		assert.Equal(t, 10.0, m["response_time_ms"])
		r := m["result"].(map[string]interface{})
		assert.Equal(t, float64(5), r["count"])
	})
}

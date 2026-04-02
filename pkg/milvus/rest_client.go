package milvus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const restAPIPrefix = "/v2/vectordb"

// RestClient is a Milvus client using RESTful v2 API
type RestClient struct {
	baseURL           string
	token             string
	dbName            string
	defaultCollection string
	httpClient        *http.Client
}

// restResponse represents the standard REST API response
type restResponse struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data,omitempty"`
	Message string          `json:"message,omitempty"`
}

// RestClient creates a new Milvus REST client (not bound to any collection)
func (m *Milvus) RestClient(address string, token ...string) *RestClient {
	return m.createRestClient(address, "", token...)
}

// RestClientWithCollection creates a new REST client bound to a specific collection
func (m *Milvus) RestClientWithCollection(address, collectionName string, token ...string) *RestClient {
	return m.createRestClient(address, collectionName, token...)
}

func (m *Milvus) createRestClient(address, collectionName string, token ...string) *RestClient {
	baseURL := address
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	rc := &RestClient{
		baseURL:           baseURL,
		defaultCollection: collectionName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if len(token) > 0 && token[0] != "" {
		rc.token = token[0]
	}

	return rc
}

// GetRestClient returns a VU-level cached REST client for connection pool reuse.
// First call creates the client; subsequent calls return the cached instance.
// Reusing the http.Client preserves TCP keep-alive connections across iterations.
//
// Usage in k6:
//
//	import milvus from 'k6/x/milvus';
//	export default function() {
//	    const client = milvus.getRestClient(host, collection, token);
//	    client.search(...);
//	    // Do NOT call client.close() - client is reused across iterations
//	}
func (m *Milvus) GetRestClient(address, collectionName string, token ...string) *RestClient {
	key := address + ":" + collectionName

	if client, ok := m.restClients[key]; ok {
		return client
	}

	client := m.createRestClient(address, collectionName, token...)
	m.restClients[key] = client
	return client
}

// Close is a no-op for REST client (stateless HTTP)
func (rc *RestClient) Close() interface{} {
	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: 0,
	})
}

// post sends a POST request to the Milvus REST API and returns an OperationResult
func (rc *RestClient) post(path string, body interface{}) (json.RawMessage, float64, error) {
	url := rc.baseURL + restAPIPrefix + path

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if rc.token != "" {
		req.Header.Set("Authorization", "Bearer "+rc.token)
	}

	start := time.Now()
	resp, err := rc.httpClient.Do(req)
	elapsed := float64(time.Since(start).Milliseconds())

	if err != nil {
		return nil, elapsed, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, elapsed, fmt.Errorf("failed to read response: %v", err)
	}

	var restResp restResponse
	if err := json.Unmarshal(respBody, &restResp); err != nil {
		return nil, elapsed, fmt.Errorf("failed to parse response: %v", err)
	}

	if restResp.Code != 0 {
		msg := restResp.Message
		if msg == "" {
			msg = fmt.Sprintf("error code: %d", restResp.Code)
		}
		return nil, elapsed, fmt.Errorf("%s", msg)
	}

	return restResp.Data, elapsed, nil
}

// getCollectionName returns collection name from params or default
func (rc *RestClient) getCollectionName(collectionName ...string) string {
	if len(collectionName) > 0 && collectionName[0] != "" {
		return collectionName[0]
	}
	return rc.defaultCollection
}

// baseBody builds a request body with collectionName and optional dbName
func (rc *RestClient) baseBody(collectionName string) map[string]interface{} {
	body := map[string]interface{}{
		"collectionName": collectionName,
	}
	if rc.dbName != "" {
		body["dbName"] = rc.dbName
	}
	return body
}

// errorResult creates a failed OperationResult with an error message
func errorResult(elapsed float64, msg string) interface{} {
	return toMap(&OperationResult{
		Success:      false,
		ResponseTime: elapsed,
		Error:        msg,
	})
}

// successResult creates a successful OperationResult
func successResult(elapsed float64, result interface{}) interface{} {
	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: elapsed,
		Result:       result,
	})
}

package milvus

import (
	"fmt"
	"strings"
	"time"

	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// CreateIndex creates an index on a field
func (c *Client) CreateIndex(fieldName string, indexParams map[string]interface{}, collectionName ...string) interface{} {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		})
	}

	idx, indexType, indexName, err := buildIndex(indexParams)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        err.Error(),
		})
	}

	option := milvusclient.NewCreateIndexOption(coll, fieldName, idx)
	if indexName != "" {
		option = option.WithIndexName(indexName)
	}
	task, err := c.client.CreateIndex(c.context(), option)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to create index: %v", err),
		})
	}

	// Wait for index creation to complete
	err = task.Await(c.context())
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to wait for index creation: %v", err),
		})
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       map[string]interface{}{"field": fieldName, "index_type": indexType},
	})
}

func buildIndex(indexParams map[string]interface{}) (index.Index, string, string, error) {
	params := flattenIndexParams(indexParams)
	indexType := "FLAT"
	if iType, ok := stringOption(params, "indexType"); ok && iType != "" {
		indexType = iType
	} else if iType, ok := stringOption(params, "index_type"); ok && iType != "" {
		indexType = iType
	}

	metricType := metricTypeOption(params)
	normalizedIndexType := strings.ToUpper(indexType)

	var idx index.Index
	switch normalizedIndexType {
	case "FLAT":
		idx = index.NewFlatIndex(metricType)
	case "BIN_FLAT":
		idx = index.NewBinFlatIndex(metricType)
	case "IVF_FLAT":
		idx = index.NewIvfFlatIndex(metricType, intIndexParam(params, "nlist", 1024))
	case "BIN_IVF_FLAT":
		idx = index.NewBinIvfFlatIndex(metricType, intIndexParam(params, "nlist", 1024))
	case "IVF_SQ8":
		idx = index.NewIvfSQ8Index(metricType, intIndexParam(params, "nlist", 1024))
	case "IVF_PQ":
		idx = index.NewIvfPQIndex(
			metricType,
			intIndexParam(params, "nlist", 1024),
			intIndexParam(params, "m", 4),
			intIndexParam(params, "nbits", 8),
		)
	case "HNSW":
		idx = index.NewHNSWIndex(
			metricType,
			intIndexParam(params, "M", 16),
			intIndexParam(params, "efConstruction", 200),
		)
	case "AUTOINDEX", "AUTO_INDEX":
		idx = index.NewAutoIndex(metricType)
	case "SPARSE_INVERTED_INDEX":
		idx = index.NewSparseInvertedIndex(metricType, floatIndexParam(params, "dropRatio", 0))
	case "SPARSE_WAND":
		idx = index.NewSparseWANDIndex(metricType, floatIndexParam(params, "dropRatio", 0))
	case "INVERTED":
		idx = index.NewInvertedIndex()
	case "STL_SORT":
		idx = index.NewSortedIndex()
	case "BITMAP":
		idx = index.NewBitmapIndex()
	case "TRIE":
		idx = index.NewTrieIndex()
	default:
		return nil, indexType, "", fmt.Errorf("unsupported index type: %s", indexType)
	}

	indexName, _ := stringOption(params, "indexName")
	if indexName == "" {
		indexName, _ = stringOption(params, "index_name")
	}
	return idx, indexType, indexName, nil
}

func flattenIndexParams(indexParams map[string]interface{}) map[string]interface{} {
	params := make(map[string]interface{}, len(indexParams))
	for key, val := range indexParams {
		params[key] = val
	}
	if nested, ok := indexParams["params"].(map[string]interface{}); ok {
		for key, val := range nested {
			if _, exists := params[key]; !exists {
				params[key] = val
			}
		}
	}
	return params
}

func metricTypeOption(params map[string]interface{}) entity.MetricType {
	metricType := entity.L2
	metricName, ok := stringOption(params, "metricType")
	if !ok || metricName == "" {
		metricName, _ = stringOption(params, "metric_type")
	}

	switch strings.ToUpper(metricName) {
	case "L2":
		metricType = entity.L2
	case "IP":
		metricType = entity.IP
	case "COSINE":
		metricType = entity.COSINE
	case "BM25":
		metricType = entity.BM25
	case "MAX_SIM":
		metricType = entity.MaxSim
	case "MAX_SIM_COSINE":
		metricType = entity.MaxSimCosine
	case "MAX_SIM_L2":
		metricType = entity.MaxSimL2
	case "MAX_SIM_IP":
		metricType = entity.MaxSimIP
	case "MAX_SIM_HAMMING":
		metricType = entity.MaxSimHamming
	case "MAX_SIM_JACCARD":
		metricType = entity.MaxSimJaccard
	}
	return metricType
}

func intIndexParam(params map[string]interface{}, key string, fallback int) int {
	if value, ok := intOption(params, key); ok {
		return value
	}
	return fallback
}

func floatIndexParam(params map[string]interface{}, key string, fallback float64) float64 {
	value, ok := params[key]
	if !ok || value == nil {
		return fallback
	}
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	}
	return fallback
}

// DropIndex drops an index by field name
func (c *Client) DropIndex(fieldName string, collectionName ...string) interface{} {
	start := time.Now()

	coll := c.getCollectionName(collectionName...)
	if coll == "" {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        "collection name required",
		})
	}

	option := milvusclient.NewDropIndexOption(coll, fieldName)
	err := c.client.DropIndex(c.context(), option)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to drop index: %v", err),
		})
	}

	return toMap(&OperationResult{
		Success:      true,
		ResponseTime: float64(time.Since(start).Milliseconds()),
		Result:       map[string]interface{}{"field": fieldName},
	})
}

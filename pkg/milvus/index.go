package milvus

import (
	"fmt"
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

	var idx index.Index

	// Default to flat index if not specified
	indexType := "FLAT"
	metricType := entity.L2

	if iType, ok := indexParams["indexType"].(string); ok {
		indexType = iType
	}

	if mType, ok := indexParams["metricType"].(string); ok {
		switch mType {
		case "L2":
			metricType = entity.L2
		case "IP":
			metricType = entity.IP
		case "COSINE":
			metricType = entity.COSINE
		case "BM25":
			metricType = entity.BM25
		}
	}

	switch indexType {
	case "FLAT":
		idx = index.NewFlatIndex(metricType)
	case "IVF_FLAT":
		nlist := 1024
		if n, ok := indexParams["nlist"].(int); ok {
			nlist = n
		}
		idx = index.NewIvfFlatIndex(metricType, nlist)
	case "IVF_SQ8":
		nlist := 1024
		if n, ok := indexParams["nlist"].(int); ok {
			nlist = n
		}
		idx = index.NewIvfSQ8Index(metricType, nlist)
	case "IVF_PQ":
		nlist := 1024
		m := 4
		nbits := 8
		if n, ok := indexParams["nlist"].(int); ok {
			nlist = n
		}
		if mVal, ok := indexParams["m"].(int); ok {
			m = mVal
		}
		if nBits, ok := indexParams["nbits"].(int); ok {
			nbits = nBits
		}
		idx = index.NewIvfPQIndex(metricType, nlist, m, nbits)
	case "HNSW":
		M := 16
		efConstruction := 200
		if mVal, ok := indexParams["M"].(int); ok {
			M = mVal
		}
		if ef, ok := indexParams["efConstruction"].(int); ok {
			efConstruction = ef
		}
		idx = index.NewHNSWIndex(metricType, M, efConstruction)
	case "SPARSE_INVERTED_INDEX":
		dropRatio := 0.0
		if dr, ok := indexParams["dropRatio"].(float64); ok {
			dropRatio = dr
		}
		idx = index.NewSparseInvertedIndex(metricType, dropRatio)
	case "SPARSE_WAND":
		dropRatio := 0.0
		if dr, ok := indexParams["dropRatio"].(float64); ok {
			dropRatio = dr
		}
		idx = index.NewSparseWANDIndex(metricType, dropRatio)
	default:
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("unsupported index type: %s", indexType),
		})
	}

	option := milvusclient.NewCreateIndexOption(coll, fieldName, idx)
	task, err := c.client.CreateIndex(c.ctx, option)
	if err != nil {
		return toMap(&OperationResult{
			Success:      false,
			ResponseTime: float64(time.Since(start).Milliseconds()),
			Error:        fmt.Sprintf("failed to create index: %v", err),
		})
	}

	// Wait for index creation to complete
	err = task.Await(c.ctx)
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

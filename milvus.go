// Package milvus is the entry point for the xk6-milvus extension.
// It imports the milvus package to register the k6 extension.
package milvus

import (
	_ "github.com/zilliz/xk6-milvus/pkg/milvus" // Import to register the milvus extension
)
package milvus

import (
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/metrics"
)

func init() {
	modules.Register("k6/x/milvus", new(RootModule))
}

// Ensure the interfaces are implemented correctly
var (
	_ modules.Module   = &RootModule{}
	_ modules.Instance = &Milvus{}
)

// RootModule is the global module instance that creates module instances for each VU
type RootModule struct {
	metrics *Metrics
}

// Milvus represents the JS module instance for each VU
type Milvus struct {
	vu      modules.VU
	metrics *Metrics
}

// NewModuleInstance implements the modules.Module interface
// It creates a new instance of the Milvus module for each VU
func (r *RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	// Register metrics once for the first VU (only if VU and InitEnv are available)
	if r.metrics == nil && vu != nil && vu.InitEnv() != nil {
		registry := vu.InitEnv().Registry
		r.metrics = &Metrics{
			// Trend metrics - for statistical distribution (percentiles, avg, min, max)
			OperationDuration: registry.MustNewMetric("milvus_operation_duration", metrics.Trend, metrics.Time),
			SearchRecall:      registry.MustNewMetric("milvus_search_recall", metrics.Trend),
			IndexBuildDuration: registry.MustNewMetric("milvus_index_build_duration", metrics.Trend, metrics.Time),

			// Counter metrics - cumulative sum
			OperationsTotal:    registry.MustNewMetric("milvus_operations_total", metrics.Counter),
			RowsInserted:       registry.MustNewMetric("milvus_rows_inserted", metrics.Counter),
			RowsDeleted:        registry.MustNewMetric("milvus_rows_deleted", metrics.Counter),
			RerankerOperations: registry.MustNewMetric("milvus_reranker_operations", metrics.Counter),
			CollectionsCreated: registry.MustNewMetric("milvus_collections_created", metrics.Counter),

			// Rate metrics - ratio of non-zero values (0-1)
			Errors:          registry.MustNewMetric("milvus_errors", metrics.Rate),
			EmptyResults:    registry.MustNewMetric("milvus_empty_results", metrics.Rate),
			FilterUsed:      registry.MustNewMetric("milvus_filter_used", metrics.Rate),
			SparseVectorOps: registry.MustNewMetric("milvus_sparse_vector_operations", metrics.Rate),

			// Gauge metrics - latest value
			ResultCount:          registry.MustNewMetric("milvus_result_count", metrics.Gauge),
			SearchTopK:           registry.MustNewMetric("milvus_search_topk", metrics.Gauge),
			OutputFieldsCount:    registry.MustNewMetric("milvus_output_fields_count", metrics.Gauge),
			CollectionLoaded:     registry.MustNewMetric("milvus_collection_loaded", metrics.Gauge),
			HybridSearchRequests: registry.MustNewMetric("milvus_hybrid_search_requests", metrics.Gauge),

			// Throughput metrics
			ThroughputMBPS:   registry.MustNewMetric("milvus_throughput_mbps", metrics.Gauge),
			ThroughputRowsPS: registry.MustNewMetric("milvus_throughput_rows_per_second", metrics.Gauge),
		}
	}

	return &Milvus{
		vu:      vu,
		metrics: r.metrics,
	}
}

// Exports implements the modules.Instance interface
// It returns the exports of the module for JavaScript
func (m *Milvus) Exports() modules.Exports {
	return modules.Exports{
		Default: m,
		Named: map[string]interface{}{
			"client":               m.Client,
			"clientWithCollection": m.ClientWithCollection,
		},
	}
}

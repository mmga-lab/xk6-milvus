// Package milvus provides a k6 extension for load testing Milvus vector databases.
// This file contains the k6 module implementation and initialization logic.
package milvus

import (
	"fmt"
	"os"
	"time"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/metrics"
)

func init() {
	modules.Register("k6/x/milvus", New())
}

// RootModule is the global module object type. It is instantiated once per test
// run and will be used to create module instances for each VU.
type RootModule struct{}

// ModuleInstance represents an instance of the Milvus module for each VU.
type ModuleInstance struct {
	vu      modules.VU
	metrics struct {
		// Milvus-specific metrics
		MilvusReqs        *metrics.Metric
		MilvusDuration    *metrics.Metric
		MilvusVectors     *metrics.Metric
		MilvusDataSize    *metrics.Metric
		MilvusErrors      *metrics.Metric
		MilvusConnections *metrics.Metric
		MilvusRecall      *metrics.Metric // Search result quality metric
	}
}


// New returns a pointer to a new RootModule instance.
func New() *RootModule {
	return &RootModule{}
}

// NewModuleInstance implements the modules.Module interface to return
// a new instance for each VU.
func (r *RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	mi := &ModuleInstance{vu: vu}

	// Register custom metrics in init context only
	if initEnv := vu.InitEnv(); initEnv != nil {
		registry := initEnv.Registry
		mi.metrics.MilvusReqs = registry.MustNewMetric("milvus_reqs", metrics.Counter)
		mi.metrics.MilvusDuration = registry.MustNewMetric("milvus_req_duration", metrics.Trend, metrics.Time)
		mi.metrics.MilvusVectors = registry.MustNewMetric("milvus_vectors", metrics.Counter)
		mi.metrics.MilvusDataSize = registry.MustNewMetric("milvus_data_size", metrics.Counter, metrics.Data)
		mi.metrics.MilvusErrors = registry.MustNewMetric("milvus_errors", metrics.Rate)
		mi.metrics.MilvusConnections = registry.MustNewMetric("milvus_connections", metrics.Gauge)
		mi.metrics.MilvusRecall = registry.MustNewMetric("milvus_recall", metrics.Trend)
	}

	return mi
}

// Exports implements the modules.Instance interface and returns the exports
// of the JS module.
func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{
		Default: mi,
	}
}

// Client creates a new Milvus client connection with proper VU context integration.
// If address is empty, it defaults to "localhost:19530" or uses MILVUS_HOST environment variable.
func (mi *ModuleInstance) Client(address string) *Client {
	state := mi.vu.State()
	if state == nil {
		common.Throw(mi.vu.Runtime(), common.NewInitContextError("milvus.Client() can only be called in the VU context"))
	}

	// Handle address configuration
	if address == "" {
		if envAddr := os.Getenv("MILVUS_HOST"); envAddr != "" {
			address = envAddr
		} else {
			address = "localhost:19530"
		}
	}

	ctx := mi.vu.Context()
	c, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: address,
	})
	if err != nil {
		// Emit error metric
		mi.emitMetric(mi.metrics.MilvusErrors, 1, map[string]string{
			"operation": "connect",
			"address":   address,
			"error":     "connection_failed",
		})
		common.Throw(mi.vu.Runtime(), fmt.Errorf("failed to create milvus client: %v", err))
	}

	// Emit connection metric
	mi.emitMetric(mi.metrics.MilvusConnections, 1, map[string]string{
		"address": address,
	})

	return &Client{
		client: c,
		vu:     mi.vu,
		mi:     mi,
	}
}

// emitMetric is a helper method to emit metrics with proper VU context
func (mi *ModuleInstance) emitMetric(metric *metrics.Metric, value float64, tags map[string]string) {
	state := mi.vu.State()
	if state == nil || metric == nil {
		return
	}

	ctx := mi.vu.Context()
	now := time.Now()

	// Get current tags and merge with custom tags
	vuTags := state.Tags.GetCurrentValues()
	for k, v := range tags {
		vuTags.Tags = vuTags.Tags.With(k, v)
	}

	sample := metrics.Sample{
		TimeSeries: metrics.TimeSeries{
			Metric: metric,
			Tags:   vuTags.Tags,
		},
		Time:     now,
		Value:    value,
		Metadata: vuTags.Metadata,
	}

	// Push sample to k6's metrics system
	metrics.PushIfNotDone(ctx, state.Samples, metrics.ConnectedSamples{
		Samples: []metrics.Sample{sample},
		Tags:    vuTags.Tags,
		Time:    now,
	})
}
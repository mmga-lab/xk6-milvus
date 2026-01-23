package milvus

import (
	"go.k6.io/k6/js/modules"
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
type RootModule struct{}

// Milvus represents the JS module instance for each VU
type Milvus struct {
	vu      modules.VU
	clients map[string]*Client // VU-level client cache (key: address:collection)
}

// NewModuleInstance implements the modules.Module interface
// It creates a new instance of the Milvus module for each VU
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &Milvus{
		vu:      vu,
		clients: make(map[string]*Client),
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
			"getClient":            m.GetClient, // VU-level cached client
		},
	}
}

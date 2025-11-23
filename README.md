# xk6-milvus

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mmga-lab/xk6-milvus)](go.mod)

A [k6 extension](https://k6.io/docs/extensions/) for load testing [Milvus](https://milvus.io/) vector databases.

## Features

- 🎯 **Unified API** - Consistent `OperationResult` return structure following Locust pattern
- 📊 **Built-in Metrics** - Automatic `response_time_ms` tracking for every operation
- 🔍 **Quality Metrics** - `recall` metric for search quality assessment
- 🚀 **Collection Binding** - Create collection-bound clients for cleaner code
- 🔧 **Flexible Schema** - Support for complex schemas with multiple field types
- 🔎 **Vector Search** - Single and multi-vector search with filtering
- 🎭 **Hybrid Search** - Multi-vector search with RRF or Weighted reranking
- 📝 **BM25 Full-Text** - Automatic sparse vector generation for text search
- ⚡ **High Performance** - Optimized for concurrent load testing scenarios
- 🛡️ **VU Isolation** - Proper k6 VU context handling for thread-safe testing

## Use Cases

- **Vector Database Load Testing** - Test Milvus performance under various loads
- **Search Quality Assessment** - Monitor recall metrics during load tests
- **Capacity Planning** - Determine optimal collection sizes and query patterns
- **Performance Benchmarking** - Compare different index types and search strategies
- **Multi-Vector Testing** - Test hybrid search with dense + sparse vectors
- **Full-Text Search** - Load test BM25-based text search capabilities

## Installation

### Download Pre-built Binaries (Recommended)

Download k6 with xk6-milvus from [GitHub Releases](https://github.com/mmga-lab/xk6-milvus/releases):

1. Go to [Releases](https://github.com/mmga-lab/xk6-milvus/releases)
2. Download the binary for your platform:
   - **Linux**: `k6-linux-amd64` or `k6-linux-arm64`
   - **macOS**: `k6-darwin-amd64` or `k6-darwin-arm64`
   - **Windows**: `k6-windows-amd64.exe` or `k6-windows-arm64.exe`
3. Make it executable (Linux/macOS): `chmod +x k6-*`
4. Run: `./k6 version`

### Build from Source

#### Prerequisites

- Go 1.24 or later
- [xk6](https://github.com/grafana/xk6) - k6 extension builder
- Milvus 2.5.4 or later (for testing)

#### Build Steps

```bash
# Install xk6
go install go.k6.io/xk6/cmd/xk6@latest

# Build k6 with milvus extension
xk6 build --with github.com/mmga-lab/xk6-milvus@latest

# Verify the build
./k6 version
```

### Build with Local Development Version

```bash
# Clone the repository
git clone https://github.com/mmga-lab/xk6-milvus.git
cd xk6-milvus

# Build with local version
xk6 build --with github.com/mmga-lab/xk6-milvus=.

# Run examples
./k6 run examples/basic-operations.js
```

### Using Makefile (Development)

```bash
# See all available commands
make help

# Build k6 with extension
make build

# Run tests
make test

# Run with coverage
make coverage

# Run linters
make lint
```

## Quick Start

### Basic Vector Search

```javascript
import milvus from 'k6/x/milvus';
import { check } from 'k6';

export default function() {
  // Create collection-bound client (recommended)
  const client = milvus.clientWithCollection('localhost:19530', 'products');

  // Insert data
  const insertResult = client.insert({
    title: ['Product A', 'Product B'],
    price: [19.99, 29.99],
    embedding: [
      [0.1, 0.2, 0.3, ...], // 128-dim vector
      [0.4, 0.5, 0.6, ...]
    ]
  });

  check(insertResult, {
    'insert successful': (r) => r.success === true,
    'insert fast': (r) => r.response_time_ms < 300,
  });

  // Search vectors
  const searchResult = client.search(
    [[0.1, 0.2, 0.3, ...]], // query vector
    10, // top 10 results
    {
      vectorField: 'embedding',
      outputFields: ['title', 'price'],
      expr: 'price > 15.0'
    }
  );

  check(searchResult, {
    'search successful': (r) => r.success === true,
    'high recall': (r) => r.recall >= 0.95,
    'not empty': (r) => r.empty === false,
    'fast response': (r) => r.response_time_ms < 100,
  });

  // Process results
  if (searchResult.success) {
    searchResult.result.forEach(hit => {
      console.log(`${hit.title}: $${hit.price} (score: ${hit.score})`);
    });
  }

  client.close();
}
```

### Creating Collections

```javascript
import milvus from 'k6/x/milvus';

export function setup() {
  const client = milvus.client('localhost:19530');

  // Create collection with schema
  const schema = {
    name: 'products',
    fields: [
      { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
      { name: 'title', dataType: 'VarChar', maxLength: 200 },
      { name: 'price', dataType: 'Float' },
      { name: 'embedding', dataType: 'FloatVector', dimension: 128 }
    ]
  };

  const createResult = client.createCollection(schema);

  check(createResult, {
    'collection created': (r) => r.success === true,
  });

  // Create index for faster search
  const indexResult = client.createIndex('embedding', {
    indexType: 'HNSW',
    metricType: 'L2',
    params: { M: 16, efConstruction: 200 }
  }, 'products');

  // Load collection into memory
  const loadResult = client.loadCollection('products');

  client.close();
}
```

### Hybrid Search (Multi-Vector)

```javascript
import milvus from 'k6/x/milvus';

export default function() {
  const client = milvus.clientWithCollection('localhost:19530', 'documents');

  const hybridResult = client.hybridSearch(
    [
      {
        vectors: denseVectors,
        vectorField: 'dense_vector',
        limit: 10,
        params: { metricType: 'L2' }
      },
      {
        vectors: sparseVectors,
        vectorField: 'sparse_vector',
        limit: 10,
        params: { metricType: 'IP' }
      }
    ],
    {
      type: 'rrf',           // RRF reranking
      params: { k: 60 }
    },
    5,                       // final top 5
    ['title', 'content']
  );

  check(hybridResult, {
    'hybrid search successful': (r) => r.success === true,
    'good recall': (r) => r.recall >= 0.9,
  });

  client.close();
}
```

### BM25 Full-Text Search

```javascript
import milvus from 'k6/x/milvus';

export function setup() {
  const client = milvus.client('localhost:19530');

  // Create collection with BM25 function
  const schema = {
    name: 'documents',
    fields: [
      { name: 'id', dataType: 'Int64', isPrimaryKey: true },
      {
        name: 'text',
        dataType: 'VarChar',
        maxLength: 25536,
        enableAnalyzer: true,
        analyzerParams: { type: 'standard' },
        enableMatch: true
      },
      { name: 'sparse', dataType: 'SparseFloatVector' }
    ],
    functions: [
      {
        name: 'text_bm25_emb',
        functionType: 'BM25',
        inputFieldNames: ['text'],
        outputFieldNames: ['sparse']
      }
    ]
  };

  client.createCollection(schema);
  client.loadCollection('documents');
  client.close();
}

export default function() {
  const client = milvus.clientWithCollection('localhost:19530', 'documents');

  // Insert text (sparse vectors generated automatically)
  client.upsert({
    id: [1, 2, 3],
    text: ['Document one', 'Document two', 'Document three']
  });

  client.close();
}
```

## API Reference

See [docs/API.md](docs/API.md) for complete API documentation.

### Client Creation

- `milvus.client(address, token?)` - Create standard client
- `milvus.clientWithCollection(address, collectionName, token?)` - Create collection-bound client

### Collection Operations

- `client.createCollection(schema)` - Create collection
- `client.dropCollection(collectionName?)` - Drop collection
- `client.hasCollection(collectionName?)` - Check existence
- `client.loadCollection(collectionName?)` - Load into memory
- `client.releaseCollection(collectionName?)` - Release from memory

### Data Operations

- `client.insert(data, collectionName?)` - Insert entities
- `client.upsert(data, collectionName?)` - Upsert entities
- `client.delete(filter, collectionName?)` - Delete by filter

### Search Operations

- `client.search(vectors, topK, params, collectionName?)` - Vector similarity search
- `client.query(filter, outputFields, collectionName?)` - Scalar query
- `client.hybridSearch(requests, reranker, limit, outputFields, collectionName?)` - Multi-vector search

### Index Operations

- `client.createIndex(fieldName, indexParams, collectionName?)` - Create index

### OperationResult Structure

All operations return `OperationResult`:

```javascript
{
  success: true,              // boolean - operation success
  response_time_ms: 123.45,   // float - duration in ms
  result: {...},              // any - operation-specific result
  error: "",                  // string - error message if failed
  empty: false,               // boolean - whether result is empty
  recall: 0.95                // float - recall metric (search only)
}
```

## Examples

### Progressive Learning

| Example | Description |
|---------|-------------|
| `examples/basic-operations.js` | Basic CRUD operations |
| `examples/collection-management.js` | Collection lifecycle |
| `examples/vector-search.js` | Vector similarity search |
| `examples/hybrid-search.js` | Multi-vector hybrid search |
| `examples/full-text-search.js` | BM25 full-text search |

See all examples in the [`examples/`](examples/) directory.

## Performance Tips

1. **Use Collection-Bound Clients** - Cleaner code and fewer parameters
2. **Load Collections First** - Collections must be loaded before searching
3. **Create Indexes** - Create indexes after inserting data for faster search
4. **Batch Operations** - Insert/upsert multiple entities at once
5. **Monitor Metrics** - Use `response_time_ms` and `recall` for observability
6. **Proper Indexing** - Choose appropriate index types (HNSW for speed, IVF_FLAT for accuracy)

## Configuration

### Environment Variables

- `MILVUS_HOST` - Milvus server address (default: `localhost:19530`)
- `MILVUS_TOKEN` - Authentication token (default: `root:Milvus`)

### k6 Options

Customize load testing behavior:

```javascript
export const options = {
  vus: 10,           // Virtual users
  duration: '30s',   // Test duration
  iterations: 1000,  // Total iterations
  thresholds: {
    'checks': ['rate>0.99'],  // 99% success rate
  },
};
```

## Development

### Project Structure

```
xk6-milvus/
├── register.go              # Extension registration
├── pkg/milvus/              # Core implementation
│   ├── module.go            # k6 module registration
│   ├── client.go            # Client management
│   ├── collection.go        # Collection operations
│   ├── data.go              # Data operations
│   ├── search.go            # Search operations
│   ├── index.go             # Index operations
│   ├── converters.go        # Type conversions
│   ├── types.go             # Type definitions
│   └── *_test.go            # Tests
├── examples/                # Usage examples
├── docs/                    # Documentation
│   └── API.md               # Complete API reference
├── .github/                 # CI/CD workflows
└── deployment/              # Deployment configs
```

### Running Tests

```bash
# All tests
make test

# With coverage
make coverage

# Specific package
go test -v ./pkg/milvus

# E2E tests (requires running Milvus)
go test -tags e2e -v ./pkg/milvus
```

### Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## Architecture

This extension follows:
- **k6 Extension Best Practices** - RootModule/ModuleInstance pattern
- **Locust Milvus Client Pattern** - Unified OperationResult, built-in metrics
- **VU Context Management** - Proper lifecycle for concurrent testing
- **Package Organization** - Clean separation in `pkg/milvus/`

## Troubleshooting

### Common Issues

**Connection Refused**
```javascript
// Ensure Milvus is running
docker ps | grep milvus

// Set correct host
export MILVUS_HOST=localhost:19530
```

**Collection Not Loaded**
```javascript
// Always load before searching
client.loadCollection('my_collection');
```

**Index Not Found**
```javascript
// Create index after inserting data
client.createIndex('embedding', {
  indexType: 'HNSW',
  metricType: 'L2'
}, 'my_collection');
```

**Low Recall**
```javascript
// Check search params and index configuration
// Increase nprobe for IVF indexes
// Increase ef for HNSW indexes
```

## Resources

- [Complete API Documentation](docs/API.md)
- [Contributing Guidelines](CONTRIBUTING.md)
- [Changelog](CHANGELOG.md)
- [Milvus Documentation](https://milvus.io/docs)
- [k6 Documentation](https://k6.io/docs/)
- [k6 Extensions](https://k6.io/docs/extensions/)

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

## Acknowledgments

- Design pattern inspired by [Locust's Milvus client](https://github.com/locustio/locust/blob/master/locust/contrib/milvus.py)
- Built with [Milvus Go SDK](https://github.com/milvus-io/milvus/tree/master/client) v2.5.4
- Powered by [k6](https://k6.io/)

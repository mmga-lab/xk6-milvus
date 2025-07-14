# xk6-milvus

[![CI](https://github.com/zilliz/xk6-milvus/workflows/CI/badge.svg)](https://github.com/zilliz/xk6-milvus/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/zilliz/xk6-milvus.svg)](https://pkg.go.dev/github.com/zilliz/xk6-milvus)

A [k6](https://k6.io/) extension for load testing [Milvus](https://milvus.io/) vector databases. This extension provides a JavaScript API to interact with Milvus from k6 test scripts, enabling comprehensive performance testing of vector operations.

## Features

- **Vector Operations**: Insert, search, and manage vector data
- **Collection Management**: Create, drop, load, and release collections
- **Index Operations**: Create and manage vector indexes
- **Flexible Schema**: Support for complex schemas with multiple field types
- **Backward Compatibility**: Simple methods for basic use cases
- **Performance Testing**: Built specifically for k6 load testing scenarios
- **Built-in Metrics**: Comprehensive Go-level metrics for all operations
- **VU Context Integration**: Proper k6 VU lifecycle and error handling

## Prerequisites

- **Go**: 1.24.2+ (required by Milvus SDK v2.5.4)
- **k6**: Latest version
- **xk6**: For building custom k6 binaries
- **Milvus**: Running instance (2.0+)

## Installation

### Option 1: Using Makefile (Recommended)

```bash
# Clone the repository
git clone https://github.com/zilliz/xk6-milvus.git
cd xk6-milvus

# Build k6 with milvus extension
make build
```

### Option 2: Manual Build

```bash
# Install xk6 if not already installed
go install go.k6.io/xk6/cmd/xk6@latest

# Build k6 with milvus extension
xk6 build --with github.com/zilliz/xk6-milvus=.
```

## Quick Start

### Simple Vector Operations

```javascript
import milvus from 'k6/x/milvus';
import { check } from 'k6';

export const options = {
    vus: 5,
    duration: '30s',
    thresholds: {
        // Built-in metrics from xk6-milvus
        'milvus_reqs': ['count>100'],
        'milvus_req_duration': ['p(95)<500'], // 95% under 500ms
        'milvus_errors': ['rate<0.05'],       // Error rate under 5%
    },
};

export function setup() {
    const client = milvus.client(); // Uses MILVUS_HOST env var or localhost:19530
    
    // Create a simple collection
    client.createCollectionSimple('test_collection', 128);
    client.createIndexSimple('test_collection', 'vector');
    client.loadCollection('test_collection');
    
    return { client };
}

export default function(data) {
    const { client } = data;
    
    // Generate random vectors
    const vectors = [];
    for (let i = 0; i < 10; i++) {
        const vector = Array(128).fill(0).map(() => Math.random());
        vectors.push(vector);
    }
    
    // Insert vectors (automatically emits metrics)
    const ids = client.insertVectors('test_collection', vectors);
    check(ids, {
        'vectors inserted successfully': (ids) => ids.length === 10,
    });
    
    // Search similar vectors (automatically emits metrics)
    const searchVectors = [vectors[0]]; // Use first vector as query
    const results = client.searchSimple('test_collection', searchVectors, 5);
    check(results, {
        'search returned results': (results) => results.length > 0,
        'search score is valid': (results) => results[0].score >= 0,
    });
}

export function teardown(data) {
    data.client.close();
}
```

### Advanced Schema Operations

```javascript
import milvus from 'k6/x/milvus';

export function setup() {
    const client = milvus.client('localhost:19530');
    
    // Define complex schema
    const schema = {
        name: 'products',
        description: 'Product catalog with embeddings',
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'title', dataType: 'VarChar', maxLength: 200 },
            { name: 'price', dataType: 'Float' },
            { name: 'category', dataType: 'VarChar', maxLength: 50 },
            { name: 'embedding', dataType: 'FloatVector', dimension: 256 }
        ]
    };
    
    // Create collection from schema
    client.createCollectionFromJSON(JSON.stringify(schema));
    
    // Create HNSW index for better performance
    const indexParams = {
        indexType: 'HNSW',
        metricType: 'L2',
        params: { M: 16, efConstruction: 256 }
    };
    client.createIndex('products', 'embedding', indexParams);
    client.loadCollection('products');
    
    return { client };
}

export default function(data) {
    const { client } = data;
    
    // Insert structured data
    const batchData = {
        title: ['Product A', 'Product B', 'Product C'],
        price: [19.99, 29.99, 39.99],
        category: ['electronics', 'books', 'clothing'],
        embedding: [
            Array(256).fill(0).map(() => Math.random()),
            Array(256).fill(0).map(() => Math.random()),
            Array(256).fill(0).map(() => Math.random())
        ]
    };
    
    const ids = client.insert('products', batchData);
    
    // Search with filters and output fields
    const searchParams = {
        vectorField: 'embedding',
        outputFields: ['title', 'price', 'category'],
        expr: 'price > 20.0 and category == "electronics"'
    };
    
    const queryVector = [Array(256).fill(0).map(() => Math.random())];
    const results = client.search('products', queryVector, 10, searchParams);
}
```

## API Reference

### Client Management

#### `milvus.client(address: string): Client`
Creates a new Milvus client connection.
- **address**: Milvus server address (default: "localhost:19530")

#### `client.close(): void`
Closes the client connection. Should be called in teardown.

### Collection Operations

#### `client.createCollectionFromJSON(schemaJSON: string): void`
Creates a collection from a JSON schema definition.

#### `client.createCollection(schema: Schema): void`
Creates a collection with a flexible schema object.

#### `client.createCollectionSimple(name: string, dimension: number): void`
Creates a simple collection with auto-ID and vector field (backward compatibility).

#### `client.dropCollection(name: string): void`
Drops a collection.

#### `client.hasCollection(name: string): boolean`
Checks if a collection exists.

#### `client.loadCollection(name: string): void`
Loads a collection into memory (required before search).

#### `client.releaseCollection(name: string): void`
Releases a collection from memory.

### Data Operations

#### `client.insert(collectionName: string, data: object): number[]`
Inserts data into a collection. Data should be an object with field names as keys.

#### `client.insertVectors(collectionName: string, vectors: number[][]): number[]`
Inserts vectors into a simple collection (backward compatibility).

### Search Operations

#### `client.search(collectionName: string, vectors: number[][], topK: number, params?: object): SearchResult[]`
Performs vector search with optional parameters.

#### `client.searchSimple(collectionName: string, vectors: number[][], topK: number): SearchResult[]`
Performs simple vector search (backward compatibility).

#### `client.searchWithRecall(collectionName: string, vectors: number[][], topK: number, params: object, groundTruth: number[][]): SearchResult[]`
Performs vector search and automatically calculates recall metrics when ground truth data is provided.

### Index Operations

#### `client.createIndex(collectionName: string, fieldName: string, indexParams: object): void`
Creates an index on a field with specified parameters.

#### `client.createIndexSimple(collectionName: string, fieldName: string): void`
Creates a default FLAT index (backward compatibility).

## Built-in Metrics

xk6-milvus automatically emits comprehensive metrics for all operations:

### Core Metrics

- **`milvus_reqs`** (Counter): Total number of Milvus requests
- **`milvus_req_duration`** (Trend, Time): Request duration in milliseconds
- **`milvus_vectors`** (Counter): Number of vectors processed
- **`milvus_data_size`** (Counter, Data): Amount of data transferred
- **`milvus_errors`** (Rate): Error rate (0-1)
- **`milvus_connections`** (Gauge): Number of active connections
- **`milvus_recall`** (Trend): Search result recall rate (0-1, quality metric)

### Metric Tags

All metrics include contextual tags:
- **`operation`**: Type of operation (`insert`, `search`, `create_collection`, etc.)
- **`collection`**: Collection name
- **`status`**: Operation status (`success`, `error`)
- **`address`**: Milvus server address

### Using Metrics in Thresholds

```javascript
export const options = {
    thresholds: {
        // Performance thresholds
        'milvus_req_duration': ['p(95)<1000', 'p(99)<2000'],
        'milvus_req_duration{operation:search}': ['p(90)<500'],
        'milvus_req_duration{operation:insert}': ['p(95)<800'],
        
        // Volume thresholds
        'milvus_reqs': ['count>1000'],
        'milvus_vectors': ['count>10000'],
        
        // Quality thresholds
        'milvus_errors': ['rate<0.01'],           // Less than 1% errors
        'milvus_errors{operation:search}': ['rate<0.005'], // Even lower for search
        'milvus_recall': ['avg>0.8', 'min>0.6'], // Average recall > 80%, min > 60%
        
        // Checks
        'checks': ['rate>0.95'],
    },
};
```

### Combining with Custom Metrics

```javascript
import { Counter, Trend } from 'k6/metrics';

// Additional custom metrics
const businessMetric = new Counter('milvus_business_operations');
const customLatency = new Trend('milvus_e2e_latency');

export default function() {
    const start = new Date();
    
    // Your Milvus operations here
    // (xk6-milvus automatically emits built-in metrics)
    
    // Add your custom metrics
    businessMetric.add(1);
    customLatency.add(new Date() - start);
}
```

## Architecture

The extension follows a clean directory structure for better maintainability:

```
xk6-milvus/
├── milvus.go                    # Entry point for xk6 extension registration
├── pkg/milvus/                  # Core implementation package
│   ├── module.go                # k6 module initialization and metrics
│   ├── types.go                 # Type definitions and data structures
│   ├── client.go                # Client implementation and data operations
│   ├── search.go                # Search operations and recall calculation
│   ├── milvus.go                # Package documentation
│   └── milvus_test.go           # Unit tests
├── examples/                    # Usage examples
├── docs/                        # Additional documentation
└── README.md                    # This file
```

This modular structure provides:
- **Clean separation**: Each file has a single responsibility
- **Easy navigation**: Related functionality is grouped together
- **Better testability**: Unit tests are co-located with implementation
- **Professional layout**: Follows Go community standards

## Development

### Building and Testing

```bash
# Format code
make fmt

# Run tests
make test

# Run linter
make lint

# Build extension
make build

# Run examples
make examples

# All checks (fmt + lint + test + build)
make all
```

### Running Examples

```bash
# Run basic example
make example-basic

# Run advanced example  
make example-advanced

# Set custom Milvus host
MILVUS_HOST=10.0.0.1:19530 make examples
```

## Configuration

### Environment Variables

- **MILVUS_HOST**: Milvus server address (default: "localhost:19530")

### k6 Options

Configure test execution with k6 options:

```javascript
export const options = {
    vus: 10,           // Virtual users
    duration: '30s',   // Test duration
    iterations: 1000,  // Total iterations
    thresholds: {
        checks: ['rate>0.95'],  // 95% of checks should pass
    },
};
```

## Performance Tips

1. **Use batch operations**: Insert multiple vectors at once for better performance
2. **Create appropriate indexes**: Use HNSW or IVF indexes for large datasets
3. **Load collections**: Ensure collections are loaded before search operations
4. **Optimize vector dimensions**: Higher dimensions increase search time
5. **Use filters wisely**: Complex expressions can impact search performance

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run `make all` to ensure all checks pass
6. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/zilliz/xk6-milvus/issues)
- **Documentation**: [Milvus Documentation](https://milvus.io/docs)
- **k6 Documentation**: [k6.io](https://k6.io/docs/)
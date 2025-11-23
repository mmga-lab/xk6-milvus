# xk6-milvus

k6 extension for load testing Milvus vector database.

## Prerequisites

- Go 1.24+
- k6
- xk6

## Build

```bash
# Install xk6
go install go.k6.io/xk6/cmd/xk6@latest

# Build k6 with milvus extension only
xk6 build --with github.com/zilliz/xk6-milvus=.

# Build with milvus and faker extensions (for upsert-test.js)
xk6 build --with github.com/zilliz/xk6-milvus=. --with github.com/grafana/xk6-faker
```

## Usage

```javascript
import milvus from 'k6/x/milvus';
import { check } from 'k6';

export default function() {
    // Connect to Milvus
    const client = milvus.client('localhost:19530');
    
    // Create collection
    client.createCollection('test_collection', 128);
    
    // Insert vectors
    const vectors = [[0.1, 0.2, ...], [0.3, 0.4, ...]];
    const ids = client.insert('test_collection', vectors);
    
    // Search vectors
    const searchVectors = [[0.1, 0.2, ...]];
    const results = client.search('test_collection', searchVectors, 10);
    
    // Close connection
    client.close();
}
```

## Run Tests

```bash
# Set Milvus host (optional, defaults to localhost:19530)
export MILVUS_HOST=your-milvus-host:19530

# Run the test
./k6 run example/test-milvus.js
```

## Available Methods

### Client Management
- `milvus.client(address)` - Create a new Milvus client
- `client.close()` - Close connection

### Collection Operations
- `client.createCollection(schema)` - Create collection with flexible schema
- `client.createCollectionFromJSON(schemaJSON)` - Create from JSON schema
- `client.createCollectionSimple(name, dimension)` - Simple collection creation
- `client.dropCollection(name)` - Drop a collection
- `client.hasCollection(name)` - Check if collection exists
- `client.loadCollection(collectionName)` - Load collection into memory
- `client.releaseCollection(collectionName)` - Release collection from memory

### Data Operations
- `client.insert(collectionName, data)` - Insert data with multiple fields
- `client.insertVectors(collectionName, vectors)` - Simple vector insertion
- `client.upsert(collectionName, data)` - Upsert (insert or update) data

### Search Operations
- `client.search(collectionName, vectors, topK, params)` - Search with filters
- `client.searchSimple(collectionName, vectors, topK)` - Simple search

### Index Operations
- `client.createIndex(collectionName, fieldName, params)` - Create index with params
- `client.createIndexSimple(collectionName, fieldName)` - Create simple index

### Advanced Features
- Support for BM25 and TextEmbedding functions
- Text analyzer and match functionality
- Sparse vectors (SparseFloatVector)
- Configurable shard numbers

## Test Options

You can customize the test behavior using k6 options:

```javascript
export const options = {
    vus: 10,        // Number of virtual users
    duration: '30s', // Test duration
    iterations: 1000, // Total iterations
};
```

## Examples

### Simple Vector Operations
See `example/test-milvus.js` for basic vector insertion and search.

### Complex Schema with Multiple Fields
See `example/flexible-test.js` for advanced schema with multiple field types.

### Upsert with BM25 Full-Text Search
See `example/upsert-test.js` and `example/UPSERT_TEST_README.md` for BM25 function usage.

## Environment Variables

- `MILVUS_HOST` / `MILVUS_URI` - Milvus server address (default: localhost:19530)
- `MILVUS_TOKEN` - Authentication token (default: root:Milvus)
- `UPSERT_BATCH_SIZE` - Batch size for upsert operations (default: 5000)
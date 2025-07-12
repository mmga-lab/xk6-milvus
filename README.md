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

# Build k6 with milvus extension
xk6 build --with github.com/zilliz/xk6-milvus=.
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

- `milvus.client(address)` - Create a new Milvus client
- `client.createCollection(name, dimension)` - Create a collection
- `client.dropCollection(name)` - Drop a collection
- `client.hasCollection(name)` - Check if collection exists
- `client.insert(collectionName, vectors)` - Insert vectors
- `client.search(collectionName, vectors, topK)` - Search vectors
- `client.createIndex(collectionName, fieldName)` - Create index
- `client.loadCollection(collectionName)` - Load collection
- `client.releaseCollection(collectionName)` - Release collection
- `client.close()` - Close connection

## Test Options

You can customize the test behavior using k6 options:

```javascript
export const options = {
    vus: 10,        // Number of virtual users
    duration: '30s', // Test duration
    iterations: 1000, // Total iterations
};
```

## Environment Variables

- `MILVUS_HOST` - Milvus server address (default: localhost:19530)
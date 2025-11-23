# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

xk6-milvus is a k6 extension for load testing Milvus vector databases. It provides a JavaScript API to interact with Milvus from k6 test scripts, following Locust's Milvus client design pattern for consistency and observability.

## Architecture

The project follows k6 extension best practices with a RootModule/ModuleInstance pattern and clean package organization:

### Project Structure
```
xk6-milvus/
├── register.go              # Extension registration entry point (6 lines)
├── pkg/milvus/              # Core implementation package
│   ├── module.go            # k6 module registration and RootModule
│   ├── client.go            # Client creation and management
│   ├── collection.go        # Collection operations
│   ├── data.go              # Insert/upsert/delete operations
│   ├── search.go            # Search and query operations
│   ├── index.go             # Index management
│   ├── converters.go        # Type conversions (schema, vectors, etc.)
│   ├── types.go             # Type definitions and structs
│   ├── errors.go            # Error handling
│   ├── config.go            # Configuration structs
│   ├── helpers.go           # Helper functions
│   └── *_test.go            # Co-located tests
├── examples/                # Usage examples
├── docs/                    # Documentation
│   └── API.md               # Complete API reference
└── .github/                 # CI/CD workflows and templates
```

### Module Pattern
- **register.go**: Minimal entry point that imports pkg/milvus for side effects
- **RootModule**: Global module instance that creates module instances for each VU
- **Milvus**: VU-specific module instance with access to VU context
- **Client**: Milvus client wrapper using VU context for proper lifecycle management
- Wraps the official Milvus Go SDK v2.5.4 to provide k6-friendly methods
- Registers using `modules.Register("k6/x/milvus", new(RootModule))`

### Key Principles
- Each VU gets its own Milvus instance for proper isolation
- VU context is used for all operations (not background context)
- Exports both default and named exports following ES module conventions
- Supports collection-bound clients for cleaner code
- Clean separation of concerns in pkg/milvus/

## Common Commands

### Build
```bash
# Using Makefile (recommended)
make help        # Show all available commands
make build       # Build k6 with extension
make test        # Run tests
make lint        # Run linters
make coverage    # Generate coverage report

# Or manually
go install go.k6.io/xk6/cmd/xk6@latest
xk6 build --with github.com/mmga-lab/xk6-milvus=.
```

### Run Tests
```bash
# Using Makefile (recommended)
make test                    # Run unit tests
make test-integration        # Run integration tests (requires MILVUS_HOST)
make test-integration-local  # Run integration tests with auto Milvus setup
make test-e2e-local          # Run E2E tests with auto Milvus setup
make test-all-local          # Run all tests (unit + integration + E2E)
make coverage                # Generate coverage HTML

# Manage Milvus for testing
make docker-up               # Start Milvus (uses --wait for readiness)
make docker-down             # Stop and cleanup Milvus
make docker-logs             # View Milvus logs
make docker-status           # Check Milvus status

# Or manually
go test -v ./pkg/milvus
go test -tags=integration -v ./pkg/milvus

# Set Milvus host for testing
export MILVUS_HOST=localhost:19530

# Deploy Milvus manually
docker compose -f deployment/docker-compose.yml up -d --wait

# Run k6 examples
./k6 run examples/basic-operations.js
./k6 run examples/vector-search.js
./k6 run examples/hybrid-search.js
```

### Development Workflow
1. Create feature branch: `git checkout -b feature/my-feature`
2. Modify code in `pkg/milvus/`
3. Add tests in `pkg/milvus/*_test.go`
4. Run tests: `make test`
5. Run linters: `make lint`
6. Rebuild: `make build`
7. Test with examples: `./k6 run examples/...`
8. Update documentation (README, API.md, CLAUDE.md)
9. Commit following Conventional Commits format
10. Submit pull request

### Dependencies
- Milvus Go SDK v2.5.4: `github.com/milvus-io/milvus/client/v2/milvusclient`
- Go 1.24+
- k6 modules system

## Key Implementation Details

### API Design (Based on Locust Pattern)

All operations return a unified `OperationResult` structure:

```javascript
{
    success: true,              // boolean - operation success
    response_time_ms: 123.45,   // float - operation duration in milliseconds
    result: {...},              // any - operation-specific result
    error: "",                  // string - error message if failed
    empty: false,               // boolean - whether result set is empty (search/query)
    recall: 0.95                // float - recall metric (search operations only)
}
```

### Available Methods

#### Client Creation
- `client(address, token?)` - Create standard client
- `clientWithCollection(address, collectionName, token?)` - Create collection-bound client

#### Collection Operations
- `createCollection(schema)` - Create collection with flexible schema
- `createCollectionFromJSON(schemaJSON)` - Create from JSON string
- `dropCollection(collectionName?)` - Drop collection
- `hasCollection(collectionName?)` - Check if collection exists
- `loadCollection(collectionName?)` - Load collection into memory
- `releaseCollection(collectionName?)` - Release collection from memory

#### Write Operations
- `insert(data, collectionName?)` - Insert data
- `upsert(data, collectionName?)` - Upsert (insert or update) data
- `delete(filter, collectionName?)` - Delete entities by filter expression

#### Read Operations
- `search(vectors, topK, params, collectionName?)` - Vector similarity search
- `query(filter, outputFields, collectionName?)` - Scalar query without vectors
- `hybridSearch(requests, reranker, limit, outputFields, collectionName?)` - Multi-vector hybrid search

#### Index Operations
- `createIndex(fieldName, indexParams, collectionName?)` - Create vector index

### SDK Implementation Details
- Uses Milvus client SDK v2.5.4: `github.com/milvus-io/milvus/client/v2/milvusclient`
- Imports: `milvusclient`, `entity`, `index`, `column` packages
- Client type: `*milvusclient.Client` (pointer type)
- Collection operations use option pattern: `milvusclient.NewCreateCollectionOption()`
- Insert operations use column-based approach: `milvusclient.NewColumnBasedInsertOption()`
- Search operations use entity.Vector interface with entity.FloatVector implementation
- Index and load operations return tasks that must be awaited: `task.Await(ctx)`

### Error Handling
All methods return `OperationResult` instead of throwing errors. Check `result.success` and `result.error` for error handling.

### Testing Approach
- Use k6 scripts to test the extension
- **New API test**: `example/new-api-test.js` demonstrates all new features
- **Collection-bound test**: `example/collection-bound-test.js` shows collection binding pattern
- **Hybrid search test**: `example/hybrid-search-test.js` demonstrates multi-vector search
- Set `MILVUS_HOST` environment variable to specify custom Milvus instance

## Important Considerations

1. **Vector Format**: Vectors are passed as JavaScript arrays and converted to entity.FloatVector
2. **Client Lifecycle**: Clients can be reused within VU iterations or recreated per iteration
3. **Collection Loading**: Collections must be loaded before search operations
4. **Index Creation**: Create indexes after inserting data, operations are async and require task.Await()
5. **Unified Results**: All operations return `OperationResult` for consistent error handling
6. **Response Time**: Built-in response_time_ms in every operation result
7. **Recall Metric**: Search operations include recall metric for quality verification
8. **Collection Binding**: Use `clientWithCollection()` to avoid repeating collection names
9. **Functions Support**: Supports BM25 and TextEmbedding functions for automatic sparse vector generation
10. **Hybrid Search**: Supports multi-vector search with RRF or Weighted reranking

## API Usage Examples

### Collection-Bound Client (Recommended)
```javascript
import milvus from 'k6/x/milvus';
import { check } from 'k6';

export default function() {
    // Create collection-bound client
    const client = milvus.clientWithCollection('localhost:19530', 'my_collection');

    // Insert
    const insertResult = client.insert({
        title: ['Product A', 'Product B'],
        price: [19.99, 29.99],
        vector: [vector1, vector2]
    });

    check(insertResult, {
        'insert successful': (r) => r.success === true,
        'insert fast': (r) => r.response_time_ms < 300,
    });

    // Search with recall
    const searchResult = client.search(searchVectors, 10, {
        vectorField: 'vector',
        outputFields: ['title', 'price'],
        expr: 'price > 15.0'
    });

    check(searchResult, {
        'search successful': (r) => r.success === true,
        'high recall': (r) => r.recall >= 0.95,
        'not empty': (r) => r.empty === false,
    });

    // Query (filter without vectors)
    const queryResult = client.query('price > 100', ['id', 'title', 'price']);

    if (queryResult.success && !queryResult.empty) {
        console.log(`Found ${queryResult.result.length} products`);
    }

    // Delete
    const deleteResult = client.delete('price < 10');
    console.log(`Deleted ${deleteResult.result.delete_count} records`);

    client.close();
}
```

### Flexible Schema
```javascript
// Create complex schema with multiple field types
const schema = {
    name: 'products',
    fields: [
        { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
        { name: 'title', dataType: 'VarChar', maxLength: 200 },
        { name: 'price', dataType: 'Float' },
        { name: 'embedding', dataType: 'FloatVector', dimension: 128 }
    ]
};

const client = milvus.client('localhost:19530');
const createResult = client.createCollection(schema);

check(createResult, {
    'collection created': (r) => r.success === true,
});
```

### Hybrid Search (Multi-Vector)
```javascript
const client = milvus.clientWithCollection('localhost:19530', 'multi_vector_collection');

const hybridResult = client.hybridSearch(
    [
        {
            vectors: denseVectors,
            vectorField: 'dense_vector',
            limit: 10,
            params: { metricType: 'L2', expr: 'price > 50' }
        },
        {
            vectors: sparseVectors,
            vectorField: 'sparse_vector',
            limit: 10,
            params: { metricType: 'IP' }
        }
    ],
    {
        type: 'rrf',           // 'rrf' or 'weighted'
        params: { k: 60 }      // RRF k parameter
    },
    5,                         // final limit after reranking
    ['title', 'price']         // output fields
);

check(hybridResult, {
    'hybrid search successful': (r) => r.success === true,
    'good recall': (r) => r.recall >= 0.9,
});
```

### BM25 Full-Text Search
```javascript
// Create collection with BM25 function
const schema = {
    name: 'documents',
    numShards: 16,
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

const client = milvus.client('localhost:19530');
client.createCollection(schema);
client.loadCollection('documents');

// Upsert data
const upsertResult = client.upsert({
    id: [1, 2, 3],
    text: ['Doc one', 'Doc two', 'Doc three']
}, 'documents');

check(upsertResult, {
    'upsert successful': (r) => r.success === true,
});
```

## Additional Resources

- **Milvus Go SDK source code**: `/Users/wxzhu/workspace/milvus/client`
- **Module path**: `github.com/mmga-lab/xk6-milvus`
- **Package path**: `github.com/mmga-lab/xk6-milvus/pkg/milvus`

### Documentation
- **README.md**: User-facing documentation with quick start
- **docs/API.md**: Complete API reference with all methods
- **CONTRIBUTING.md**: Development setup and contribution guidelines
- **CHANGELOG.md**: Version history and migration guides

### Examples (examples/ directory)
- `basic-operations.js` - Basic CRUD operations
- `collection-management.js` - Collection lifecycle management
- `vector-search.js` - Vector similarity search patterns
- `hybrid-search.js` - Multi-vector hybrid search
- `full-text-search.js` - BM25 full-text search
- Older examples in root for backward compatibility

### GitHub Configuration
- **Workflows**: `.github/workflows/test.yml` - Multi-job CI (test, lint, build)
- **Issue Templates**: Bug reports, feature requests in `.github/ISSUE_TEMPLATE/`
- **PR Template**: Pull request template for consistent submissions
- **Security**: Security policy in `.github/SECURITY.md`

## Design Pattern

This implementation follows [Locust's Milvus client](https://github.com/locustio/locust/blob/master/locust/contrib/milvus.py) design pattern:
- Unified `OperationResult` return structure
- Built-in response time tracking
- Collection-bound clients
- Recall metric exposure
- Empty detection
- Consistent error handling

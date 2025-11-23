# API Documentation

Complete API reference for xk6-milvus extension.

## Module Import

```javascript
import milvus from 'k6/x/milvus';
```

## Client Creation

### client()

Creates a standard Milvus client for interacting with a Milvus server.

#### Signature

```javascript
client(address: string, token?: string): Client
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `address` | string | Yes | Milvus server address (e.g., "localhost:19530") |
| `token` | string | No | Authentication token |

#### Returns

Client object for executing Milvus operations.

#### Example

```javascript
const client = milvus.client('localhost:19530');
const clientWithAuth = milvus.client('localhost:19530', 'my-token');
```

---

### clientWithCollection()

Creates a collection-bound Milvus client that automatically uses the specified collection for all operations.

#### Signature

```javascript
clientWithCollection(address: string, collectionName: string, token?: string): Client
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `address` | string | Yes | Milvus server address |
| `collectionName` | string | Yes | Default collection name for all operations |
| `token` | string | No | Authentication token |

#### Returns

Collection-bound Client object.

#### Example

```javascript
const client = milvus.clientWithCollection('localhost:19530', 'products');
// All operations now default to 'products' collection
```

---

## Collection Operations

### createCollection()

Creates a new collection with the specified schema.

#### Signature

```javascript
createCollection(schema: CollectionSchema): OperationResult
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `schema` | CollectionSchema | Yes | Collection schema definition |

#### CollectionSchema

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | Collection name |
| `fields` | FieldSchema[] | Yes | Array of field definitions |
| `numShards` | number | No | Number of shards (default: 2) |
| `functions` | Function[] | No | Functions for automatic processing |

#### FieldSchema

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | Field name |
| `dataType` | string | Yes | Data type (Int64, Float, VarChar, FloatVector, etc.) |
| `isPrimaryKey` | boolean | No | Whether this is the primary key field |
| `isAutoID` | boolean | No | Auto-generate IDs for primary key |
| `maxLength` | number | Conditional | Max length for VarChar fields |
| `dimension` | number | Conditional | Dimension for vector fields |
| `enableAnalyzer` | boolean | No | Enable text analyzer (for BM25) |
| `analyzerParams` | object | No | Analyzer configuration |
| `enableMatch` | boolean | No | Enable text matching |

#### Returns

`OperationResult` with the following properties:

| Property | Type | Description |
|----------|------|-------------|
| `success` | boolean | Whether operation succeeded |
| `response_time_ms` | number | Operation duration in milliseconds |
| `result` | any | Operation-specific result |
| `error` | string | Error message if failed |

#### Example

```javascript
const schema = {
  name: 'products',
  fields: [
    { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
    { name: 'title', dataType: 'VarChar', maxLength: 200 },
    { name: 'price', dataType: 'Float' },
    { name: 'embedding', dataType: 'FloatVector', dimension: 128 }
  ]
};

const result = client.createCollection(schema);
check(result, {
  'collection created': (r) => r.success === true,
  'fast creation': (r) => r.response_time_ms < 1000,
});
```

---

### createCollectionFromJSON()

Creates a collection from a JSON string schema definition.

#### Signature

```javascript
createCollectionFromJSON(schemaJSON: string): OperationResult
```

#### Example

```javascript
const schemaJSON = JSON.stringify({
  name: 'products',
  fields: [
    { name: 'id', dataType: 'Int64', isPrimaryKey: true }
  ]
});

const result = client.createCollectionFromJSON(schemaJSON);
```

---

### dropCollection()

Drops (deletes) a collection.

#### Signature

```javascript
dropCollection(collectionName?: string): OperationResult
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `collectionName` | string | Conditional | Collection name (required for standard client, optional for collection-bound) |

#### Example

```javascript
// Standard client
const result = client.dropCollection('products');

// Collection-bound client
const boundClient = milvus.clientWithCollection('localhost:19530', 'products');
const result = boundClient.dropCollection(); // Uses 'products'
```

---

### hasCollection()

Checks if a collection exists.

#### Signature

```javascript
hasCollection(collectionName?: string): OperationResult
```

#### Returns

`OperationResult` where `result` contains a boolean indicating existence.

#### Example

```javascript
const result = client.hasCollection('products');
if (result.success && result.result) {
  console.log('Collection exists');
}
```

---

### loadCollection()

Loads a collection into memory for search operations.

#### Signature

```javascript
loadCollection(collectionName?: string): OperationResult
```

#### Example

```javascript
const result = client.loadCollection('products');
check(result, {
  'collection loaded': (r) => r.success === true,
});
```

---

### releaseCollection()

Releases a collection from memory.

#### Signature

```javascript
releaseCollection(collectionName?: string): OperationResult
```

---

## Write Operations

### insert()

Inserts data into a collection.

#### Signature

```javascript
insert(data: ColumnData, collectionName?: string): OperationResult
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `data` | ColumnData | Yes | Column-based data to insert |
| `collectionName` | string | Conditional | Collection name |

#### ColumnData Format

Data should be organized by columns (not rows):

```javascript
{
  field1: [value1, value2, value3],
  field2: [value1, value2, value3],
  vector: [[0.1, 0.2], [0.3, 0.4], [0.5, 0.6]]
}
```

#### Returns

`OperationResult` where `result` contains:
- `insert_count`: Number of entities inserted
- `ids`: Array of inserted IDs

#### Example

```javascript
const insertResult = client.insert({
  title: ['Product A', 'Product B', 'Product C'],
  price: [19.99, 29.99, 39.99],
  embedding: [
    [0.1, 0.2, 0.3, ...], // 128-dim vector
    [0.4, 0.5, 0.6, ...],
    [0.7, 0.8, 0.9, ...]
  ]
}, 'products');

check(insertResult, {
  'insert successful': (r) => r.success === true,
  'inserted 3 items': (r) => r.result.insert_count === 3,
});
```

---

### upsert()

Inserts or updates data in a collection.

#### Signature

```javascript
upsert(data: ColumnData, collectionName?: string): OperationResult
```

#### Example

```javascript
const upsertResult = client.upsert({
  id: [1, 2, 3],
  title: ['Updated A', 'Updated B', 'Updated C'],
  price: [18.99, 28.99, 38.99]
}, 'products');
```

---

### delete()

Deletes entities matching a filter expression.

#### Signature

```javascript
delete(filter: string, collectionName?: string): OperationResult
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `filter` | string | Yes | Boolean filter expression |
| `collectionName` | string | Conditional | Collection name |

#### Returns

`OperationResult` where `result` contains:
- `delete_count`: Number of entities deleted

#### Example

```javascript
const deleteResult = client.delete('price < 20', 'products');
console.log(`Deleted ${deleteResult.result.delete_count} items`);

// Complex filters
client.delete('price > 100 && title like "Premium%"', 'products');
client.delete('id in [1, 2, 3]', 'products');
```

---

## Read Operations

### search()

Performs vector similarity search.

#### Signature

```javascript
search(
  vectors: number[][] | number[],
  topK: number,
  params: SearchParams,
  collectionName?: string
): OperationResult
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `vectors` | number[][] or number[] | Yes | Query vector(s) |
| `topK` | number | Yes | Number of results to return |
| `params` | SearchParams | Yes | Search parameters |
| `collectionName` | string | Conditional | Collection name |

#### SearchParams

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `vectorField` | string | Yes | Name of the vector field to search |
| `metricType` | string | No | Distance metric (L2, IP, COSINE) |
| `outputFields` | string[] | No | Fields to return in results |
| `expr` | string | No | Filter expression |
| `params` | object | No | Index-specific search params |

#### Returns

`OperationResult` where:
- `result`: Array of search results
- `recall`: Recall metric (for quality assessment)
- `empty`: Boolean indicating if results are empty

#### Example

```javascript
const searchResult = client.search(
  [[0.1, 0.2, 0.3, ...]], // Single query vector
  10, // Top 10 results
  {
    vectorField: 'embedding',
    metricType: 'L2',
    outputFields: ['title', 'price'],
    expr: 'price > 20'
  },
  'products'
);

check(searchResult, {
  'search successful': (r) => r.success === true,
  'high recall': (r) => r.recall >= 0.95,
  'has results': (r) => !r.empty,
  'fast search': (r) => r.response_time_ms < 100,
});

// Process results
searchResult.result.forEach(hit => {
  console.log(`Title: ${hit.title}, Price: ${hit.price}, Score: ${hit.score}`);
});
```

---

### query()

Performs scalar query without vectors (filter-based retrieval).

#### Signature

```javascript
query(
  filter: string,
  outputFields: string[],
  collectionName?: string
): OperationResult
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `filter` | string | Yes | Boolean filter expression |
| `outputFields` | string[] | Yes | Fields to return |
| `collectionName` | string | Conditional | Collection name |

#### Example

```javascript
const queryResult = client.query(
  'price > 100 && price < 200',
  ['id', 'title', 'price'],
  'products'
);

if (queryResult.success && !queryResult.empty) {
  console.log(`Found ${queryResult.result.length} products`);
  queryResult.result.forEach(item => {
    console.log(`${item.title}: $${item.price}`);
  });
}
```

---

### hybridSearch()

Performs multi-vector hybrid search with reranking.

#### Signature

```javascript
hybridSearch(
  requests: SearchRequest[],
  reranker: Reranker,
  limit: number,
  outputFields: string[],
  collectionName?: string
): OperationResult
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `requests` | SearchRequest[] | Yes | Array of search requests |
| `reranker` | Reranker | Yes | Reranking strategy |
| `limit` | number | Yes | Final number of results after reranking |
| `outputFields` | string[] | Yes | Fields to return |
| `collectionName` | string | Conditional | Collection name |

#### SearchRequest

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `vectors` | number[][] | Yes | Query vectors for this search |
| `vectorField` | string | Yes | Vector field name |
| `limit` | number | Yes | Results per search |
| `params` | object | No | Search params (metricType, expr, etc.) |

#### Reranker

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `type` | string | Yes | Reranker type: "rrf" or "weighted" |
| `params` | object | No | Reranker-specific params |

For RRF: `{ k: 60 }` (default k value)
For Weighted: `{ weights: [0.7, 0.3] }` (weights for each search)

#### Example

```javascript
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
    type: 'rrf',
    params: { k: 60 }
  },
  5, // Final top 5 results
  ['title', 'price']
);

check(hybridResult, {
  'hybrid search successful': (r) => r.success === true,
  'good recall': (r) => r.recall >= 0.9,
});
```

---

## Index Operations

### createIndex()

Creates an index on a vector field for faster searches.

#### Signature

```javascript
createIndex(
  fieldName: string,
  indexParams: IndexParams,
  collectionName?: string
): OperationResult
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `fieldName` | string | Yes | Field to index |
| `indexParams` | IndexParams | Yes | Index configuration |
| `collectionName` | string | Conditional | Collection name |

#### IndexParams

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `indexType` | string | Yes | Index type (FLAT, IVF_FLAT, HNSW, etc.) |
| `metricType` | string | Yes | Distance metric (L2, IP, COSINE) |
| `params` | object | No | Index-specific parameters |

Common index params:
- IVF_FLAT: `{ nlist: 128 }`
- HNSW: `{ M: 16, efConstruction: 200 }`

#### Example

```javascript
const indexResult = client.createIndex(
  'embedding',
  {
    indexType: 'HNSW',
    metricType: 'L2',
    params: { M: 16, efConstruction: 200 }
  },
  'products'
);

check(indexResult, {
  'index created': (r) => r.success === true,
});
```

---

## Advanced Features

### BM25 Full-Text Search

Create collections with BM25 function for automatic sparse vector generation:

```javascript
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

client.createCollection(schema);
client.loadCollection('documents');

// Insert text data (sparse vectors generated automatically)
client.upsert({
  id: [1, 2, 3],
  text: ['Document one', 'Document two', 'Document three']
}, 'documents');
```

---

## Error Handling

All methods return `OperationResult` instead of throwing errors. Always check `success` and `error`:

```javascript
const result = client.createCollection(schema);

if (!result.success) {
  console.error(`Operation failed: ${result.error}`);
  console.error(`Response time: ${result.response_time_ms}ms`);
  return;
}

console.log('Success!');
```

---

## Performance Tips

1. **Use Collection-Bound Clients** for cleaner code when working with single collections
2. **Load Collections** before searching - unloaded collections cannot be searched
3. **Create Indexes** after inserting data for better search performance
4. **Batch Inserts** - insert multiple entities at once instead of one-by-one
5. **Monitor Response Times** - use `response_time_ms` to identify slow operations
6. **Check Recall** - use `recall` metric to verify search quality

---

## Type Mapping

| Milvus Type | JavaScript Type | Example |
|-------------|----------------|---------|
| Int64 | number | 12345 |
| Float | number | 19.99 |
| Double | number | 3.14159 |
| VarChar | string | "Product Name" |
| Bool | boolean | true |
| FloatVector | number[] | [0.1, 0.2, 0.3] |
| SparseFloatVector | object | {0: 0.5, 12: 0.8} |

---

## Client Methods Summary

| Method | Purpose | Returns |
|--------|---------|---------|
| `client()` | Create standard client | Client |
| `clientWithCollection()` | Create collection-bound client | Client |
| `createCollection()` | Create new collection | OperationResult |
| `dropCollection()` | Delete collection | OperationResult |
| `hasCollection()` | Check existence | OperationResult |
| `loadCollection()` | Load to memory | OperationResult |
| `releaseCollection()` | Unload from memory | OperationResult |
| `insert()` | Insert data | OperationResult |
| `upsert()` | Insert or update | OperationResult |
| `delete()` | Delete by filter | OperationResult |
| `search()` | Vector search | OperationResult |
| `query()` | Scalar query | OperationResult |
| `hybridSearch()` | Multi-vector search | OperationResult |
| `createIndex()` | Create index | OperationResult |
| `close()` | Close connection | OperationResult |

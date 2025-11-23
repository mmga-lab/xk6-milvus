# xk6-milvus Examples

This directory contains progressive examples demonstrating xk6-milvus features.

## Examples Overview

| Example | Description | Difficulty | Duration |
|---------|-------------|------------|----------|
| `basic-operations.js` | CRUD operations walkthrough | Beginner | ~1 min |
| `collection-management.js` | Collection lifecycle | Beginner | ~1 min |
| `vector-search.js` | Vector similarity search patterns | Intermediate | 10s |
| `hybrid-search.js` | Multi-vector hybrid search | Advanced | 10s |
| `full-text-search.js` | BM25 full-text search | Advanced | 10s |

## Prerequisites

1. **Milvus Server** - Running Milvus instance
2. **k6 with xk6-milvus** - Build the extension or download binary

### Start Milvus (using Docker)

```bash
# Using standalone Docker
docker run -d --name milvus \
  -p 19530:19530 \
  -p 9091:9091 \
  milvusdb/milvus:latest

# Or using docker-compose
cd deployment
docker-compose up -d
```

### Build k6 with Extension

```bash
xk6 build --with github.com/mmga-lab/xk6-milvus=.
```

## Running Examples

### Basic Operations (Recommended Start)

Learn fundamental CRUD operations:

```bash
./k6 run examples/basic-operations.js
```

**What you'll learn:**

- Creating collections
- Inserting data
- Vector search
- Querying with filters
- Deleting data

### Collection Management

Understand collection lifecycle:

```bash
./k6 run examples/collection-management.js
```

**What you'll learn:**

- Different schema types
- Collection existence checks
- Loading/releasing collections
- Creating from JSON

### Vector Search

Master vector similarity search:

```bash
# Set Milvus host if not default
export MILVUS_HOST=localhost:19530

./k6 run examples/vector-search.js
```

**What you'll learn:**

- Basic vector search
- Search with filters
- Batch search
- Performance monitoring
- Recall metrics

### Hybrid Search

Multi-vector search with reranking:

```bash
./k6 run examples/hybrid-search.js
```

**What you'll learn:**

- Dense + sparse vectors
- RRF reranking
- Weighted reranking
- Multi-modal search

### Full-Text Search

BM25-based text search:

```bash
./k6 run examples/full-text-search.js
```

**What you'll learn:**

- BM25 functions
- Automatic sparse vector generation
- Text analyzers
- Document insertion and search

## Customization

### Environment Variables

```bash
# Set Milvus server address
export MILVUS_HOST=your-milvus-host:19530

# Set authentication token (if needed)
export MILVUS_TOKEN=username:password
```

### k6 Options

Modify `options` in each script:

```javascript
export const options = {
    vus: 10,           // Number of virtual users
    duration: '30s',   // Test duration
    iterations: 100,   // Total iterations (alternative to duration)
};
```

## Example Output

```text
✓ collection created
✓ index created
✓ collection loaded
✓ insert successful
✓ search successful
✓ has results

Search took 45.2ms, recall: 0.98
Top 3 results:
  1. Apple - $1.99 (distance: 0.23)
  2. Orange - $2.49 (distance: 0.45)
  3. Banana - $0.99 (distance: 0.67)
```

## Load Testing

To use these examples for load testing, modify the options:

```javascript
export const options = {
    stages: [
        { duration: '30s', target: 10 },   // Ramp up to 10 VUs
        { duration: '1m', target: 10 },    // Stay at 10 VUs
        { duration: '30s', target: 50 },   // Ramp up to 50 VUs
        { duration: '1m', target: 50 },    // Stay at 50 VUs
        { duration: '30s', target: 0 },    // Ramp down to 0
    ],
    thresholds: {
        'checks': ['rate>0.99'],           // 99% success rate
        'http_req_duration': ['p(95)<200'], // 95% under 200ms
    },
};
```

## Data Generators

The `data/` directory contains utilities for generating test data:

```bash
cd examples/data
go run generate_vectors.go
```

## Troubleshooting

### Connection Refused

```bash
# Check Milvus is running
docker ps | grep milvus

# Check connection
telnet localhost 19530
```

### Collection Already Exists

Examples include cleanup (setup/teardown), but if needed:

```javascript
// Manually drop collection
const client = milvus.client('localhost:19530');
client.dropCollection('collection_name');
client.close();
```

### Performance Issues

- Reduce `vus` (virtual users)
- Increase `duration` for longer warm-up
- Check Milvus server resources
- Use appropriate index types

## Learn More

- [API Documentation](../docs/API.md)
- [Contributing Guide](../CONTRIBUTING.md)
- [Milvus Documentation](https://milvus.io/docs)
- [k6 Documentation](https://k6.io/docs/)

## Contributing

Have a great example to share? See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines!

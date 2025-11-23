# Test Data Generators

This directory contains utilities for generating test data for xk6-milvus examples.

## Available Generators

### Vector Data Generator (Go)

Generate sample vector data with customizable dimensions and count.

**Usage:**

```bash
# Generate default data (1000 vectors, 128 dimensions)
go run generate_vectors.go

# Output: sample_data.json
```

**Customize:**

Edit `generate_vectors.go`:

```go
count := 1000      // Number of samples
dimension := 128   // Vector dimension
```

**Output Format:**

```json
[
  {
    "id": 1,
    "title": "Laptop Computer",
    "category": "Electronics",
    "price": 299.99,
    "vector": [0.123, 0.456, ...]
  },
  ...
]
```

## Using Generated Data

### In k6 Scripts

```javascript
import { open } from 'k6/data';

const data = JSON.parse(open('../examples/data/sample_data.json'));

export default function() {
    // Use the data
    const randomItem = data[Math.floor(Math.random() * data.length)];
    console.log(randomItem.title, randomItem.vector);
}
```

### Direct Insertion

```javascript
import milvus from 'k6/x/milvus';
const data = JSON.parse(open('../examples/data/sample_data.json'));

export function setup() {
    const client = milvus.client('localhost:19530');

    // Transform data for insertion
    const insertData = {
        id: data.map(d => d.id),
        title: data.map(d => d.title),
        category: data.map(d => d.category),
        price: data.map(d => d.price),
        vector: data.map(d => d.vector)
    };

    client.insert(insertData, 'my_collection');
    client.close();
}
```

## Creating Custom Generators

### Python Example

```python
import json
import random

def generate_vector(dim):
    return [random.random() for _ in range(dim)]

data = [
    {
        "id": i,
        "vector": generate_vector(128)
    }
    for i in range(1000)
]

with open('custom_data.json', 'w') as f:
    json.dump(data, f, indent=2)
```

### JavaScript Example

```javascript
const fs = require('fs');

function generateVector(dim) {
    return Array.from({length: dim}, () => Math.random());
}

const data = Array.from({length: 1000}, (_, i) => ({
    id: i + 1,
    vector: generateVector(128)
}));

fs.writeFileSync('custom_data.json', JSON.stringify(data, null, 2));
```

## Large Dataset Generation

For load testing with large datasets:

```go
// Modify generate_vectors.go
count := 100000     // 100K vectors
dimension := 768    // Larger dimension

// Or generate in batches
for batch := 0; batch < 10; batch++ {
    data := generateSampleData(10000, 768)
    filename := fmt.Sprintf("batch_%d.json", batch)
    // Save to file
}
```

## Performance Considerations

- **Memory**: Large datasets may require streaming
- **File Size**: 10K x 128-dim vectors ≈ 5-10 MB
- **Generation Time**: ~1-5 seconds for 10K vectors

## Clean Up

```bash
# Remove generated files
rm -f sample_data.json
rm -f batch_*.json
```

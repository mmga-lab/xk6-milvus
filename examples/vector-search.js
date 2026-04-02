// Vector Similarity Search Example
// This example demonstrates various vector search patterns:
// - Basic vector search
// - Search with filtering
// - Batch search
// - Different distance metrics
// - Monitoring recall and performance

import milvus from 'k6/x/milvus';
import { check } from 'k6';

export const options = {
    vus: 5,
    duration: '10s',
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
const COLLECTION_NAME = 'search_demo';
const VECTOR_DIM = 128;

function generateRandomVector(dim) {
    return Array.from({ length: dim }, () => Math.random());
}

export function setup() {
    const client = milvus.client(MILVUS_HOST);

    // Drop if exists
    const hasResult = client.hasCollection(COLLECTION_NAME);
    if (hasResult.success && hasResult.result.exists) {
        client.dropCollection(COLLECTION_NAME);
    }

    // Create collection
    const schema = {
        name: COLLECTION_NAME,
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'category', dataType: 'VarChar', maxLength: 50 },
            { name: 'price', dataType: 'Float' },
            { name: 'rating', dataType: 'Float' },
            { name: 'embedding', dataType: 'FloatVector', dimension: VECTOR_DIM }
        ]
    };

    client.createCollection(schema);

    // Create HNSW index for fast search
    client.createIndex('embedding', {
        indexType: 'HNSW',
        metricType: 'L2',
        params: { M: 16, efConstruction: 200 }
    }, COLLECTION_NAME);

    client.loadCollection(COLLECTION_NAME);

    // Insert sample data
    const categories = ['Electronics', 'Clothing', 'Books', 'Food', 'Toys'];
    const batchSize = 1000;

    for (let batch = 0; batch < 10; batch++) {
        const data = {
            category: [],
            price: [],
            rating: [],
            embedding: []
        };

        for (let i = 0; i < batchSize; i++) {
            data.category.push(categories[Math.floor(Math.random() * categories.length)]);
            data.price.push(Math.random() * 100);
            data.rating.push(Math.random() * 5);
            data.embedding.push(generateRandomVector(VECTOR_DIM));
        }

        client.insert(data, COLLECTION_NAME);
    }

    console.log('Setup complete: 10,000 items inserted');
    client.close();

    return { ready: true };
}

export default function() {
    // Use getClient for VU-level connection reuse (one gRPC connection per VU)
    const client = milvus.getClient(MILVUS_HOST, COLLECTION_NAME);

    // Example 1: Basic search
    const queryVector = generateRandomVector(VECTOR_DIM);
    let searchResult = client.search(
        [queryVector],
        10,  // Top 10
        {
            vectorField: 'embedding',
            outputFields: ['category', 'price', 'rating'],
            metricType: 'L2'
        }
    );

    check(searchResult, {
        'basic search successful': (r) => r.success === true,
        'has results': (r) => !r.empty,
        'fast search': (r) => r.response_time_ms < 100,
    });

    // Example 2: Search with price filter
    searchResult = client.search(
        [queryVector],
        5,
        {
            vectorField: 'embedding',
            outputFields: ['category', 'price', 'rating'],
            expr: 'price > 50 && price < 80',
            metricType: 'L2'
        }
    );

    check(searchResult, {
        'filtered search successful': (r) => r.success === true,
    });

    // Example 3: Search with category filter
    searchResult = client.search(
        [queryVector],
        10,
        {
            vectorField: 'embedding',
            outputFields: ['category', 'price', 'rating'],
            expr: 'category == "Electronics"',
            metricType: 'L2'
        }
    );

    check(searchResult, {
        'category filtered search successful': (r) => r.success === true,
    });

    // Example 4: Search highly-rated items
    searchResult = client.search(
        [queryVector],
        10,
        {
            vectorField: 'embedding',
            outputFields: ['category', 'price', 'rating'],
            expr: 'rating > 4.0',
            metricType: 'L2',
            params: { ef: 64 }  // HNSW search parameter
        }
    );

    check(searchResult, {
        'rating filtered search successful': (r) => r.success === true,
        'good performance': (r) => r.response_time_ms < 150,
    });

    // Example 5: Batch search (multiple query vectors)
    const batchVectors = [
        generateRandomVector(VECTOR_DIM),
        generateRandomVector(VECTOR_DIM),
        generateRandomVector(VECTOR_DIM)
    ];

    searchResult = client.search(
        batchVectors,
        5,
        {
            vectorField: 'embedding',
            outputFields: ['category', 'price'],
            metricType: 'L2'
        }
    );

    check(searchResult, {
        'batch search successful': (r) => r.success === true,
        'batch search fast': (r) => r.response_time_ms < 200,
    });

    // Do NOT close - connection is reused across iterations
}

export function teardown(data) {
    if (data.ready) {
        const client = milvus.client(MILVUS_HOST);
        client.dropCollection(COLLECTION_NAME);
        client.close();
        console.log('Teardown complete');
    }
}

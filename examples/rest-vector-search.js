// REST API Vector Search Example
// Demonstrates various vector search patterns via Milvus RESTful v2 API.
// Uses milvus.restClient() for REST and milvus.client() for gRPC.

import milvus from 'k6/x/milvus';
import { check, sleep } from 'k6';

export const options = {
    vus: 3,
    duration: '10s',
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
const COLLECTION_NAME = 'rest_search_demo';
const VECTOR_DIM = 128;
const NUM_ENTITIES = 1000;

function generateRandomVector(dim) {
    return Array.from({ length: dim }, () => Math.random());
}

export function setup() {
    const client = milvus.restClient(MILVUS_HOST);

    // Drop if exists
    const hasResult = client.hasCollection(COLLECTION_NAME);
    if (hasResult.success && hasResult.result.exists) {
        client.dropCollection(COLLECTION_NAME);
    }

    // Create collection
    const createResult = client.createCollection({
        name: COLLECTION_NAME,
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'category', dataType: 'VarChar', maxLength: 50 },
            { name: 'price', dataType: 'Float' },
            { name: 'embedding', dataType: 'FloatVector', dimension: VECTOR_DIM },
        ],
    });
    check(createResult, { 'collection created': (r) => r.success });

    // Create index
    client.createIndex('embedding', {
        indexType: 'HNSW',
        metricType: 'L2',
        params: { M: 16, efConstruction: 200 },
    }, COLLECTION_NAME);

    // Load collection
    client.loadCollection(COLLECTION_NAME);

    // Insert data in batches
    const categories = ['electronics', 'clothing', 'food', 'books', 'toys'];
    const batchSize = 100;

    for (let batch = 0; batch < NUM_ENTITIES / batchSize; batch++) {
        const data = {
            category: [],
            price: [],
            embedding: [],
        };
        for (let i = 0; i < batchSize; i++) {
            data.category.push(categories[Math.floor(Math.random() * categories.length)]);
            data.price.push(Math.round(Math.random() * 500 * 100) / 100);
            data.embedding.push(generateRandomVector(VECTOR_DIM));
        }
        const insertRes = client.insert(data, COLLECTION_NAME);
        check(insertRes, {
            [`batch ${batch + 1} inserted`]: (r) => r.success,
        });
    }

    sleep(2);
    client.close();
    return { ready: true };
}

export default function () {
    const client = milvus.restClientWithCollection(MILVUS_HOST, COLLECTION_NAME);

    // Basic vector search
    const basicSearch = client.search(
        [generateRandomVector(VECTOR_DIM)],
        10,
        {
            vectorField: 'embedding',
            outputFields: ['category', 'price'],
            metricType: 'L2',
        },
    );
    check(basicSearch, {
        'basic search ok': (r) => r.success,
        'basic search has results': (r) => !r.empty,
    });

    // Search with filter
    const filteredSearch = client.search(
        [generateRandomVector(VECTOR_DIM)],
        5,
        {
            vectorField: 'embedding',
            outputFields: ['category', 'price'],
            metricType: 'L2',
            expr: 'price > 100 && category == "electronics"',
        },
    );
    check(filteredSearch, {
        'filtered search ok': (r) => r.success,
    });

    // Multi-vector search (batch)
    const batchVectors = [
        generateRandomVector(VECTOR_DIM),
        generateRandomVector(VECTOR_DIM),
        generateRandomVector(VECTOR_DIM),
    ];
    const batchSearch = client.search(
        batchVectors,
        5,
        {
            vectorField: 'embedding',
            outputFields: ['category', 'price'],
            metricType: 'L2',
        },
    );
    check(batchSearch, {
        'batch search ok': (r) => r.success,
    });

    // Query with filter (no vectors)
    const queryResult = client.query(
        'price > 400',
        ['id', 'category', 'price'],
    );
    check(queryResult, {
        'query ok': (r) => r.success,
    });

    client.close();
}

export function teardown() {
    const client = milvus.restClient(MILVUS_HOST);
    client.dropCollection(COLLECTION_NAME);
    client.close();
}

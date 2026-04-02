// Hybrid Search Example (Multi-Vector Search)
// This example demonstrates hybrid search with multiple vector fields:
// - Dense + Sparse vector search
// - RRF (Reciprocal Rank Fusion) reranking
// - Weighted reranking
// - Multi-modal search scenarios

import milvus from 'k6/x/milvus';
import { check } from 'k6';

export const options = {
    vus: 3,
    duration: '10s',
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
const COLLECTION_NAME = 'hybrid_search_demo';
const DENSE_DIM = 128;
const SPARSE_DIM = 1000;

function generateDenseVector(dim) {
    return Array.from({ length: dim }, () => Math.random());
}

function generateSparseVector(dim, sparsity = 0.01) {
    // Generate sparse vector with ~1% non-zero values
    const sparse = {};
    const numNonZero = Math.floor(dim * sparsity);

    for (let i = 0; i < numNonZero; i++) {
        const index = Math.floor(Math.random() * dim);
        sparse[index] = Math.random();
    }

    return sparse;
}

export function setup() {
    const client = milvus.client(MILVUS_HOST);

    // Drop if exists
    const hasResult = client.hasCollection(COLLECTION_NAME);
    if (hasResult.success && hasResult.result.exists) {
        client.dropCollection(COLLECTION_NAME);
    }

    // Create collection with multiple vector fields
    const schema = {
        name: COLLECTION_NAME,
        numShards: 2,
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'title', dataType: 'VarChar', maxLength: 200 },
            { name: 'price', dataType: 'Float' },
            // Dense vector (e.g., image embedding)
            { name: 'dense_vector', dataType: 'FloatVector', dimension: DENSE_DIM },
            // Sparse vector (e.g., text features)
            { name: 'sparse_vector', dataType: 'SparseFloatVector' }
        ]
    };

    client.createCollection(schema);

    // Create indexes for both vector fields
    client.createIndex('dense_vector', {
        indexType: 'HNSW',
        metricType: 'L2',
        params: { M: 16, efConstruction: 200 }
    }, COLLECTION_NAME);

    client.createIndex('sparse_vector', {
        indexType: 'SPARSE_INVERTED_INDEX',
        metricType: 'IP'
    }, COLLECTION_NAME);

    client.loadCollection(COLLECTION_NAME);

    // Insert sample data
    const titles = [
        'Modern Laptop', 'Wireless Mouse', 'Mechanical Keyboard',
        'USB-C Cable', 'Monitor Stand', 'Webcam HD',
        'Gaming Headset', 'External SSD', 'Laptop Bag'
    ];

    const batchSize = 100;
    for (let batch = 0; batch < 5; batch++) {
        const data = {
            title: [],
            price: [],
            dense_vector: [],
            sparse_vector: []
        };

        for (let i = 0; i < batchSize; i++) {
            data.title.push(titles[Math.floor(Math.random() * titles.length)]);
            data.price.push(Math.random() * 200);
            data.dense_vector.push(generateDenseVector(DENSE_DIM));
            data.sparse_vector.push(generateSparseVector(SPARSE_DIM));
        }

        client.insert(data, COLLECTION_NAME);
    }

    console.log('Setup complete: 500 items with multi-vector embeddings');
    client.close();

    return { ready: true };
}

export default function() {
    // VU-level connection reuse - one gRPC connection per VU
    const client = milvus.getClient(MILVUS_HOST, COLLECTION_NAME);

    // Generate query vectors
    const queryDense = generateDenseVector(DENSE_DIM);
    const querySparse = generateSparseVector(SPARSE_DIM);

    // Example 1: Hybrid search with RRF (Reciprocal Rank Fusion)
    let hybridResult = client.hybridSearch(
        [
            {
                vectors: [queryDense],
                vectorField: 'dense_vector',
                limit: 20,
                params: { metricType: 'L2', ef: 64 }
            },
            {
                vectors: [querySparse],
                vectorField: 'sparse_vector',
                limit: 20,
                params: { metricType: 'IP' }
            }
        ],
        {
            type: 'rrf',
            params: { k: 60 }  // RRF k parameter
        },
        10,  // Final top 10 results
        ['title', 'price']
    );

    check(hybridResult, {
        'RRF hybrid search successful': (r) => r.success === true,
        'has results': (r) => !r.empty,
        'fast response': (r) => r.response_time_ms < 200,
    });

    // Example 2: Hybrid search with Weighted reranking
    hybridResult = client.hybridSearch(
        [
            {
                vectors: [queryDense],
                vectorField: 'dense_vector',
                limit: 20,
                params: { metricType: 'L2' }
            },
            {
                vectors: [querySparse],
                vectorField: 'sparse_vector',
                limit: 20,
                params: { metricType: 'IP' }
            }
        ],
        {
            type: 'weighted',
            params: { weights: [0.7, 0.3] }  // 70% dense, 30% sparse
        },
        10,
        ['title', 'price']
    );

    check(hybridResult, {
        'weighted hybrid search successful': (r) => r.success === true,
    });

    // Example 3: Hybrid search with price filter
    hybridResult = client.hybridSearch(
        [
            {
                vectors: [queryDense],
                vectorField: 'dense_vector',
                limit: 20,
                params: { metricType: 'L2', expr: 'price > 50' }
            },
            {
                vectors: [querySparse],
                vectorField: 'sparse_vector',
                limit: 20,
                params: { metricType: 'IP', expr: 'price > 50' }
            }
        ],
        {
            type: 'rrf',
            params: { k: 60 }
        },
        5,
        ['title', 'price']
    );

    check(hybridResult, {
        'filtered hybrid search successful': (r) => r.success === true,
    });

    // Example 4: Different weight combinations
    const weightCombinations = [
        [0.5, 0.5],  // Equal weights
        [0.8, 0.2],  // Dense-heavy
        [0.3, 0.7]   // Sparse-heavy
    ];

    weightCombinations.forEach((weights, index) => {
        const result = client.hybridSearch(
            [
                {
                    vectors: [queryDense],
                    vectorField: 'dense_vector',
                    limit: 15,
                    params: { metricType: 'L2' }
                },
                {
                    vectors: [querySparse],
                    vectorField: 'sparse_vector',
                    limit: 15,
                    params: { metricType: 'IP' }
                }
            ],
            {
                type: 'weighted',
                params: { weights: weights }
            },
            5,
            ['title', 'price']
        );

        check(result, {
            [`weight combination ${index + 1} successful`]: (r) => r.success === true,
        });
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

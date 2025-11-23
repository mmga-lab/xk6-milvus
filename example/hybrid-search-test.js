import milvus from 'k6/x/milvus';
import { check } from 'k6';

// This example demonstrates HybridSearch with multiple vector fields
// Hybrid search combines results from multiple vector searches using reranking

export const options = {
    vus: 5,
    duration: '10s',
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
const COLLECTION_NAME = 'hybrid_search_collection';
const DENSE_DIM = 128;
const SPARSE_DIM = 64;

function generateRandomVectors(count, dim) {
    const vectors = [];
    for (let i = 0; i < count; i++) {
        const vector = [];
        for (let j = 0; j < dim; j++) {
            vector.push(Math.random());
        }
        vectors.push(vector);
    }
    return vectors;
}

export function setup() {
    const client = milvus.client(MILVUS_HOST);

    // Clean up
    const hasResult = client.hasCollection(COLLECTION_NAME);
    if (hasResult.success && hasResult.result.exists) {
        client.dropCollection(COLLECTION_NAME);
    }

    // Create collection with multiple vector fields
    const schema = {
        name: COLLECTION_NAME,
        description: "Collection with multiple vector fields for hybrid search",
        fields: [
            {
                name: "id",
                dataType: "Int64",
                isPrimaryKey: true,
                isAutoID: true
            },
            {
                name: "title",
                dataType: "VarChar",
                maxLength: 200
            },
            {
                name: "dense_vector",
                dataType: "FloatVector",
                dimension: DENSE_DIM
            },
            {
                name: "sparse_vector",
                dataType: "FloatVector",
                dimension: SPARSE_DIM
            }
        ]
    };

    const createResult = client.createCollection(schema);
    check(createResult, {
        'collection created': (r) => r.success === true,
    });

    // Create indexes for both vector fields
    client.createIndex("dense_vector", {
        indexType: "FLAT",
        metricType: "L2"
    }, COLLECTION_NAME);

    client.createIndex("sparse_vector", {
        indexType: "FLAT",
        metricType: "L2"
    }, COLLECTION_NAME);

    // Load collection
    client.loadCollection(COLLECTION_NAME);

    // Insert test data
    const denseVectors = generateRandomVectors(100, DENSE_DIM);
    const sparseVectors = generateRandomVectors(100, SPARSE_DIM);
    const titles = [];
    for (let i = 0; i < 100; i++) {
        titles.push(`Document ${i}`);
    }

    const insertResult = client.insert({
        title: titles,
        dense_vector: denseVectors,
        sparse_vector: sparseVectors
    }, COLLECTION_NAME);

    check(insertResult, {
        'data inserted': (r) => r.success === true,
        'inserted 100 records': (r) => r.result.insert_count === 100,
    });

    client.close();
    return { ready: true };
}

export default function() {
    const client = milvus.clientWithCollection(MILVUS_HOST, COLLECTION_NAME);

    // Generate query vectors
    const denseQueryVectors = generateRandomVectors(1, DENSE_DIM);
    const sparseQueryVectors = generateRandomVectors(1, SPARSE_DIM);

    // Example 1: HybridSearch with RRF (Reciprocal Rank Fusion) reranker
    const rrfResult = client.hybridSearch(
        [
            {
                vectors: denseQueryVectors,
                vectorField: 'dense_vector',
                limit: 10,
                params: { metricType: 'L2' }
            },
            {
                vectors: sparseQueryVectors,
                vectorField: 'sparse_vector',
                limit: 10,
                params: { metricType: 'L2' }
            }
        ],
        {
            type: 'rrf',
            params: { k: 60 }  // RRF k parameter
        },
        5,  // final limit after reranking
        ['title']  // output fields
    );

    check(rrfResult, {
        'RRF hybrid search successful': (r) => r.success === true,
        'RRF not empty': (r) => r.empty === false,
        'RRF fast': (r) => r.response_time_ms < 500,
    });

    if (rrfResult.success) {
        console.log(`RRF HybridSearch: ${rrfResult.result.length} results, RT: ${rrfResult.response_time_ms.toFixed(2)}ms`);
    }

    // Example 2: HybridSearch with Weighted reranker
    const weightedResult = client.hybridSearch(
        [
            {
                vectors: denseQueryVectors,
                vectorField: 'dense_vector',
                limit: 10,
            },
            {
                vectors: sparseQueryVectors,
                vectorField: 'sparse_vector',
                limit: 10,
            }
        ],
        {
            type: 'weighted',
            params: {
                weights: [0.7, 0.3]  // 70% weight to dense, 30% to sparse
            }
        },
        5,
        ['title']
    );

    check(weightedResult, {
        'Weighted hybrid search successful': (r) => r.success === true,
        'Weighted not empty': (r) => r.empty === false,
    });

    if (weightedResult.success) {
        console.log(`Weighted HybridSearch: ${weightedResult.result.length} results, RT: ${weightedResult.response_time_ms.toFixed(2)}ms`);
    }

    // Example 3: HybridSearch with filter
    const filteredResult = client.hybridSearch(
        [
            {
                vectors: denseQueryVectors,
                vectorField: 'dense_vector',
                limit: 10,
                params: {
                    expr: 'id > 50',  // Filter results
                    metricType: 'L2'
                }
            },
            {
                vectors: sparseQueryVectors,
                vectorField: 'sparse_vector',
                limit: 10,
                params: {
                    expr: 'id > 50',
                    metricType: 'L2'
                }
            }
        ],
        {
            type: 'rrf',
            params: { k: 60 }
        },
        5,
        ['title']
    );

    check(filteredResult, {
        'Filtered hybrid search successful': (r) => r.success === true,
    });

    if (filteredResult.success) {
        console.log(`Filtered HybridSearch: ${filteredResult.result.length} results`);
    }

    client.close();
}

export function teardown() {
    const client = milvus.client(MILVUS_HOST);
    const dropResult = client.dropCollection(COLLECTION_NAME);
    check(dropResult, {
        'collection dropped': (r) => r.success === true,
    });
    client.close();
}

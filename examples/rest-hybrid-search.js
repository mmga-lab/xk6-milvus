// REST API Hybrid Search Example
// Demonstrates multi-vector hybrid search via Milvus RESTful v2 API.

import milvus from 'k6/x/milvus';
import { check, sleep } from 'k6';

export const options = {
    vus: 1,
    iterations: 1,
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
const COLLECTION_NAME = 'rest_hybrid_demo';
const DENSE_DIM = 128;

function generateDenseVector(dim) {
    return Array.from({ length: dim }, () => Math.random());
}

function generateSparseVector(dim, sparsity) {
    sparsity = sparsity || 0.01;
    const sparse = {};
    const numNonZero = Math.floor(dim * sparsity);
    for (let i = 0; i < numNonZero; i++) {
        const index = Math.floor(Math.random() * dim);
        sparse[index] = Math.random();
    }
    return sparse;
}

export default function () {
    const client = milvus.restClient(MILVUS_HOST);

    // Clean up if exists
    const hasResult = client.hasCollection(COLLECTION_NAME);
    if (hasResult.success && hasResult.result.exists) {
        client.dropCollection(COLLECTION_NAME);
    }

    // Create collection with dense and sparse vector fields
    console.log('Creating hybrid collection...');
    const createResult = client.createCollection({
        name: COLLECTION_NAME,
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'title', dataType: 'VarChar', maxLength: 200 },
            { name: 'price', dataType: 'Float' },
            { name: 'dense_vector', dataType: 'FloatVector', dimension: DENSE_DIM },
            { name: 'sparse_vector', dataType: 'SparseFloatVector' },
        ],
    });
    check(createResult, { 'collection created': (r) => r.success });

    // Create indexes for both vector fields
    client.createIndex('dense_vector', {
        indexType: 'HNSW',
        metricType: 'L2',
        params: { M: 16, efConstruction: 200 },
    }, COLLECTION_NAME);

    client.createIndex('sparse_vector', {
        indexType: 'SPARSE_INVERTED_INDEX',
        metricType: 'IP',
    }, COLLECTION_NAME);

    // Load collection
    client.loadCollection(COLLECTION_NAME);

    // Insert data
    console.log('Inserting data...');
    const titles = ['Laptop Pro', 'Wireless Mouse', 'Mechanical Keyboard', 'USB Hub',
        'Monitor Stand', 'Webcam HD', 'Headset Pro', 'Desk Lamp',
        'Cable Organizer', 'Mouse Pad XL'];

    const data = {
        title: titles,
        price: titles.map(() => Math.round((10 + Math.random() * 490) * 100) / 100),
        dense_vector: titles.map(() => generateDenseVector(DENSE_DIM)),
        sparse_vector: titles.map(() => generateSparseVector(1000, 0.02)),
    };

    const insertResult = client.insert(data, COLLECTION_NAME);
    check(insertResult, { 'insert ok': (r) => r.success });

    sleep(1);

    // Hybrid Search with RRF reranking
    console.log('Performing hybrid search (RRF)...');
    const hybridResult = client.hybridSearch(
        [
            {
                vectors: [generateDenseVector(DENSE_DIM)],
                vectorField: 'dense_vector',
                limit: 10,
                params: { metricType: 'L2' },
            },
            {
                vectors: [generateSparseVector(1000, 0.02)],
                vectorField: 'sparse_vector',
                limit: 10,
                params: { metricType: 'IP' },
            },
        ],
        { type: 'rrf', params: { k: 60 } },
        5,
        ['title', 'price'],
        COLLECTION_NAME,
    );

    check(hybridResult, {
        'hybrid search ok': (r) => r.success,
        'hybrid has results': (r) => !r.empty,
    });

    if (hybridResult.success && !hybridResult.empty) {
        console.log('Hybrid search results (RRF):');
        hybridResult.result.forEach((hit, i) => {
            console.log(`  ${i + 1}. ${hit.title} - $${hit.price} (distance: ${hit.distance})`);
        });
    }

    // Clean up
    client.dropCollection(COLLECTION_NAME);
    client.close();
    console.log('Done!');
}

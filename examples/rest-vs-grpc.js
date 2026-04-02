// REST vs gRPC Comparison Example
// Shows how to use both gRPC and REST clients side by side.
// Useful for comparing performance between the two protocols.

import milvus from 'k6/x/milvus';
import milvusRest from '../lib/milvus-rest.js';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';

export const options = {
    scenarios: {
        grpc: {
            executor: 'constant-vus',
            vus: 2,
            duration: '15s',
            exec: 'grpcSearch',
            tags: { protocol: 'grpc' },
        },
        rest: {
            executor: 'constant-vus',
            vus: 2,
            duration: '15s',
            exec: 'restSearch',
            tags: { protocol: 'rest' },
        },
    },
};

const MILVUS_GRPC = __ENV.MILVUS_HOST || 'localhost:19530';
const MILVUS_REST = __ENV.MILVUS_REST_HOST || 'localhost:19530';
const COLLECTION_NAME = 'protocol_compare';
const VECTOR_DIM = 128;

const grpcSearchTime = new Trend('grpc_search_time', true);
const restSearchTime = new Trend('rest_search_time', true);

function generateRandomVector(dim) {
    return Array.from({ length: dim }, () => Math.random());
}

export function setup() {
    // Use REST client for setup (works with standard k6 too)
    const client = milvusRest.client(MILVUS_REST);

    const hasResult = client.hasCollection(COLLECTION_NAME);
    if (hasResult.success && hasResult.result.exists) {
        client.dropCollection(COLLECTION_NAME);
    }

    client.createCollection({
        name: COLLECTION_NAME,
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'category', dataType: 'VarChar', maxLength: 50 },
            { name: 'embedding', dataType: 'FloatVector', dimension: VECTOR_DIM },
        ],
    });

    client.createIndex('embedding', {
        indexType: 'HNSW',
        metricType: 'L2',
        params: { M: 16, efConstruction: 200 },
    }, COLLECTION_NAME);

    client.loadCollection(COLLECTION_NAME);

    // Insert test data
    const categories = ['A', 'B', 'C'];
    for (let batch = 0; batch < 10; batch++) {
        const rows = [];
        for (let i = 0; i < 100; i++) {
            rows.push({
                category: categories[Math.floor(Math.random() * 3)],
                embedding: generateRandomVector(VECTOR_DIM),
            });
        }
        client.insert(rows, COLLECTION_NAME);
    }

    sleep(2);
    client.close();
}

// gRPC search scenario
export function grpcSearch() {
    const client = milvus.clientWithCollection(MILVUS_GRPC, COLLECTION_NAME);

    const result = client.search(
        [generateRandomVector(VECTOR_DIM)],
        10,
        {
            vectorField: 'embedding',
            outputFields: ['category'],
            metricType: 'L2',
        },
    );

    check(result, {
        'grpc search ok': (r) => r.success,
        'grpc has results': (r) => !r.empty,
    });

    grpcSearchTime.add(result.response_time_ms);
    client.close();
}

// REST search scenario
export function restSearch() {
    const client = milvusRest.clientWithCollection(MILVUS_REST, COLLECTION_NAME);

    const result = client.search(
        [generateRandomVector(VECTOR_DIM)],
        10,
        {
            vectorField: 'embedding',
            outputFields: ['category'],
            metricType: 'L2',
        },
    );

    check(result, {
        'rest search ok': (r) => r.success,
        'rest has results': (r) => !r.empty,
    });

    restSearchTime.add(result.response_time_ms);
    client.close();
}

export function teardown() {
    const client = milvusRest.client(MILVUS_REST);
    client.dropCollection(COLLECTION_NAME);
    client.close();
}

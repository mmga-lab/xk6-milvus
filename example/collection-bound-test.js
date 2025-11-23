import milvus from 'k6/x/milvus';
import { check } from 'k6';

// This example demonstrates the collection-bound client pattern
// Following Locust's design where client is tied to a specific collection

export const options = {
    vus: 5,
    duration: '10s',
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
const COLLECTION_NAME = 'bound_collection';
const VECTOR_DIM = 64;

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

    // Create collection
    const schema = {
        name: COLLECTION_NAME,
        fields: [
            { name: "id", dataType: "Int64", isPrimaryKey: true, isAutoID: true },
            { name: "vector", dataType: "FloatVector", dimension: VECTOR_DIM }
        ]
    };

    client.createCollection(schema);
    client.createIndex("vector", { indexType: "FLAT", metricType: "L2" }, COLLECTION_NAME);
    client.loadCollection(COLLECTION_NAME);

    client.close();
    return { ready: true };
}

export default function() {
    // Create collection-bound client - no need to specify collection in every call!
    const client = milvus.clientWithCollection(MILVUS_HOST, COLLECTION_NAME);

    // Insert - notice we don't pass collection name
    const vectors = generateRandomVectors(5, VECTOR_DIM);
    const insertResult = client.insert({ vector: vectors });

    check(insertResult, {
        'insert successful': (r) => r.success === true,
        'insert fast': (r) => r.response_time_ms < 200,
    });

    // Search - no collection name needed
    const searchVectors = generateRandomVectors(1, VECTOR_DIM);
    const searchResult = client.search(searchVectors, 3, {
        vectorField: 'vector'
    });

    check(searchResult, {
        'search successful': (r) => r.success === true,
        'search not empty': (r) => !r.empty,
        'recall available': (r) => r.recall !== undefined,
    });

    console.log(`Recall: ${searchResult.recall.toFixed(4)}, RT: ${searchResult.response_time_ms.toFixed(2)}ms`);

    // Query - no collection name needed
    const queryResult = client.query('id > 0', ['id']);

    check(queryResult, {
        'query successful': (r) => r.success === true,
    });

    client.close();
}

export function teardown() {
    const client = milvus.client(MILVUS_HOST);
    client.dropCollection(COLLECTION_NAME);
    client.close();
}

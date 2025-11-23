import milvus from 'k6/x/milvus';
import { check } from 'k6';

export const options = {
    vus: 10,
    duration: '30s',
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
const COLLECTION_NAME = 'test_collection_new';
const VECTOR_DIM = 128;

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
    // Using traditional client (not bound to collection)
    const client = milvus.client(MILVUS_HOST);

    // Drop collection if exists
    const hasResult = client.hasCollection(COLLECTION_NAME);
    if (hasResult.success && hasResult.result.exists) {
        client.dropCollection(COLLECTION_NAME);
    }

    // Create collection with schema
    const schema = {
        name: COLLECTION_NAME,
        description: "New API test collection",
        fields: [
            {
                name: "id",
                dataType: "Int64",
                isPrimaryKey: true,
                isAutoID: true
            },
            {
                name: "price",
                dataType: "Float"
            },
            {
                name: "title",
                dataType: "VarChar",
                maxLength: 200
            },
            {
                name: "vector",
                dataType: "FloatVector",
                dimension: VECTOR_DIM
            }
        ]
    };

    const createResult = client.createCollection(schema);
    check(createResult, {
        'collection created successfully': (r) => r.success === true,
        'create response time < 5000ms': (r) => r.response_time_ms < 5000,
    });

    // Create index
    const indexResult = client.createIndex("vector", {
        indexType: "FLAT",
        metricType: "L2"
    }, COLLECTION_NAME);
    check(indexResult, {
        'index created successfully': (r) => r.success === true,
    });

    // Load collection
    const loadResult = client.loadCollection(COLLECTION_NAME);
    check(loadResult, {
        'collection loaded successfully': (r) => r.success === true,
    });

    client.close();

    return { collectionCreated: true };
}

export default function() {
    // NEW: Using collection-bound client (Locust pattern)
    const client = milvus.clientWithCollection(MILVUS_HOST, COLLECTION_NAME);

    // Insert data
    const vectors = generateRandomVectors(10, VECTOR_DIM);
    const titles = [];
    const prices = [];

    for (let i = 0; i < 10; i++) {
        titles.push(`Product ${i}`);
        prices.push(Math.random() * 100);
    }

    const insertResult = client.insert({
        title: titles,
        price: prices,
        vector: vectors
    });

    check(insertResult, {
        'insert successful': (r) => r.success === true,
        'insert not empty': (r) => r.empty !== true,
        'insert response time < 500ms': (r) => r.response_time_ms < 500,
        'inserted 10 records': (r) => r.result.insert_count === 10,
    });

    // NEW: Query operation (without vectors)
    const queryResult = client.query('price > 50', ['id', 'title', 'price']);
    check(queryResult, {
        'query successful': (r) => r.success === true,
        'query response time < 200ms': (r) => r.response_time_ms < 200,
    });

    if (queryResult.success && !queryResult.empty) {
        console.log(`Query found ${queryResult.result.length} products with price > 50`);
    }

    // Search with Recall
    const searchVectors = generateRandomVectors(1, VECTOR_DIM);
    const searchResult = client.search(searchVectors, 5, {
        vectorField: 'vector',
        outputFields: ['title', 'price'],
        expr: 'price > 20'
    });

    check(searchResult, {
        'search successful': (r) => r.success === true,
        'search not empty': (r) => r.empty !== true,
        'search response time < 300ms': (r) => r.response_time_ms < 300,
    });

    // NEW: Recall metric is now available!
    if (searchResult.success) {
        console.log(`Search recall: ${searchResult.recall.toFixed(4)}`);
        console.log(`Search response time: ${searchResult.response_time_ms.toFixed(2)}ms`);
    }

    // NEW: Delete operation
    const deleteResult = client.delete('price < 10');
    check(deleteResult, {
        'delete successful': (r) => r.success === true,
        'delete response time < 200ms': (r) => r.response_time_ms < 200,
    });

    if (deleteResult.success) {
        console.log(`Deleted ${deleteResult.result.delete_count} records`);
    }

    // Upsert operation
    const upsertData = {
        id: [1, 2],
        title: ['Updated Product 1', 'Updated Product 2'],
        price: [99.99, 89.99],
        vector: generateRandomVectors(2, VECTOR_DIM)
    };

    const upsertResult = client.upsert(upsertData);
    check(upsertResult, {
        'upsert successful': (r) => r.success === true,
        'upsert response time < 500ms': (r) => r.response_time_ms < 500,
    });

    client.close();
}

export function teardown(data) {
    if (data.collectionCreated) {
        const client = milvus.client(MILVUS_HOST);
        const dropResult = client.dropCollection(COLLECTION_NAME);
        check(dropResult, {
            'collection dropped successfully': (r) => r.success === true,
        });
        client.close();
    }
}

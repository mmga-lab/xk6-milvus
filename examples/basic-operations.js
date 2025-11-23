// Basic CRUD Operations Example
// This example demonstrates the fundamental operations in Milvus:
// - Creating a collection
// - Inserting data
// - Searching vectors
// - Deleting data

import milvus from 'k6/x/milvus';
import { check, sleep } from 'k6';

export const options = {
    vus: 1,
    iterations: 1,
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
const COLLECTION_NAME = 'basic_example';
const VECTOR_DIM = 8;  // Small dimension for clarity

// Helper function to generate random vectors
function generateRandomVector(dim) {
    const vector = [];
    for (let i = 0; i < dim; i++) {
        vector.push(Math.random());
    }
    return vector;
}

export default function() {
    // Step 1: Create a client
    console.log('Step 1: Creating Milvus client...');
    const client = milvus.client(MILVUS_HOST);

    // Step 2: Create a collection
    console.log('Step 2: Creating collection...');
    const schema = {
        name: COLLECTION_NAME,
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'title', dataType: 'VarChar', maxLength: 100 },
            { name: 'price', dataType: 'Float' },
            { name: 'embedding', dataType: 'FloatVector', dimension: VECTOR_DIM }
        ]
    };

    const createResult = client.createCollection(schema);
    check(createResult, {
        'collection created': (r) => r.success === true,
    });
    console.log(`Collection created in ${createResult.response_time_ms}ms`);

    // Step 3: Create index for faster search
    console.log('Step 3: Creating index...');
    const indexResult = client.createIndex('embedding', {
        indexType: 'FLAT',
        metricType: 'L2'
    }, COLLECTION_NAME);
    check(indexResult, {
        'index created': (r) => r.success === true,
    });

    // Step 4: Load collection into memory
    console.log('Step 4: Loading collection...');
    const loadResult = client.loadCollection(COLLECTION_NAME);
    check(loadResult, {
        'collection loaded': (r) => r.success === true,
    });

    // Step 5: Insert data
    console.log('Step 5: Inserting data...');
    const insertResult = client.insert({
        title: ['Apple', 'Banana', 'Orange', 'Grape'],
        price: [1.99, 0.99, 2.49, 3.99],
        embedding: [
            generateRandomVector(VECTOR_DIM),
            generateRandomVector(VECTOR_DIM),
            generateRandomVector(VECTOR_DIM),
            generateRandomVector(VECTOR_DIM)
        ]
    }, COLLECTION_NAME);

    check(insertResult, {
        'insert successful': (r) => r.success === true,
        'inserted 4 items': (r) => r.result.insert_count === 4,
    });
    console.log(`Inserted ${insertResult.result.insert_count} items in ${insertResult.response_time_ms}ms`);

    // Wait for data to be indexed and available for search
    sleep(0.5);

    // Step 6: Search for similar vectors
    console.log('Step 6: Searching for similar items...');
    const queryVector = generateRandomVector(VECTOR_DIM);
    const searchResult = client.search(
        [queryVector],  // Single query vector
        3,              // Top 3 results
        {
            vectorField: 'embedding',
            outputFields: ['title', 'price'],
            metricType: 'L2'
        },
        COLLECTION_NAME
    );

    check(searchResult, {
        'search successful': (r) => r.success === true,
        'has results': (r) => !r.empty,
    });

    if (searchResult.success && !searchResult.empty) {
        console.log(`Search took ${searchResult.response_time_ms}ms, recall: ${searchResult.recall}`);
        console.log('Top 3 results:');
        searchResult.result.forEach((hit, index) => {
            console.log(`  ${index + 1}. ${hit.title} - $${hit.price} (distance: ${hit.distance})`);
        });
    }

    // Step 7: Query with filter (no vectors)
    console.log('Step 7: Querying with price filter...');
    const queryResult = client.query(
        'price > 1.5',
        ['id', 'title', 'price'],
        COLLECTION_NAME
    );

    check(queryResult, {
        'query successful': (r) => r.success === true,
    });

    if (queryResult.success && !queryResult.empty) {
        console.log(`Found ${queryResult.result.length} items with price > 1.5:`);
        queryResult.result.forEach(item => {
            console.log(`  - ${item.title}: $${item.price}`);
        });
    }

    // Step 8: Delete data
    console.log('Step 8: Deleting expensive items...');
    const deleteResult = client.delete('price > 3.0', COLLECTION_NAME);
    check(deleteResult, {
        'delete successful': (r) => r.success === true,
    });
    console.log(`Deleted ${deleteResult.result.delete_count} items`);

    // Step 9: Clean up
    console.log('Step 9: Cleaning up...');
    const dropResult = client.dropCollection(COLLECTION_NAME);
    check(dropResult, {
        'collection dropped': (r) => r.success === true,
    });

    client.close();
    console.log('Done! ✓');
}

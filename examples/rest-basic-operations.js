// REST API Basic Operations Example
// Demonstrates CRUD operations using Milvus RESTful v2 API with k6's built-in HTTP.
// No custom k6 binary needed - works with standard k6.

import milvusRest from '../lib/milvus-rest.js';
import { check, sleep } from 'k6';

export const options = {
    vus: 1,
    iterations: 1,
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
const COLLECTION_NAME = 'rest_basic_example';
const VECTOR_DIM = 8;

function generateRandomVector(dim) {
    return Array.from({ length: dim }, () => Math.random());
}

export default function () {
    // Step 1: Create a REST client
    console.log('Step 1: Creating Milvus REST client...');
    const client = milvusRest.client(MILVUS_HOST);

    // Step 2: Create a collection
    console.log('Step 2: Creating collection...');
    const createResult = client.createCollection({
        name: COLLECTION_NAME,
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'title', dataType: 'VarChar', maxLength: 100 },
            { name: 'price', dataType: 'Float' },
            { name: 'embedding', dataType: 'FloatVector', dimension: VECTOR_DIM },
        ],
    });
    check(createResult, {
        'collection created': (r) => r.success === true,
    });
    console.log(`Collection created in ${createResult.response_time_ms}ms`);

    // Step 3: Create index
    console.log('Step 3: Creating index...');
    const indexResult = client.createIndex('embedding', {
        indexType: 'FLAT',
        metricType: 'L2',
    }, COLLECTION_NAME);
    check(indexResult, {
        'index created': (r) => r.success === true,
    });

    // Step 4: Load collection
    console.log('Step 4: Loading collection...');
    const loadResult = client.loadCollection(COLLECTION_NAME);
    check(loadResult, {
        'collection loaded': (r) => r.success === true,
    });

    // Step 5: Insert data (column-based format, auto-converted to row-based)
    console.log('Step 5: Inserting data (column format)...');
    const insertResult = client.insert({
        title: ['Apple', 'Banana', 'Orange', 'Grape'],
        price: [1.99, 0.99, 2.49, 3.99],
        embedding: [
            generateRandomVector(VECTOR_DIM),
            generateRandomVector(VECTOR_DIM),
            generateRandomVector(VECTOR_DIM),
            generateRandomVector(VECTOR_DIM),
        ],
    }, COLLECTION_NAME);

    check(insertResult, {
        'insert successful': (r) => r.success === true,
        'inserted 4 items': (r) => r.result.insert_count === 4,
    });
    console.log(`Inserted ${insertResult.result.insert_count} items in ${insertResult.response_time_ms}ms`);

    // Step 5b: Insert data (row-based format, native REST format)
    console.log('Step 5b: Inserting data (row format)...');
    const insertRowResult = client.insert([
        { title: 'Mango', price: 4.99, embedding: generateRandomVector(VECTOR_DIM) },
        { title: 'Peach', price: 2.99, embedding: generateRandomVector(VECTOR_DIM) },
    ], COLLECTION_NAME);

    check(insertRowResult, {
        'row insert successful': (r) => r.success === true,
        'inserted 2 items': (r) => r.result.insert_count === 2,
    });

    sleep(1);

    // Step 6: Search
    console.log('Step 6: Searching for similar items...');
    const queryVector = generateRandomVector(VECTOR_DIM);
    const searchResult = client.search(
        [queryVector],
        3,
        {
            vectorField: 'embedding',
            outputFields: ['title', 'price'],
            metricType: 'L2',
        },
        COLLECTION_NAME,
    );

    check(searchResult, {
        'search successful': (r) => r.success === true,
        'has results': (r) => !r.empty,
    });

    if (searchResult.success && !searchResult.empty) {
        console.log(`Search took ${searchResult.response_time_ms}ms`);
        console.log('Top 3 results:');
        searchResult.result.forEach((hit, index) => {
            console.log(`  ${index + 1}. ${hit.title} - $${hit.price} (distance: ${hit.distance})`);
        });
    }

    // Step 7: Query with filter
    console.log('Step 7: Querying with price filter...');
    const queryResult = client.query(
        'price > 1.5',
        ['id', 'title', 'price'],
        COLLECTION_NAME,
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

    // Step 8: Get entities by IDs
    if (insertResult.success && insertResult.result.ids.length > 0) {
        console.log('Step 8: Getting entities by IDs...');
        const ids = insertResult.result.ids.slice(0, 2);
        const getResult = client.get(ids, ['id', 'title', 'price'], COLLECTION_NAME);
        check(getResult, {
            'get successful': (r) => r.success === true,
        });
        if (getResult.success && getResult.result) {
            console.log(`Got ${getResult.result.length} entities by ID`);
        }
    }

    // Step 9: Delete
    console.log('Step 9: Deleting expensive items...');
    const deleteResult = client.delete('price > 3.0', COLLECTION_NAME);
    check(deleteResult, {
        'delete successful': (r) => r.success === true,
    });

    // Step 10: Clean up
    console.log('Step 10: Cleaning up...');
    const dropResult = client.dropCollection(COLLECTION_NAME);
    check(dropResult, {
        'collection dropped': (r) => r.success === true,
    });

    client.close();
    console.log('Done!');
}

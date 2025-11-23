// Collection Management Example
// This example demonstrates collection lifecycle operations:
// - Creating collections with different schemas
// - Checking collection existence
// - Loading and releasing collections
// - Dropping collections

import milvus from 'k6/x/milvus';
import { check } from 'k6';

export const options = {
    vus: 1,
    iterations: 1,
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';

export default function() {
    const client = milvus.client(MILVUS_HOST);

    // Example 1: Simple collection
    console.log('=== Example 1: Simple Collection ===');
    const simpleSchema = {
        name: 'simple_collection',
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true },
            { name: 'vector', dataType: 'FloatVector', dimension: 128 }
        ]
    };

    let result = client.createCollection(simpleSchema);
    check(result, { 'simple collection created': (r) => r.success });

    // Check if collection exists
    result = client.hasCollection('simple_collection');
    check(result, {
        'check successful': (r) => r.success,
        'collection exists': (r) => r.result === true,
    });
    console.log('Collection exists:', result.result);

    // Example 2: Collection with multiple field types
    console.log('\n=== Example 2: Complex Collection ===');
    const complexSchema = {
        name: 'product_catalog',
        fields: [
            { name: 'product_id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'name', dataType: 'VarChar', maxLength: 200 },
            { name: 'category', dataType: 'VarChar', maxLength: 50 },
            { name: 'price', dataType: 'Float' },
            { name: 'stock', dataType: 'Int32' },
            { name: 'rating', dataType: 'Double' },
            { name: 'is_available', dataType: 'Bool' },
            { name: 'image_embedding', dataType: 'FloatVector', dimension: 256 },
            { name: 'text_embedding', dataType: 'FloatVector', dimension: 768 }
        ]
    };

    result = client.createCollection(complexSchema);
    check(result, { 'complex collection created': (r) => r.success });

    // Example 3: Collection from JSON
    console.log('\n=== Example 3: Collection from JSON ===');
    const jsonSchema = JSON.stringify({
        name: 'json_collection',
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true },
            { name: 'data', dataType: 'VarChar', maxLength: 1000 },
            { name: 'embedding', dataType: 'FloatVector', dimension: 64 }
        ]
    });

    result = client.createCollectionFromJSON(jsonSchema);
    check(result, { 'JSON collection created': (r) => r.success });

    // Example 4: Load and release collection
    console.log('\n=== Example 4: Load/Release Collection ===');

    // Create index before loading (required)
    result = client.createIndex('vector', {
        indexType: 'FLAT',
        metricType: 'L2'
    }, 'simple_collection');
    check(result, { 'index created': (r) => r.success });

    // Load collection into memory (required for search)
    result = client.loadCollection('simple_collection');
    check(result, { 'collection loaded': (r) => r.success });
    console.log(`Collection loaded in ${result.response_time_ms}ms`);

    // Release collection from memory
    result = client.releaseCollection('simple_collection');
    check(result, { 'collection released': (r) => r.success });
    console.log('Collection released from memory');

    // Example 5: Clean up - Drop all collections
    console.log('\n=== Example 5: Cleanup ===');

    const collections = ['simple_collection', 'product_catalog', 'json_collection'];
    collections.forEach(name => {
        const dropResult = client.dropCollection(name);
        check(dropResult, { [`${name} dropped`]: (r) => r.success });
        console.log(`Dropped ${name}`);
    });

    client.close();
    console.log('\nDone! All collections cleaned up ✓');
}

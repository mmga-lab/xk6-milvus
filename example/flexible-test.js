import milvus from 'k6/x/milvus';
import { check } from 'k6';

export const options = {
    vus: 5,
    duration: '20s',
};

export function setup() {
    const host = __ENV.MILVUS_HOST || 'localhost:19530';
    console.log(`Connecting to Milvus at ${host}`);
    
    const client = milvus.client(host);
    const collectionName = 'flexible_test_collection';
    
    // Drop collection if exists
    try {
        if (client.hasCollection(collectionName)) {
            client.dropCollection(collectionName);
        }
    } catch (e) {
        console.log('Collection does not exist, continuing...');
    }
    
    // Define a flexible schema with multiple field types
    const schema = {
        name: collectionName,
        description: 'Flexible test collection with multiple field types',
        fields: [
            {
                name: 'id',
                dataType: 'Int64',
                isPrimaryKey: true,
                isAutoID: true,
                description: 'Primary key field'
            },
            {
                name: 'title',
                dataType: 'VarChar',
                maxLength: 200,
                description: 'Title field'
            },
            {
                name: 'category',
                dataType: 'VarChar', 
                maxLength: 50,
                description: 'Category field'
            },
            {
                name: 'price',
                dataType: 'Float',
                description: 'Price field'
            },
            {
                name: 'rating',
                dataType: 'Double',
                description: 'Rating field'
            },
            {
                name: 'in_stock',
                dataType: 'Bool',
                description: 'Stock status'
            },
            {
                name: 'embedding',
                dataType: 'FloatVector',
                dimension: 128,
                description: 'Product embedding vector'
            }
        ]
    };
    
    // Create collection with flexible schema
    client.createCollectionFromJSON(JSON.stringify(schema));
    console.log('Collection created with flexible schema');
    
    // Create HNSW index for better performance
    const indexParams = {
        indexType: 'HNSW',
        metricType: 'L2',
        M: 16,
        efConstruction: 200
    };
    client.createIndex(collectionName, 'embedding', indexParams);
    console.log('HNSW index created');
    
    // Load collection
    client.loadCollection(collectionName);
    console.log('Collection loaded');
    
    client.close();
    
    return { collectionName: collectionName };
}

export default function(data) {
    const host = __ENV.MILVUS_HOST || 'localhost:19530';
    const client = milvus.client(host);
    const collectionName = data.collectionName;
    
    // Generate test data with multiple fields
    const batchSize = 10;
    const testData = {
        title: [],
        category: [],
        price: [],
        rating: [],
        in_stock: [],
        embedding: []
    };
    
    for (let i = 0; i < batchSize; i++) {
        testData.title.push(`Product ${Math.floor(Math.random() * 1000)}`);
        testData.category.push(['Electronics', 'Books', 'Clothing', 'Sports'][Math.floor(Math.random() * 4)]);
        testData.price.push(Math.random() * 100);
        testData.rating.push(Math.random() * 5);
        testData.in_stock.push(Math.random() > 0.3);
        
        // Generate random 128-dim vector
        const vector = [];
        for (let j = 0; j < 128; j++) {
            vector.push(Math.random());
        }
        testData.embedding.push(vector);
    }
    
    // Insert multi-field data
    const insertIds = client.insert(collectionName, testData);
    check(insertIds, {
        'insert successful': (ids) => ids && ids.length === batchSize,
    });
    
    // Search with advanced parameters
    const queryVector = [];
    for (let i = 0; i < 128; i++) {
        queryVector.push(Math.random());
    }
    
    const searchParams = {
        vectorField: 'embedding',
        outputFields: ['title', 'category', 'price', 'rating', 'in_stock'],
        expr: 'price > 10.0 and in_stock == true', // Filter condition
    };
    
    const searchResults = client.search(collectionName, [queryVector], 5, searchParams);
    
    check(searchResults, {
        'search successful': (results) => results && results.length >= 0,
        'search returns fields': (results) => {
            if (results.length > 0) {
                const result = results[0];
                return result.fields && 
                       result.fields.hasOwnProperty('title') &&
                       result.fields.hasOwnProperty('category') &&
                       result.fields.hasOwnProperty('price') &&
                       result.fields.hasOwnProperty('rating') &&
                       result.fields.hasOwnProperty('in_stock');
            }
            return true; // No results is also valid due to filter
        }
    });
    
    if (searchResults.length > 0) {
        console.log(`Found ${searchResults.length} results:`);
        searchResults.forEach((result, index) => {
            console.log(`  ${index + 1}. ID: ${result.id}, Score: ${result.score}`);
            console.log(`     Title: ${result.fields.title}, Category: ${result.fields.category}`);
            console.log(`     Price: $${result.fields.price.toFixed(2)}, Rating: ${result.fields.rating.toFixed(1)}`);
            console.log(`     In Stock: ${result.fields.in_stock}`);
        });
    }
    
    client.close();
}

export function teardown(data) {
    const host = __ENV.MILVUS_HOST || 'localhost:19530';
    const client = milvus.client(host);
    
    // Clean up
    client.dropCollection(data.collectionName);
    client.close();
    console.log('Test cleanup completed');
}
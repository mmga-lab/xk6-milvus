// Check what's failing in the examples
import milvus from 'k6/x/milvus';
import { check, sleep } from 'k6';

export const options = {
    vus: 1,
    iterations: 1,
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';

function generateRandomVector(dim) {
    const vector = [];
    for (let i = 0; i < dim; i++) {
        vector.push(Math.random());
    }
    return vector;
}

export default function() {
    console.log('=== Test 1: Basic Search ===');
    const client1 = milvus.client(MILVUS_HOST);

    const schema1 = {
        name: 'test_basic_search',
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'title', dataType: 'VarChar', maxLength: 100 },
            { name: 'embedding', dataType: 'FloatVector', dimension: 8 }
        ]
    };

    client1.createCollection(schema1);
    client1.createIndex('embedding', { indexType: 'FLAT', metricType: 'L2' }, 'test_basic_search');
    client1.loadCollection('test_basic_search');

    // Insert data
    client1.insert({
        title: ['Apple', 'Banana'],
        embedding: [
            generateRandomVector(8),
            generateRandomVector(8)
        ]
    }, 'test_basic_search');

    // Immediate search (might be empty)
    const searchResult1 = client1.search(
        [generateRandomVector(8)],
        2,
        { vectorField: 'embedding', outputFields: ['title'], metricType: 'L2' },
        'test_basic_search'
    );

    console.log('Immediate search - success:', searchResult1.success);
    console.log('Immediate search - empty:', searchResult1.empty);
    console.log('Immediate search - recall:', searchResult1.recall);

    // Wait and search again
    sleep(1);
    const searchResult2 = client1.search(
        [generateRandomVector(8)],
        2,
        { vectorField: 'embedding', outputFields: ['title'], metricType: 'L2' },
        'test_basic_search'
    );

    console.log('After 1s search - success:', searchResult2.success);
    console.log('After 1s search - empty:', searchResult2.empty);
    console.log('After 1s search - result count:', searchResult2.result ? searchResult2.result.length : 0);

    client1.dropCollection('test_basic_search');
    client1.close();

    console.log('\n=== Test 2: Hybrid Search ===');
    const client2 = milvus.client(MILVUS_HOST);

    const schema2 = {
        name: 'test_hybrid_search',
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'dense_vector', dataType: 'FloatVector', dimension: 4 },
            { name: 'sparse_vector', dataType: 'FloatVector', dimension: 4 }
        ]
    };

    client2.createCollection(schema2);
    client2.createIndex('dense_vector', { indexType: 'FLAT', metricType: 'L2' }, 'test_hybrid_search');
    client2.createIndex('sparse_vector', { indexType: 'FLAT', metricType: 'IP' }, 'test_hybrid_search');
    client2.loadCollection('test_hybrid_search');

    client2.insert({
        dense_vector: [[0.1, 0.2, 0.3, 0.4], [0.5, 0.6, 0.7, 0.8]],
        sparse_vector: [[0.9, 0.8, 0.7, 0.6], [0.5, 0.4, 0.3, 0.2]]
    }, 'test_hybrid_search');

    sleep(1);

    const hybridResult = client2.hybridSearch(
        [
            {
                vectors: [[0.1, 0.2, 0.3, 0.4]],
                vectorField: 'dense_vector',
                limit: 2,
                params: { metricType: 'L2' }
            },
            {
                vectors: [[0.9, 0.8, 0.7, 0.6]],
                vectorField: 'sparse_vector',
                limit: 2,
                params: { metricType: 'IP' }
            }
        ],
        { type: 'rrf', params: { k: 60 } },
        2,
        ['id'],
        'test_hybrid_search'
    );

    console.log('Hybrid search - success:', hybridResult.success);
    console.log('Hybrid search - empty:', hybridResult.empty);
    console.log('Hybrid search - error:', hybridResult.error);
    console.log('Hybrid search - recall:', hybridResult.recall);
    console.log('Hybrid search - result count:', hybridResult.result ? hybridResult.result.length : 0);

    const checks = check(hybridResult, {
        'hybrid search successful': (r) => r.success === true,
        'has results': (r) => !r.empty,
        'good recall': (r) => r.recall >= 0.9,
    });

    console.log('Checks passed:', checks);

    client2.dropCollection('test_hybrid_search');
    client2.close();

    console.log('\n=== Test 3: Collection Load ===');
    const client3 = milvus.client(MILVUS_HOST);

    const schema3 = {
        name: 'test_load',
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
            { name: 'vector', dataType: 'FloatVector', dimension: 4 }
        ]
    };

    const createResult = client3.createCollection(schema3);
    console.log('Create collection - success:', createResult.success);
    console.log('Create collection - response_time_ms:', createResult.response_time_ms);

    const loadResult = client3.loadCollection('test_load');
    console.log('Load collection - success:', loadResult.success);
    console.log('Load collection - response_time_ms:', loadResult.response_time_ms);
    console.log('Load collection - error:', loadResult.error);

    const loadCheck = check(loadResult, {
        'collection loaded': (r) => r.success === true && r.response_time_ms < 2000,
    });
    console.log('Load check passed:', loadCheck);

    client3.dropCollection('test_load');
    client3.close();
}

// Recall Validation Example
// This example demonstrates how to validate search recall using ground truth data
// Recall = (Number of relevant items retrieved) / (Total number of relevant items)
//
// Data is pre-generated using Go tool: tools/generate-recall-data/main.go
// - train.json: 200 vectors (10 groups × 10 similar vectors + 100 noise vectors)
// - test.json: 10 query vectors (one per group)
// - neighbors.json: ground truth (expected neighbors for each query)

import milvus from 'k6/x/milvus';
import { check, sleep } from 'k6';

export const options = {
    vus: 1,
    iterations: 1,
};

const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
const COLLECTION_NAME = 'recall_validation';
const VECTOR_DIM = 128;

// Load pre-generated data
const trainData = JSON.parse(open('./recall-data/train.json'));
const testData = JSON.parse(open('./recall-data/test.json'));
const neighborsData = JSON.parse(open('./recall-data/neighbors.json'));

export function setup() {
    const client = milvus.client(MILVUS_HOST);

    // Drop if exists
    const hasResult = client.hasCollection(COLLECTION_NAME);
    if (hasResult.success && hasResult.result.exists) {
        client.dropCollection(COLLECTION_NAME);
    }

    // Create collection
    const schema = {
        name: COLLECTION_NAME,
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true },
            { name: 'group_id', dataType: 'Int64' },
            { name: 'category', dataType: 'VarChar', maxLength: 50 },
            { name: 'embedding', dataType: 'FloatVector', dimension: VECTOR_DIM }
        ]
    };

    const createResult = client.createCollection(schema);
    check(createResult, {
        'collection created': (r) => r.success === true,
    });

    // Create index
    const indexResult = client.createIndex('embedding', {
        indexType: 'HNSW',
        metricType: 'L2',
        M: 16,
        efConstruction: 200
    }, COLLECTION_NAME);

    check(indexResult, {
        'index created': (r) => r.success === true,
    });

    // Load collection
    const loadResult = client.loadCollection(COLLECTION_NAME);
    check(loadResult, {
        'collection loaded': (r) => r.success === true,
    });

    // Insert train data
    const insertResult = client.insert(trainData, COLLECTION_NAME);
    check(insertResult, {
        'data inserted': (r) => r.success === true,
        'correct count': (r) => r.result.insert_count === 200,
    });

    console.log(`Inserted ${insertResult.result.insert_count} vectors`);
    console.log('Ground truth: 10 groups with 10 similar vectors each, plus 100 noise vectors');

    client.close();

    // Wait for data to be indexed
    sleep(2);

    return { ready: true };
}

export default function() {
    const client = milvus.clientWithCollection(MILVUS_HOST, COLLECTION_NAME);

    console.log('\n=== Recall Validation Test ===\n');

    let totalRecall = 0;
    let testCount = 0;

    // Test recall for each query
    for (let i = 0; i < testData.query_id.length; i++) {
        const queryId = testData.query_id[i];
        const groupId = testData.group_id[i];
        const queryVector = testData.embedding[i];
        const expectedNeighbors = neighborsData[i].neighbors;

        // Search for top 10 (should retrieve all vectors from the same group)
        const searchResult = client.search(
            [queryVector],
            10,
            {
                vectorField: 'embedding',
                outputFields: ['id', 'group_id', 'category'],
                metricType: 'L2'
            }
        );

        if (searchResult.success && !searchResult.empty) {
            const retrievedIds = searchResult.result.map(r => r.id);

            // Calculate recall: how many expected IDs were retrieved
            let foundCount = 0;
            for (const expectedId of expectedNeighbors) {
                if (retrievedIds.includes(expectedId)) {
                    foundCount++;
                }
            }

            const recall = foundCount / expectedNeighbors.length;
            totalRecall += recall;
            testCount++;

            console.log(`Query ${queryId} (Group ${groupId}):`);
            console.log(`  Expected IDs: ${expectedNeighbors.slice(0, 3).join(', ')}... (${expectedNeighbors.length} total)`);
            console.log(`  Retrieved IDs: ${retrievedIds.slice(0, 3).join(', ')}... (${retrievedIds.length} total)`);
            console.log(`  Found: ${foundCount}/${expectedNeighbors.length}`);
            console.log(`  Recall: ${(recall * 100).toFixed(2)}%`);
            console.log(`  Response time: ${searchResult.response_time_ms.toFixed(2)}ms\n`);

            check(searchResult, {
                [`Query ${queryId} - search successful`]: (r) => r.success === true,
                [`Query ${queryId} - has results`]: (r) => !r.empty,
                [`Query ${queryId} - high recall (>= 80%)`]: () => recall >= 0.8,
                [`Query ${queryId} - perfect recall (100%)`]: () => recall === 1.0,
            });
        }
    }

    const averageRecall = totalRecall / testCount;
    console.log('===========================================');
    console.log(`Average Recall across all queries: ${(averageRecall * 100).toFixed(2)}%`);
    console.log('===========================================\n');

    check({ avgRecall: averageRecall }, {
        'Overall average recall >= 90%': (r) => r.avgRecall >= 0.9,
        'Overall average recall >= 95%': (r) => r.avgRecall >= 0.95,
        'Overall average recall == 100%': (r) => r.avgRecall === 1.0,
    });

    // Test with different top-k values
    console.log('\n=== Testing Recall with Different Top-K Values ===\n');

    const topKValues = [5, 10, 20];
    const queryVector = testData.embedding[0];
    const expectedNeighbors = neighborsData[0].neighbors;

    for (const topK of topKValues) {
        const searchResult = client.search(
            [queryVector],
            topK,
            {
                vectorField: 'embedding',
                outputFields: ['id', 'group_id'],
                metricType: 'L2'
            }
        );

        if (searchResult.success) {
            const retrievedIds = searchResult.result.map(r => r.id);
            let foundCount = 0;
            for (const expectedId of expectedNeighbors) {
                if (retrievedIds.includes(expectedId)) {
                    foundCount++;
                }
            }

            // Recall calculation: among the top-K results, how many are relevant?
            const expectedInTopK = Math.min(topK, expectedNeighbors.length);
            const recall = foundCount / expectedInTopK;

            console.log(`Top-${topK}:`);
            console.log(`  Found ${foundCount}/${expectedInTopK} relevant items`);
            console.log(`  Recall@${topK}: ${(recall * 100).toFixed(2)}%\n`);
        }
    }

    client.close();
}

export function teardown(data) {
    if (data.ready) {
        const client = milvus.client(MILVUS_HOST);
        client.dropCollection(COLLECTION_NAME);
        console.log('Teardown complete');
        client.close();
    }
}

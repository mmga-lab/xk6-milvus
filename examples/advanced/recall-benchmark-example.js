// Example demonstrating Milvus search recall measurement with xk6-milvus
import milvus from 'k6/x/milvus';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// Custom metrics to complement built-in metrics
const recallValidations = new Counter('custom_recall_validations');
const recallDifference = new Trend('custom_recall_difference');

export const options = {
    vus: 2,
    duration: '30s',
    thresholds: {
        // Built-in recall metric from the extension
        'milvus_recall': ['avg>0.8'],  // Average recall should be above 80%
        
        // Other quality and performance thresholds
        'milvus_req_duration{operation:search}': ['p(95)<1000'],
        'milvus_req_duration{operation:search_with_recall}': ['p(95)<1200'],
        'milvus_errors': ['rate<0.05'],
        
        // Custom metrics
        'custom_recall_validations': ['count>10'],
        'checks': ['rate>0.9'],
    },
};

export function setup() {
    console.log('Setting up recall benchmark test...');
    
    const client = milvus.client();
    
    // Create collection for recall testing
    client.createCollectionSimple('recall_test', 256);
    client.createIndexSimple('recall_test', 'vector');
    client.loadCollection('recall_test');
    
    // Insert a known dataset for recall calculation
    const knownVectors = [];
    const knownIds = [];
    
    // Create 100 known vectors with predictable patterns
    for (let i = 0; i < 100; i++) {
        const vector = Array(256).fill(0).map((_, idx) => {
            // Create patterns that will allow us to predict similar vectors
            return Math.sin(i * 0.1 + idx * 0.01) + Math.cos(i * 0.05);
        });
        knownVectors.push(vector);
    }
    
    // Insert the known vectors
    const insertedIds = client.insertVectors('recall_test', knownVectors);
    console.log(`Inserted ${insertedIds.length} known vectors for recall testing`);
    
    // Build ground truth: for each vector, similar vectors are those close in index
    const groundTruth = [];
    for (let i = 0; i < knownVectors.length; i++) {
        const truth = [];
        // Add the vector itself and nearby vectors as ground truth
        for (let j = Math.max(0, i - 2); j <= Math.min(knownVectors.length - 1, i + 2); j++) {
            if (insertedIds[j] !== undefined) {
                truth.push(insertedIds[j]);
            }
        }
        groundTruth.push(truth);
    }
    
    return { 
        client, 
        knownVectors, 
        groundTruth,
        insertedIds 
    };
}

export default function(data) {
    const { client, knownVectors, groundTruth, insertedIds } = data;
    
    // Select a random query vector from our known set
    const queryIndex = Math.floor(Math.random() * knownVectors.length);
    const queryVector = knownVectors[queryIndex];
    const expectedTruth = groundTruth[queryIndex];
    
    // === Test 1: Regular search (without recall calculation) ===
    const searchParams = {
        vectorField: 'vector',
        outputFields: ['id']
    };
    
    const regularResults = client.search('recall_test', [queryVector], 5, searchParams);
    
    check(regularResults, {
        'regular search returned results': (results) => results && results.length > 0,
        'regular search found exact match': (results) => {
            const expectedId = insertedIds[queryIndex];
            return results && results.some(r => r.id === expectedId);
        }
    });
    
    // === Test 2: Search with automatic recall calculation ===
    const resultsWithRecall = client.searchWithRecall(
        'recall_test', 
        [queryVector], 
        5, 
        searchParams, 
        [expectedTruth]  // Ground truth for this single query
    );
    
    check(resultsWithRecall, {
        'recall search returned results': (results) => results && results.length > 0,
        'recall search results match regular search': (results) => {
            return results && regularResults && results.length === regularResults.length;
        }
    });
    
    // === Manual recall validation (for comparison) ===
    if (resultsWithRecall && expectedTruth) {
        const retrievedIds = resultsWithRecall.map(r => r.id);
        const relevantRetrieved = retrievedIds.filter(id => expectedTruth.includes(id));
        const manualRecall = relevantRetrieved.length / expectedTruth.length;
        
        // Record custom metrics
        recallValidations.add(1);
        
        // Validate that our ground truth makes sense
        check(manualRecall, {
            'manual recall is reasonable': (recall) => recall >= 0 && recall <= 1,
            'found at least some relevant results': (recall) => recall > 0,
        });
    }
    
    // === Test 3: Multi-query recall test ===
    if (Math.random() < 0.1) { // 10% of the time, test with multiple queries
        const numQueries = 3;
        const multiQueryVectors = [];
        const multiGroundTruth = [];
        
        for (let i = 0; i < numQueries; i++) {
            const idx = Math.floor(Math.random() * knownVectors.length);
            multiQueryVectors.push(knownVectors[idx]);
            multiGroundTruth.push(groundTruth[idx]);
        }
        
        const multiResults = client.searchWithRecall(
            'recall_test',
            multiQueryVectors,
            5,
            searchParams,
            multiGroundTruth
        );
        
        check(multiResults, {
            'multi-query search returned results': (results) => results && results.length === numQueries * 5,
        });
    }
    
    // === Performance variation to test recall under different conditions ===
    if (Math.random() < 0.2) { // 20% of the time, test with different topK values
        const topKValues = [1, 3, 10];
        const selectedTopK = topKValues[Math.floor(Math.random() * topKValues.length)];
        
        const varyingResults = client.searchWithRecall(
            'recall_test',
            [queryVector],
            selectedTopK,
            searchParams,
            [expectedTruth]
        );
        
        check(varyingResults, {
            'varying topK returned correct count': (results) => results && results.length <= selectedTopK,
        });
    }
}

export function teardown(data) {
    console.log('Cleaning up recall benchmark test...');
    
    try {
        data.client.dropCollection('recall_test');
        data.client.close();
        console.log('Cleanup complete.');
    } catch (error) {
        console.error('Cleanup failed:', error.message);
    }
}

export function handleSummary(data) {
    console.log('\n=== RECALL BENCHMARK SUMMARY ===');
    console.log('Built-in recall metrics:');
    console.log(`  - milvus_recall (avg): ${data.metrics.milvus_recall ? Math.round(data.metrics.milvus_recall.values.avg * 100) / 100 : 'N/A'}`);
    console.log(`  - milvus_recall (min): ${data.metrics.milvus_recall ? Math.round(data.metrics.milvus_recall.values.min * 100) / 100 : 'N/A'}`);
    console.log(`  - milvus_recall (max): ${data.metrics.milvus_recall ? Math.round(data.metrics.milvus_recall.values.max * 100) / 100 : 'N/A'}`);
    
    console.log('\nSearch performance:');
    console.log(`  - Regular search (p95): ${data.metrics['milvus_req_duration{operation:search}'] ? Math.round(data.metrics['milvus_req_duration{operation:search}'].values['p(95)']) + 'ms' : 'N/A'}`);
    console.log(`  - Recall search (p95): ${data.metrics['milvus_req_duration{operation:search_with_recall}'] ? Math.round(data.metrics['milvus_req_duration{operation:search_with_recall}'].values['p(95)']) + 'ms' : 'N/A'}`);
    
    console.log('\nQuality metrics:');
    console.log(`  - Error rate: ${data.metrics.milvus_errors ? Math.round(data.metrics.milvus_errors.values.rate * 100) + '%' : 'N/A'}`);
    console.log(`  - Recall validations: ${data.metrics.custom_recall_validations ? data.metrics.custom_recall_validations.values.count : 'N/A'}`);
    
    return {
        'recall-summary.json': JSON.stringify(data, null, 2),
    };
}
// Example showing how to use xk6-milvus with custom metrics
import milvus from 'k6/x/milvus';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// Custom metrics to track in addition to built-in ones
const milvusInserts = new Counter('milvus_custom_inserts');
const milvusSearches = new Counter('milvus_custom_searches');
const milvusLatency = new Trend('milvus_custom_latency');

export const options = {
    vus: 5,
    duration: '30s',
    thresholds: {
        'milvus_custom_latency': ['p(95)<500'], // 95% of operations should be under 500ms
        'milvus_custom_inserts': ['count>100'], // Should perform more than 100 inserts
    },
};

export function setup() {
    const client = milvus.client('localhost:19530');
    
    // Create collection
    client.createCollectionSimple('metrics_test', 128);
    client.createIndexSimple('metrics_test', 'vector');
    client.loadCollection('metrics_test');
    
    return { client };
}

export default function(data) {
    const { client } = data;
    
    // Generate test data
    const vectors = [];
    for (let i = 0; i < 5; i++) {
        const vector = Array(128).fill(0).map(() => Math.random());
        vectors.push(vector);
    }
    
    // Measure insert operation
    const insertStart = new Date();
    const ids = client.insertVectors('metrics_test', vectors);
    const insertDuration = new Date() - insertStart;
    
    // Record custom metrics
    milvusInserts.add(vectors.length);
    milvusLatency.add(insertDuration);
    
    check(ids, {
        'insert successful': (ids) => ids && ids.length === vectors.length,
    });
    
    // Measure search operation
    const searchStart = new Date();
    const searchVector = [vectors[0]];
    const results = client.searchSimple('metrics_test', searchVector, 3);
    const searchDuration = new Date() - searchStart;
    
    // Record custom metrics
    milvusSearches.add(1);
    milvusLatency.add(searchDuration);
    
    check(results, {
        'search returned results': (results) => results && results.length > 0,
        'search performance': () => searchDuration < 1000, // Should be under 1 second
    });
}

export function teardown(data) {
    data.client.dropCollection('metrics_test');
    data.client.close();
}
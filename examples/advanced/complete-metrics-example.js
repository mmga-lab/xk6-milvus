// Complete example showing xk6-milvus with comprehensive metrics tracking
import milvus from 'k6/x/milvus';
import { check } from 'k6';
import { Counter, Trend, Rate } from 'k6/metrics';

// Additional JavaScript-level metrics (complementing Go-level metrics)
const customInserts = new Counter('custom_milvus_inserts');
const customSearches = new Counter('custom_milvus_searches');
const customLatency = new Trend('custom_milvus_latency');
const customErrorRate = new Rate('custom_milvus_error_rate');

export const options = {
    vus: 3,
    duration: '20s',
    thresholds: {
        // Built-in Go metrics from the extension
        'milvus_reqs': ['count>10'],                    // Should perform more than 10 requests
        'milvus_req_duration': ['p(95)<1000'],         // 95% of operations under 1s
        'milvus_vectors': ['count>50'],                // Should process more than 50 vectors
        'milvus_errors': ['rate<0.1'],                 // Error rate under 10%
        
        // Custom JavaScript metrics
        'custom_milvus_latency': ['p(90)<500'],        // 90% of custom measured latency under 500ms
        'custom_milvus_error_rate': ['rate<0.05'],     // Custom error rate under 5%
        
        // Standard k6 checks
        'checks': ['rate>0.9'],                        // 90% of checks should pass
    },
};

export function setup() {
    console.log('Setting up Milvus test with comprehensive metrics...');
    
    const client = milvus.client(); // Will use MILVUS_HOST env var or default to localhost:19530
    
    // Create collection with auto-generated schema (simple approach)
    client.createCollectionSimple('metrics_test', 128);
    client.createIndexSimple('metrics_test', 'vector');
    client.loadCollection('metrics_test');
    
    console.log('Setup complete. Collection "metrics_test" is ready.');
    
    return { client };
}

export default function(data) {
    const { client } = data;
    
    // Generate test vectors
    const batchSize = 3;
    const vectors = [];
    for (let i = 0; i < batchSize; i++) {
        const vector = Array(128).fill(0).map(() => Math.random() * 2 - 1); // Range: -1 to 1
        vectors.push(vector);
    }
    
    // === INSERT OPERATION WITH DUAL METRICS ===
    const insertStart = new Date();
    let insertSuccess = true;
    let insertedIds;
    
    try {
        // This will emit Go-level metrics: milvus_reqs, milvus_req_duration, milvus_vectors, milvus_errors
        insertedIds = client.insertVectors('metrics_test', vectors);
        
        // Verify insert
        check(insertedIds, {
            'insert returned IDs': (ids) => ids && ids.length === batchSize,
            'insert IDs are valid': (ids) => ids && ids.every(id => typeof id === 'number'),
        });
        
    } catch (error) {
        console.error('Insert failed:', error.message);
        insertSuccess = false;
        customErrorRate.add(1); // Record error
    }
    
    // Record custom JavaScript metrics for insert
    const insertDuration = new Date() - insertStart;
    customInserts.add(batchSize);
    customLatency.add(insertDuration);
    customErrorRate.add(insertSuccess ? 0 : 1);
    
    // === SEARCH OPERATION WITH DUAL METRICS ===
    if (insertSuccess) {
        const searchStart = new Date();
        let searchSuccess = true;
        
        try {
            // Use first vector as query vector
            const queryVectors = [vectors[0]];
            
            // This will emit Go-level metrics: milvus_reqs, milvus_req_duration, milvus_errors
            const results = client.searchSimple('metrics_test', queryVectors, 5);
            
            // Verify search results
            check(results, {
                'search returned results': (results) => results && results.length > 0,
                'search results have scores': (results) => results && results[0] && typeof results[0].score === 'number',
                'search results have IDs': (results) => results && results[0] && typeof results[0].id === 'number',
                'search score is reasonable': (results) => results && results[0] && results[0].score >= 0 && results[0].score <= 1,
            });
            
        } catch (error) {
            console.error('Search failed:', error.message);
            searchSuccess = false;
            customErrorRate.add(1); // Record error
        }
        
        // Record custom JavaScript metrics for search
        const searchDuration = new Date() - searchStart;
        customSearches.add(1);
        customLatency.add(searchDuration);
        customErrorRate.add(searchSuccess ? 0 : 1);
    }
    
    // === PERFORMANCE VALIDATION ===
    // Add some variability to test threshold conditions
    if (Math.random() < 0.1) {
        // Occasionally add a small delay to test latency thresholds
        const delay = Math.random() * 100; // 0-100ms
        const start = Date.now();
        while (Date.now() - start < delay) {
            // Busy wait
        }
    }
}

export function teardown(data) {
    console.log('Cleaning up...');
    
    try {
        data.client.dropCollection('metrics_test');
        data.client.close();
        console.log('Cleanup complete.');
    } catch (error) {
        console.error('Cleanup failed:', error.message);
    }
}

export function handleSummary(data) {
    console.log('\n=== METRICS SUMMARY ===');
    console.log('Built-in Go metrics from xk6-milvus:');
    console.log(`  - milvus_reqs: ${data.metrics.milvus_reqs ? data.metrics.milvus_reqs.values.count : 'N/A'}`);
    console.log(`  - milvus_req_duration (avg): ${data.metrics.milvus_req_duration ? Math.round(data.metrics.milvus_req_duration.values.avg) + 'ms' : 'N/A'}`);
    console.log(`  - milvus_vectors: ${data.metrics.milvus_vectors ? data.metrics.milvus_vectors.values.count : 'N/A'}`);
    console.log(`  - milvus_errors (rate): ${data.metrics.milvus_errors ? Math.round(data.metrics.milvus_errors.values.rate * 100) + '%' : 'N/A'}`);
    
    console.log('\nCustom JavaScript metrics:');
    console.log(`  - custom_milvus_inserts: ${data.metrics.custom_milvus_inserts ? data.metrics.custom_milvus_inserts.values.count : 'N/A'}`);
    console.log(`  - custom_milvus_searches: ${data.metrics.custom_milvus_searches ? data.metrics.custom_milvus_searches.values.count : 'N/A'}`);
    console.log(`  - custom_milvus_latency (p95): ${data.metrics.custom_milvus_latency ? Math.round(data.metrics.custom_milvus_latency.values['p(95)']) + 'ms' : 'N/A'}`);
    console.log(`  - custom_milvus_error_rate: ${data.metrics.custom_milvus_error_rate ? Math.round(data.metrics.custom_milvus_error_rate.values.rate * 100) + '%' : 'N/A'}`);
    
    return {
        'summary.json': JSON.stringify(data, null, 2),
    };
}
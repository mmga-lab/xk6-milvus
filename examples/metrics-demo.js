import milvus from 'k6/x/milvus';
import { check } from 'k6';

export const options = {
    vus: 2,
    iterations: 10,
    thresholds: {
        // Operation duration thresholds
        'milvus_operation_duration': ['p(95)<500'],  // 95% of operations under 500ms
        'milvus_operation_duration{operation:insert}': ['p(95)<200'],  // Insert under 200ms
        'milvus_operation_duration{operation:search}': ['p(95)<100'],  // Search under 100ms

        // Error rate threshold
        'milvus_errors': ['rate<0.01'],  // Less than 1% errors

        // Search quality threshold
        'milvus_search_recall': ['avg>=0.95'],  // Average recall >= 95%

        // Empty results threshold
        'milvus_empty_results': ['rate<0.1'],  // Less than 10% empty results

        // Throughput thresholds (NEW!)
        'milvus_throughput_mbps{operation:insert}': ['value>10'],  // Insert throughput > 10 MB/s
        'milvus_throughput_rows_per_second{operation:insert}': ['value>1000'],  // > 1000 rows/s
        'milvus_throughput_mbps{operation:upsert}': ['value>8'],  // Upsert throughput > 8 MB/s

        // Collection operations
        'milvus_collections_created': ['count<=10'],  // Limit collection creation
    },
};

// Generate random vector
function randomVector(dim) {
    const vector = [];
    for (let i = 0; i < dim; i++) {
        vector.push(Math.random());
    }
    return vector;
}

export default function () {
    const MILVUS_HOST = __ENV.MILVUS_HOST || 'localhost:19530';
    const collectionName = `metrics_demo_${__VU}_${__ITER}`;

    // Connect to Milvus
    const client = milvus.client(MILVUS_HOST);

    try {
        // 1. Create Collection (metrics: milvus_collections_created, milvus_operation_duration)
        const createResult = client.createCollection({
            name: collectionName,
            fields: [
                { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
                { name: 'vector', dataType: 'FloatVector', dimension: 128 },
                { name: 'price', dataType: 'Float' },
            ],
        });

        check(createResult, {
            'collection created': (r) => r.success === true,
        });

        // 2. Insert Data (metrics: milvus_rows_inserted, milvus_batch_size, milvus_vector_dimension)
        const vectors = [];
        const prices = [];
        const batchSize = 100;

        for (let i = 0; i < batchSize; i++) {
            vectors.push(randomVector(128));
            prices.push(Math.random() * 1000);
        }

        const insertResult = client.insert({
            vector: vectors,
            price: prices,
        }, collectionName);

        check(insertResult, {
            'insert successful': (r) => r.success === true,
            'inserted 100 rows': (r) => r.result.insert_count === batchSize,
        });

        // 3. Create Index (metrics: milvus_index_build_duration)
        const indexResult = client.createIndex('vector', {
            indexType: 'HNSW',
            metricType: 'L2',
            M: 16,
            efConstruction: 200,
        }, collectionName);

        check(indexResult, {
            'index created': (r) => r.success === true,
        });

        // 4. Load Collection (metrics: milvus_collection_loaded)
        const loadResult = client.loadCollection(collectionName);

        check(loadResult, {
            'collection loaded': (r) => r.success === true,
        });

        // 5. Search (metrics: milvus_search_recall, milvus_search_topk, milvus_filter_used)
        const searchVectors = [randomVector(128)];
        const searchResult = client.search(searchVectors, 10, {
            vectorField: 'vector',
            outputFields: ['price'],
            metricType: 'L2',
            expr: 'price > 100',  // Filter usage tracked
        }, collectionName);

        check(searchResult, {
            'search successful': (r) => r.success === true,
            'high recall': (r) => r.recall >= 0.9,
            'not empty': (r) => r.empty === false,
        });

        // 6. Query (metrics: milvus_output_fields_count, milvus_result_count)
        const queryResult = client.query('price > 500', ['id', 'price'], collectionName);

        check(queryResult, {
            'query successful': (r) => r.success === true,
        });

        // 7. Update with Upsert (metrics: milvus_rows_inserted)
        const upsertResult = client.upsert({
            vector: [randomVector(128)],
            price: [999.99],
        }, collectionName);

        check(upsertResult, {
            'upsert successful': (r) => r.success === true,
        });

        // 8. Delete (metrics: milvus_rows_deleted)
        const deleteResult = client.delete('price < 50', collectionName);

        check(deleteResult, {
            'delete successful': (r) => r.success === true,
        });

        // 9. Release Collection (metrics: milvus_collection_loaded = 0)
        const releaseResult = client.releaseCollection(collectionName);

        check(releaseResult, {
            'collection released': (r) => r.success === true,
        });

        // 10. Drop Collection
        const dropResult = client.dropCollection(collectionName);

        check(dropResult, {
            'collection dropped': (r) => r.success === true,
        });

    } finally {
        client.close();
    }
}

// Custom summary to display metrics
export function handleSummary(data) {
    // Standard k6 output
    return {
        'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    };
}

// Simple text summary helper
function textSummary(data, options) {
    const indent = options?.indent || '';
    const lines = [];

    lines.push('\n' + indent + '=== Milvus Metrics Summary ===\n');

    // Extract key metrics
    const metrics = data.metrics;

    // Operation Duration
    if (metrics.milvus_operation_duration) {
        const m = metrics.milvus_operation_duration;
        lines.push(indent + 'Operation Duration:');
        lines.push(indent + `  avg=${m.values.avg?.toFixed(2)}ms p95=${m.values['p(95)']?.toFixed(2)}ms`);
    }

    // Search Recall
    if (metrics.milvus_search_recall) {
        const m = metrics.milvus_search_recall;
        lines.push(indent + 'Search Recall:');
        lines.push(indent + `  avg=${m.values.avg?.toFixed(4)} min=${m.values.min?.toFixed(4)} max=${m.values.max?.toFixed(4)}`);
    }

    // Operations Total
    if (metrics.milvus_operations_total) {
        const m = metrics.milvus_operations_total;
        lines.push(indent + `Total Operations: ${m.values.count}`);
    }

    // Rows Inserted/Deleted
    if (metrics.milvus_rows_inserted) {
        lines.push(indent + `Rows Inserted: ${metrics.milvus_rows_inserted.values.count}`);
    }
    if (metrics.milvus_rows_deleted) {
        lines.push(indent + `Rows Deleted: ${metrics.milvus_rows_deleted.values.count}`);
    }

    // Error Rate
    if (metrics.milvus_errors) {
        const m = metrics.milvus_errors;
        const rate = (m.values.rate * 100).toFixed(2);
        lines.push(indent + `Error Rate: ${rate}%`);
    }

    // Collections Created
    if (metrics.milvus_collections_created) {
        lines.push(indent + `Collections Created: ${metrics.milvus_collections_created.values.count}`);
    }

    // Throughput Metrics (NEW!)
    if (metrics.milvus_throughput_mbps) {
        const m = metrics.milvus_throughput_mbps;
        lines.push(indent + 'Throughput (MB/s):');
        lines.push(indent + `  avg=${m.values.avg?.toFixed(2)} min=${m.values.min?.toFixed(2)} max=${m.values.max?.toFixed(2)}`);
    }
    if (metrics.milvus_throughput_rows_per_second) {
        const m = metrics.milvus_throughput_rows_per_second;
        lines.push(indent + 'Throughput (Rows/s):');
        lines.push(indent + `  avg=${m.values.avg?.toFixed(0)} min=${m.values.min?.toFixed(0)} max=${m.values.max?.toFixed(0)}`);
    }

    lines.push('');

    return lines.join('\n');
}

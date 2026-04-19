// Workload Checker — Diverse Milvus workload generator with all-datatype schema
//
// Generates weighted mixed workloads (Int64, Float, VarChar, JSON, Array, StructArray,
// FloatVector, BM25 SparseVector) across 17 operation types using configurable profiles.
//
// Usage:
//   ./k6 run examples/workload-checker.js -e MILVUS_HOST=host:19530 -e PROFILE=all -e DURATION=10m
//   ./k6 run examples/workload-checker.js -e PROFILE=search_heavy -e VUS_DATA=10
//   ./k6 run examples/workload-checker.js -e PROFILE=custom -e WEIGHTS_DATA='{"search":80,"insert":20}'

import milvus from 'k6/x/milvus';
import { check, sleep } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';

// ============================================================================
// Section A: Workload Profiles & Configuration
// ============================================================================

const PROFILES = {
    // Read-intensive: mostly search/query operations
    search_heavy: {
        data: { search: 40, full_text_search: 15, hybrid_search: 10, query: 15, text_match: 10, json_query: 5, insert: 5 },
        ddl: {},
    },
    // Write-intensive: mostly insert/upsert/delete operations
    write_heavy: {
        data: { insert: 35, upsert: 25, delete: 15, search: 15, query: 10 },
        ddl: {},
    },
    // Balanced read/write with DDL operations
    mixed: {
        data: { search: 20, full_text_search: 8, hybrid_search: 8, query: 10, text_match: 5, json_query: 5,
                insert: 20, upsert: 12, delete: 7 },
        ddl: { collection_create: 20, collection_drop: 20, collection_load: 15, collection_release: 15,
               partition_create: 10, partition_drop: 10, index_create: 5, index_drop: 5 },
    },
    // All operation types with equal emphasis — maximum coverage
    all: {
        data: { search: 18, full_text_search: 8, hybrid_search: 8, query: 8, text_match: 5, json_query: 5,
                insert: 20, upsert: 12, delete: 10 },
        ddl: { collection_create: 15, collection_drop: 15, collection_load: 10, collection_release: 10,
               partition_create: 15, partition_drop: 15, index_create: 10, index_drop: 10 },
    },
    // DML only: data manipulation without DDL
    dml: {
        data: { search: 18, full_text_search: 8, hybrid_search: 8, query: 10, text_match: 5, json_query: 5,
                insert: 22, upsert: 12, delete: 10 },
        ddl: {},
    },
    // DDL only: schema/index/partition management
    ddl: {
        data: {},
        ddl: { collection_create: 20, collection_drop: 20, collection_load: 10, collection_release: 10,
               partition_create: 15, partition_drop: 15, index_create: 5, index_drop: 5 },
    },
};

const CONFIG = {
    host: __ENV.MILVUS_HOST || 'localhost:19530',
    token: __ENV.MILVUS_TOKEN || '',
    dim: parseInt(__ENV.VECTOR_DIM || '128'),
    batchSize: parseInt(__ENV.BATCH_SIZE || '100'),
    topK: parseInt(__ENV.TOPK || '10'),
    duration: __ENV.DURATION || '5m',
    vusData: parseInt(__ENV.VUS_DATA || '3'),
    vusDdl: parseInt(__ENV.VUS_DDL || '1'),
    initialRows: parseInt(__ENV.INITIAL_ROWS || '3000'),
};

function resolveProfile() {
    const name = __ENV.PROFILE || 'mixed';
    if (name === 'custom') {
        return {
            data: __ENV.WEIGHTS_DATA ? JSON.parse(__ENV.WEIGHTS_DATA) : {},
            ddl: __ENV.WEIGHTS_DDL ? JSON.parse(__ENV.WEIGHTS_DDL) : {},
        };
    }
    return PROFILES[name] || PROFILES.mixed;
}

const activeProfile = resolveProfile();
const hasDataOps = Object.keys(activeProfile.data).length > 0;
const hasDdlOps = Object.keys(activeProfile.ddl).length > 0;

// ============================================================================
// Section B: Metrics — tagged metrics
// ============================================================================

const opLatency = new Trend('checker_op_latency', true);
const opCount = new Counter('checker_op_count');
const opErrors = new Rate('checker_op_errors');

function recordOp(opName, result) {
    const tags = { op: opName };
    opLatency.add(result.response_time_ms, tags);
    opCount.add(1, tags);
    opErrors.add(!result.success, tags);
}

// ============================================================================
// Section C: Data Generation — matching Python gen_row_data_by_schema
// ============================================================================

// Word pool for text generation (BM25 needs real words for tokenization)
const WORDS = [
    'milvus', 'vector', 'database', 'search', 'index', 'collection', 'partition',
    'segment', 'query', 'insert', 'delete', 'upsert', 'flush', 'compact',
    'load', 'release', 'schema', 'field', 'dimension', 'metric', 'distance',
    'similarity', 'embedding', 'neural', 'network', 'model', 'training',
    'inference', 'batch', 'stream', 'cluster', 'node', 'replica', 'shard',
    'performance', 'latency', 'throughput', 'scalable', 'distributed', 'cloud',
    'hybrid', 'sparse', 'dense', 'float', 'binary', 'scalar', 'filter',
    'expression', 'boolean', 'range', 'match', 'phrase', 'token', 'analyzer',
];

const NAMES = ['Alice', 'Bob', 'Charlie', 'Diana', 'Eve', 'Frank', 'Grace', 'Hank',
               'Iris', 'Jack', 'Kate', 'Leo', 'Mia', 'Nick', 'Olivia', 'Paul'];

const ADDRESSES = ['123 Main St', '456 Oak Ave', '789 Pine Rd', '321 Elm Blvd',
                   '654 Cedar Ln', '987 Birch Way', '147 Maple Dr', '258 Walnut Ct'];

function randomWord() {
    return WORDS[Math.floor(Math.random() * WORDS.length)];
}

function randomSentence(wordCount) {
    const n = wordCount || (5 + Math.floor(Math.random() * 15));
    const words = [];
    for (let i = 0; i < n; i++) {
        words.push(randomWord());
    }
    return words.join(' ');
}

function randomString(len) {
    const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
    let s = '';
    for (let i = 0; i < len; i++) {
        s += chars[Math.floor(Math.random() * chars.length)];
    }
    return s;
}

function generateVector(dim) {
    return Array.from({ length: dim }, () => Math.random());
}

function generateVectors(count, dim) {
    return Array.from({ length: count }, () => generateVector(dim));
}

// Timestamp-based IDs within JS safe integer range.
// Date.now() ~1.7e12, * 1000 = ~1.7e15 < MAX_SAFE_INTEGER(9e15)
// VU offset (0-999) goes into the hundreds digit, iter provides uniqueness across calls.
function generateTimestampIds(count) {
    const vu = typeof __VU !== 'undefined' ? __VU : 0;
    const iter = typeof __ITER !== 'undefined' ? __ITER : 0;
    const base = Date.now() * 1000 + (vu % 100) * 10 + (iter % 10);
    const ids = [];
    for (let i = 0; i < count; i++) {
        ids.push(base + i * 1000);
    }
    return ids;
}

// Generate column-based data matching Python's gen_all_datatype_collection_schema
function generateCheckerData(count, dim) {
    const data = {
        int64: generateTimestampIds(count),
        float: Array.from({ length: count }, () => Math.random() * 999.99 + 0.01),
        varchar: Array.from({ length: count }, () => randomString(10 + Math.floor(Math.random() * 20))),
        document: Array.from({ length: count }, () => randomSentence(10 + Math.floor(Math.random() * 20))),
        text: Array.from({ length: count }, () => randomSentence(5 + Math.floor(Math.random() * 15))),
        json_field: Array.from({ length: count }, () => ({
            name: NAMES[Math.floor(Math.random() * NAMES.length)],
            address: ADDRESSES[Math.floor(Math.random() * ADDRESSES.length)],
            count: Math.floor(Math.random() * 100),
        })),
        array_int: Array.from({ length: count }, () =>
            Array.from({ length: 5 + Math.floor(Math.random() * 5) }, () => Math.floor(Math.random() * 1000))),
        array_float: Array.from({ length: count }, () =>
            Array.from({ length: 5 + Math.floor(Math.random() * 5) }, () => Math.random() * 100)),
        array_varchar: Array.from({ length: count }, () =>
            Array.from({ length: 3 + Math.floor(Math.random() * 3) }, () => randomString(8))),
        array_bool: Array.from({ length: count }, () =>
            Array.from({ length: 5 + Math.floor(Math.random() * 5) }, () => Math.random() > 0.5)),
        float_vector: generateVectors(count, dim),
        // text_sparse_emb is BM25 function output — skip in insert
        array_struct: Array.from({ length: count }, () => {
            const n = 1 + Math.floor(Math.random() * 5);
            return Array.from({ length: n }, () => ({
                name: NAMES[Math.floor(Math.random() * NAMES.length)],
                age: 18 + Math.floor(Math.random() * 60),
            }));
        }),
    };
    return data;
}

function weightedSelect(weights) {
    const entries = Object.entries(weights);
    if (entries.length === 0) return null;
    const total = entries.reduce((sum, [, w]) => sum + w, 0);
    if (total === 0) return entries[0][0];
    let r = Math.random() * total;
    for (const [name, weight] of entries) {
        r -= weight;
        if (r <= 0) return name;
    }
    return entries[entries.length - 1][0];
}

// Minimal schema for DDL test collections
function minimalSchema(name) {
    return {
        name: name,
        fields: [
            { name: 'id', dataType: 'Int64', isPrimaryKey: true },
            { name: 'v', dataType: 'FloatVector', dimension: CONFIG.dim },
        ],
    };
}

function uniqueName(prefix) {
    return `${prefix}_${Date.now()}_${Math.floor(Math.random() * 100000)}`;
}

// ============================================================================
// Section D: Schema Definition — matching Python gen_all_datatype_collection_schema
// ============================================================================

function checkerSchema(name) {
    const fields = [
        { name: 'int64', dataType: 'Int64', isPrimaryKey: true },
        { name: 'float', dataType: 'Float', nullable: true },
        { name: 'varchar', dataType: 'VarChar', maxLength: 500, nullable: true },
        { name: 'document', dataType: 'VarChar', maxLength: 2000,
          enableAnalyzer: true, enableMatch: true, nullable: true },
        { name: 'text', dataType: 'VarChar', maxLength: 2000,
          enableAnalyzer: true, enableMatch: true,
          analyzerParams: { tokenizer: 'standard' }, nullable: true },
        { name: 'json_field', dataType: 'JSON', nullable: true },
        { name: 'array_int', dataType: 'Array', elementType: 'Int64', maxCapacity: 100 },
        { name: 'array_float', dataType: 'Array', elementType: 'Float', maxCapacity: 100 },
        { name: 'array_varchar', dataType: 'Array', elementType: 'VarChar', maxLength: 200, maxCapacity: 100 },
        { name: 'array_bool', dataType: 'Array', elementType: 'Bool', maxCapacity: 100 },
        { name: 'float_vector', dataType: 'FloatVector', dimension: CONFIG.dim },
        { name: 'text_sparse_emb', dataType: 'SparseFloatVector' },
        { name: 'array_struct', dataType: 'Array', elementType: 'Struct', maxCapacity: 10,
          structFields: [
              { name: 'name', dataType: 'VarChar', maxLength: 200 },
              { name: 'age', dataType: 'Int64' },
          ] },
    ];
    return {
        name: name,
        fields: fields,
        functions: [
            { name: 'text_bm25', functionType: 'BM25',
              inputFieldNames: ['text'], outputFieldNames: ['text_sparse_emb'] },
        ],
    };
}

// ============================================================================
// Section E: Operation Functions
// ============================================================================

// --- Data-plane: read operations ---

function doSearch(data) {
    const client = milvus.getClient(CONFIG.host, data.sharedCollection, CONFIG.token);
    const result = client.search(
        [generateVector(CONFIG.dim)],
        CONFIG.topK,
        {
            vectorField: 'float_vector',
            metricType: 'L2',
            outputFields: ['varchar', 'float'],
            params: { ef: 64 },
        },
    );
    recordOp('search', result);
    check(result, { 'search ok': (r) => r.success });
}

function doFullTextSearch(data) {
    // BM25 full-text search: pass text query string to search
    const client = milvus.getClient(CONFIG.host, data.sharedCollection, CONFIG.token);
    const queryText = randomSentence(3);
    const result = client.search(
        [queryText],
        CONFIG.topK,
        {
            vectorField: 'text_sparse_emb',
            metricType: 'BM25',
            outputFields: ['int64', 'varchar'],
        },
    );
    recordOp('full_text_search', result);
    check(result, { 'full_text_search ok': (r) => r.success });
}

function doHybridSearch(data) {
    // Hybrid search: combines dense vector + BM25 text with RRF reranking
    const client = milvus.getClient(CONFIG.host, data.sharedCollection, CONFIG.token);
    const result = client.hybridSearch(
        [
            {
                vectors: [generateVector(CONFIG.dim)],
                vectorField: 'float_vector',
                limit: 20,
                params: { metricType: 'L2' },
            },
            {
                vectors: [randomSentence(3)],
                vectorField: 'text_sparse_emb',
                limit: 20,
                params: { metricType: 'BM25' },
            },
        ],
        { type: 'rrf', params: { k: 60 } },
        CONFIG.topK,
        ['varchar', 'float'],
    );
    recordOp('hybrid_search', result);
    check(result, { 'hybrid_search ok': (r) => r.success });
}

function doQuery(data) {
    const client = milvus.getClient(CONFIG.host, data.sharedCollection, CONFIG.token);
    const filters = [
        'int64 > 0',
        'float > 50',
        'float < 100',
    ];
    const filter = filters[Math.floor(Math.random() * filters.length)];
    const result = client.query(filter, ['int64', 'varchar', 'float']);
    recordOp('query', result);
    check(result, { 'query ok': (r) => r.success });
}

function doTextMatch(data) {
    const client = milvus.getClient(CONFIG.host, data.sharedCollection, CONFIG.token);
    const keyword = randomWord();
    const filter = `TEXT_MATCH(text, '${keyword}')`;
    const result = client.search(
        [generateVector(CONFIG.dim)],
        CONFIG.topK,
        {
            vectorField: 'float_vector',
            metricType: 'L2',
            outputFields: ['text'],
            expr: filter,
        },
    );
    recordOp('text_match', result);
    check(result, { 'text_match ok': (r) => r.success });
}

function doJsonQuery(data) {
    const client = milvus.getClient(CONFIG.host, data.sharedCollection, CONFIG.token);
    const name = NAMES[Math.floor(Math.random() * NAMES.length)];
    const count = Math.floor(Math.random() * 100);
    const filters = [
        `json_field['name'] == '${name}'`,
        `json_field['count'] <= ${count}`,
    ];
    const filter = filters[Math.floor(Math.random() * filters.length)];
    const result = client.query(filter, ['int64', 'json_field']);
    recordOp('json_query', result);
    check(result, { 'json_query ok': (r) => r.success });
}

// --- Data-plane: write operations ---

function doInsert(data) {
    const client = milvus.getClient(CONFIG.host, data.sharedCollection, CONFIG.token);
    const result = client.insert(generateCheckerData(CONFIG.batchSize, CONFIG.dim));
    recordOp('insert', result);
    check(result, { 'insert ok': (r) => r.success });
}

function doUpsert(data) {
    const client = milvus.getClient(CONFIG.host, data.sharedCollection, CONFIG.token);
    // Upsert with explicit PK (half existing range, half new)
    const batchData = generateCheckerData(CONFIG.batchSize, CONFIG.dim);
    const result = client.upsert(batchData);
    recordOp('upsert', result);
    check(result, { 'upsert ok': (r) => r.success });
}

function doDelete(data) {
    const client = milvus.getClient(CONFIG.host, data.sharedCollection, CONFIG.token);
    // Delete by filter expression (simpler, no need to query IDs first)
    const minVal = Math.floor(Math.random() * 90);
    const result = client.delete(`float > ${minVal} && float < ${minVal + 5}`);
    recordOp('delete', result);
    check(result, { 'delete ok': (r) => r.success });
    // Replenish data
    client.insert(generateCheckerData(50, CONFIG.dim));
}

function doFlush(data) {
    // Milvus has a flush rate limiter (default 0.1/s). Errors from rate limiting are expected.
    const client = milvus.getClient(CONFIG.host, data.sharedCollection, CONFIG.token);
    const result = client.flush();
    recordOp('flush', result);
    check(result, { 'flush ok': (r) => r.success });
}

// --- DDL operations ---

function doCollectionCreate() {
    const client = milvus.client(CONFIG.host, CONFIG.token);
    const name = uniqueName('chk_cc');
    const result = client.createCollection(minimalSchema(name));
    recordOp('collection_create', result);
    check(result, { 'collection_create ok': (r) => r.success });
    if (result.success) client.dropCollection(name);
    client.close();
}

function doCollectionDrop() {
    const client = milvus.client(CONFIG.host, CONFIG.token);
    const name = uniqueName('chk_cd');
    client.createCollection(minimalSchema(name));
    const result = client.dropCollection(name);
    recordOp('collection_drop', result);
    check(result, { 'collection_drop ok': (r) => r.success });
    client.close();
}

function doCollectionLoad(data) {
    if (!data.loadCheckCollection) return;
    const client = milvus.getClient(CONFIG.host, data.loadCheckCollection, CONFIG.token);
    const result = client.loadCollection();
    recordOp('collection_load', result);
    check(result, { 'collection_load ok': (r) => r.success });
    if (result.success) client.releaseCollection();
}

function doCollectionRelease(data) {
    if (!data.loadCheckCollection) return;
    const client = milvus.getClient(CONFIG.host, data.loadCheckCollection, CONFIG.token);
    const result = client.releaseCollection();
    recordOp('collection_release', result);
    check(result, { 'collection_release ok': (r) => r.success });
}

function doPartitionCreate(data) {
    if (!data.partitionCheckCollection) return;
    const client = milvus.getClient(CONFIG.host, data.partitionCheckCollection, CONFIG.token);
    const name = uniqueName('part');
    const result = client.createPartition(name);
    recordOp('partition_create', result);
    check(result, { 'partition_create ok': (r) => r.success });
    if (result.success) client.dropPartition(name);
}

function doPartitionDrop(data) {
    if (!data.partitionCheckCollection) return;
    const client = milvus.getClient(CONFIG.host, data.partitionCheckCollection, CONFIG.token);
    const name = uniqueName('part_d');
    client.createPartition(name);
    const result = client.dropPartition(name);
    recordOp('partition_drop', result);
    check(result, { 'partition_drop ok': (r) => r.success });
}

function doIndexCreate() {
    const client = milvus.client(CONFIG.host, CONFIG.token);
    const name = uniqueName('chk_ic');
    client.createCollection(minimalSchema(name));
    const result = client.createIndex('v', {
        indexType: 'HNSW', metricType: 'L2', params: { M: 16, efConstruction: 200 },
    }, name);
    recordOp('index_create', result);
    check(result, { 'index_create ok': (r) => r.success });
    client.dropCollection(name);
    client.close();
}

function doIndexDrop() {
    const client = milvus.client(CONFIG.host, CONFIG.token);
    const name = uniqueName('chk_id');
    client.createCollection(minimalSchema(name));
    client.createIndex('v', {
        indexType: 'HNSW', metricType: 'L2', params: { M: 16, efConstruction: 200 },
    }, name);
    const result = client.dropIndex('v', name);
    recordOp('index_drop', result);
    check(result, { 'index_drop ok': (r) => r.success });
    client.dropCollection(name);
    client.close();
}

// ============================================================================
// Section F: k6 Options
// ============================================================================

function buildScenarios() {
    const scenarios = {};
    if (hasDataOps) {
        scenarios.data_ops = {
            executor: 'constant-vus',
            vus: CONFIG.vusData,
            duration: CONFIG.duration,
            exec: 'dataOps',
            tags: { plane: 'data' },
            gracefulStop: '30s',
        };
    }
    if (hasDdlOps) {
        scenarios.ddl_ops = {
            executor: 'constant-vus',
            vus: CONFIG.vusDdl,
            duration: CONFIG.duration,
            exec: 'ddlOps',
            tags: { plane: 'ddl' },
            gracefulStop: '30s',
        };
    }
    return scenarios;
}

export const options = {
    scenarios: buildScenarios(),
    thresholds: {
        'checker_op_errors': ['rate<0.15'],
        'checker_op_latency{op:search}': ['p(95)<500'],
        'checker_op_latency{op:insert}': ['p(95)<1000'],
        'checker_op_latency{op:query}': ['p(95)<500'],
    },
};

// ============================================================================
// Section G: Scenario Entry Functions
// ============================================================================

const DATA_OPS = {
    search: doSearch,
    full_text_search: doFullTextSearch,
    hybrid_search: doHybridSearch,
    query: doQuery,
    text_match: doTextMatch,
    json_query: doJsonQuery,
    insert: doInsert,
    upsert: doUpsert,
    delete: doDelete,
    flush: doFlush,
};

const DDL_OPS = {
    collection_create: doCollectionCreate,
    collection_drop: doCollectionDrop,
    collection_load: doCollectionLoad,
    collection_release: doCollectionRelease,
    partition_create: doPartitionCreate,
    partition_drop: doPartitionDrop,
    index_create: doIndexCreate,
    index_drop: doIndexDrop,
};

export function dataOps(data) {
    const op = weightedSelect(activeProfile.data);
    const fn = DATA_OPS[op];
    if (fn) fn(data);
    sleep(0.1);
}

export function ddlOps(data) {
    const op = weightedSelect(activeProfile.ddl);
    const fn = DDL_OPS[op];
    if (fn) fn(data);
    sleep(2);
}

// ============================================================================
// Section H: setup() — create collection with Python-matching schema
// ============================================================================

export function setup() {
    let client = milvus.client(CONFIG.host, CONFIG.token);
    const result = { collections: [] };
    const profileName = __ENV.PROFILE || 'mixed';

    console.log(`[setup] Profile: ${profileName}`);
    console.log(`[setup] Data ops: ${JSON.stringify(activeProfile.data)}`);
    console.log(`[setup] DDL ops: ${JSON.stringify(activeProfile.ddl)}`);

    // 1. Create shared collection with all-datatype schema
    if (hasDataOps) {
        const sharedName = `checker_shared_${Date.now()}`;
        console.log(`[setup] Creating shared collection: ${sharedName}`);

        const schema = checkerSchema(sharedName);
        const createResult = client.createCollection(schema);
        if (!createResult.success) {
            console.error(`[setup] Failed to create collection: ${createResult.error}`);
            client.close();
            return result;
        }
        result.collections.push(sharedName);

        // Create indexes matching Python checker
        console.log('[setup] Creating indexes...');

        client.createIndex('float_vector', {
            indexType: 'HNSW', metricType: 'L2',
            params: { M: 48, efConstruction: 500 },
        }, sharedName);

        client.createIndex('text_sparse_emb', {
            indexType: 'SPARSE_INVERTED_INDEX', metricType: 'BM25',
            params: { bm25_k1: 1.5, bm25_b: 0.75 },
        }, sharedName);

        client.loadCollection(sharedName);
        console.log(`[setup] Collection loaded: ${sharedName}`);

        // Use a fresh client for insert to refresh schema cache (struct array needs re-fetch)
        client.close();
        client = milvus.client(CONFIG.host, CONFIG.token);

        // Insert initial data
        const batchCount = Math.ceil(CONFIG.initialRows / CONFIG.batchSize);
        for (let i = 0; i < batchCount; i++) {
            const count = Math.min(CONFIG.batchSize, CONFIG.initialRows - i * CONFIG.batchSize);
            const insertResult = client.insert(generateCheckerData(count, CONFIG.dim), sharedName);
            if (!insertResult.success) {
                console.warn(`[setup] Insert batch ${i} failed: ${insertResult.error}`);
            }
        }
        console.log(`[setup] Inserted ${CONFIG.initialRows} initial rows`);

        result.sharedCollection = sharedName;
    }

    // 2. Dedicated collection for load/release checker
    const needsLoadCheck = 'collection_load' in activeProfile.ddl || 'collection_release' in activeProfile.ddl;
    if (needsLoadCheck) {
        const loadName = `checker_loadrel_${Date.now()}`;
        client.createCollection(minimalSchema(loadName));
        client.createIndex('v', { indexType: 'FLAT', metricType: 'L2' }, loadName);
        result.collections.push(loadName);
        result.loadCheckCollection = loadName;
    }

    // 3. Dedicated collection for partition checker
    const needsPartitionCheck = 'partition_create' in activeProfile.ddl || 'partition_drop' in activeProfile.ddl;
    if (needsPartitionCheck) {
        const partName = `checker_part_${Date.now()}`;
        client.createCollection(minimalSchema(partName));
        client.createIndex('v', { indexType: 'FLAT', metricType: 'L2' }, partName);
        result.collections.push(partName);
        result.partitionCheckCollection = partName;
    }

    client.close();
    console.log(`[setup] Complete. Collections: ${result.collections.join(', ')}`);
    return result;
}

// ============================================================================
// Section I: teardown()
// ============================================================================

export function teardown(data) {
    if (!data || !data.collections) return;
    const client = milvus.client(CONFIG.host, CONFIG.token);
    for (const name of data.collections) {
        const r = client.dropCollection(name);
        if (r.success) {
            console.log(`[teardown] Dropped: ${name}`);
        } else {
            console.warn(`[teardown] Failed to drop ${name}: ${r.error}`);
        }
    }
    client.close();
    console.log('[teardown] Complete');
}

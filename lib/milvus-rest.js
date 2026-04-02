// xk6-milvus REST Client
// JavaScript library wrapping k6's built-in HTTP module for Milvus RESTful v2 API.
// Provides the same OperationResult interface as the gRPC client for consistency.

import http from 'k6/http';

const API_PREFIX = '/v2/vectordb';

// Convert column-based data to row-based data for REST API
function columnsToRows(data) {
    const keys = Object.keys(data);
    if (keys.length === 0) return [];

    const length = Array.isArray(data[keys[0]]) ? data[keys[0]].length : 0;
    const rows = [];

    for (let i = 0; i < length; i++) {
        const row = {};
        for (const key of keys) {
            row[key] = data[key][i];
        }
        rows.push(row);
    }

    return rows;
}

// Build OperationResult matching the gRPC client's structure
function makeResult(success, responseTimeMs, result, error, extra) {
    const r = {
        success: success,
        response_time_ms: responseTimeMs,
        result: result,
        error: error || '',
    };
    if (extra) {
        if (extra.empty !== undefined) r.empty = extra.empty;
        if (extra.recall !== undefined) r.recall = extra.recall;
    }
    return r;
}

// Parse REST API response and return OperationResult
function parseResponse(res, startTime) {
    const elapsed = (Date.now() - startTime);
    let body;
    try {
        body = JSON.parse(res.body);
    } catch (e) {
        return makeResult(false, elapsed, null, `Failed to parse response: ${e.message}`);
    }

    if (body.code !== 0) {
        return makeResult(false, elapsed, null, body.message || `Error code: ${body.code}`);
    }

    return makeResult(true, elapsed, body.data, '');
}

// Convert our schema format to REST API schema format
function convertSchemaToRest(schema) {
    const restSchema = {
        collectionName: schema.name,
    };

    // If fields are provided, build custom schema
    if (schema.fields && schema.fields.length > 0) {
        const restFields = schema.fields.map(f => {
            const field = {
                fieldName: f.name,
                dataType: f.dataType,
            };

            if (f.isPrimaryKey) field.isPrimary = true;
            if (f.isAutoID) field.autoId = true;
            if (f.description) field.description = f.description;
            if (f.nullable) field.nullable = true;
            if (f.isPartitionKey) field.isPartitionKey = true;
            if (f.isClusteringKey) field.isClusteringKey = true;
            if (f.enableAnalyzer) field.enableAnalyzer = true;
            if (f.enableMatch) field.enableMatch = true;

            // elementTypeParams for dimension, max_length, etc.
            const typeParams = {};
            if (f.dimension) typeParams.dim = String(f.dimension);
            if (f.maxLength) typeParams.max_length = String(f.maxLength);

            if (f.analyzerParams) {
                field.analyzerParams = f.analyzerParams;
            }

            if (Object.keys(typeParams).length > 0) {
                field.elementTypeParams = typeParams;
            }

            return field;
        });

        restSchema.schema = {
            autoId: schema.fields.some(f => f.isAutoID),
            enableDynamicField: schema.enableDynamicField || false,
            fields: restFields,
        };

        // Functions (BM25, TextEmbedding, etc.)
        if (schema.functions && schema.functions.length > 0) {
            restSchema.schema.functions = schema.functions.map(fn => ({
                name: fn.name,
                type: fn.functionType,
                inputFieldNames: fn.inputFieldNames,
                outputFieldNames: fn.outputFieldNames,
                params: fn.params || {},
            }));
        }
    } else if (schema.dimension) {
        // Quick setup mode
        restSchema.dimension = schema.dimension;
        if (schema.metricType) restSchema.metricType = schema.metricType;
    }

    if (schema.numShards) restSchema.numShards = schema.numShards;

    return restSchema;
}

export class MilvusRestClient {
    constructor(baseUrl, options) {
        // Normalize base URL
        if (baseUrl.indexOf('http') !== 0) {
            baseUrl = 'http://' + baseUrl;
        }
        // Remove trailing slash
        if (baseUrl.endsWith('/')) {
            baseUrl = baseUrl.slice(0, -1);
        }
        this._baseUrl = baseUrl;
        this._defaultCollection = (options && options.collectionName) || '';
        this._dbName = (options && options.dbName) || '';

        // Build headers
        this._headers = {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
        };
        if (options && options.token) {
            this._headers['Authorization'] = 'Bearer ' + options.token;
        }

        // HTTP request params for k6
        this._params = {
            headers: this._headers,
        };
        if (options && options.timeout) {
            this._params.timeout = options.timeout;
        }
        if (options && options.tags) {
            this._params.tags = options.tags;
        }
    }

    // Internal: perform POST request to a REST endpoint
    _post(path, body) {
        const url = this._baseUrl + API_PREFIX + path;
        const startTime = Date.now();

        try {
            const res = http.post(url, JSON.stringify(body), this._params);
            return parseResponse(res, startTime);
        } catch (e) {
            const elapsed = Date.now() - startTime;
            return makeResult(false, elapsed, null, e.message || String(e));
        }
    }

    // Resolve collection name: explicit > default
    _collection(collectionName) {
        const name = collectionName || this._defaultCollection;
        if (!name) {
            throw new Error('Collection name is required. Pass it as an argument or use clientWithCollection().');
        }
        return name;
    }

    // Build base body with optional dbName
    _baseBody(collectionName) {
        const body = { collectionName: this._collection(collectionName) };
        if (this._dbName) body.dbName = this._dbName;
        return body;
    }

    // ==================== Collection Operations ====================

    listCollections(dbName) {
        const body = {};
        if (dbName || this._dbName) body.dbName = dbName || this._dbName;
        return this._post('/collections/list', body);
    }

    createCollection(schema) {
        const body = convertSchemaToRest(schema);
        if (this._dbName) body.dbName = this._dbName;
        return this._post('/collections/create', body);
    }

    describeCollection(collectionName) {
        return this._post('/collections/describe', this._baseBody(collectionName));
    }

    dropCollection(collectionName) {
        return this._post('/collections/drop', this._baseBody(collectionName));
    }

    hasCollection(collectionName) {
        const res = this._post('/collections/has', this._baseBody(collectionName));
        if (res.success && res.result) {
            // Wrap to match gRPC client format: { exists: true/false }
            res.result = { exists: res.result.has === true };
        }
        return res;
    }

    loadCollection(collectionName) {
        return this._post('/collections/load', this._baseBody(collectionName));
    }

    releaseCollection(collectionName) {
        return this._post('/collections/release', this._baseBody(collectionName));
    }

    getLoadState(collectionName) {
        return this._post('/collections/get_load_state', this._baseBody(collectionName));
    }

    getCollectionStats(collectionName) {
        return this._post('/collections/get_stats', this._baseBody(collectionName));
    }

    flush(collectionName) {
        return this._post('/collections/flush', this._baseBody(collectionName));
    }

    renameCollection(collectionName, newCollectionName) {
        const body = this._baseBody(collectionName);
        body.newCollectionName = newCollectionName;
        return this._post('/collections/rename', body);
    }

    // ==================== Data Operations ====================

    insert(data, collectionName) {
        const body = this._baseBody(collectionName);

        // Support both column-based (like gRPC client) and row-based data
        if (Array.isArray(data)) {
            body.data = data;
        } else {
            body.data = columnsToRows(data);
        }

        const res = this._post('/entities/insert', body);

        // Normalize result to match gRPC client format
        if (res.success && res.result) {
            res.result = {
                insert_count: res.result.insertCount || 0,
                ids: res.result.insertIds || [],
            };
        }
        return res;
    }

    upsert(data, collectionName) {
        const body = this._baseBody(collectionName);

        if (Array.isArray(data)) {
            body.data = data;
        } else {
            body.data = columnsToRows(data);
        }

        const res = this._post('/entities/upsert', body);

        if (res.success && res.result) {
            res.result = {
                upsert_count: res.result.upsertCount || 0,
                ids: res.result.upsertIds || [],
            };
        }
        return res;
    }

    delete(filter, collectionName) {
        const body = this._baseBody(collectionName);
        body.filter = filter;
        const res = this._post('/entities/delete', body);
        if (res.success) {
            res.result = { delete_count: 0 };
        }
        return res;
    }

    get(ids, outputFields, collectionName) {
        const body = this._baseBody(collectionName);
        body.id = ids;
        if (outputFields) body.outputFields = outputFields;
        return this._post('/entities/get', body);
    }

    // ==================== Search Operations ====================

    search(vectors, topK, params, collectionName) {
        const body = this._baseBody(collectionName);

        // Ensure vectors is array of arrays
        if (vectors.length > 0 && !Array.isArray(vectors[0]) && typeof vectors[0] === 'number') {
            body.data = [vectors];
        } else {
            body.data = vectors;
        }

        body.limit = topK;

        if (params) {
            if (params.vectorField) body.annsField = params.vectorField;
            if (params.metricType) body.metricType = params.metricType;
            if (params.outputFields) body.outputFields = params.outputFields;
            if (params.expr) body.filter = params.expr;
            if (params.offset) body.offset = params.offset;
            if (params.groupingField) body.groupingField = params.groupingField;
            if (params.params) body.searchParams = params.params;
        }

        const res = this._post('/entities/search', body);

        // Add empty flag
        if (res.success) {
            const results = res.result || [];
            res.empty = results.length === 0;
            res.recall = 0; // REST API does not return recall natively
        }

        return res;
    }

    query(filter, outputFields, collectionName, queryParams) {
        const body = this._baseBody(collectionName);
        body.filter = filter;
        if (outputFields) body.outputFields = outputFields;
        if (queryParams) {
            if (queryParams.limit) body.limit = queryParams.limit;
            if (queryParams.offset) body.offset = queryParams.offset;
        }

        const res = this._post('/entities/query', body);

        if (res.success) {
            res.empty = !res.result || res.result.length === 0;
        }

        return res;
    }

    hybridSearch(requests, reranker, limit, outputFields, collectionName) {
        const body = this._baseBody(collectionName);

        // Convert search requests to REST format
        body.search = requests.map(req => {
            const s = {
                data: req.vectors,
                annsField: req.vectorField,
                limit: req.limit,
            };
            if (req.params) {
                if (req.params.metricType) s.metricType = req.params.metricType;
                if (req.params.expr) s.filter = req.params.expr;
            }
            return s;
        });

        // Convert reranker to REST format
        if (reranker) {
            body.rerank = {
                strategy: reranker.type,
                params: reranker.params || {},
            };
        }

        body.limit = limit;
        if (outputFields) body.outputFields = outputFields;

        const res = this._post('/entities/hybrid_search', body);

        if (res.success) {
            const results = res.result || [];
            res.empty = results.length === 0;
            res.recall = 0;
        }

        return res;
    }

    // ==================== Index Operations ====================

    createIndex(fieldName, indexParams, collectionName) {
        const body = this._baseBody(collectionName);

        const idx = {
            fieldName: fieldName,
        };
        if (indexParams.indexType) idx.indexType = indexParams.indexType;
        if (indexParams.metricType) idx.metricType = indexParams.metricType;
        if (indexParams.indexName) idx.indexName = indexParams.indexName;
        if (indexParams.params) idx.params = indexParams.params;

        body.indexParams = [idx];

        return this._post('/indexes/create', body);
    }

    describeIndex(indexName, collectionName) {
        const body = this._baseBody(collectionName);
        body.indexName = indexName;
        return this._post('/indexes/describe', body);
    }

    dropIndex(indexName, collectionName) {
        const body = this._baseBody(collectionName);
        body.indexName = indexName;
        return this._post('/indexes/drop', body);
    }

    // ==================== Partition Operations ====================

    listPartitions(collectionName) {
        return this._post('/partitions/list', this._baseBody(collectionName));
    }

    createPartition(partitionName, collectionName) {
        const body = this._baseBody(collectionName);
        body.partitionName = partitionName;
        return this._post('/partitions/create', body);
    }

    dropPartition(partitionName, collectionName) {
        const body = this._baseBody(collectionName);
        body.partitionName = partitionName;
        return this._post('/partitions/drop', body);
    }

    hasPartition(partitionName, collectionName) {
        const body = this._baseBody(collectionName);
        body.partitionName = partitionName;
        return this._post('/partitions/has', body);
    }

    // ==================== Alias Operations ====================

    createAlias(aliasName, collectionName) {
        const body = this._baseBody(collectionName);
        body.aliasName = aliasName;
        return this._post('/aliases/create', body);
    }

    dropAlias(aliasName) {
        const body = {};
        body.aliasName = aliasName;
        if (this._dbName) body.dbName = this._dbName;
        return this._post('/aliases/drop', body);
    }

    listAliases(collectionName) {
        return this._post('/aliases/list', this._baseBody(collectionName));
    }

    // ==================== Database Operations ====================

    listDatabases() {
        return this._post('/databases/list', {});
    }

    createDatabase(dbName) {
        return this._post('/databases/create', { dbName: dbName });
    }

    dropDatabase(dbName) {
        return this._post('/databases/drop', { dbName: dbName });
    }

    // ==================== Import Operations ====================

    createImportJob(files, collectionName, options) {
        const body = this._baseBody(collectionName);
        body.files = files;
        if (options) {
            if (options.partitionName) body.partitionName = options.partitionName;
            if (options.options) body.options = options.options;
        }
        return this._post('/jobs/import/create', body);
    }

    getImportJobProgress(jobId) {
        const body = { jobId: jobId };
        if (this._dbName) body.dbName = this._dbName;
        return this._post('/jobs/import/get_progress', body);
    }

    listImportJobs(collectionName) {
        return this._post('/jobs/import/list', this._baseBody(collectionName));
    }

    // ==================== User Operations ====================

    listUsers() {
        return this._post('/users/list', {});
    }

    createUser(userName, password) {
        return this._post('/users/create', { userName, password });
    }

    dropUser(userName) {
        return this._post('/users/drop', { userName });
    }

    // ==================== Role Operations ====================

    listRoles() {
        return this._post('/roles/list', {});
    }

    createRole(roleName) {
        return this._post('/roles/create', { roleName });
    }

    dropRole(roleName) {
        return this._post('/roles/drop', { roleName });
    }

    grantRoleToUser(userName, roleName) {
        return this._post('/users/grant_role', { userName, roleName });
    }

    revokeRoleFromUser(userName, roleName) {
        return this._post('/users/revoke_role', { userName, roleName });
    }

    grantPrivilege(roleName, objectType, objectName, privilege) {
        return this._post('/roles/grant_privilege', {
            roleName, objectType, objectName, privilege,
        });
    }

    revokePrivilege(roleName, objectType, objectName, privilege) {
        return this._post('/roles/revoke_privilege', {
            roleName, objectType, objectName, privilege,
        });
    }

    // ==================== Lifecycle ====================

    // REST is stateless, close is a no-op but matches gRPC client interface
    close() {
        return makeResult(true, 0, null, '');
    }
}

// Factory functions matching gRPC client naming conventions
export function client(address, token) {
    return new MilvusRestClient(address, { token });
}

export function clientWithCollection(address, collectionName, token) {
    return new MilvusRestClient(address, { collectionName, token });
}

// Default export matching gRPC module pattern
export default {
    client,
    clientWithCollection,
    MilvusRestClient,
};

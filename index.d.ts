/**
 * xk6-milvus - k6 extension for load testing Milvus vector databases
 *
 * This module provides a JavaScript API to interact with Milvus from k6 test scripts.
 * It follows Locust's Milvus client design pattern for consistency and observability.
 *
 * @module k6/x/milvus
 * @example
 * ```javascript
 * import milvus from 'k6/x/milvus';
 * import { check } from 'k6';
 *
 * export default function() {
 *   // Create client
 *   const client = milvus.client('localhost:19530');
 *
 *   // Insert data
 *   const insertResult = client.insert({
 *     title: ['Product A', 'Product B'],
 *     price: [19.99, 29.99],
 *     vector: [[0.1, 0.2], [0.3, 0.4]]
 *   }, 'products');
 *
 *   check(insertResult, {
 *     'insert successful': (r) => r.success === true,
 *   });
 *
 *   client.close();
 * }
 * ```
 */
declare module 'k6/x/milvus' {
  /**
   * Creates a standard Milvus client for interacting with a Milvus server.
   *
   * @param address - Milvus server address (e.g., "localhost:19530")
   * @param token - Optional authentication token
   * @returns Client object for executing Milvus operations
   * @example
   * ```javascript
   * const client = milvus.client('localhost:19530');
   * const clientWithAuth = milvus.client('localhost:19530', 'my-token');
   * ```
   */
  export function client(address: string, token?: string): Client;

  /**
   * Creates a collection-bound Milvus client that automatically uses the specified
   * collection for all operations.
   *
   * @param address - Milvus server address
   * @param collectionName - Default collection name for all operations
   * @param token - Optional authentication token
   * @returns Collection-bound Client object
   * @example
   * ```javascript
   * const client = milvus.clientWithCollection('localhost:19530', 'products');
   * // All operations now default to 'products' collection
   * client.insert({ title: ['A'], price: [19.99] });
   * ```
   */
  export function clientWithCollection(address: string, collectionName: string, token?: string): Client;

  /**
   * Milvus client interface providing all database operations.
   */
  export interface Client {
    // Collection Operations

    /**
     * Creates a new collection with the specified schema.
     *
     * @param schema - Collection schema definition
     * @returns OperationResult with creation status
     * @example
     * ```javascript
     * const result = client.createCollection({
     *   name: 'products',
     *   fields: [
     *     { name: 'id', dataType: 'Int64', isPrimaryKey: true, isAutoID: true },
     *     { name: 'title', dataType: 'VarChar', maxLength: 200 },
     *     { name: 'embedding', dataType: 'FloatVector', dimension: 128 }
     *   ]
     * });
     * ```
     */
    createCollection(schema: CollectionSchema): OperationResult;

    /**
     * Creates a collection from a JSON string schema definition.
     *
     * @param schemaJSON - JSON string containing collection schema
     * @returns OperationResult with creation status
     * @example
     * ```javascript
     * const schemaJSON = JSON.stringify({
     *   name: 'products',
     *   fields: [{ name: 'id', dataType: 'Int64', isPrimaryKey: true }]
     * });
     * const result = client.createCollectionFromJSON(schemaJSON);
     * ```
     */
    createCollectionFromJSON(schemaJSON: string): OperationResult;

    /**
     * Drops (deletes) a collection.
     *
     * @param collectionName - Collection name (optional for collection-bound clients)
     * @returns OperationResult with deletion status
     * @example
     * ```javascript
     * const result = client.dropCollection('products');
     * ```
     */
    dropCollection(collectionName?: string): OperationResult;

    /**
     * Checks if a collection exists.
     *
     * @param collectionName - Collection name (optional for collection-bound clients)
     * @returns OperationResult where result contains boolean indicating existence
     * @example
     * ```javascript
     * const result = client.hasCollection('products');
     * if (result.success && result.result) {
     *   console.log('Collection exists');
     * }
     * ```
     */
    hasCollection(collectionName?: string): OperationResult;

    /**
     * Loads a collection into memory for search operations.
     *
     * @param collectionName - Collection name (optional for collection-bound clients)
     * @returns OperationResult with load status
     * @example
     * ```javascript
     * const result = client.loadCollection('products');
     * ```
     */
    loadCollection(collectionName?: string): OperationResult;

    /**
     * Releases a collection from memory.
     *
     * @param collectionName - Collection name (optional for collection-bound clients)
     * @returns OperationResult with release status
     * @example
     * ```javascript
     * const result = client.releaseCollection('products');
     * ```
     */
    releaseCollection(collectionName?: string): OperationResult;

    // Data Operations

    /**
     * Inserts data into a collection.
     * Data should be organized by columns (not rows).
     *
     * @param data - Column-based data to insert
     * @param collectionName - Collection name (optional for collection-bound clients)
     * @returns OperationResult with insert_count and ids
     * @example
     * ```javascript
     * const result = client.insert({
     *   title: ['Product A', 'Product B'],
     *   price: [19.99, 29.99],
     *   embedding: [[0.1, 0.2, 0.3], [0.4, 0.5, 0.6]]
     * }, 'products');
     * ```
     */
    insert(data: ColumnData, collectionName?: string): OperationResult;

    /**
     * Inserts or updates data in a collection.
     *
     * @param data - Column-based data to upsert
     * @param collectionName - Collection name (optional for collection-bound clients)
     * @returns OperationResult with upsert_count and ids
     * @example
     * ```javascript
     * const result = client.upsert({
     *   id: [1, 2],
     *   title: ['Updated A', 'Updated B']
     * }, 'products');
     * ```
     */
    upsert(data: ColumnData, collectionName?: string): OperationResult;

    /**
     * Deletes entities matching a filter expression.
     *
     * @param filter - Boolean filter expression
     * @param collectionName - Collection name (optional for collection-bound clients)
     * @returns OperationResult with delete_count
     * @example
     * ```javascript
     * const result = client.delete('price < 20', 'products');
     * console.log(`Deleted ${result.result.delete_count} items`);
     * ```
     */
    delete(filter: string, collectionName?: string): OperationResult;

    // Search Operations

    /**
     * Performs vector similarity search.
     *
     * @param vectors - Query vector(s) as number[][] or number[]
     * @param topK - Number of results to return
     * @param params - Search parameters
     * @param collectionName - Collection name (optional for collection-bound clients)
     * @returns OperationResult with search results, recall, and empty flag
     * @example
     * ```javascript
     * const result = client.search(
     *   [[0.1, 0.2, 0.3]],
     *   10,
     *   {
     *     vectorField: 'embedding',
     *     metricType: 'L2',
     *     outputFields: ['title', 'price'],
     *     expr: 'price > 20'
     *   },
     *   'products'
     * );
     * ```
     */
    search(
      vectors: number[][] | number[],
      topK: number,
      params: SearchParams,
      collectionName?: string
    ): OperationResult;

    /**
     * Performs scalar query without vectors (filter-based retrieval).
     *
     * @param filter - Boolean filter expression
     * @param outputFields - Fields to return in results
     * @param collectionName - Collection name (optional for collection-bound clients)
     * @returns OperationResult with query results
     * @example
     * ```javascript
     * const result = client.query(
     *   'price > 100 && price < 200',
     *   ['id', 'title', 'price'],
     *   'products'
     * );
     * ```
     */
    query(filter: string, outputFields: string[], collectionName?: string): OperationResult;

    /**
     * Performs multi-vector hybrid search with reranking.
     *
     * @param requests - Array of search requests for different vector fields
     * @param reranker - Reranking strategy (RRF or Weighted)
     * @param limit - Final number of results after reranking
     * @param outputFields - Fields to return in results
     * @param collectionName - Collection name (optional for collection-bound clients)
     * @returns OperationResult with hybrid search results and recall
     * @example
     * ```javascript
     * const result = client.hybridSearch(
     *   [
     *     {
     *       vectors: denseVectors,
     *       vectorField: 'dense_vector',
     *       limit: 10,
     *       params: { metricType: 'L2' }
     *     },
     *     {
     *       vectors: sparseVectors,
     *       vectorField: 'sparse_vector',
     *       limit: 10,
     *       params: { metricType: 'IP' }
     *     }
     *   ],
     *   { type: 'rrf', params: { k: 60 } },
     *   5,
     *   ['title', 'price']
     * );
     * ```
     */
    hybridSearch(
      requests: SearchRequest[],
      reranker: Reranker,
      limit: number,
      outputFields: string[],
      collectionName?: string
    ): OperationResult;

    // Index Operations

    /**
     * Creates an index on a vector field for faster searches.
     *
     * @param fieldName - Field to index
     * @param indexParams - Index configuration
     * @param collectionName - Collection name (optional for collection-bound clients)
     * @returns OperationResult with index creation status
     * @example
     * ```javascript
     * const result = client.createIndex(
     *   'embedding',
     *   {
     *     indexType: 'HNSW',
     *     metricType: 'L2',
     *     params: { M: 16, efConstruction: 200 }
     *   },
     *   'products'
     * );
     * ```
     */
    createIndex(fieldName: string, indexParams: IndexParams, collectionName?: string): OperationResult;

    // Lifecycle

    /**
     * Closes the client connection.
     *
     * @returns OperationResult with close status
     * @example
     * ```javascript
     * const result = client.close();
     * ```
     */
    close(): OperationResult;
  }

  // Type Definitions

  /**
   * Unified result structure returned by all operations.
   */
  export interface OperationResult {
    /** Whether the operation succeeded */
    success: boolean;

    /** Operation duration in milliseconds */
    response_time_ms: number;

    /** Operation-specific result data */
    result: any;

    /** Error message if operation failed */
    error: string;

    /** Whether result set is empty (search/query operations) */
    empty?: boolean;

    /** Recall metric for quality assessment (search operations) */
    recall?: number;
  }

  /**
   * Collection schema definition.
   */
  export interface CollectionSchema {
    /** Collection name */
    name: string;

    /** Array of field definitions */
    fields: FieldSchema[];

    /** Number of shards (default: 2) */
    numShards?: number;

    /** Functions for automatic processing (e.g., BM25) */
    functions?: FunctionSchema[];
  }

  /**
   * Field schema definition.
   */
  export interface FieldSchema {
    /** Field name */
    name: string;

    /** Data type (Int64, Float, VarChar, FloatVector, SparseFloatVector, etc.) */
    dataType: string;

    /** Whether this is the primary key field */
    isPrimaryKey?: boolean;

    /** Auto-generate IDs for primary key */
    isAutoID?: boolean;

    /** Max length for VarChar fields */
    maxLength?: number;

    /** Dimension for vector fields */
    dimension?: number;

    /** Enable text analyzer (for BM25) */
    enableAnalyzer?: boolean;

    /** Analyzer configuration */
    analyzerParams?: {
      /** Analyzer type (e.g., 'standard') */
      type: string;
    };

    /** Enable text matching */
    enableMatch?: boolean;
  }

  /**
   * Function schema for automatic processing.
   */
  export interface FunctionSchema {
    /** Function name */
    name: string;

    /** Function type (e.g., 'BM25', 'TextEmbedding') */
    functionType: string;

    /** Input field names */
    inputFieldNames: string[];

    /** Output field names */
    outputFieldNames: string[];
  }

  /**
   * Column-based data format for insert/upsert operations.
   * Data should be organized by columns, not rows.
   *
   * @example
   * ```javascript
   * {
   *   field1: [value1, value2, value3],
   *   field2: [value1, value2, value3],
   *   vector: [[0.1, 0.2], [0.3, 0.4], [0.5, 0.6]]
   * }
   * ```
   */
  export interface ColumnData {
    [fieldName: string]: any[] | number[][];
  }

  /**
   * Search parameters for vector similarity search.
   */
  export interface SearchParams {
    /** Name of the vector field to search */
    vectorField: string;

    /** Distance metric (L2, IP, COSINE) */
    metricType?: string;

    /** Fields to return in results */
    outputFields?: string[];

    /** Filter expression */
    expr?: string;

    /** Index-specific search parameters */
    params?: Record<string, any>;
  }

  /**
   * Search request for hybrid search.
   */
  export interface SearchRequest {
    /** Query vectors */
    vectors: number[][] | number[];

    /** Vector field name */
    vectorField: string;

    /** Number of results for this search */
    limit: number;

    /** Search parameters */
    params?: {
      /** Distance metric (L2, IP, COSINE) */
      metricType?: string;

      /** Filter expression */
      expr?: string;

      /** Index-specific parameters */
      [key: string]: any;
    };
  }

  /**
   * Reranking strategy for hybrid search.
   */
  export interface Reranker {
    /** Reranker type: 'rrf' (Reciprocal Rank Fusion) or 'weighted' */
    type: 'rrf' | 'weighted';

    /** Reranker-specific parameters */
    params?: {
      /** RRF k parameter (default: 60) */
      k?: number;

      /** Weights for each search (for weighted reranker) */
      weights?: number[];
    };
  }

  /**
   * Index parameters for creating indexes.
   */
  export interface IndexParams {
    /** Index type (FLAT, IVF_FLAT, HNSW, etc.) */
    indexType: string;

    /** Distance metric (L2, IP, COSINE) */
    metricType: string;

    /** Index-specific parameters */
    params?: {
      /** Number of cluster units (for IVF_FLAT) */
      nlist?: number;

      /** Max number of connections per layer (for HNSW) */
      M?: number;

      /** Size of dynamic candidate list (for HNSW) */
      efConstruction?: number;

      [key: string]: any;
    };
  }

  // REST Client

  /**
   * Creates a Milvus REST client using RESTful v2 API.
   * Uses HTTP instead of gRPC - same OperationResult interface.
   *
   * @param address - Milvus server address (e.g., "localhost:19530")
   * @param token - Optional authentication token (format: "username:password")
   * @returns RestClient object for executing Milvus operations via REST API
   * @example
   * ```javascript
   * const client = milvus.restClient('localhost:19530');
   * const authClient = milvus.restClient('localhost:19530', 'root:Milvus');
   * ```
   */
  export function restClient(address: string, token?: string): RestClient;

  /**
   * Creates a collection-bound Milvus REST client.
   *
   * @param address - Milvus server address
   * @param collectionName - Default collection name for all operations
   * @param token - Optional authentication token
   * @returns Collection-bound RestClient object
   * @example
   * ```javascript
   * const client = milvus.restClientWithCollection('localhost:19530', 'products');
   * client.search([[0.1, 0.2]], 10, { vectorField: 'embedding' });
   * ```
   */
  export function restClientWithCollection(address: string, collectionName: string, token?: string): RestClient;

  /**
   * Milvus REST client interface using RESTful v2 API.
   * Provides the same core operations as the gRPC Client, plus additional REST-only operations.
   */
  export interface RestClient {
    // Collection Operations
    listCollections(dbName?: string): OperationResult;
    createCollection(schema: CollectionSchema): OperationResult;
    createCollectionFromJSON(schemaJSON: string): OperationResult;
    describeCollection(collectionName?: string): OperationResult;
    dropCollection(collectionName?: string): OperationResult;
    hasCollection(collectionName?: string): OperationResult;
    loadCollection(collectionName?: string): OperationResult;
    releaseCollection(collectionName?: string): OperationResult;
    getLoadState(collectionName?: string): OperationResult;
    getCollectionStats(collectionName?: string): OperationResult;
    flush(collectionName?: string): OperationResult;
    renameCollection(collectionName: string, newCollectionName: string): OperationResult;

    // Data Operations
    insert(data: ColumnData, collectionName?: string): OperationResult;
    upsert(data: ColumnData, collectionName?: string): OperationResult;
    delete(filter: string, collectionName?: string): OperationResult;
    get(ids: any, outputFields: string[], collectionName?: string): OperationResult;

    // Search Operations
    search(vectors: number[][], topK: number, params: SearchParams, collectionName?: string): OperationResult;
    query(filter: string, outputFields: string[], collectionName?: string): OperationResult;
    hybridSearch(requests: SearchRequest[], reranker: Reranker, limit: number, outputFields: string[], collectionName?: string): OperationResult;

    // Index Operations
    createIndex(fieldName: string, indexParams: IndexParams, collectionName?: string): OperationResult;
    describeIndex(indexName: string, collectionName?: string): OperationResult;
    dropIndex(indexName: string, collectionName?: string): OperationResult;

    // Partition Operations
    listPartitions(collectionName?: string): OperationResult;
    createPartition(partitionName: string, collectionName?: string): OperationResult;
    dropPartition(partitionName: string, collectionName?: string): OperationResult;
    hasPartition(partitionName: string, collectionName?: string): OperationResult;

    // Lifecycle
    close(): OperationResult;
  }

  // Default export
  const milvus: {
    client: typeof client;
    clientWithCollection: typeof clientWithCollection;
    restClient: typeof restClient;
    restClientWithCollection: typeof restClientWithCollection;
  };

  export default milvus;
}

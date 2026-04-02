/**
 * xk6-milvus REST Client - TypeScript definitions
 *
 * JavaScript library wrapping k6's built-in HTTP module for Milvus RESTful v2 API.
 * Provides the same OperationResult interface as the gRPC extension client.
 *
 * @module lib/milvus-rest
 * @example
 * ```javascript
 * import milvusRest from '../lib/milvus-rest.js';
 *
 * export default function() {
 *   const client = milvusRest.client('localhost:19530');
 *   const result = client.search([[0.1, 0.2]], 10, { vectorField: 'embedding' }, 'my_collection');
 *   client.close();
 * }
 * ```
 */

export interface OperationResult {
  success: boolean;
  response_time_ms: number;
  result: any;
  error: string;
  empty?: boolean;
  recall?: number;
}

export interface RestClientOptions {
  /** Default collection name for all operations */
  collectionName?: string;
  /** Authentication token (username:password or API key) */
  token?: string;
  /** Default database name */
  dbName?: string;
  /** HTTP request timeout (e.g., '30s') */
  timeout?: string;
  /** Custom k6 tags for HTTP metrics */
  tags?: Record<string, string>;
}

export interface CollectionSchema {
  name: string;
  fields: FieldSchema[];
  numShards?: number;
  enableDynamicField?: boolean;
  functions?: FunctionSchema[];
  /** Quick-setup mode: dimension for auto-created vector field */
  dimension?: number;
  /** Quick-setup mode: metric type for auto-created index */
  metricType?: string;
}

export interface FieldSchema {
  name: string;
  dataType: string;
  isPrimaryKey?: boolean;
  isAutoID?: boolean;
  maxLength?: number;
  dimension?: number;
  enableAnalyzer?: boolean;
  analyzerParams?: { type: string };
  enableMatch?: boolean;
  nullable?: boolean;
  isPartitionKey?: boolean;
  isClusteringKey?: boolean;
  description?: string;
}

export interface FunctionSchema {
  name: string;
  functionType: string;
  inputFieldNames: string[];
  outputFieldNames: string[];
  params?: Record<string, any>;
}

export interface SearchParams {
  vectorField: string;
  metricType?: string;
  outputFields?: string[];
  expr?: string;
  offset?: number;
  groupingField?: string;
  params?: Record<string, any>;
}

export interface SearchRequest {
  vectors: number[][] | number[] | Record<number, number>[];
  vectorField: string;
  limit: number;
  params?: {
    metricType?: string;
    expr?: string;
    [key: string]: any;
  };
}

export interface Reranker {
  type: 'rrf' | 'weighted';
  params?: {
    k?: number;
    weights?: number[];
  };
}

export interface IndexParams {
  indexType: string;
  metricType: string;
  indexName?: string;
  params?: Record<string, any>;
}

export interface QueryParams {
  limit?: number;
  offset?: number;
}

export interface ImportOptions {
  partitionName?: string;
  options?: Record<string, any>;
}

export class MilvusRestClient {
  constructor(baseUrl: string, options?: RestClientOptions);

  // Collection operations
  listCollections(dbName?: string): OperationResult;
  createCollection(schema: CollectionSchema): OperationResult;
  describeCollection(collectionName?: string): OperationResult;
  dropCollection(collectionName?: string): OperationResult;
  hasCollection(collectionName?: string): OperationResult;
  loadCollection(collectionName?: string): OperationResult;
  releaseCollection(collectionName?: string): OperationResult;
  getLoadState(collectionName?: string): OperationResult;
  getCollectionStats(collectionName?: string): OperationResult;
  flush(collectionName?: string): OperationResult;
  renameCollection(collectionName: string, newCollectionName: string): OperationResult;

  // Data operations
  insert(data: Record<string, any[]> | Record<string, any>[], collectionName?: string): OperationResult;
  upsert(data: Record<string, any[]> | Record<string, any>[], collectionName?: string): OperationResult;
  delete(filter: string, collectionName?: string): OperationResult;
  get(ids: any | any[], outputFields?: string[], collectionName?: string): OperationResult;

  // Search operations
  search(vectors: number[][] | number[], topK: number, params: SearchParams, collectionName?: string): OperationResult;
  query(filter: string, outputFields: string[], collectionName?: string, queryParams?: QueryParams): OperationResult;
  hybridSearch(requests: SearchRequest[], reranker: Reranker, limit: number, outputFields: string[], collectionName?: string): OperationResult;

  // Index operations
  createIndex(fieldName: string, indexParams: IndexParams, collectionName?: string): OperationResult;
  describeIndex(indexName: string, collectionName?: string): OperationResult;
  dropIndex(indexName: string, collectionName?: string): OperationResult;

  // Partition operations
  listPartitions(collectionName?: string): OperationResult;
  createPartition(partitionName: string, collectionName?: string): OperationResult;
  dropPartition(partitionName: string, collectionName?: string): OperationResult;
  hasPartition(partitionName: string, collectionName?: string): OperationResult;

  // Alias operations
  createAlias(aliasName: string, collectionName?: string): OperationResult;
  dropAlias(aliasName: string): OperationResult;
  listAliases(collectionName?: string): OperationResult;

  // Database operations
  listDatabases(): OperationResult;
  createDatabase(dbName: string): OperationResult;
  dropDatabase(dbName: string): OperationResult;

  // Import operations
  createImportJob(files: string[][], collectionName?: string, options?: ImportOptions): OperationResult;
  getImportJobProgress(jobId: string): OperationResult;
  listImportJobs(collectionName?: string): OperationResult;

  // User & role operations
  listUsers(): OperationResult;
  createUser(userName: string, password: string): OperationResult;
  dropUser(userName: string): OperationResult;
  listRoles(): OperationResult;
  createRole(roleName: string): OperationResult;
  dropRole(roleName: string): OperationResult;
  grantRoleToUser(userName: string, roleName: string): OperationResult;
  revokeRoleFromUser(userName: string, roleName: string): OperationResult;
  grantPrivilege(roleName: string, objectType: string, objectName: string, privilege: string): OperationResult;
  revokePrivilege(roleName: string, objectType: string, objectName: string, privilege: string): OperationResult;

  // Lifecycle
  close(): OperationResult;
}

export function client(address: string, token?: string): MilvusRestClient;
export function clientWithCollection(address: string, collectionName: string, token?: string): MilvusRestClient;

declare const milvusRest: {
  client: typeof client;
  clientWithCollection: typeof clientWithCollection;
  MilvusRestClient: typeof MilvusRestClient;
};

export default milvusRest;

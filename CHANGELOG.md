# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Reorganized project structure to follow k6 extension best practices
- Moved all implementation code to `pkg/milvus/` directory
- Added comprehensive documentation (API.md, CONTRIBUTING.md)
- Improved examples organization

## [0.2.0] - 2025-11-23

### Added

- Unified `OperationResult` return structure for all operations
- Built-in `response_time_ms` metric in every operation
- `recall` metric for search operations quality assessment
- `empty` detection for search and query results
- Collection-bound clients via `clientWithCollection()`
- Comprehensive API following Locust's Milvus client design pattern
- Support for BM25 full-text search with automatic sparse vector generation
- Support for TextEmbedding functions for automatic dense vector generation
- Hybrid search with multiple vector fields
- RRF (Reciprocal Rank Fusion) and Weighted reranking strategies
- Flexible schema creation with support for complex field types
- JSON-based schema creation via `createCollectionFromJSON()`
- Support for analyzer configuration in VarChar fields
- Function support for automatic data processing

### Changed

- Refactored API to return `OperationResult` instead of throwing errors
- Improved error handling with consistent error messages
- Enhanced type conversions for schema and data operations
- Optimized client creation and management

### Fixed

- Index creation now properly awaits task completion
- Collection loading properly awaits task completion
- Improved handling of vector field types

## [0.1.0] - 2025-11-XX

### Added

- Initial implementation of xk6-milvus extension
- Basic Milvus client creation with authentication support
- Collection operations:
  - Create collection with schema
  - Drop collection
  - Check collection existence
  - Load/release collection
- Data operations:
  - Insert entities with column-based data
  - Upsert entities
  - Delete entities by filter expression
- Vector search operations:
  - Vector similarity search with filtering
  - Scalar query without vectors
  - Multi-vector hybrid search
- Index operations:
  - Create vector indexes (FLAT, IVF_FLAT, HNSW, etc.)
- k6 RootModule/ModuleInstance pattern for VU isolation
- Basic examples and documentation
- Support for Milvus Go SDK v2.5.4

### Technical Details

- Go 1.24+ required
- Milvus Go SDK v2.5.4 dependency
- k6 v0.49.0 compatibility
- Proper VU context usage for all operations
- Thread-safe concurrent testing support

## [0.0.1] - Initial Development

### Added

- Project scaffolding
- Basic module structure
- Initial Milvus SDK integration

---

## Version History Notes

### Unreleased

Changes in `main` branch not yet released

### [0.2.0] - Major API Redesign

Complete redesign following Locust's Milvus client pattern with unified operation results, built-in metrics, and comprehensive feature support.

### [0.1.0] - Initial Release

First functional release with basic Milvus operations and k6 integration.

---

## Migration Guides

### Migrating from 0.1.0 to 0.2.0

**Breaking Changes:**

- All operations now return `OperationResult` instead of raw values or throwing errors
- Client methods use different signatures for consistency

**Before (0.1.0):**

```javascript
try {
  const result = client.search(vectors, 10, "products");
  console.log(result);
} catch (error) {
  console.error(error);
}
```

**After (0.2.0):**

```javascript
const result = client.search(
  vectors,
  10,
  {
    vectorField: "embedding",
    outputFields: ["title", "price"],
  },
  "products",
);

if (!result.success) {
  console.error(result.error);
  return;
}

console.log(`Search took ${result.response_time_ms}ms`);
console.log(`Recall: ${result.recall}`);
console.log(result.result);
```

**Benefits:**

- Consistent error handling across all operations
- Built-in performance metrics
- Quality metrics (recall) for search operations
- Better observability in load testing scenarios

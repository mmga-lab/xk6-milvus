# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

xk6-milvus is a k6 extension for load testing Milvus vector databases. It provides a JavaScript API to interact with Milvus from k6 test scripts.

## Architecture

The project follows a simple, single-file architecture:
- **milvus.go**: Contains the entire k6 extension implementation
- Wraps the official Milvus Go SDK to provide k6-friendly methods
- Registers as a k6 module using `modules.Register("k6/x/milvus", new(Milvus))`

## Common Commands

### Build
```bash
# Install xk6 if not already installed
go install go.k6.io/xk6/cmd/xk6@latest

# Build k6 with milvus extension
xk6 build --with github.com/zilliz/xk6-milvus=.
```

### Run Tests
```bash
# Set Milvus host (optional, defaults to localhost:19530)
export MILVUS_HOST=your-milvus-host:19530

# Run example test
./k6 run example/test-milvus.js

# Run with custom options
./k6 run -u 10 -d 30s example/test-milvus.js
```

### Development Workflow
1. Modify milvus.go to add/update functionality
2. Rebuild using `xk6 build --with github.com/zilliz/xk6-milvus=.`
3. Test changes using example script or custom k6 scripts

### Dependencies
- Uses latest Milvus client SDK v2.5.4: `github.com/milvus-io/milvus/client/v2/milvusclient`
- Requires Go 1.24+ (upgraded from 1.23.5)

## Key Implementation Details

### Module Structure
The module exports a single `Milvus` struct that implements all methods. Each method follows this pattern:
1. Parse JavaScript arguments using goja
2. Convert to Go types
3. Call Milvus SDK v2.5.4
4. Convert response back to JavaScript types

### Available Methods
- Client: `client(address)`, `close()`
- Collections: `createCollection()`, `dropCollection()`, `hasCollection()`, `loadCollection()`, `releaseCollection()`
- Data: `insert()`, `search()`
- Index: `createIndex()`

### SDK Implementation Details
- Uses latest Milvus client SDK: `github.com/milvus-io/milvus/client/v2/milvusclient`
- Imports: `milvusclient`, `entity`, `index`, `column` packages
- Client type: `*milvusclient.Client` (pointer type)
- Collection operations use option pattern: `milvusclient.NewCreateCollectionOption()`
- Insert operations use column-based approach: `milvusclient.NewColumnBasedInsertOption()`
- Search operations use entity.Vector interface with entity.FloatVector implementation
- Index and load operations return tasks that must be awaited: `task.Await(ctx)`

### Error Handling
All methods return errors to k6's JavaScript runtime. The new SDK provides better error context and task-based operations for async operations.

### Testing Approach
- Use k6 scripts to test the extension
- Example test in `example/test-milvus.js` demonstrates common usage patterns
- Successfully tested with Milvus instance at 10.104.13.2:19530
- Performance: 99.66% success rate with 594 iterations, 10 VUs, 30s duration

## Important Considerations

1. **Vector Format**: Vectors are passed as JavaScript arrays and converted to entity.FloatVector
2. **Client Lifecycle**: Clients should be created in setup() and closed in teardown()
3. **Collection Loading**: Collections must be loaded before search operations using task.Await()
4. **Index Creation**: Create indexes after inserting data, operations are async and require task.Await()
5. **SDK Upgrade**: Updated from v2.4.2 to v2.5.4 with breaking API changes requiring option pattern

## Memorized Notes

- to memorize
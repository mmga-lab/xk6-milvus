# TypeScript Support for xk6-milvus

xk6-milvus provides TypeScript type definitions to enhance your development experience with IDE autocompletion, type checking, and inline documentation.

## Quick Start

### 1. Download the Type Definition File

The TypeScript declaration file is available at the root of the repository:

- **File**: `index.d.ts`
- **Location**: <https://github.com/mmga-lab/xk6-milvus/blob/main/index.d.ts>

### 2. Configure Your IDE

Create a `jsconfig.json` (for JavaScript) or `tsconfig.json` (for TypeScript) file in your project root:

#### For JavaScript Projects (jsconfig.json)

```json
{
  "compilerOptions": {
    "target": "ES6",
    "module": "ES6",
    "paths": {
      "k6/x/milvus": ["./typings/xk6-milvus/index.d.ts"]
    }
  }
}
```

#### For TypeScript Projects (tsconfig.json)

```json
{
  "compilerOptions": {
    "target": "ES6",
    "module": "ES6",
    "moduleResolution": "node",
    "strict": true,
    "paths": {
      "k6/x/milvus": ["./typings/xk6-milvus/index.d.ts"]
    }
  }
}
```

### 3. Setup Project Structure

Organize your project files:

```text
your-project/
├── jsconfig.json (or tsconfig.json)
├── typings/
│   └── xk6-milvus/
│       └── index.d.ts (copy from xk6-milvus repository)
└── tests/
    └── your-test.js
```

### 4. Copy the Type Definition

```bash
# Create typings directory
mkdir -p typings/xk6-milvus

# Copy the type definition file
cp /path/to/xk6-milvus/index.d.ts typings/xk6-milvus/
```

Or download directly:

```bash
curl -o typings/xk6-milvus/index.d.ts \
  https://raw.githubusercontent.com/mmga-lab/xk6-milvus/main/index.d.ts
```

## IDE Support

### Visual Studio Code

VS Code automatically detects `jsconfig.json` and `tsconfig.json` files. After setup:

✅ **Autocompletion**: Type `client.` to see all available methods
✅ **Parameter hints**: See parameter types and descriptions as you type
✅ **Type checking**: Get warnings for incorrect types
✅ **IntelliSense**: View inline documentation from JSDoc comments

### JetBrains IDEs (WebStorm, IntelliJ)

These IDEs also support TypeScript definitions automatically:

1. Place `index.d.ts` in your project
2. Configure paths in `tsconfig.json` or `jsconfig.json`
3. Enjoy full type support

## Example Usage

### JavaScript with Type Support

```javascript
import milvus from "k6/x/milvus";
import { check } from "k6";

export default function () {
  // IDE shows all available parameters and types
  const client = milvus.client("localhost:19530");

  // Autocompletion for client methods
  const result = client.createCollection({
    name: "products",
    fields: [
      {
        name: "id",
        dataType: "Int64", // IDE suggests valid data types
        isPrimaryKey: true,
        isAutoID: true,
      },
      {
        name: "embedding",
        dataType: "FloatVector",
        dimension: 128,
      },
    ],
  });

  // Type-safe result handling
  check(result, {
    "collection created": (r) => r.success === true,
    "fast creation": (r) => r.response_time_ms < 1000,
  });

  client.close();
}
```

### TypeScript Usage

```typescript
import milvus, { Client, OperationResult, CollectionSchema } from "k6/x/milvus";
import { check } from "k6";

export default function (): void {
  const client: Client = milvus.client("localhost:19530");

  const schema: CollectionSchema = {
    name: "products",
    fields: [
      { name: "id", dataType: "Int64", isPrimaryKey: true, isAutoID: true },
      { name: "title", dataType: "VarChar", maxLength: 200 },
      { name: "embedding", dataType: "FloatVector", dimension: 128 },
    ],
  };

  const result: OperationResult = client.createCollection(schema);

  check(result, {
    success: (r: OperationResult) => r.success === true,
  });

  client.close();
}
```

## Available Type Exports

The `k6/x/milvus` module exports the following types:

### Functions

- `client(address, token?)` - Create standard client
- `clientWithCollection(address, collectionName, token?)` - Create collection-bound client

### Interfaces

- `Client` - Milvus client interface with all methods
- `OperationResult` - Unified result structure for all operations
- `CollectionSchema` - Collection schema definition
- `FieldSchema` - Field schema definition
- `FunctionSchema` - Function schema (e.g., BM25)
- `ColumnData` - Column-based data format
- `SearchParams` - Vector search parameters
- `SearchRequest` - Hybrid search request
- `Reranker` - Reranking strategy configuration
- `IndexParams` - Index configuration

## Benefits

### 🚀 **Faster Development**

Autocomplete reduces typing and helps discover available methods.

### 🐛 **Fewer Errors**

Type checking catches mistakes before runtime.

### 📚 **Better Documentation**

Inline JSDoc comments provide context-aware help.

### 🔍 **Improved Readability**

Explicit types make code intentions clearer.

## Troubleshooting

### IDE Not Showing Suggestions

1. **Verify file path**: Ensure `index.d.ts` is in the correct location
2. **Check config**: Confirm `jsconfig.json` or `tsconfig.json` paths match
3. **Restart IDE**: Reload the window or restart your IDE
4. **Clear cache**: In VS Code: Cmd/Ctrl + Shift + P → "Reload Window"

### Type Errors

If you see type errors:

1. **Update definitions**: Ensure you have the latest `index.d.ts`
2. **Check k6 version**: Some features require specific k6 versions
3. **Disable strict mode**: In `tsconfig.json`, set `"strict": false` if needed

### Module Not Found

If you see "Cannot find module 'k6/x/milvus'":

1. Verify the `paths` configuration in your config file
2. Ensure the path is relative to your config file location
3. Use `./` prefix for relative paths

## Resources

- **API Documentation**: [docs/API.md](../docs/API.md)
- **Examples**: [examples/](../examples/)
- **Source Code**: [pkg/milvus/](../pkg/milvus/)
- **k6 TypeScript Guide**: <https://grafana.com/docs/k6/latest/using-k6/javascript-typescript-compatibility-mode/>

## Contributing

If you find type definition issues or want to improve them:

1. Edit `index.d.ts` in the repository root
2. Test with your IDE to ensure types work correctly
3. Submit a pull request with your improvements

For questions or issues, please open an issue on GitHub.

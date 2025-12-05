# 第10章：动手实践

## 学习目标

完成本章后，你将能够：

- ✅ 独立为项目添加新功能
- ✅ 编写单元测试
- ✅ 运行完整的测试示例
- ✅ 调试和排查问题

---

## 📖 10.1 实践准备

### 环境检查

```bash
# 确保在项目目录
cd ~/projects/xk6-milvus

# 检查 Go 版本
go version  # 需要 1.24+

# 检查依赖
go mod tidy

# 运行现有测试
make test
```

### 理解现有代码结构

```
pkg/milvus/
├── module.go      # 模块注册（不需要修改）
├── client.go      # 客户端创建（可能需要扩展）
├── types.go       # 类型定义（添加新类型）
├── collection.go  # 集合操作（参考实现）
├── data.go        # 数据操作（参考实现）
├── search.go      # 搜索操作（参考实现）
├── index.go       # 索引操作（参考实现）
├── converters.go  # 类型转换
├── errors.go      # 错误定义
├── config.go      # 配置选项
└── helpers.go     # 辅助函数
```

---

## ✏️ 10.2 练习一：添加 DescribeCollection 功能

### 需求

添加一个 `DescribeCollection` 方法，返回集合的详细信息。

### 步骤 1：查看 Milvus SDK

首先了解 SDK 提供的功能：

```go
// Milvus SDK 提供的方法
client.DescribeCollection(ctx, milvusclient.NewDescribeCollectionOption(collectionName))
```

### 步骤 2：定义返回类型（types.go）

```go
// 在 pkg/milvus/types.go 添加

// CollectionInfo 表示集合信息
type CollectionInfo struct {
    Name         string `json:"name"`
    Description  string `json:"description"`
    NumShards    int32  `json:"num_shards"`
    NumFields    int    `json:"num_fields"`
    NumPartitions int   `json:"num_partitions"`
    Loaded       bool   `json:"loaded"`
}
```

### 步骤 3：实现方法（collection.go）

```go
// 在 pkg/milvus/collection.go 添加

// DescribeCollection 获取集合详细信息
func (c *Client) DescribeCollection(collectionName ...string) interface{} {
    start := time.Now()

    // 获取集合名
    name := c.defaultCollection
    if len(collectionName) > 0 && collectionName[0] != "" {
        name = collectionName[0]
    }

    if name == "" {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        ErrCollectionNameRequired.Error(),
        })
    }

    // 调用 SDK
    option := milvusclient.NewDescribeCollectionOption(name)
    coll, err := c.client.DescribeCollection(c.ctx, option)

    if err != nil {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        fmt.Sprintf("failed to describe collection: %v", err),
        })
    }

    // 构建返回信息
    info := CollectionInfo{
        Name:        coll.Name,
        Description: coll.Schema.Description,
        NumFields:   len(coll.Schema.Fields),
        // 根据 SDK 返回的结构填充其他字段
    }

    return toMap(&OperationResult{
        Success:      true,
        ResponseTime: float64(time.Since(start).Milliseconds()),
        Result:       info,
    })
}
```

### 步骤 4：编写测试

```go
// 创建 pkg/milvus/collection_test.go（如果不存在）

package milvus

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestDescribeCollection_EmptyName(t *testing.T) {
    // 创建一个没有默认集合的客户端
    client := &Client{
        defaultCollection: "",
    }

    result := client.DescribeCollection()
    resultMap := result.(map[string]interface{})

    assert.False(t, resultMap["success"].(bool))
    assert.Contains(t, resultMap["error"].(string), "collection name required")
}
```

### 步骤 5：测试你的实现

```bash
# 运行测试
go test -v ./pkg/milvus/ -run TestDescribeCollection

# 构建
make build

# 在 JavaScript 中测试
cat > test-describe.js << 'EOF'
import milvus from 'k6/x/milvus';

export default function() {
    const client = milvus.client('localhost:19530');

    // 假设集合已存在
    const info = client.describeCollection('your_collection');

    console.log('Collection info:', JSON.stringify(info, null, 2));

    if (info.success) {
        console.log('Name:', info.result.name);
        console.log('Fields:', info.result.num_fields);
    }
}
EOF

./k6 run test-describe.js
```

---

## ✏️ 10.3 练习二：添加 ListCollections 功能

### 需求

添加一个 `ListCollections` 方法，返回所有集合的列表。

### 步骤 1：实现方法

```go
// 在 pkg/milvus/collection.go 添加

// ListCollections 列出所有集合
func (c *Client) ListCollections() interface{} {
    start := time.Now()

    option := milvusclient.NewListCollectionOption()
    collections, err := c.client.ListCollections(c.ctx, option)

    if err != nil {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        fmt.Sprintf("failed to list collections: %v", err),
        })
    }

    // 提取集合名
    names := make([]string, len(collections))
    for i, coll := range collections {
        names[i] = coll
    }

    return toMap(&OperationResult{
        Success:      true,
        ResponseTime: float64(time.Since(start).Milliseconds()),
        Result: map[string]interface{}{
            "collections": names,
            "count":       len(names),
        },
        Empty: len(names) == 0,
    })
}
```

### 步骤 2：JavaScript 使用

```javascript
import milvus from 'k6/x/milvus';
import { check } from 'k6';

export default function() {
    const client = milvus.client('localhost:19530');

    const result = client.listCollections();

    check(result, {
        'list successful': (r) => r.success === true,
        'is array': (r) => Array.isArray(r.result.collections),
    });

    if (result.success) {
        console.log('Found', result.result.count, 'collections');
        result.result.collections.forEach(name => {
            console.log(' -', name);
        });
    }

    client.close();
}
```

---

## ✏️ 10.4 练习三：添加 GetCollectionStats 功能

### 需求

获取集合的统计信息（如实体数量）。

### 步骤 1：实现

```go
// 在 pkg/milvus/collection.go 添加

// GetCollectionStats 获取集合统计信息
func (c *Client) GetCollectionStats(collectionName ...string) interface{} {
    start := time.Now()

    name := c.getCollectionName(collectionName...)
    if name == "" {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        ErrCollectionNameRequired.Error(),
        })
    }

    // 使用 Query 计算实体数量
    // 或使用 SDK 的统计方法（如果可用）
    option := milvusclient.NewQueryOption(name).
        WithOutputFields("count(*)").
        WithFilter("")  // 空过滤器

    // 注意：具体实现取决于 Milvus SDK 版本
    // 这里是示例逻辑

    return toMap(&OperationResult{
        Success:      true,
        ResponseTime: float64(time.Since(start).Milliseconds()),
        Result: map[string]interface{}{
            "collection":   name,
            "entity_count": 0, // 填充实际值
        },
    })
}
```

---

## ✏️ 10.5 练习四：编写集成测试

### 创建集成测试文件

```go
// pkg/milvus/collection_integration_test.go

//go:build integration
// +build integration

package milvus

import (
    "os"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func getTestClient(t *testing.T) *Client {
    host := os.Getenv("MILVUS_HOST")
    if host == "" {
        host = "localhost:19530"
    }

    // 注意：这里需要模拟 VU 上下文
    // 实际测试中可能需要更复杂的设置
    // ...
}

func TestListCollections_Integration(t *testing.T) {
    client := getTestClient(t)
    require.NotNil(t, client)
    defer client.Close()

    result := client.ListCollections()
    resultMap := result.(map[string]interface{})

    assert.True(t, resultMap["success"].(bool))
    assert.Contains(t, resultMap, "result")
}
```

### 运行集成测试

```bash
# 启动 Milvus
make docker-up

# 运行集成测试
make test-integration-local

# 或手动
MILVUS_HOST=localhost:19530 go test -tags=integration -v ./pkg/milvus/
```

---

## ✏️ 10.6 练习五：完整功能开发流程

### 任务：添加 CompactCollection 功能

Compact 操作用于压缩集合，合并小的数据段。

### 完整步骤

1. **研究 SDK**
   ```bash
   # 查看 Milvus SDK 文档或源码
   # 找到 Compact 相关方法
   ```

2. **定义接口**
   ```go
   // 思考：需要什么参数？返回什么？
   func (c *Client) CompactCollection(collectionName ...string) interface{}
   ```

3. **实现功能**
   ```go
   func (c *Client) CompactCollection(collectionName ...string) interface{} {
       start := time.Now()

       name := c.getCollectionName(collectionName...)
       if name == "" {
           return toMap(&OperationResult{
               Success:      false,
               ResponseTime: float64(time.Since(start).Milliseconds()),
               Error:        ErrCollectionNameRequired.Error(),
           })
       }

       // 调用 SDK 的 Compact 方法
       // compactionID, err := c.client.Compact(c.ctx, name)
       // ...

       return toMap(&OperationResult{
           Success:      true,
           ResponseTime: float64(time.Since(start).Milliseconds()),
           Result: map[string]interface{}{
               "collection": name,
               "message":    "compaction started",
           },
       })
   }
   ```

4. **编写单元测试**
   ```go
   func TestCompactCollection_EmptyName(t *testing.T) {
       client := &Client{defaultCollection: ""}
       result := client.CompactCollection()
       resultMap := result.(map[string]interface{})
       assert.False(t, resultMap["success"].(bool))
   }
   ```

5. **运行测试**
   ```bash
   go test -v ./pkg/milvus/ -run TestCompact
   ```

6. **构建并验证**
   ```bash
   make build
   ```

7. **编写 JavaScript 示例**
   ```javascript
   import milvus from 'k6/x/milvus';

   export default function() {
       const client = milvus.clientWithCollection('localhost:19530', 'test');
       const result = client.compactCollection();
       console.log(JSON.stringify(result));
   }
   ```

---

## 📖 10.7 调试技巧

### 打印调试

```go
import "fmt"

func (c *Client) SomeMethod() interface{} {
    // 打印变量
    fmt.Printf("DEBUG: value = %+v\n", someValue)
    fmt.Printf("DEBUG: type = %T\n", someValue)

    // 打印 JSON
    jsonBytes, _ := json.MarshalIndent(data, "", "  ")
    fmt.Println("DEBUG:", string(jsonBytes))
}
```

### 使用 delve 调试器

```bash
# 安装 delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 调试测试
dlv test ./pkg/milvus/ -- -test.run TestSomeFunction
```

### 常见问题排查

1. **编译错误**
   ```bash
   go build ./...
   # 查看详细错误
   ```

2. **测试失败**
   ```bash
   go test -v ./pkg/milvus/ -run TestFailing
   ```

3. **类型断言失败**
   ```go
   // 添加类型检查
   if val, ok := data["field"].(string); ok {
       // 使用 val
   } else {
       fmt.Printf("unexpected type: %T\n", data["field"])
   }
   ```

---

## 📖 10.8 代码风格指南

### 遵循项目约定

1. **错误处理**
   ```go
   // 使用项目的 OperationResult 模式
   return toMap(&OperationResult{
       Success:      false,
       ResponseTime: float64(time.Since(start).Milliseconds()),
       Error:        fmt.Sprintf("operation failed: %v", err),
   })
   ```

2. **方法签名**
   ```go
   // 可选参数使用 ...string
   func (c *Client) SomeMethod(required string, optional ...string) interface{}
   ```

3. **获取集合名**
   ```go
   name := c.getCollectionName(collectionName...)
   if name == "" {
       // 返回错误
   }
   ```

4. **计时**
   ```go
   start := time.Now()
   // ... 操作 ...
   ResponseTime: float64(time.Since(start).Milliseconds())
   ```

### 运行 Linter

```bash
make lint
```

---

## ❓ 自测挑战

完成以下挑战来巩固所学：

### 挑战 1：添加 RenameCollection
- 实现重命名集合功能
- 参数：旧名称，新名称
- 编写测试

### 挑战 2：添加 GetIndexInfo
- 获取集合的索引信息
- 返回索引类型、参数等
- 处理没有索引的情况

### 挑战 3：添加批量操作
- 实现 BatchInsert 方法
- 支持分批插入大量数据
- 返回每批的统计信息

### 挑战 4：性能优化
- 分析 `convertNestedVectors` 函数
- 思考如何减少内存分配
- 实现优化版本并对比性能

---

## 💡 本章要点

1. **开发流程**：研究 SDK → 定义接口 → 实现 → 测试 → 验证
2. **遵循约定**：使用项目现有的模式和风格
3. **测试先行**：先写测试，再实现功能
4. **调试技巧**：打印日志、使用调试器
5. **代码风格**：运行 linter，保持一致性

---

## 🎉 恭喜完成！

你已经完成了整个 Golang 学习教程！现在你应该：

- 理解 Go 语言的核心概念
- 能够阅读和理解 xk6-milvus 项目代码
- 具备为项目添加新功能的能力
- 知道如何编写测试和调试代码

### 下一步学习建议

1. **深入 Go**
   - [Go 官方文档](https://go.dev/doc/)
   - [Effective Go](https://go.dev/doc/effective_go)
   - [Go by Example](https://gobyexample.com/)

2. **k6 扩展开发**
   - [xk6 文档](https://github.com/grafana/xk6)
   - [k6 文档](https://k6.io/docs/)

3. **实践项目**
   - 为 xk6-milvus 贡献代码
   - 创建自己的 k6 扩展

### 获取帮助

- 项目 Issues: https://github.com/mmga-lab/xk6-milvus/issues
- Go 社区: https://forum.golangbridge.org/
- k6 社区: https://community.k6.io/

---

[← 返回目录](./README.md)

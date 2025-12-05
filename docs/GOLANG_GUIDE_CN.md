# xk6-milvus Golang 初学者指南

本文档专为不熟悉 Golang 的开发者编写，详细解释项目中使用的 Go 语言概念和代码结构。

## 目录

1. [Go 语言基础概念](#1-go-语言基础概念)
2. [项目结构详解](#2-项目结构详解)
3. [核心代码逐行解析](#3-核心代码逐行解析)
4. [k6 扩展模式详解](#4-k6-扩展模式详解)
5. [数据类型转换](#5-数据类型转换)
6. [错误处理模式](#6-错误处理模式)
7. [开发工作流程](#7-开发工作流程)
8. [常见问题解答](#8-常见问题解答)

---

## 1. Go 语言基础概念

### 1.1 包（Package）

Go 程序由包组成。每个 `.go` 文件的第一行必须声明它属于哪个包。

```go
package milvus  // 声明这个文件属于 milvus 包
```

**本项目中的包：**

| 包路径 | 说明 |
|-------|------|
| `github.com/mmga-lab/xk6-milvus` | 主模块（入口点） |
| `github.com/mmga-lab/xk6-milvus/pkg/milvus` | 核心实现包 |

### 1.2 导入（Import）

导入其他包的代码：

```go
import (
    "fmt"           // 标准库：格式化输入输出
    "time"          // 标准库：时间操作
    "encoding/json" // 标准库：JSON 编解码

    // 第三方包
    "github.com/milvus-io/milvus/client/v2/milvusclient"  // Milvus 官方 SDK
    "go.k6.io/k6/js/modules"                              // k6 模块系统
)
```

**特殊导入方式：**

```go
import (
    _ "github.com/mmga-lab/xk6-milvus/pkg/milvus"  // 下划线导入：只执行包的 init() 函数
)
```

### 1.3 变量与类型

#### 基本类型

```go
var name string = "hello"    // 显式声明类型
age := 25                    // 短声明（类型推断）
var isActive bool            // 默认值：false
var count int64              // 默认值：0
var price float64            // 默认值：0.0
```

#### 复合类型

```go
// 切片（Slice）- 动态数组
var vectors []float32                    // 一维切片
var matrix [][]float32                   // 二维切片
data := []string{"a", "b", "c"}          // 初始化切片

// 映射（Map）- 键值对
params := map[string]interface{}{        // interface{} 表示可以是任何类型
    "vectorField": "embedding",
    "topK":        10,
}

// 数组（Array）- 固定长度
var arr [3]int = [3]int{1, 2, 3}
```

### 1.4 结构体（Struct）

结构体是 Go 中定义自定义类型的方式：

```go
// 定义结构体
type Client struct {
    client            *milvusclient.Client   // 指针类型字段
    ctx               context.Context        // 上下文
    vu                modules.VU             // 接口类型
    config            *ClientConfig          // 指向配置的指针
    defaultCollection string                 // 字符串字段
}

// 创建结构体实例
client := &Client{                         // & 取地址，返回指针
    client:            milvusClient,
    ctx:               ctx,
    defaultCollection: "my_collection",
}
```

**JSON 标签：**

```go
type OperationResult struct {
    Success      bool        `json:"success"`           // JSON 序列化时使用 "success"
    ResponseTime float64     `json:"response_time_ms"`  // 使用下划线命名
    Result       interface{} `json:"result,omitempty"`  // omitempty：值为空时不输出
    Error        string      `json:"error,omitempty"`
}
```

### 1.5 函数与方法

#### 普通函数

```go
// 函数定义：func 函数名(参数) 返回值类型
func add(a, b int) int {
    return a + b
}

// 多返回值
func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}
```

#### 方法（绑定到类型）

```go
// 方法：func (接收者) 方法名(参数) 返回值
// 接收者类型决定了哪个类型可以调用这个方法

// 值接收者
func (c Client) GetName() string {
    return c.defaultCollection
}

// 指针接收者（可以修改结构体）
func (c *Client) Close() error {
    return c.client.Close(c.ctx)
}
```

#### 可变参数

```go
// ... 表示可变参数，可以传入 0 个或多个
func (c *Client) DropCollection(collectionName ...string) interface{} {
    name := c.defaultCollection
    if len(collectionName) > 0 && collectionName[0] != "" {
        name = collectionName[0]
    }
    // ...
}

// 调用方式
client.DropCollection()                    // 不传参数
client.DropCollection("my_collection")     // 传入一个参数
```

### 1.6 接口（Interface）

接口定义了一组方法签名：

```go
// 定义接口
type Module interface {
    NewModuleInstance(vu VU) Instance
}

type Instance interface {
    Exports() Exports
}

// 实现接口（隐式实现，无需声明）
type RootModule struct{}

func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
    return &Milvus{vu: vu}
}

// 接口断言：确保类型实现了接口
var _ modules.Module = &RootModule{}    // 编译时检查
var _ modules.Instance = &Milvus{}
```

### 1.7 错误处理

Go 使用显式错误返回而非异常：

```go
// 函数返回 error
func doSomething() error {
    return errors.New("something went wrong")
}

// 处理错误
result, err := someOperation()
if err != nil {
    return fmt.Errorf("operation failed: %v", err)  // 包装错误
}

// 错误类型定义
var ErrCollectionNameRequired = errors.New("collection name required")
```

### 1.8 指针

```go
var x int = 10
var p *int = &x    // p 是指向 x 的指针，& 取地址
*p = 20            // * 解引用，修改 x 的值

// 结构体指针
config := &ClientConfig{Address: "localhost:19530"}  // 创建并返回指针
fmt.Println(config.Address)  // 自动解引用，等价于 (*config).Address
```

### 1.9 init 函数

`init()` 函数在包被导入时自动执行：

```go
func init() {
    // 包初始化代码
    modules.Register("k6/x/milvus", new(RootModule))
}
```

执行顺序：
1. 导入的包的 `init()` 函数
2. 包级变量初始化
3. 当前包的 `init()` 函数
4. `main()` 函数（如果是主程序）

---

## 2. 项目结构详解

### 2.1 目录结构

```
xk6-milvus/
├── register.go              # 入口点（6 行代码）
├── go.mod                   # Go 模块定义
├── go.sum                   # 依赖校验和
├── Makefile                 # 构建自动化
│
├── pkg/milvus/              # 核心实现包
│   ├── module.go            # k6 模块注册
│   ├── client.go            # 客户端创建
│   ├── collection.go        # 集合操作
│   ├── data.go              # 数据操作
│   ├── search.go            # 搜索操作
│   ├── index.go             # 索引管理
│   ├── converters.go        # 类型转换
│   ├── types.go             # 类型定义
│   ├── errors.go            # 错误处理
│   ├── config.go            # 配置结构
│   └── helpers.go           # 辅助函数
│
├── examples/                # 使用示例
└── docs/                    # 文档
```

### 2.2 文件职责

| 文件 | 职责 | 代码行数 |
|------|------|----------|
| `register.go` | 导入 pkg/milvus 触发注册 | ~6 |
| `module.go` | k6 模块注册，VU 实例创建 | ~42 |
| `client.go` | 客户端工厂方法 | ~65 |
| `types.go` | 类型定义（Schema、Field 等） | ~87 |
| `collection.go` | 集合 CRUD 操作 | ~320 |
| `data.go` | Insert/Upsert/Delete | ~124 |
| `search.go` | Search/Query/HybridSearch | ~430 |
| `converters.go` | JS 类型转 Milvus 类型 | ~237 |
| `errors.go` | 错误类型和包装器 | ~56 |
| `config.go` | 配置选项模式 | ~79 |
| `helpers.go` | 通用辅助函数 | ~32 |

### 2.3 代码流向

```
JavaScript 代码
    ↓
register.go (导入 pkg/milvus)
    ↓
module.go init() → 注册到 k6
    ↓
module.go Exports() → 暴露 Client/ClientWithCollection
    ↓
client.go → 创建 Client 实例
    ↓
collection.go / data.go / search.go → 具体操作
    ↓
converters.go → 类型转换
    ↓
Milvus Go SDK
    ↓
Milvus 服务器 (gRPC)
```

---

## 3. 核心代码逐行解析

### 3.1 register.go - 入口点

```go
package milvus

import (
    _ "github.com/mmga-lab/xk6-milvus/pkg/milvus" // register the extension
)
```

**解析：**
- `package milvus`：声明包名
- `_` 下划线导入：只执行 `pkg/milvus` 包的 `init()` 函数，不使用其导出的任何内容
- 这是 k6 扩展的标准模式，xk6 构建工具会自动导入这个包

### 3.2 module.go - k6 模块注册

```go
package milvus

import (
    "go.k6.io/k6/js/modules"
)

// init 函数在包导入时自动执行
func init() {
    // 将扩展注册到 k6 模块系统
    // "k6/x/milvus" 是 JavaScript 中的导入路径
    modules.Register("k6/x/milvus", new(RootModule))
}

// 编译时接口检查
// 确保 RootModule 实现了 modules.Module 接口
// 确保 Milvus 实现了 modules.Instance 接口
var (
    _ modules.Module   = &RootModule{}
    _ modules.Instance = &Milvus{}
)

// RootModule 是全局模块实例
// 它为每个 VU（Virtual User）创建独立的模块实例
type RootModule struct{}

// Milvus 是每个 VU 的模块实例
// vu 字段保存了 VU 的上下文信息
type Milvus struct {
    vu modules.VU  // VU 上下文，用于获取运行时信息
}

// NewModuleInstance 实现 modules.Module 接口
// k6 为每个 VU 调用此方法创建新实例
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
    return &Milvus{vu: vu}
}

// Exports 实现 modules.Instance 接口
// 返回 JavaScript 可以访问的导出内容
func (m *Milvus) Exports() modules.Exports {
    return modules.Exports{
        Default: m,  // 默认导出：import milvus from "k6/x/milvus"
        Named: map[string]interface{}{
            // 命名导出：import { client } from "k6/x/milvus"
            "client":               m.Client,
            "clientWithCollection": m.ClientWithCollection,
        },
    }
}
```

### 3.3 types.go - 类型定义

```go
package milvus

import (
    "context"
    "github.com/milvus-io/milvus/client/v2/milvusclient"
    "go.k6.io/k6/js/modules"
)

// OperationResult 是所有操作的统一返回结构
// 遵循 Locust 的设计模式，确保一致的指标收集
type OperationResult struct {
    Success      bool        `json:"success"`           // 操作是否成功
    ResponseTime float64     `json:"response_time_ms"`  // 响应时间（毫秒）
    Result       interface{} `json:"result,omitempty"`  // 操作结果（可选）
    Error        string      `json:"error,omitempty"`   // 错误信息（可选）
    Empty        bool        `json:"empty"`             // 结果集是否为空
    Recall       float32     `json:"recall"`            // 召回率（仅搜索操作）
}

// Client 封装 Milvus 客户端
type Client struct {
    client            *milvusclient.Client  // Milvus SDK 客户端（指针类型）
    ctx               context.Context       // 操作上下文
    vu                modules.VU            // k6 VU 引用
    config            *ClientConfig         // 客户端配置
    defaultCollection string                // 默认集合名（集合绑定模式）
}

// Field 表示 Schema 中的字段定义
type Field struct {
    Name           string                 `json:"name"`                     // 字段名
    DataType       string                 `json:"dataType"`                 // 数据类型
    IsPrimaryKey   bool                   `json:"isPrimaryKey,omitempty"`   // 是否主键
    IsAutoID       bool                   `json:"isAutoID,omitempty"`       // 是否自动生成 ID
    Dimension      int64                  `json:"dimension,omitempty"`      // 向量维度
    Description    string                 `json:"description,omitempty"`    // 字段描述
    MaxLength      int64                  `json:"maxLength,omitempty"`      // 最大长度
    EnableAnalyzer bool                   `json:"enableAnalyzer,omitempty"` // 启用分析器
    EnableMatch    bool                   `json:"enableMatch,omitempty"`    // 启用匹配
    AnalyzerParams map[string]interface{} `json:"analyzerParams,omitempty"` // 分析器参数
}

// Schema 表示集合的模式定义
type Schema struct {
    Name        string     `json:"name"`                // 集合名
    Description string     `json:"description"`         // 描述
    Fields      []Field    `json:"fields"`              // 字段列表
    Functions   []Function `json:"functions,omitempty"` // 函数列表（BM25等）
    NumShards   int32      `json:"numShards,omitempty"` // 分片数
}
```

### 3.4 client.go - 客户端创建

```go
package milvus

import (
    "fmt"
    "strings"
    "github.com/milvus-io/milvus/client/v2/milvusclient"
)

// Client 创建普通 Milvus 客户端（不绑定集合）
// address: Milvus 服务器地址，如 "localhost:19530"
// token: 可选的认证令牌，格式 "username:password"
func (m *Milvus) Client(address string, token ...string) (*Client, error) {
    return m.createClient(address, "", token...)
}

// ClientWithCollection 创建绑定到特定集合的客户端
// 这遵循 Locust 的模式，简化后续操作（不需要重复传入集合名）
func (m *Milvus) ClientWithCollection(address, collectionName string, token ...string) (*Client, error) {
    return m.createClient(address, collectionName, token...)
}

// createClient 是内部客户端创建函数
func (m *Milvus) createClient(address, collectionName string, token ...string) (*Client, error) {
    // 从 VU 获取上下文（用于请求取消和超时）
    ctx := m.vu.Context()

    // 创建配置对象
    clientConfig := DefaultClientConfig()
    clientConfig.Address = address
    clientConfig.DefaultCollection = collectionName

    // 解析认证令牌（格式："username:password"）
    if len(token) > 0 && token[0] != "" {
        parts := strings.Split(token[0], ":")
        if len(parts) == 2 {
            clientConfig.Username = parts[0]
            clientConfig.Password = parts[1]
        }
    }

    // 创建 Milvus SDK 配置
    milvusConfig := &milvusclient.ClientConfig{
        Address: clientConfig.Address,
    }

    // 设置认证信息
    if clientConfig.Username != "" {
        milvusConfig.Username = clientConfig.Username
        milvusConfig.Password = clientConfig.Password
    }

    // 创建 Milvus 客户端
    c, err := milvusclient.New(ctx, milvusConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create milvus client: %v", err)
    }

    // 返回封装的 Client
    return &Client{
        client:            c,
        ctx:               ctx,
        vu:                m.vu,
        config:            clientConfig,
        defaultCollection: collectionName,
    }, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
    return c.client.Close(c.ctx)
}
```

### 3.5 collection.go - 集合操作（部分示例）

```go
package milvus

import (
    "encoding/json"
    "fmt"
    "time"
    "github.com/milvus-io/milvus/client/v2/entity"
    "github.com/milvus-io/milvus/client/v2/milvusclient"
)

// CreateCollection 创建集合
// schemaInput: 可以是 Go 结构体或 JavaScript 对象（通过 JSON 转换）
func (c *Client) CreateCollection(schemaInput interface{}) interface{} {
    start := time.Now()  // 记录开始时间，用于计算响应时间

    // 步骤 1：将 interface{} 转换为 Schema 结构体
    // 使用 JSON 序列化/反序列化确保正确处理 JavaScript 对象
    var schema Schema
    schemaBytes, err := json.Marshal(schemaInput)
    if err != nil {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        fmt.Sprintf("failed to marshal schema: %v", err),
        })
    }
    err = json.Unmarshal(schemaBytes, &schema)
    if err != nil {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        fmt.Sprintf("failed to unmarshal schema: %v", err),
        })
    }

    // 步骤 2：创建 Milvus Entity Schema
    // 使用建造者模式（Builder Pattern）构建 Schema
    entitySchema := entity.NewSchema().
        WithName(schema.Name).
        WithDescription(schema.Description)

    // 步骤 3：添加字段
    for _, field := range schema.Fields {
        entityField := entity.NewField().
            WithName(field.Name).
            WithDescription(field.Description)

        // 根据数据类型设置字段类型
        switch field.DataType {
        case "Int64":
            entityField = entityField.WithDataType(entity.FieldTypeInt64)
        case "Float":
            entityField = entityField.WithDataType(entity.FieldTypeFloat)
        case "VarChar":
            entityField = entityField.WithDataType(entity.FieldTypeVarChar)
        case "FloatVector":
            entityField = entityField.WithDataType(entity.FieldTypeFloatVector).
                WithDim(field.Dimension)  // 设置向量维度
        case "SparseFloatVector":
            entityField = entityField.WithDataType(entity.FieldTypeSparseVector)
        // ... 其他类型
        default:
            return toMap(&OperationResult{
                Success:      false,
                ResponseTime: float64(time.Since(start).Milliseconds()),
                Error:        fmt.Sprintf("unsupported data type: '%s'", field.DataType),
            })
        }

        // 设置主键和自动 ID
        if field.IsPrimaryKey {
            entityField = entityField.WithIsPrimaryKey(true)
        }
        if field.IsAutoID {
            entityField = entityField.WithIsAutoID(true)
        }

        entitySchema = entitySchema.WithField(entityField)
    }

    // 步骤 4：创建集合
    option := milvusclient.NewCreateCollectionOption(schema.Name, entitySchema)
    err = c.client.CreateCollection(c.ctx, option)
    if err != nil {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        fmt.Sprintf("failed to create collection: %v", err),
        })
    }

    // 步骤 5：返回成功结果
    return toMap(&OperationResult{
        Success:      true,
        ResponseTime: float64(time.Since(start).Milliseconds()),
        Result:       map[string]interface{}{"collection": schema.Name},
    })
}

// LoadCollection 将集合加载到内存
func (c *Client) LoadCollection(collectionName ...string) interface{} {
    start := time.Now()

    // 获取集合名（支持默认集合）
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

    // 创建加载选项
    option := milvusclient.NewLoadCollectionOption(name)

    // LoadCollection 返回一个 Task，需要等待完成
    task, err := c.client.LoadCollection(c.ctx, option)
    if err != nil {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        fmt.Sprintf("failed to load collection: %v", err),
        })
    }

    // 等待加载完成
    err = task.Await(c.ctx)
    if err != nil {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        fmt.Sprintf("failed to wait for collection load: %v", err),
        })
    }

    return toMap(&OperationResult{
        Success:      true,
        ResponseTime: float64(time.Since(start).Milliseconds()),
        Result:       map[string]interface{}{"collection": name},
    })
}
```

### 3.6 converters.go - 类型转换

```go
package milvus

import (
    "fmt"
    "github.com/milvus-io/milvus/client/v2/column"
    "github.com/milvus-io/milvus/client/v2/entity"
)

// convertDataToColumns 将 JavaScript 数据转换为 Milvus 列
// 这是核心转换函数，处理 JS 和 Go 之间的类型差异
func (c *Client) convertDataToColumns(data map[string]interface{}) ([]column.Column, error) {
    var columns []column.Column

    // 遍历每个字段
    for fieldName, fieldData := range data {
        col, err := c.convertFieldToColumn(fieldName, fieldData)
        if err != nil {
            return nil, wrapError("convertDataToColumns", err)
        }
        if col != nil {
            columns = append(columns, col)
        }
    }

    if len(columns) == 0 {
        return nil, wrapError("convertDataToColumns", ErrEmptyData)
    }

    return columns, nil
}

// convertFieldToColumn 转换单个字段
// 使用类型断言（Type Assertion）处理不同类型
func (c *Client) convertFieldToColumn(fieldName string, fieldData interface{}) (column.Column, error) {
    // switch v := fieldData.(type) 是类型开关（Type Switch）
    // v 获取转换后的值，fieldData.(type) 获取类型
    switch v := fieldData.(type) {

    case [][]float32:
        // 二维 float32 切片 - 通常是向量数据
        if len(v) == 0 {
            return nil, nil
        }
        dim := len(v[0])  // 向量维度
        return column.NewColumnFloatVector(fieldName, dim, v), nil

    case []int64:
        // int64 切片
        return column.NewColumnInt64(fieldName, v), nil

    case []string:
        // 字符串切片
        return column.NewColumnVarChar(fieldName, v), nil

    case []interface{}:
        // interface{} 切片 - 需要进一步检查元素类型
        // JavaScript 数组通常会以这种形式传入
        return c.convertInterfaceSlice(fieldName, v)

    default:
        return nil, newError("convertFieldToColumn", ErrUnsupportedType,
            fmt.Sprintf("field %s has type %T", fieldName, fieldData))
    }
}

// convertInterfaceSlice 处理 []interface{} 类型
// 这是处理 JavaScript 数组的关键函数
func (c *Client) convertInterfaceSlice(fieldName string, v []interface{}) (column.Column, error) {
    if len(v) == 0 {
        return nil, nil
    }

    // 根据第一个元素的类型决定如何处理整个数组
    switch v[0].(type) {

    case int64:
        // 转换为 int64 切片
        ids := make([]int64, len(v))
        for i, val := range v {
            if id, ok := val.(int64); ok {
                ids[i] = id
            }
        }
        return column.NewColumnInt64(fieldName, ids), nil

    case string:
        // 转换为 string 切片
        strs := make([]string, len(v))
        for i, val := range v {
            if str, ok := val.(string); ok {
                strs[i] = str
            }
        }
        return column.NewColumnVarChar(fieldName, strs), nil

    case float64:
        // JSON 数字默认解析为 float64
        return c.convertFloat64Slice(fieldName, v)

    case []interface{}:
        // 嵌套数组 - 通常是向量数据
        return c.convertNestedVectors(fieldName, v)

    case map[string]interface{}:
        // 对象数组 - 通常是稀疏向量
        maps := make([]map[string]interface{}, len(v))
        for i, val := range v {
            if m, ok := val.(map[string]interface{}); ok {
                maps[i] = m
            }
        }
        return c.convertSparseVectors(fieldName, maps)

    default:
        return nil, newError("convertInterfaceSlice", ErrUnsupportedType,
            fmt.Sprintf("field %s has element type %T", fieldName, v[0]))
    }
}
```

### 3.7 helpers.go - 辅助函数

```go
package milvus

import "encoding/json"

// getCollectionName 获取集合名
// 支持可选参数，如果未提供则使用默认集合
func (c *Client) getCollectionName(collectionName ...string) string {
    if len(collectionName) > 0 && collectionName[0] != "" {
        return collectionName[0]
    }
    return c.defaultCollection
}

// toMap 将 OperationResult 转换为 map[string]interface{}
// 这确保 JavaScript 可以正确访问结果字段
func toMap(result *OperationResult) map[string]interface{} {
    // 序列化为 JSON
    data, err := json.Marshal(result)
    if err != nil {
        return map[string]interface{}{
            "success": false,
            "error":   err.Error(),
        }
    }

    // 反序列化为 map
    var m map[string]interface{}
    if err := json.Unmarshal(data, &m); err != nil {
        return map[string]interface{}{
            "success": false,
            "error":   err.Error(),
        }
    }
    return m
}
```

### 3.8 config.go - 选项模式

```go
package milvus

import "time"

// ClientConfig 客户端配置
type ClientConfig struct {
    Address           string
    Username          string
    Password          string
    DefaultCollection string
    Timeout           time.Duration
    MaxRetries        int
    Debug             bool
}

// ClientOption 是配置函数类型
// 这是 Go 中常见的"函数选项"模式
type ClientOption func(*ClientConfig)

// DefaultClientConfig 返回默认配置
func DefaultClientConfig() *ClientConfig {
    return &ClientConfig{
        Timeout:    30 * time.Second,
        MaxRetries: 3,
        Debug:      false,
    }
}

// WithAddress 设置地址的选项函数
func WithAddress(address string) ClientOption {
    return func(c *ClientConfig) {
        c.Address = address
    }
}

// WithTimeout 设置超时的选项函数
func WithTimeout(timeout time.Duration) ClientOption {
    return func(c *ClientConfig) {
        c.Timeout = timeout
    }
}

// ApplyOptions 应用所有选项
func (c *ClientConfig) ApplyOptions(opts ...ClientOption) {
    for _, opt := range opts {
        opt(c)  // 调用每个选项函数
    }
}

// 使用示例：
// config := DefaultClientConfig()
// config.ApplyOptions(
//     WithAddress("localhost:19530"),
//     WithTimeout(60 * time.Second),
// )
```

### 3.9 errors.go - 错误处理

```go
package milvus

import (
    "errors"
    "fmt"
)

// 预定义错误（哨兵错误）
var (
    ErrCollectionNameRequired = errors.New("collection name required")
    ErrEmptyData              = errors.New("no valid columns provided")
    ErrEmptyVectorArray       = errors.New("empty vector array")
    ErrNoSearchRequests       = errors.New("at least one search request required")
    ErrInvalidDataType        = errors.New("invalid data type")
    ErrUnsupportedType        = errors.New("unsupported type")
)

// MilvusError 自定义错误类型
// 包含额外的上下文信息
type MilvusError struct {
    Op      string  // 失败的操作
    Err     error   // 底层错误
    Context string  // 额外上下文
}

// Error 实现 error 接口
func (e *MilvusError) Error() string {
    if e.Context != "" {
        return fmt.Sprintf("%s: %s: %v", e.Op, e.Context, e.Err)
    }
    return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap 支持错误链
// 允许使用 errors.Is() 和 errors.As()
func (e *MilvusError) Unwrap() error {
    return e.Err
}

// newError 创建新的 MilvusError
func newError(op string, err error, context string) error {
    return &MilvusError{
        Op:      op,
        Err:     err,
        Context: context,
    }
}

// wrapError 包装现有错误
func wrapError(op string, err error) error {
    if err == nil {
        return nil
    }
    return &MilvusError{
        Op:  op,
        Err: err,
    }
}
```

---

## 4. k6 扩展模式详解

### 4.1 RootModule / ModuleInstance 模式

k6 扩展使用两层模式：

```
+------------------+
|   RootModule     |  ← 全局单例，程序启动时创建
+------------------+
        |
        | NewModuleInstance() 为每个 VU 创建
        ↓
+------------------+     +------------------+     +------------------+
|  Milvus (VU 1)   |     |  Milvus (VU 2)   |     |  Milvus (VU N)   |
+------------------+     +------------------+     +------------------+
        |                        |                        |
        ↓                        ↓                        ↓
+------------------+     +------------------+     +------------------+
|  Client (VU 1)   |     |  Client (VU 2)   |     |  Client (VU N)   |
+------------------+     +------------------+     +------------------+
```

### 4.2 VU 上下文

VU（Virtual User）上下文提供：

```go
type VU interface {
    Context() context.Context  // 操作上下文（支持取消）
    State() *State            // VU 状态
    Runtime() *goja.Runtime   // JavaScript 运行时
}
```

**为什么使用 VU 上下文？**
- 每个 VU 有独立的上下文
- k6 测试结束时可以取消所有操作
- 确保资源正确清理

### 4.3 导出到 JavaScript

```go
func (m *Milvus) Exports() modules.Exports {
    return modules.Exports{
        // 默认导出
        Default: m,

        // 命名导出
        Named: map[string]interface{}{
            "client":               m.Client,
            "clientWithCollection": m.ClientWithCollection,
        },
    }
}
```

**JavaScript 中的使用：**

```javascript
// 默认导出
import milvus from "k6/x/milvus";
const client = milvus.client("localhost:19530");

// 命名导出
import { client, clientWithCollection } from "k6/x/milvus";
const c = client("localhost:19530");
```

---

## 5. 数据类型转换

### 5.1 JavaScript 到 Go 类型映射

| JavaScript 类型 | Go 类型 | 说明 |
|----------------|---------|------|
| `number` (整数) | `float64` → `int64` | JSON 数字默认为 float64 |
| `number` (小数) | `float64` → `float32` | 转换为 Milvus 的 Float |
| `string` | `string` | 直接映射 |
| `boolean` | `bool` | 直接映射 |
| `Array<number>` | `[]interface{}` → `[]float32` | 需要类型转换 |
| `Array<Array<number>>` | `[]interface{}` → `[][]float32` | 向量数组 |
| `Object` | `map[string]interface{}` | 任意对象 |

### 5.2 类型断言示例

```go
// 类型断言基本语法
value, ok := someInterface.(TargetType)
if !ok {
    // 类型不匹配
}

// 实际使用
func processData(data interface{}) {
    switch v := data.(type) {
    case float64:
        fmt.Printf("Float: %f\n", v)
    case string:
        fmt.Printf("String: %s\n", v)
    case []interface{}:
        fmt.Printf("Array with %d elements\n", len(v))
    case map[string]interface{}:
        fmt.Printf("Object with %d keys\n", len(v))
    default:
        fmt.Printf("Unknown type: %T\n", v)
    }
}
```

### 5.3 JSON 作为中间格式

```go
// JavaScript 对象 → Go 结构体
func parseSchema(input interface{}) (Schema, error) {
    var schema Schema

    // 步骤 1：序列化为 JSON
    bytes, err := json.Marshal(input)
    if err != nil {
        return schema, err
    }

    // 步骤 2：反序列化为目标类型
    err = json.Unmarshal(bytes, &schema)
    return schema, err
}
```

---

## 6. 错误处理模式

### 6.1 统一返回结构

所有操作返回 `OperationResult`，而不是抛出异常：

```go
func (c *Client) SomeOperation() interface{} {
    start := time.Now()

    // 执行操作
    result, err := c.doSomething()
    if err != nil {
        // 返回错误结果（不是 panic 或 throw）
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        err.Error(),
        })
    }

    // 返回成功结果
    return toMap(&OperationResult{
        Success:      true,
        ResponseTime: float64(time.Since(start).Milliseconds()),
        Result:       result,
    })
}
```

### 6.2 JavaScript 中的错误处理

```javascript
import milvus from "k6/x/milvus";
import { check } from "k6";

export default function() {
    const client = milvus.client("localhost:19530");

    const result = client.createCollection({
        name: "test",
        fields: [/* ... */]
    });

    // 检查结果
    check(result, {
        "operation successful": (r) => r.success === true,
        "no error": (r) => r.error === undefined || r.error === "",
        "fast response": (r) => r.response_time_ms < 1000,
    });

    // 条件处理
    if (!result.success) {
        console.error(`Operation failed: ${result.error}`);
        return;
    }

    console.log(`Created collection: ${result.result.collection}`);
}
```

---

## 7. 开发工作流程

### 7.1 添加新功能

**步骤 1：定义类型（types.go）**

```go
// 在 types.go 中添加新类型
type MyNewResult struct {
    Count int    `json:"count"`
    Items []Item `json:"items"`
}
```

**步骤 2：实现功能（新文件或现有文件）**

```go
// 在适当的文件中添加方法
func (c *Client) MyNewMethod(param string) interface{} {
    start := time.Now()

    // 实现逻辑...
    result, err := c.client.SomeSDKMethod(c.ctx, param)
    if err != nil {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        err.Error(),
        })
    }

    return toMap(&OperationResult{
        Success:      true,
        ResponseTime: float64(time.Since(start).Milliseconds()),
        Result:       result,
    })
}
```

**步骤 3：添加测试**

```go
// 在 *_test.go 文件中
func TestMyNewMethod(t *testing.T) {
    // 测试代码...
}
```

### 7.2 常用命令

```bash
# 构建
make build

# 运行测试
make test

# 运行 linter
make lint

# 生成覆盖率报告
make coverage

# 启动 Milvus（用于集成测试）
make docker-up

# 运行集成测试
make test-integration-local

# 运行示例
./k6 run examples/basic-operations.js
```

### 7.3 调试技巧

```go
// 打印调试信息
fmt.Printf("Debug: %+v\n", someStruct)  // 打印结构体所有字段
fmt.Printf("Type: %T\n", someValue)      // 打印类型

// JSON 格式化输出
bytes, _ := json.MarshalIndent(data, "", "  ")
fmt.Println(string(bytes))
```

---

## 8. 常见问题解答

### Q1: 为什么使用 `interface{}` 而不是具体类型？

**答：** JavaScript 是动态类型语言，传入 Go 的数据类型在编译时未知。`interface{}` 可以接受任何类型，然后在运行时进行类型断言。

### Q2: 为什么要用 JSON 进行类型转换？

**答：** JavaScript 对象和 Go 结构体之间没有直接的映射方式。JSON 是两者的公共格式：
- JavaScript 对象可以自然地序列化为 JSON
- Go 结构体通过 JSON 标签定义序列化规则

### Q3: 什么是建造者模式（Builder Pattern）？

**答：** Milvus SDK 使用建造者模式构建复杂对象：

```go
// 链式调用
schema := entity.NewSchema().
    WithName("my_collection").
    WithDescription("test").
    WithField(field1).
    WithField(field2)
```

每个 `WithXxx` 方法返回修改后的对象，允许链式调用。

### Q4: 为什么有些方法返回 `interface{}` 而不是 `*OperationResult`？

**答：** 返回 `interface{}` 并通过 `toMap()` 转换为 `map[string]interface{}` 是因为：
- k6 的 JavaScript 引擎（Sobek）更容易处理 map 类型
- 确保 JSON 标签被正确应用（驼峰命名等）

### Q5: 什么是上下文（Context）？

**答：** `context.Context` 用于：
- 请求取消（测试停止时取消进行中的操作）
- 超时控制
- 传递请求范围的值

```go
ctx := m.vu.Context()  // 获取 VU 的上下文
c.client.Search(ctx, option)  // 传递给 SDK 操作
```

### Q6: 如何添加新的数据类型支持？

**答：** 在 `converters.go` 中：

1. 在 `convertFieldToColumn` 的 switch 语句中添加新 case
2. 如果是复杂类型，添加新的转换函数
3. 在 `collection.go` 的 `CreateCollection` 中添加对应的类型处理

### Q7: 为什么使用指针接收者 `(c *Client)` 而不是值接收者 `(c Client)`？

**答：**
- 指针接收者可以修改结构体的字段
- 避免复制大型结构体
- 保持一致性（如果一个方法使用指针，通常所有方法都使用指针）

---

## 附录：Go 语言速查表

### 常用语法

```go
// 变量声明
var x int = 10
y := 20                           // 短声明

// 切片操作
s := []int{1, 2, 3}
s = append(s, 4)                  // 添加元素
len(s)                            // 长度

// Map 操作
m := map[string]int{"a": 1}
m["b"] = 2                        // 添加
delete(m, "a")                    // 删除
v, ok := m["c"]                   // 检查存在

// 错误处理
if err != nil {
    return err
}

// 类型断言
v, ok := x.(string)

// 类型开关
switch v := x.(type) {
case string:
    // ...
case int:
    // ...
}

// 循环
for i := 0; i < 10; i++ { }
for i, v := range slice { }
for k, v := range map { }
```

### 常用包

```go
import (
    "fmt"           // 格式化 I/O
    "time"          // 时间
    "encoding/json" // JSON
    "errors"        // 错误
    "strings"       // 字符串操作
    "context"       // 上下文
)
```

---

*文档版本: 1.0.0*
*最后更新: 2024*

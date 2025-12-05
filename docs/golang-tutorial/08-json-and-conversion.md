# 第8章：JSON 与类型转换

## 学习目标

完成本章后，你将能够：

- ✅ 掌握 Go 的 JSON 编解码
- ✅ 理解结构体标签的作用
- ✅ 了解 JavaScript 与 Go 的类型映射
- ✅ 深入理解项目中的类型转换逻辑

---

## 📖 8.1 JSON 基础

Go 的 `encoding/json` 包提供 JSON 处理功能。

### 序列化（Marshal）

```go
import "encoding/json"

type Person struct {
    Name string
    Age  int
}

p := Person{Name: "Alice", Age: 25}

// 序列化为 JSON
jsonBytes, err := json.Marshal(p)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(jsonBytes))
// 输出: {"Name":"Alice","Age":25}

// 格式化输出
jsonBytes, _ := json.MarshalIndent(p, "", "  ")
fmt.Println(string(jsonBytes))
// 输出:
// {
//   "Name": "Alice",
//   "Age": 25
// }
```

### 反序列化（Unmarshal）

```go
jsonStr := `{"Name":"Bob","Age":30}`

var p Person
err := json.Unmarshal([]byte(jsonStr), &p)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%+v\n", p)  // {Name:Bob Age:30}
```

---

## 📖 8.2 结构体标签

### JSON 标签语法

```go
type User struct {
    ID        int64  `json:"id"`                    // 重命名
    Name      string `json:"name"`                  // 重命名
    Email     string `json:"email,omitempty"`       // 空值不输出
    Password  string `json:"-"`                     // 完全忽略
    CreatedAt string `json:"created_at,omitempty"`  // 下划线命名
}
```

### 标签选项

| 选项 | 说明 | 示例 |
|------|------|------|
| `json:"name"` | 字段重命名 | `Name string` → `"name"` |
| `json:",omitempty"` | 零值时省略 | `""` 或 `0` 时不输出 |
| `json:"-"` | 完全忽略字段 | 不参与序列化/反序列化 |
| `json:",string"` | 数字作为字符串 | `123` → `"123"` |

### 🔍 项目中的 JSON 标签

```go
// pkg/milvus/types.go

type OperationResult struct {
    Success      bool        `json:"success"`           // success
    ResponseTime float64     `json:"response_time_ms"`  // response_time_ms
    Result       interface{} `json:"result,omitempty"`  // 空时省略
    Error        string      `json:"error,omitempty"`   // 空时省略
    Empty        bool        `json:"empty"`             // empty
    Recall       float32     `json:"recall"`            // recall
}

type Field struct {
    Name           string                 `json:"name"`
    DataType       string                 `json:"dataType"`             // 驼峰命名
    IsPrimaryKey   bool                   `json:"isPrimaryKey,omitempty"`
    IsAutoID       bool                   `json:"isAutoID,omitempty"`
    Dimension      int64                  `json:"dimension,omitempty"`  // 非向量字段时省略
    MaxLength      int64                  `json:"maxLength,omitempty"`  // 非字符串字段时省略
    EnableAnalyzer bool                   `json:"enableAnalyzer,omitempty"`
    AnalyzerParams map[string]interface{} `json:"analyzerParams,omitempty"`
}
```

---

## 📖 8.3 动态 JSON

### map[string]interface{}

```go
// 反序列化到 map
jsonStr := `{"name":"Alice","age":25,"scores":[90,85,88]}`
var data map[string]interface{}
json.Unmarshal([]byte(jsonStr), &data)

// 访问（需要类型断言）
name := data["name"].(string)
age := data["age"].(float64)  // JSON 数字默认是 float64!
scores := data["scores"].([]interface{})
```

### interface{} 接收任意 JSON

```go
jsonStr := `[1, "hello", true, null]`
var data interface{}
json.Unmarshal([]byte(jsonStr), &data)

// 类型断言
arr := data.([]interface{})
for _, v := range arr {
    switch val := v.(type) {
    case float64:
        fmt.Println("数字:", val)
    case string:
        fmt.Println("字符串:", val)
    case bool:
        fmt.Println("布尔:", val)
    case nil:
        fmt.Println("空值")
    }
}
```

---

## 📖 8.4 JavaScript 与 Go 类型映射

当 JavaScript 数据传入 Go 时，经过 JSON 序列化/反序列化后的类型映射：

| JavaScript 类型 | JSON 类型 | Go 类型 |
|----------------|-----------|---------|
| `number` (整数) | number | `float64` |
| `number` (小数) | number | `float64` |
| `string` | string | `string` |
| `boolean` | boolean | `bool` |
| `null` | null | `nil` |
| `Array` | array | `[]interface{}` |
| `Object` | object | `map[string]interface{}` |

⚠️ **重要**：JSON 中的所有数字都会变成 `float64`！

### 示例

```javascript
// JavaScript 中
const data = {
    id: 123,          // number
    name: "test",     // string
    price: 19.99,     // number
    active: true,     // boolean
    tags: ["a", "b"], // array
    meta: null        // null
};
```

```go
// Go 中接收为 map[string]interface{}
data := map[string]interface{}{
    "id":     float64(123),      // 注意：是 float64！
    "name":   "test",
    "price":  float64(19.99),
    "active": true,
    "tags":   []interface{}{"a", "b"},
    "meta":   nil,
}
```

---

## 📖 8.5 项目的类型转换器

### 转换流程

```
JavaScript Object
       ↓
   JSON 传输
       ↓
map[string]interface{}
       ↓
   类型转换器
       ↓
Milvus SDK 类型
```

### 🔍 核心转换代码分析

```go
// pkg/milvus/converters.go

// 入口函数：将 map 数据转换为 Milvus 列
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

// 单字段转换
func (c *Client) convertFieldToColumn(fieldName string, fieldData interface{}) (column.Column, error) {
    switch v := fieldData.(type) {
    case [][]float32:
        // 直接传入的 float32 向量（较少见）
        if len(v) == 0 {
            return nil, nil
        }
        dim := len(v[0])
        return column.NewColumnFloatVector(fieldName, dim, v), nil

    case []int64:
        return column.NewColumnInt64(fieldName, v), nil

    case []string:
        return column.NewColumnVarChar(fieldName, v), nil

    case []interface{}:
        // 从 JavaScript 传入的数组通常是这种类型
        return c.convertInterfaceSlice(fieldName, v)

    default:
        return nil, newError("convertFieldToColumn", ErrUnsupportedType,
            fmt.Sprintf("field %s has type %T", fieldName, fieldData))
    }
}
```

### 处理 JavaScript 数组

```go
// pkg/milvus/converters.go

func (c *Client) convertInterfaceSlice(fieldName string, v []interface{}) (column.Column, error) {
    if len(v) == 0 {
        return nil, nil
    }

    // 根据第一个元素判断整个数组的类型
    switch v[0].(type) {
    case int64:
        // Go 原生 int64 数组
        ids := make([]int64, len(v))
        for i, val := range v {
            if id, ok := val.(int64); ok {
                ids[i] = id
            }
        }
        return column.NewColumnInt64(fieldName, ids), nil

    case float64:
        // JavaScript 数字都是 float64
        return c.convertFloat64Slice(fieldName, v)

    case string:
        strs := make([]string, len(v))
        for i, val := range v {
            if str, ok := val.(string); ok {
                strs[i] = str
            }
        }
        return column.NewColumnVarChar(fieldName, strs), nil

    case []interface{}:
        // 嵌套数组 = 向量数据
        return c.convertNestedVectors(fieldName, v)

    case map[string]interface{}:
        // 对象数组 = 稀疏向量
        // ...
    }
}
```

### float64 转换逻辑

```go
// pkg/milvus/converters.go

func (c *Client) convertFloat64Slice(fieldName string, v []interface{}) (column.Column, error) {
    // 检查是否所有值都是整数
    isInteger := true
    for _, val := range v {
        f, ok := val.(float64)
        if !ok {
            return nil, wrapError("convertFloat64Slice", ErrInvalidDataType)
        }
        // 检查是否是整数（没有小数部分）
        if f != float64(int64(f)) {
            isInteger = false
            break
        }
    }

    // ID 字段且都是整数 → 转为 int64
    if isInteger && fieldName == "id" {
        ids := make([]int64, len(v))
        for i, val := range v {
            f := val.(float64)
            ids[i] = int64(f)
        }
        return column.NewColumnInt64(fieldName, ids), nil
    }

    // 其他数字 → 转为 float32
    floats := make([]float32, len(v))
    for i, val := range v {
        f := val.(float64)
        floats[i] = float32(f)
    }
    return column.NewColumnFloat(fieldName, floats), nil
}
```

### 向量转换

```go
// pkg/milvus/converters.go

func (c *Client) convertNestedVectors(fieldName string, v []interface{}) (column.Column, error) {
    if len(v) == 0 {
        return nil, wrapError("convertNestedVectors", ErrEmptyVectorArray)
    }

    // 获取第一个向量来确定维度
    firstVec, ok := v[0].([]interface{})
    if !ok {
        return nil, newError("convertNestedVectors", ErrInvalidDataType,
            fmt.Sprintf("field %s: expected []interface{}, got %T", fieldName, v[0]))
    }

    dim := len(firstVec)
    vectors := make([][]float32, len(v))

    // 转换每个向量
    for i, vecInterface := range v {
        vec, ok := vecInterface.([]interface{})
        if !ok {
            return nil, newError("convertNestedVectors", ErrInvalidDataType,
                fmt.Sprintf("field %s: vector %d is not []interface{}", fieldName, i))
        }

        floatVec := make([]float32, len(vec))
        for j, val := range vec {
            // 处理不同的数值类型
            switch num := val.(type) {
            case float64:
                floatVec[j] = float32(num)
            case int:
                floatVec[j] = float32(num)
            case int64:
                floatVec[j] = float32(num)
            default:
                return nil, newError("convertNestedVectors", ErrInvalidDataType,
                    fmt.Sprintf("vector element is not number: %T", val))
            }
        }
        vectors[i] = floatVec
    }

    return column.NewColumnFloatVector(fieldName, dim, vectors), nil
}
```

---

## 📖 8.6 使用 JSON 作为中间格式

### 为什么使用 JSON 转换？

```go
// pkg/milvus/collection.go

func (c *Client) CreateCollection(schemaInput interface{}) interface{} {
    // schemaInput 从 JavaScript 传入，是 map[string]interface{}
    // 但我们想用强类型的 Schema 结构体

    var schema Schema

    // 方式1：直接类型断言？
    // schema := schemaInput.(Schema)  // 会失败！类型不匹配

    // 方式2：使用 JSON 作为中间格式
    schemaBytes, err := json.Marshal(schemaInput)  // map → JSON
    if err != nil {
        // 处理错误
    }
    err = json.Unmarshal(schemaBytes, &schema)     // JSON → struct
    // 现在 schema 是强类型的 Schema 结构体
}
```

### 🔍 helpers.go 中的 toMap

```go
// pkg/milvus/helpers.go

// 将 OperationResult 转换为 map，确保 JavaScript 可以访问
func toMap(result *OperationResult) map[string]interface{} {
    // 序列化为 JSON（应用 json 标签）
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

**为什么这样做？**

1. 确保字段名按 JSON 标签转换（如 `ResponseTime` → `response_time_ms`）
2. 处理 `omitempty`（空值字段不包含在输出中）
3. JavaScript 可以直接访问这些字段

---

## ✏️ 动手练习

### 练习 1：JSON 序列化

```go
package main

import (
    "encoding/json"
    "fmt"
)

type Product struct {
    ID          int64   `json:"id"`
    Name        string  `json:"name"`
    Price       float64 `json:"price"`
    Description string  `json:"description,omitempty"`
    InStock     bool    `json:"in_stock"`
    Tags        []string `json:"tags,omitempty"`
}

func main() {
    // 创建产品
    p1 := Product{
        ID:      1,
        Name:    "Laptop",
        Price:   999.99,
        InStock: true,
        Tags:    []string{"electronics", "computer"},
    }

    p2 := Product{
        ID:      2,
        Name:    "Mouse",
        Price:   29.99,
        InStock: false,
        // Description 和 Tags 为空
    }

    // 序列化
    for _, p := range []Product{p1, p2} {
        jsonBytes, _ := json.MarshalIndent(p, "", "  ")
        fmt.Printf("Product %d:\n%s\n\n", p.ID, string(jsonBytes))
    }
}
```

### 练习 2：动态 JSON 处理

```go
package main

import (
    "encoding/json"
    "fmt"
)

func main() {
    // 模拟 JavaScript 传入的数据
    jsonStr := `{
        "collection": "products",
        "data": {
            "id": [1, 2, 3],
            "name": ["A", "B", "C"],
            "price": [10.5, 20.0, 30.99],
            "vector": [[0.1, 0.2], [0.3, 0.4], [0.5, 0.6]]
        }
    }`

    // 反序列化为 map
    var input map[string]interface{}
    json.Unmarshal([]byte(jsonStr), &input)

    // 提取集合名
    collection := input["collection"].(string)
    fmt.Println("Collection:", collection)

    // 处理数据
    data := input["data"].(map[string]interface{})
    for fieldName, fieldData := range data {
        fmt.Printf("\nField: %s\n", fieldName)

        arr := fieldData.([]interface{})
        fmt.Printf("  Type of first element: %T\n", arr[0])
        fmt.Printf("  Values: %v\n", arr)
    }
}
```

### 练习 3：类型转换练习

```go
package main

import (
    "encoding/json"
    "fmt"
)

// 模拟 JavaScript 传入的向量数据
func simulateJSVectors() []interface{} {
    jsonStr := `[[0.1, 0.2, 0.3], [0.4, 0.5, 0.6], [0.7, 0.8, 0.9]]`
    var data []interface{}
    json.Unmarshal([]byte(jsonStr), &data)
    return data
}

// 转换为 [][]float32
func convertToFloatVectors(data []interface{}) [][]float32 {
    vectors := make([][]float32, len(data))

    for i, vecInterface := range data {
        vec := vecInterface.([]interface{})
        floatVec := make([]float32, len(vec))

        for j, val := range vec {
            floatVec[j] = float32(val.(float64))
        }

        vectors[i] = floatVec
    }

    return vectors
}

func main() {
    jsData := simulateJSVectors()
    fmt.Printf("原始数据类型: %T\n", jsData)
    fmt.Printf("元素类型: %T\n", jsData[0])

    vectors := convertToFloatVectors(jsData)
    fmt.Printf("\n转换后类型: %T\n", vectors)
    fmt.Printf("向量: %v\n", vectors)
}
```

### 练习 4：深入分析项目代码

阅读 `pkg/milvus/search.go` 中的 `HybridSearch` 方法：

1. 找出哪些地方使用了 `json.Marshal/Unmarshal`
2. 为什么需要将 `requestsInput interface{}` 转换为 `[]HybridSearchRequest`？
3. 稀疏向量是如何从 `map[string]interface{}` 转换的？

---

## ❓ 自测问题

1. JavaScript 中的数字 `123` 在 Go 中解析为什么类型？
   <details>
   <summary>查看答案</summary>
   float64。JSON 中的所有数字默认解析为 float64。
   </details>

2. `json:"name,omitempty"` 中的 `omitempty` 什么时候生效？
   <details>
   <summary>查看答案</summary>
   当字段值为零值（空字符串、0、false、nil、空切片、空map）时，序列化不输出该字段。
   </details>

3. 为什么项目使用 JSON 序列化/反序列化来转换类型？
   <details>
   <summary>查看答案</summary>
   因为 JavaScript 对象传入 Go 时是 map[string]interface{}，无法直接转换为 Go 结构体。JSON 作为中间格式，可以利用 json 标签进行正确的字段映射。
   </details>

4. 如何安全地从 `interface{}` 获取 `float64` 值？
   <details>
   <summary>查看答案</summary>
   使用带检查的类型断言：`if val, ok := v.(float64); ok { ... }`
   </details>

5. 项目中 `toMap()` 函数的作用是什么？
   <details>
   <summary>查看答案</summary>
   将 OperationResult 结构体转换为 map[string]interface{}，确保 JSON 标签被应用，使 JavaScript 可以正确访问字段。
   </details>

---

## 💡 本章要点

1. **JSON 序列化**：`json.Marshal()` 和 `json.Unmarshal()`
2. **结构体标签**：控制序列化行为
3. **类型映射**：JSON 数字 → `float64`
4. **动态 JSON**：`map[string]interface{}` 和 `[]interface{}`
5. **类型转换**：需要遍历和逐个转换
6. **JSON 作为桥梁**：map → JSON → struct

---

## 下一步

在下一章，我们将学习 k6 扩展架构：

- k6 模块系统
- VU（Virtual User）概念
- 扩展注册机制
- 上下文管理

[继续第9章：k6 扩展架构 →](./09-k6-extension-pattern.md)

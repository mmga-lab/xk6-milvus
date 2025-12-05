# 第6章：接口

## 学习目标

完成本章后，你将能够：

- ✅ 理解接口的概念和作用
- ✅ 掌握接口的定义和隐式实现
- ✅ 熟练使用空接口 interface{}
- ✅ 掌握类型断言和类型开关
- ✅ 理解项目中的接口应用

---

## 📖 6.1 接口基础

接口定义了一组方法签名，任何实现了这些方法的类型都实现了该接口。

### 接口定义

```go
// 定义接口
type Writer interface {
    Write(data []byte) (int, error)
}

type Reader interface {
    Read(data []byte) (int, error)
}

// 组合接口
type ReadWriter interface {
    Reader
    Writer
}
```

### 隐式实现

Go 的接口实现是隐式的——不需要 `implements` 关键字：

```go
type Writer interface {
    Write(data []byte) (int, error)
}

// File 类型实现了 Writer 接口
type File struct {
    name string
}

// 只要实现了 Write 方法，File 就自动实现了 Writer
func (f *File) Write(data []byte) (int, error) {
    fmt.Printf("Writing to %s: %s\n", f.name, data)
    return len(data), nil
}

// 使用
var w Writer = &File{name: "test.txt"}
w.Write([]byte("hello"))
```

### 🔍 项目中的接口

```go
// pkg/milvus/module.go

// k6 定义的接口
type modules.Module interface {
    NewModuleInstance(vu VU) Instance
}

type modules.Instance interface {
    Exports() Exports
}

// 项目实现这些接口

// RootModule 实现 modules.Module
type RootModule struct{}

func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
    return &Milvus{vu: vu}
}

// Milvus 实现 modules.Instance
type Milvus struct {
    vu modules.VU
}

func (m *Milvus) Exports() modules.Exports {
    return modules.Exports{
        Default: m,
        Named: map[string]interface{}{
            "client":               m.Client,
            "clientWithCollection": m.ClientWithCollection,
        },
    }
}

// 编译时验证接口实现
var (
    _ modules.Module   = &RootModule{}  // 确保 RootModule 实现了 Module
    _ modules.Instance = &Milvus{}      // 确保 Milvus 实现了 Instance
)
```

---

## 📖 6.2 接口值

接口值由两部分组成：具体类型和具体值。

```go
type Writer interface {
    Write([]byte) (int, error)
}

var w Writer

// w 是 nil（类型和值都是 nil）
fmt.Printf("类型: %T, 值: %v\n", w, w)  // 类型: <nil>, 值: <nil>

// 赋值后
w = &File{name: "test.txt"}
fmt.Printf("类型: %T, 值: %v\n", w, w)  // 类型: *main.File, 值: &{test.txt}
```

```
接口值内部结构：

var w Writer = &File{name: "test"}

w (interface)
┌─────────────┐
│ type: *File │  ← 具体类型
├─────────────┤
│ value: ────┼──→ File{name: "test"}  ← 具体值
└─────────────┘
```

---

## 📖 6.3 空接口 interface{}

空接口没有任何方法，因此所有类型都实现了空接口。

```go
// 空接口可以持有任何类型的值
var any interface{}

any = 42
fmt.Println(any)  // 42

any = "hello"
fmt.Println(any)  // hello

any = []int{1, 2, 3}
fmt.Println(any)  // [1 2 3]

any = struct{ Name string }{"Alice"}
fmt.Println(any)  // {Alice}
```

### Go 1.18+ 的 any 类型

```go
// any 是 interface{} 的别名
var x any = 42
var y any = "hello"
```

### 🔍 项目中的空接口

```go
// pkg/milvus/types.go

type OperationResult struct {
    // Result 可以是任何类型
    Result interface{} `json:"result,omitempty"`
    // ...
}

type HybridSearchRequest struct {
    // Vectors 可以是 [][]float32 或 []map[string]interface{}
    Vectors interface{} `json:"vectors"`
    // Params 可以包含任意参数
    Params map[string]interface{} `json:"params,omitempty"`
}

// pkg/milvus/data.go

// data 参数是 map[string]interface{}
// 值可以是 []int64、[]string、[][]float32 等
func (c *Client) Insert(data map[string]interface{}, ...) interface{} {
    // ...
}

// pkg/milvus/collection.go

// schemaInput 可以是任何类型（JavaScript 对象会变成 map）
func (c *Client) CreateCollection(schemaInput interface{}) interface{} {
    // 通过 JSON 序列化/反序列化来处理
    var schema Schema
    schemaBytes, err := json.Marshal(schemaInput)
    // ...
}
```

---

## 📖 6.4 类型断言

类型断言用于从接口值中提取具体类型的值。

### 基本语法

```go
var i interface{} = "hello"

// 类型断言
s := i.(string)
fmt.Println(s)  // hello

// 如果类型不匹配，会 panic
// n := i.(int)  // panic!

// 安全的类型断言
s, ok := i.(string)
if ok {
    fmt.Println("是字符串:", s)
}

n, ok := i.(int)
if !ok {
    fmt.Println("不是整数")
}
```

### 🔍 项目中的类型断言

```go
// pkg/milvus/converters.go

func (c *Client) convertFieldToColumn(fieldName string, fieldData interface{}) (column.Column, error) {
    // 使用类型开关进行多类型断言
    switch v := fieldData.(type) {
    case [][]float32:
        // v 的类型是 [][]float32
        return column.NewColumnFloatVector(fieldName, len(v[0]), v), nil

    case []int64:
        // v 的类型是 []int64
        return column.NewColumnInt64(fieldName, v), nil

    case []string:
        return column.NewColumnVarChar(fieldName, v), nil

    case []interface{}:
        // 需要进一步检查元素类型
        return c.convertInterfaceSlice(fieldName, v)

    default:
        return nil, newError("convertFieldToColumn", ErrUnsupportedType,
            fmt.Sprintf("field %s has type %T", fieldName, fieldData))
    }
}

// pkg/milvus/search.go

func (c *Client) Search(vectors [][]float32, topK int, params map[string]interface{}, ...) interface{} {
    // 从 params 中提取值
    vectorField := "vector"
    if field, ok := params["vectorField"].(string); ok {
        vectorField = field
    }

    // 提取切片
    var outputFields []string
    if fields, ok := params["outputFields"].([]interface{}); ok {
        outputFields = make([]string, len(fields))
        for i, field := range fields {
            if fieldStr, ok := field.(string); ok {
                outputFields[i] = fieldStr
            }
        }
    }
}
```

---

## 📖 6.5 类型开关（Type Switch）

类型开关是处理多种类型的优雅方式。

```go
func describe(i interface{}) {
    switch v := i.(type) {
    case int:
        fmt.Printf("整数: %d\n", v)
    case string:
        fmt.Printf("字符串: %s\n", v)
    case bool:
        fmt.Printf("布尔: %t\n", v)
    case []int:
        fmt.Printf("整数切片: %v\n", v)
    default:
        fmt.Printf("未知类型: %T\n", v)
    }
}

describe(42)           // 整数: 42
describe("hello")      // 字符串: hello
describe(true)         // 布尔: true
describe([]int{1,2,3}) // 整数切片: [1 2 3]
describe(3.14)         // 未知类型: float64
```

### 🔍 项目中的类型开关

```go
// pkg/milvus/converters.go

// 处理 []interface{} 切片
func (c *Client) convertInterfaceSlice(fieldName string, v []interface{}) (column.Column, error) {
    if len(v) == 0 {
        return nil, nil
    }

    // 根据第一个元素的类型决定处理方式
    switch v[0].(type) {
    case int64:
        ids := make([]int64, len(v))
        for i, val := range v {
            if id, ok := val.(int64); ok {
                ids[i] = id
            }
        }
        return column.NewColumnInt64(fieldName, ids), nil

    case string:
        strs := make([]string, len(v))
        for i, val := range v {
            if str, ok := val.(string); ok {
                strs[i] = str
            }
        }
        return column.NewColumnVarChar(fieldName, strs), nil

    case float64:
        return c.convertFloat64Slice(fieldName, v)

    case []interface{}:
        // 嵌套数组（向量）
        return c.convertNestedVectors(fieldName, v)

    case map[string]interface{}:
        // 对象数组（稀疏向量）
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

---

## 📖 6.6 常用标准库接口

### fmt.Stringer

```go
type Stringer interface {
    String() string
}

// 实现 Stringer
type Person struct {
    Name string
    Age  int
}

func (p Person) String() string {
    return fmt.Sprintf("%s (%d years old)", p.Name, p.Age)
}

p := Person{"Alice", 25}
fmt.Println(p)  // Alice (25 years old)
```

### error 接口

```go
type error interface {
    Error() string
}

// 实现自定义错误
type MyError struct {
    Code    int
    Message string
}

func (e *MyError) Error() string {
    return fmt.Sprintf("Error %d: %s", e.Code, e.Message)
}
```

### io.Reader 和 io.Writer

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}
```

---

## 📖 6.7 接口组合

```go
// 小接口
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

type Closer interface {
    Close() error
}

// 组合接口
type ReadWriter interface {
    Reader
    Writer
}

type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}
```

### 🔍 项目中接口组合的体现

```go
// k6 模块系统中的接口

// VU 接口组合了多个功能
type VU interface {
    Context() context.Context
    State() *State
    Runtime() *goja.Runtime
    // ...
}

// 项目中使用 VU 接口
type Milvus struct {
    vu modules.VU  // 使用接口类型，而不是具体类型
}

// 通过 vu 可以访问上下文
func (m *Milvus) createClient(...) (*Client, error) {
    ctx := m.vu.Context()  // 调用接口方法
    // ...
}
```

---

## ✏️ 动手练习

### 练习 1：定义和实现接口

```go
package main

import "fmt"

// 定义接口
type Shape interface {
    Area() float64
    Perimeter() float64
}

// 实现接口：圆形
type Circle struct {
    Radius float64
}

func (c Circle) Area() float64 {
    return 3.14159 * c.Radius * c.Radius
}

func (c Circle) Perimeter() float64 {
    return 2 * 3.14159 * c.Radius
}

// 实现接口：矩形
type Rectangle struct {
    Width, Height float64
}

func (r Rectangle) Area() float64 {
    return r.Width * r.Height
}

func (r Rectangle) Perimeter() float64 {
    return 2 * (r.Width + r.Height)
}

// 使用接口
func printShapeInfo(s Shape) {
    fmt.Printf("面积: %.2f, 周长: %.2f\n", s.Area(), s.Perimeter())
}

func main() {
    shapes := []Shape{
        Circle{Radius: 5},
        Rectangle{Width: 4, Height: 6},
    }

    for _, s := range shapes {
        fmt.Printf("类型: %T\n", s)
        printShapeInfo(s)
        fmt.Println()
    }
}
```

### 练习 2：类型断言实践

```go
package main

import "fmt"

func processValue(v interface{}) {
    // 使用类型开关处理不同类型
    switch val := v.(type) {
    case int:
        fmt.Printf("整数: %d, 平方: %d\n", val, val*val)
    case float64:
        fmt.Printf("浮点数: %.2f, 平方: %.2f\n", val, val*val)
    case string:
        fmt.Printf("字符串: %s, 长度: %d\n", val, len(val))
    case []int:
        sum := 0
        for _, n := range val {
            sum += n
        }
        fmt.Printf("整数切片: %v, 总和: %d\n", val, sum)
    case map[string]interface{}:
        fmt.Printf("Map: %v\n", val)
        for k, v := range val {
            fmt.Printf("  %s: %v (%T)\n", k, v, v)
        }
    default:
        fmt.Printf("未知类型: %T = %v\n", val, val)
    }
}

func main() {
    processValue(42)
    processValue(3.14)
    processValue("hello")
    processValue([]int{1, 2, 3, 4, 5})
    processValue(map[string]interface{}{
        "name": "Alice",
        "age":  25,
    })
    processValue(true)
}
```

### 练习 3：模拟项目的转换逻辑

```go
package main

import (
    "encoding/json"
    "fmt"
)

// 模拟 JavaScript 传入的数据
func simulateJSInput() map[string]interface{} {
    jsonStr := `{
        "id": [1, 2, 3],
        "title": ["Product A", "Product B", "Product C"],
        "price": [19.99, 29.99, 39.99],
        "vector": [[0.1, 0.2, 0.3], [0.4, 0.5, 0.6], [0.7, 0.8, 0.9]]
    }`

    var data map[string]interface{}
    json.Unmarshal([]byte(jsonStr), &data)
    return data
}

// 处理数据
func processData(data map[string]interface{}) {
    for fieldName, fieldData := range data {
        fmt.Printf("\n字段: %s\n", fieldName)

        switch v := fieldData.(type) {
        case []interface{}:
            if len(v) == 0 {
                continue
            }

            // 检查第一个元素的类型
            switch v[0].(type) {
            case float64:
                // 可能是 ID 或价格
                fmt.Printf("  类型: 数字数组\n")
                fmt.Printf("  值: %v\n", v)

            case string:
                fmt.Printf("  类型: 字符串数组\n")
                fmt.Printf("  值: %v\n", v)

            case []interface{}:
                // 嵌套数组（向量）
                fmt.Printf("  类型: 向量数组\n")
                for i, vec := range v {
                    fmt.Printf("  向量 %d: %v\n", i, vec)
                }
            }

        default:
            fmt.Printf("  未知类型: %T\n", fieldData)
        }
    }
}

func main() {
    data := simulateJSInput()
    processData(data)
}
```

### 练习 4：接口验证

查看 `pkg/milvus/module.go` 中的接口验证：

```go
var (
    _ modules.Module   = &RootModule{}
    _ modules.Instance = &Milvus{}
)
```

尝试理解：
1. 这行代码的作用是什么？
2. 如果删除 `Exports()` 方法会发生什么？

---

## ❓ 自测问题

1. Go 接口实现的特点是什么？
   <details>
   <summary>查看答案</summary>
   隐式实现。只要类型实现了接口定义的所有方法，就自动实现了该接口，不需要显式声明。
   </details>

2. `interface{}` 可以接受什么类型？
   <details>
   <summary>查看答案</summary>
   任何类型。因为空接口没有方法要求，所有类型都实现了空接口。
   </details>

3. `v, ok := i.(Type)` 和 `v := i.(Type)` 的区别是什么？
   <details>
   <summary>查看答案</summary>
   前者是安全的类型断言，类型不匹配时 ok 为 false；后者类型不匹配时会 panic。
   </details>

4. 项目中为什么大量使用 `interface{}` 和 `map[string]interface{}`？
   <details>
   <summary>查看答案</summary>
   因为需要与 JavaScript 交互，JavaScript 是动态类型语言，数据类型在编译时未知。使用 interface{} 可以接受任何类型，然后在运行时进行类型断言和转换。
   </details>

5. `var _ Interface = &Type{}` 的作用是什么？
   <details>
   <summary>查看答案</summary>
   编译时检查 Type 是否实现了 Interface。如果没有实现，编译会失败。这是一种静态验证技巧。
   </details>

---

## 💡 本章要点

1. **接口定义方法签名** - 不包含实现
2. **隐式实现** - 实现方法即实现接口
3. **空接口 interface{}** - 可以持有任何类型
4. **类型断言**：
   - `v := i.(Type)` - 不安全，失败 panic
   - `v, ok := i.(Type)` - 安全，失败返回零值
5. **类型开关** - 优雅处理多种类型
6. **接口验证** - `var _ Interface = &Type{}`

---

## 下一步

在下一章，我们将深入学习错误处理：

- error 接口
- 自定义错误类型
- 错误包装
- 项目中的错误处理模式

[继续第7章：错误处理 →](./07-error-handling.md)

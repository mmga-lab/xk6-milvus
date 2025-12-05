# 第3章：基础类型与变量

## 学习目标

完成本章后，你将能够：

- ✅ 掌握 Go 的基本数据类型
- ✅ 理解变量声明的多种方式
- ✅ 学会使用类型推断
- ✅ 理解零值概念
- ✅ 认识项目中使用的类型

---

## 📖 3.1 基本数据类型

### 数值类型

```go
// 整数类型
var a int     // 平台相关（32位或64位）
var b int8    // -128 到 127
var c int16   // -32768 到 32767
var d int32   // 约 -21亿 到 21亿
var e int64   // 很大的范围

// 无符号整数
var f uint    // 平台相关
var g uint8   // 0 到 255（别名 byte）
var h uint16  // 0 到 65535
var i uint32  // 0 到 约42亿
var j uint64  // 更大

// 浮点数
var k float32  // 单精度
var l float64  // 双精度（推荐）

// 复数（较少使用）
var m complex64
var n complex128
```

### 🔍 项目中的数值类型

```go
// pkg/milvus/types.go

type OperationResult struct {
    Success      bool        `json:"success"`
    ResponseTime float64     `json:"response_time_ms"`  // 使用 float64 存储毫秒
    Recall       float32     `json:"recall"`            // 使用 float32 足够
    // ...
}

type Field struct {
    Dimension int64  `json:"dimension,omitempty"`  // 向量维度用 int64
    MaxLength int64  `json:"maxLength,omitempty"`  // 最大长度用 int64
    // ...
}
```

💡 **为什么用 int64？** Milvus 数据库使用 int64 作为 ID 类型，保持一致性。

### 字符串类型

```go
var s string = "Hello, 世界"  // UTF-8 编码

// 字符串是不可变的
s[0] = 'h'  // 编译错误！

// 使用 + 拼接
greeting := "Hello" + ", " + "World"

// 多行字符串（反引号）
query := `
    SELECT *
    FROM users
    WHERE id = 1
`
```

### 🔍 项目中的字符串

```go
// pkg/milvus/types.go

type Field struct {
    Name        string `json:"name"`         // 字段名
    DataType    string `json:"dataType"`     // 数据类型名称
    Description string `json:"description"`  // 描述
    // ...
}

// pkg/milvus/errors.go

type MilvusError struct {
    Op      string  // 操作名
    Context string  // 上下文信息
    // ...
}
```

### 布尔类型

```go
var isActive bool = true
var isDeleted bool        // 默认值 false

// 布尔运算
result := true && false   // false
result = true || false    // true
result = !true            // false
```

### 🔍 项目中的布尔类型

```go
// pkg/milvus/types.go

type OperationResult struct {
    Success bool `json:"success"`  // 操作是否成功
    Empty   bool `json:"empty"`    // 结果是否为空
}

type Field struct {
    IsPrimaryKey   bool `json:"isPrimaryKey,omitempty"`
    IsAutoID       bool `json:"isAutoID,omitempty"`
    EnableAnalyzer bool `json:"enableAnalyzer,omitempty"`
    EnableMatch    bool `json:"enableMatch,omitempty"`
}
```

---

## 📖 3.2 变量声明

### 方式一：var 声明

```go
// 声明并初始化
var name string = "Alice"
var age int = 25

// 声明多个同类型变量
var x, y, z int = 1, 2, 3

// 声明不同类型（分组）
var (
    name   string = "Bob"
    age    int    = 30
    active bool   = true
)
```

### 方式二：类型推断

```go
// 省略类型，编译器自动推断
var name = "Alice"     // 推断为 string
var age = 25           // 推断为 int
var price = 19.99      // 推断为 float64
var isOK = true        // 推断为 bool
```

### 方式三：短声明（推荐）

```go
// := 只能在函数内使用
name := "Alice"
age := 25
x, y := 10, 20

// 不能用于已声明的变量（除非有新变量）
name := "Bob"          // 错误！name 已存在
name, email := "Bob", "bob@example.com"  // OK，email 是新变量
```

### 🔍 项目中的变量声明

```go
// pkg/milvus/errors.go - 包级变量用 var
var (
    ErrCollectionNameRequired = errors.New("collection name required")
    ErrEmptyData              = errors.New("no valid columns provided")
    ErrEmptyVectorArray       = errors.New("empty vector array")
    // ...
)

// pkg/milvus/config.go - 函数内用短声明
func DefaultClientConfig() *ClientConfig {
    return &ClientConfig{
        Timeout:    30 * time.Second,  // 字面量
        MaxRetries: 3,
        Debug:      false,
    }
}

// pkg/milvus/collection.go
func (c *Client) CreateCollection(schemaInput interface{}) interface{} {
    start := time.Now()  // 短声明

    var schema Schema    // var 声明（稍后赋值）
    // ...
}
```

---

## 📖 3.3 零值（Zero Value）

Go 中声明变量但不初始化时，会自动赋予零值：

```go
var i int        // 0
var f float64    // 0.0
var s string     // ""（空字符串）
var b bool       // false
var p *int       // nil（空指针）
var slice []int  // nil
var m map[string]int  // nil
```

### 🔍 项目中利用零值

```go
// pkg/milvus/search.go

func (c *Client) Search(...) interface{} {
    // ...
    var results []SearchResult  // nil slice
    var recall float32          // 0
    isEmpty := true             // 显式初始化为 true

    // 后续根据条件填充
    if totalResults > 0 {
        results = make([]SearchResult, 0, totalResults)
    }
    // ...
}
```

---

## 📖 3.4 常量

```go
// 单个常量
const Pi = 3.14159
const MaxSize = 100

// 常量组
const (
    StatusOK      = 200
    StatusCreated = 201
    StatusError   = 500
)

// iota - 常量计数器
const (
    Sunday    = iota  // 0
    Monday           // 1
    Tuesday          // 2
    Wednesday        // 3
    // ...
)

// iota 应用
const (
    _  = iota             // 0，忽略
    KB = 1 << (10 * iota) // 1 << 10 = 1024
    MB                    // 1 << 20 = 1048576
    GB                    // 1 << 30
    TB                    // 1 << 40
)
```

---

## 📖 3.5 类型转换

Go 不支持隐式类型转换，必须显式转换：

```go
var i int = 42
var f float64 = float64(i)    // int → float64
var u uint = uint(f)          // float64 → uint

// 字符串转换
s := string(65)               // "A"（ASCII）
b := []byte("hello")          // 字符串 → 字节切片
s2 := string(b)               // 字节切片 → 字符串

// 数字与字符串转换需要 strconv 包
import "strconv"
s := strconv.Itoa(42)         // int → string: "42"
i, _ := strconv.Atoi("42")    // string → int: 42
```

### 🔍 项目中的类型转换

```go
// pkg/milvus/converters.go

// float64 转 float32（JavaScript 数字默认是 float64）
floats := make([]float32, len(v))
for i, val := range v {
    f, ok := val.(float64)     // 类型断言
    if !ok {
        return nil, wrapError("convertFloat64Slice", ErrInvalidDataType)
    }
    floats[i] = float32(f)     // 显式转换
}

// float64 转 int64（ID 字段）
ids := make([]int64, len(v))
for i, val := range v {
    f, ok := val.(float64)
    ids[i] = int64(f)          // 显式转换
}

// pkg/milvus/collection.go

// time.Duration 转 float64（毫秒）
ResponseTime: float64(time.Since(start).Milliseconds()),
```

---

## 📖 3.6 类型别名与自定义类型

```go
// 类型别名（完全等价）
type MyInt = int
var a MyInt = 10
var b int = a      // OK，同一类型

// 自定义类型（新类型）
type UserID int64
var id UserID = 100
var num int64 = int64(id)  // 需要转换

// 自定义类型可以添加方法
type Celsius float64

func (c Celsius) ToFahrenheit() float64 {
    return float64(c)*1.8 + 32
}
```

---

## ✏️ 动手练习

### 练习 1：类型实验

创建 `~/go-learning/types/main.go`：

```go
package main

import "fmt"

func main() {
    // 1. 测试零值
    var i int
    var s string
    var b bool
    var f float64
    fmt.Printf("int零值: %d\n", i)
    fmt.Printf("string零值: %q\n", s)  // %q 会显示引号
    fmt.Printf("bool零值: %t\n", b)
    fmt.Printf("float64零值: %f\n", f)

    // 2. 测试类型转换
    var x int = 100
    var y float64 = float64(x)
    var z int = int(y)
    fmt.Printf("\n转换: int(%d) → float64(%f) → int(%d)\n", x, y, z)

    // 3. 测试精度损失
    var big float64 = 1.9999999
    var small int = int(big)
    fmt.Printf("\n精度损失: float64(%f) → int(%d)\n", big, small)
}
```

运行：
```bash
cd ~/go-learning/types
go mod init types
go run main.go
```

### 练习 2：分析项目类型

打开 `pkg/milvus/types.go`，回答：

1. `OperationResult` 有哪些字段？
2. 为什么 `ResponseTime` 用 `float64` 而不是 `int64`？
3. `Result` 字段的类型是什么？为什么？

### 练习 3：常量练习

```go
package main

import "fmt"

const (
    // 定义文件大小常量
    _  = iota
    KB = 1 << (10 * iota)
    MB
    GB
)

func main() {
    fileSize := 5 * MB
    fmt.Printf("文件大小: %d bytes\n", fileSize)
    fmt.Printf("即: %.2f MB\n", float64(fileSize)/float64(MB))
}
```

---

## ❓ 自测问题

1. `var x int` 的初始值是什么？
   <details>
   <summary>查看答案</summary>
   0（零值）
   </details>

2. `:=` 和 `var` 的区别是什么？
   <details>
   <summary>查看答案</summary>
   := 是短声明，只能在函数内使用，自动推断类型；var 可以在任何地方使用，可以不初始化
   </details>

3. 下面的代码有什么问题？
   ```go
   var i int = 10
   var f float64 = i
   ```
   <details>
   <summary>查看答案</summary>
   Go 不支持隐式类型转换，需要显式转换：var f float64 = float64(i)
   </details>

4. 为什么项目中 JavaScript 传入的数字要转换为 `float32`？
   <details>
   <summary>查看答案</summary>
   因为 JSON 中的数字默认解析为 float64，而 Milvus 向量使用 float32，需要转换以匹配 SDK 要求
   </details>

---

## 💡 本章要点

1. **基本类型**：int、float64、string、bool 是最常用的
2. **三种声明方式**：
   - `var name Type = value`
   - `var name = value`（类型推断）
   - `name := value`（短声明，仅函数内）
3. **零值**：未初始化的变量有默认值
4. **显式转换**：Go 不支持隐式类型转换
5. **iota**：用于定义递增常量

---

## 下一步

在下一章，我们将学习 Go 的复合类型：

- 数组与切片
- 映射（Map）
- 结构体

[继续第4章：复合类型 →](./04-composite-types.md)

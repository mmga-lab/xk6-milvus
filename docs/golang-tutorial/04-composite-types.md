# 第4章：复合类型

## 学习目标

完成本章后，你将能够：

- ✅ 理解数组与切片的区别
- ✅ 熟练使用切片操作
- ✅ 掌握 Map 的使用
- ✅ 理解结构体的定义和使用
- ✅ 认识项目中的复合类型应用

---

## 📖 4.1 数组（Array）

数组是固定长度的同类型元素序列。

### 数组声明

```go
// 声明固定长度数组
var arr1 [5]int                    // 5个int，初始值都是0
var arr2 [3]string = [3]string{"a", "b", "c"}
arr3 := [4]int{1, 2, 3, 4}

// 让编译器计算长度
arr4 := [...]int{1, 2, 3, 4, 5}    // 长度为5

// 指定索引初始化
arr5 := [5]int{0: 100, 4: 400}     // [100, 0, 0, 0, 400]
```

### 数组特点

```go
arr := [3]int{1, 2, 3}

// 1. 长度是类型的一部分
var a [3]int
var b [4]int
// a = b  // 编译错误！[3]int 和 [4]int 是不同类型

// 2. 数组是值类型，赋值会复制
arr2 := arr     // 创建副本
arr2[0] = 100   // 不影响 arr
fmt.Println(arr[0])  // 仍然是 1

// 3. 访问元素
fmt.Println(arr[0])  // 第一个元素
fmt.Println(len(arr)) // 长度：3
```

💡 **实际开发中，数组较少使用，切片更常见。**

---

## 📖 4.2 切片（Slice）

切片是动态长度的序列，是 Go 中最常用的数据结构之一。

### 切片声明

```go
// 方式1：从数组创建
arr := [5]int{1, 2, 3, 4, 5}
slice1 := arr[1:4]     // [2, 3, 4]

// 方式2：直接声明
var slice2 []int       // nil 切片
slice3 := []int{}      // 空切片（非 nil）
slice4 := []int{1, 2, 3}

// 方式3：make 创建
slice5 := make([]int, 5)      // 长度5，容量5
slice6 := make([]int, 3, 10)  // 长度3，容量10
```

### 切片操作

```go
s := []int{1, 2, 3, 4, 5}

// 访问
fmt.Println(s[0])    // 1
fmt.Println(s[1:3])  // [2, 3]

// 长度和容量
fmt.Println(len(s))  // 5
fmt.Println(cap(s))  // 5

// 追加元素
s = append(s, 6)           // [1, 2, 3, 4, 5, 6]
s = append(s, 7, 8, 9)     // 追加多个

// 追加另一个切片
other := []int{10, 11}
s = append(s, other...)    // ... 展开切片

// 复制
dst := make([]int, len(s))
copy(dst, s)
```

### 🔍 项目中的切片

```go
// pkg/milvus/types.go

type Schema struct {
    Fields    []Field    `json:"fields"`              // Field 切片
    Functions []Function `json:"functions,omitempty"` // Function 切片
}

// pkg/milvus/search.go

func (c *Client) Search(...) interface{} {
    // 将 [][]float32 转换为 []entity.Vector
    searchVectors := make([]entity.Vector, len(vectors))  // 预分配
    for i, v := range vectors {
        searchVectors[i] = entity.FloatVector(v)
    }

    // 动态追加结果
    var results []SearchResult
    if totalResults > 0 {
        results = make([]SearchResult, 0, totalResults)  // 预分配容量
    }

    for _, resultSet := range resultSets {
        // ...
        results = append(results, result)  // 追加
    }
}

// pkg/milvus/converters.go

// 二维切片：向量数据
func (c *Client) convertNestedVectors(fieldName string, v []interface{}) (column.Column, error) {
    vectors := make([][]float32, len(v))  // 二维切片
    for i, vecInterface := range v {
        vec, ok := vecInterface.([]interface{})
        // ...
        floatVec := make([]float32, len(vec))
        // ...
        vectors[i] = floatVec
    }
    return column.NewColumnFloatVector(fieldName, dim, vectors), nil
}
```

### 切片内部结构

```go
// 切片的底层结构（理解即可）
type slice struct {
    array unsafe.Pointer  // 指向底层数组
    len   int             // 当前长度
    cap   int             // 容量
}
```

```
切片 s := []int{1, 2, 3, 4, 5}

s (slice header)
┌─────────────┐
│ ptr ────────┼───→ [1][2][3][4][5]  ← 底层数组
│ len: 5      │
│ cap: 5      │
└─────────────┘
```

---

## 📖 4.3 映射（Map）

Map 是键值对集合，类似其他语言的字典或哈希表。

### Map 声明

```go
// 方式1：make 创建
m1 := make(map[string]int)
m1["age"] = 25

// 方式2：字面量
m2 := map[string]int{
    "alice": 25,
    "bob":   30,
}

// 方式3：声明（nil map，不能写入）
var m3 map[string]int  // nil
// m3["key"] = 1  // panic！
```

### Map 操作

```go
m := map[string]int{"a": 1, "b": 2}

// 读取
val := m["a"]     // 1
val = m["x"]      // 0（不存在返回零值）

// 检查是否存在
val, ok := m["a"]
if ok {
    fmt.Println("存在:", val)
}

// 简洁写法
if val, ok := m["a"]; ok {
    fmt.Println("存在:", val)
}

// 写入
m["c"] = 3

// 删除
delete(m, "a")

// 遍历（顺序不确定）
for key, value := range m {
    fmt.Println(key, value)
}

// 长度
fmt.Println(len(m))
```

### 🔍 项目中的 Map

```go
// pkg/milvus/types.go

type Field struct {
    AnalyzerParams map[string]interface{} `json:"analyzerParams,omitempty"`
}

type HybridSearchRequest struct {
    Params map[string]interface{} `json:"params,omitempty"`
}

// pkg/milvus/data.go

// 接收 JavaScript 对象作为 map
func (c *Client) Insert(data map[string]interface{}, ...) interface{} {
    // data 的结构类似：
    // {
    //     "id": [1, 2, 3],
    //     "title": ["a", "b", "c"],
    //     "vector": [[0.1, 0.2], [0.3, 0.4], [0.5, 0.6]]
    // }

    for fieldName, fieldData := range data {
        // 处理每个字段
    }
}

// pkg/milvus/helpers.go

// 返回 map 给 JavaScript
func toMap(result *OperationResult) map[string]interface{} {
    // ...
    var m map[string]interface{}
    json.Unmarshal(data, &m)
    return m
}
```

### interface{} 作为值类型

```go
// map[string]interface{} 可以存储任意类型的值
m := map[string]interface{}{
    "name":   "Alice",           // string
    "age":    25,                // int
    "active": true,              // bool
    "scores": []int{90, 85, 88}, // []int
}

// 读取时需要类型断言
name := m["name"].(string)
age := m["age"].(int)
```

---

## 📖 4.4 结构体（Struct）

结构体是自定义的复合类型，可以包含多个不同类型的字段。

### 结构体定义

```go
// 定义结构体类型
type Person struct {
    Name    string
    Age     int
    Email   string
    Active  bool
}

// 创建实例
p1 := Person{
    Name:   "Alice",
    Age:    25,
    Email:  "alice@example.com",
    Active: true,
}

// 部分初始化（其他字段为零值）
p2 := Person{Name: "Bob"}

// 按顺序初始化（不推荐）
p3 := Person{"Charlie", 30, "charlie@example.com", true}

// 访问字段
fmt.Println(p1.Name)
p1.Age = 26
```

### 结构体指针

```go
// 创建指针
p := &Person{Name: "Alice", Age: 25}

// 访问字段（自动解引用）
fmt.Println(p.Name)   // 等价于 (*p).Name
p.Age = 26

// new 创建（字段为零值）
p2 := new(Person)     // *Person
p2.Name = "Bob"
```

### 🔍 项目中的结构体

```go
// pkg/milvus/types.go

// 统一返回结构
type OperationResult struct {
    Success      bool        `json:"success"`           // JSON 标签
    ResponseTime float64     `json:"response_time_ms"`
    Result       interface{} `json:"result,omitempty"`  // omitempty: 空值不输出
    Error        string      `json:"error,omitempty"`
    Empty        bool        `json:"empty"`
    Recall       float32     `json:"recall"`
}

// 客户端结构
type Client struct {
    client            *milvusclient.Client  // 指针字段
    ctx               context.Context       // 接口字段
    vu                modules.VU            // 接口字段
    config            *ClientConfig         // 指针字段
    defaultCollection string
}

// 字段定义
type Field struct {
    Name           string                 `json:"name"`
    DataType       string                 `json:"dataType"`
    IsPrimaryKey   bool                   `json:"isPrimaryKey,omitempty"`
    IsAutoID       bool                   `json:"isAutoID,omitempty"`
    Dimension      int64                  `json:"dimension,omitempty"`
    MaxLength      int64                  `json:"maxLength,omitempty"`
    EnableAnalyzer bool                   `json:"enableAnalyzer,omitempty"`
    EnableMatch    bool                   `json:"enableMatch,omitempty"`
    AnalyzerParams map[string]interface{} `json:"analyzerParams,omitempty"`
}
```

### JSON 标签详解

```go
type Example struct {
    // 基本映射
    Name string `json:"name"`          // JSON 中用 "name"

    // 忽略空值
    Email string `json:"email,omitempty"`  // 空字符串时不输出

    // 忽略字段
    Password string `json:"-"`          // 不序列化

    // 字符串形式
    ID int64 `json:"id,string"`        // 输出为字符串 "123"
}
```

### 嵌套结构体

```go
type Address struct {
    City    string
    Country string
}

type Person struct {
    Name    string
    Address Address  // 嵌套
}

// 使用
p := Person{
    Name: "Alice",
    Address: Address{
        City:    "Beijing",
        Country: "China",
    },
}
fmt.Println(p.Address.City)

// 匿名嵌套（继承效果）
type Employee struct {
    Person        // 匿名嵌套
    Department string
}

e := Employee{
    Person: Person{Name: "Bob"},
    Department: "Engineering",
}
fmt.Println(e.Name)  // 直接访问 Person 的字段
```

---

## 📖 4.5 类型组合示例

让我们看一个综合的例子：

```go
// 定义类型
type Vector []float32

type SearchRequest struct {
    Vectors     []Vector               // 切片的切片
    TopK        int
    Params      map[string]interface{} // Map
    OutputFields []string              // 字符串切片
}

type SearchResult struct {
    ID     int64
    Score  float32
    Fields map[string]interface{}
}

type SearchResponse struct {
    Success bool
    Results []SearchResult  // 结构体切片
    Error   string
}
```

---

## ✏️ 动手练习

### 练习 1：切片操作

```go
package main

import "fmt"

func main() {
    // 1. 创建切片
    numbers := []int{1, 2, 3, 4, 5}

    // 2. 切片操作
    fmt.Println("原始:", numbers)
    fmt.Println("前三个:", numbers[:3])
    fmt.Println("后三个:", numbers[2:])
    fmt.Println("中间:", numbers[1:4])

    // 3. 追加
    numbers = append(numbers, 6, 7, 8)
    fmt.Println("追加后:", numbers)

    // 4. 复制
    copied := make([]int, len(numbers))
    copy(copied, numbers)
    copied[0] = 100
    fmt.Println("原始:", numbers)
    fmt.Println("复制:", copied)

    // 5. 二维切片
    matrix := [][]int{
        {1, 2, 3},
        {4, 5, 6},
        {7, 8, 9},
    }
    fmt.Println("矩阵:", matrix)
    fmt.Println("第二行:", matrix[1])
    fmt.Println("元素[1][2]:", matrix[1][2])
}
```

### 练习 2：Map 操作

```go
package main

import "fmt"

func main() {
    // 1. 创建学生成绩表
    scores := map[string]int{
        "Alice": 95,
        "Bob":   87,
        "Carol": 92,
    }

    // 2. 添加和修改
    scores["David"] = 88
    scores["Alice"] = 98

    // 3. 检查是否存在
    if score, ok := scores["Eve"]; ok {
        fmt.Println("Eve的分数:", score)
    } else {
        fmt.Println("Eve不在列表中")
    }

    // 4. 删除
    delete(scores, "Bob")

    // 5. 遍历
    fmt.Println("\n所有成绩:")
    for name, score := range scores {
        fmt.Printf("%s: %d\n", name, score)
    }
}
```

### 练习 3：结构体实践

```go
package main

import (
    "encoding/json"
    "fmt"
)

// 定义结构体
type Field struct {
    Name     string `json:"name"`
    DataType string `json:"dataType"`
    Dimension int   `json:"dimension,omitempty"`
}

type Schema struct {
    Name   string  `json:"name"`
    Fields []Field `json:"fields"`
}

func main() {
    // 创建 Schema
    schema := Schema{
        Name: "products",
        Fields: []Field{
            {Name: "id", DataType: "Int64"},
            {Name: "title", DataType: "VarChar"},
            {Name: "embedding", DataType: "FloatVector", Dimension: 128},
        },
    }

    // 序列化为 JSON
    jsonBytes, _ := json.MarshalIndent(schema, "", "  ")
    fmt.Println("JSON 输出:")
    fmt.Println(string(jsonBytes))

    // 从 JSON 反序列化
    jsonStr := `{"name":"test","fields":[{"name":"id","dataType":"Int64"}]}`
    var parsed Schema
    json.Unmarshal([]byte(jsonStr), &parsed)
    fmt.Println("\n解析结果:", parsed)
}
```

### 练习 4：分析项目代码

打开 `pkg/milvus/converters.go`，找出：

1. 哪些函数处理切片？
2. `convertNestedVectors` 是如何处理二维切片的？
3. Map 是如何用于稀疏向量转换的？

---

## ❓ 自测问题

1. 数组和切片的主要区别是什么？
   <details>
   <summary>查看答案</summary>
   数组长度固定，是值类型；切片长度可变，是引用类型
   </details>

2. `make([]int, 3, 10)` 创建的切片，len 和 cap 分别是多少？
   <details>
   <summary>查看答案</summary>
   len = 3, cap = 10
   </details>

3. 如何检查 map 中是否存在某个 key？
   <details>
   <summary>查看答案</summary>
   使用双返回值：value, ok := m[key]，ok 为 true 表示存在
   </details>

4. 结构体标签 `json:"name,omitempty"` 中 omitempty 的作用是什么？
   <details>
   <summary>查看答案</summary>
   当字段值为零值（空字符串、0、false、nil等）时，JSON 序列化时不输出该字段
   </details>

5. `map[string]interface{}` 有什么特点？
   <details>
   <summary>查看答案</summary>
   键是字符串，值可以是任意类型。常用于处理动态 JSON 数据
   </details>

---

## 💡 本章要点

1. **切片比数组常用** - 动态长度，更灵活
2. **切片操作**：
   - `make([]T, len, cap)` 创建
   - `append()` 追加
   - `copy()` 复制
3. **Map 是引用类型** - nil map 不能写入
4. **结构体是值类型** - 用指针传递更高效
5. **JSON 标签** - 控制序列化行为

---

## 下一步

在下一章，我们将学习函数与方法：

- 函数定义和调用
- 多返回值
- 方法和接收者
- 值接收者 vs 指针接收者

[继续第5章：函数与方法 →](./05-functions-and-methods.md)

# 第5章：函数与方法

## 学习目标

完成本章后，你将能够：

- ✅ 定义和调用函数
- ✅ 理解多返回值
- ✅ 掌握方法和接收者
- ✅ 区分值接收者和指针接收者
- ✅ 理解闭包和匿名函数

---

## 📖 5.1 函数基础

### 函数定义

```go
// 基本语法
func functionName(parameter1 type1, parameter2 type2) returnType {
    // 函数体
    return value
}

// 示例
func add(a int, b int) int {
    return a + b
}

// 相同类型参数可以简写
func add(a, b int) int {
    return a + b
}

// 无返回值
func greet(name string) {
    fmt.Println("Hello,", name)
}

// 无参数
func sayHello() {
    fmt.Println("Hello!")
}
```

### 多返回值

```go
// 返回多个值
func divide(a, b int) (int, int) {
    quotient := a / b
    remainder := a % b
    return quotient, remainder
}

// 调用
q, r := divide(10, 3)
fmt.Println(q, r)  // 3 1

// 忽略某个返回值
q, _ := divide(10, 3)

// 命名返回值
func divide(a, b int) (quotient, remainder int) {
    quotient = a / b
    remainder = a % b
    return  // 裸返回，返回命名变量的值
}
```

### 🔍 项目中的多返回值

```go
// pkg/milvus/client.go

// 返回 (*Client, error)
func (m *Milvus) Client(address string, token ...string) (*Client, error) {
    return m.createClient(address, "", token...)
}

// 内部创建函数
func (m *Milvus) createClient(address, collectionName string, token ...string) (*Client, error) {
    ctx := m.vu.Context()

    // ... 创建配置 ...

    c, err := milvusclient.New(ctx, milvusConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create milvus client: %v", err)
    }

    return &Client{
        client:            c,
        ctx:               ctx,
        vu:                m.vu,
        config:            clientConfig,
        defaultCollection: collectionName,
    }, nil
}
```

### 可变参数

```go
// ... 表示可变参数
func sum(numbers ...int) int {
    total := 0
    for _, n := range numbers {
        total += n
    }
    return total
}

// 调用
sum(1, 2, 3)       // 6
sum(1, 2, 3, 4, 5) // 15
sum()              // 0

// 展开切片
nums := []int{1, 2, 3}
sum(nums...)       // 6
```

### 🔍 项目中的可变参数

```go
// pkg/milvus/client.go

// token 是可选参数
func (m *Milvus) Client(address string, token ...string) (*Client, error) {
    return m.createClient(address, "", token...)
}

// 调用方式
client, _ := milvus.Client("localhost:19530")                    // 无 token
client, _ := milvus.Client("localhost:19530", "user:password")  // 有 token

// pkg/milvus/collection.go

// collectionName 是可选参数
func (c *Client) DropCollection(collectionName ...string) interface{} {
    name := c.defaultCollection
    if len(collectionName) > 0 && collectionName[0] != "" {
        name = collectionName[0]
    }
    // ...
}
```

---

## 📖 5.2 方法

方法是绑定到特定类型的函数。

### 方法定义

```go
type Rectangle struct {
    Width  float64
    Height float64
}

// 定义方法：func (接收者) 方法名(参数) 返回值
func (r Rectangle) Area() float64 {
    return r.Width * r.Height
}

func (r Rectangle) Perimeter() float64 {
    return 2 * (r.Width + r.Height)
}

// 调用
rect := Rectangle{Width: 10, Height: 5}
area := rect.Area()         // 50
perimeter := rect.Perimeter() // 30
```

### 值接收者 vs 指针接收者

```go
type Counter struct {
    Value int
}

// 值接收者：操作的是副本
func (c Counter) IncrementValue() {
    c.Value++  // 修改的是副本，不影响原始值
}

// 指针接收者：操作的是原始值
func (c *Counter) IncrementPointer() {
    c.Value++  // 修改原始值
}

// 使用
counter := Counter{Value: 0}
counter.IncrementValue()
fmt.Println(counter.Value)  // 0（未改变）

counter.IncrementPointer()
fmt.Println(counter.Value)  // 1（已改变）
```

### 何时使用指针接收者？

1. **需要修改接收者** - 修改结构体字段
2. **接收者很大** - 避免复制开销
3. **保持一致性** - 如果有一个方法用指针，通常全部用指针

### 🔍 项目中的方法

```go
// pkg/milvus/collection.go

// 所有 Client 方法都使用指针接收者
func (c *Client) CreateCollection(schemaInput interface{}) interface{} {
    start := time.Now()
    // ...
}

func (c *Client) DropCollection(collectionName ...string) interface{} {
    start := time.Now()
    // ...
}

func (c *Client) LoadCollection(collectionName ...string) interface{} {
    start := time.Now()
    // ...
}

// pkg/milvus/client.go

func (c *Client) Close() error {
    return c.client.Close(c.ctx)  // 使用 c.client 字段
}

// pkg/milvus/helpers.go

// 辅助方法
func (c *Client) getCollectionName(collectionName ...string) string {
    if len(collectionName) > 0 && collectionName[0] != "" {
        return collectionName[0]
    }
    return c.defaultCollection  // 访问 c.defaultCollection
}
```

---

## 📖 5.3 函数类型

函数在 Go 中是一等公民，可以赋值给变量。

### 函数作为类型

```go
// 定义函数类型
type Operation func(int, int) int

// 使用函数类型
func add(a, b int) int { return a + b }
func multiply(a, b int) int { return a * b }

var op Operation
op = add
fmt.Println(op(2, 3))  // 5

op = multiply
fmt.Println(op(2, 3))  // 6
```

### 函数作为参数

```go
func apply(op func(int, int) int, a, b int) int {
    return op(a, b)
}

result := apply(add, 2, 3)      // 5
result = apply(multiply, 2, 3)  // 6
```

### 🔍 项目中的函数类型

```go
// pkg/milvus/config.go

// ClientOption 是函数类型
type ClientOption func(*ClientConfig)

// 返回 ClientOption 的函数
func WithAddress(address string) ClientOption {
    return func(c *ClientConfig) {
        c.Address = address
    }
}

func WithTimeout(timeout time.Duration) ClientOption {
    return func(c *ClientConfig) {
        c.Timeout = timeout
    }
}

// 应用选项
func (c *ClientConfig) ApplyOptions(opts ...ClientOption) {
    for _, opt := range opts {
        opt(c)  // 调用每个选项函数
    }
}

// 使用
config := DefaultClientConfig()
config.ApplyOptions(
    WithAddress("localhost:19530"),
    WithTimeout(60 * time.Second),
)
```

这是 Go 中常见的 **函数选项模式（Functional Options Pattern）**。

---

## 📖 5.4 匿名函数与闭包

### 匿名函数

```go
// 直接定义并调用
result := func(a, b int) int {
    return a + b
}(2, 3)
fmt.Println(result)  // 5

// 赋值给变量
add := func(a, b int) int {
    return a + b
}
fmt.Println(add(2, 3))  // 5
```

### 闭包

闭包可以访问外部作用域的变量：

```go
func counter() func() int {
    count := 0  // 被闭包捕获的变量
    return func() int {
        count++
        return count
    }
}

c := counter()
fmt.Println(c())  // 1
fmt.Println(c())  // 2
fmt.Println(c())  // 3

c2 := counter()  // 新的计数器
fmt.Println(c2()) // 1
```

### 🔍 项目中的闭包

```go
// pkg/milvus/config.go

// WithAddress 返回一个闭包
func WithAddress(address string) ClientOption {
    // 返回的函数"捕获"了 address 变量
    return func(c *ClientConfig) {
        c.Address = address
    }
}

// 每次调用 WithAddress 会创建一个新的闭包
opt1 := WithAddress("localhost:19530")  // 捕获 "localhost:19530"
opt2 := WithAddress("localhost:19531")  // 捕获 "localhost:19531"
```

---

## 📖 5.5 defer 语句

defer 延迟执行，常用于清理资源。

### 基本用法

```go
func main() {
    defer fmt.Println("3")
    defer fmt.Println("2")
    fmt.Println("1")
}
// 输出：
// 1
// 2
// 3
// defer 按 LIFO（后进先出）顺序执行
```

### 典型应用

```go
// 文件操作
func readFile(filename string) error {
    f, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer f.Close()  // 确保文件关闭

    // 读取文件...
    return nil
}

// 解锁
func (s *SafeMap) Get(key string) string {
    s.mu.Lock()
    defer s.mu.Unlock()  // 确保解锁
    return s.data[key]
}
```

### 🔍 项目中的 defer

```go
// 虽然项目中没有直接使用 defer，但在使用 Client 时应该：

func testMilvus() {
    client, err := milvus.Client("localhost:19530")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()  // 确保关闭连接

    // 执行操作...
}
```

---

## ✏️ 动手练习

### 练习 1：函数基础

```go
package main

import "fmt"

// 1. 基本函数
func greet(name string) string {
    return "Hello, " + name + "!"
}

// 2. 多返回值
func minMax(numbers []int) (min, max int) {
    if len(numbers) == 0 {
        return 0, 0
    }
    min, max = numbers[0], numbers[0]
    for _, n := range numbers {
        if n < min {
            min = n
        }
        if n > max {
            max = n
        }
    }
    return
}

// 3. 可变参数
func joinStrings(sep string, strs ...string) string {
    result := ""
    for i, s := range strs {
        if i > 0 {
            result += sep
        }
        result += s
    }
    return result
}

func main() {
    // 测试
    fmt.Println(greet("Gopher"))

    min, max := minMax([]int{3, 1, 4, 1, 5, 9, 2, 6})
    fmt.Printf("Min: %d, Max: %d\n", min, max)

    fmt.Println(joinStrings(", ", "a", "b", "c"))
}
```

### 练习 2：方法实践

```go
package main

import "fmt"

// 定义结构体
type Vector struct {
    X, Y float64
}

// 值接收者：不修改
func (v Vector) Length() float64 {
    return v.X*v.X + v.Y*v.Y
}

// 指针接收者：修改
func (v *Vector) Scale(factor float64) {
    v.X *= factor
    v.Y *= factor
}

// 指针接收者：添加
func (v *Vector) Add(other Vector) {
    v.X += other.X
    v.Y += other.Y
}

func main() {
    v := Vector{3, 4}
    fmt.Printf("初始: %+v\n", v)
    fmt.Printf("长度平方: %.2f\n", v.Length())

    v.Scale(2)
    fmt.Printf("缩放后: %+v\n", v)

    v.Add(Vector{1, 1})
    fmt.Printf("添加后: %+v\n", v)
}
```

### 练习 3：函数选项模式

```go
package main

import "fmt"

// 配置结构
type ServerConfig struct {
    Host    string
    Port    int
    Timeout int
    Debug   bool
}

// 选项类型
type ServerOption func(*ServerConfig)

// 默认配置
func DefaultServerConfig() *ServerConfig {
    return &ServerConfig{
        Host:    "localhost",
        Port:    8080,
        Timeout: 30,
        Debug:   false,
    }
}

// 选项函数
func WithHost(host string) ServerOption {
    return func(c *ServerConfig) {
        c.Host = host
    }
}

func WithPort(port int) ServerOption {
    return func(c *ServerConfig) {
        c.Port = port
    }
}

func WithDebug(debug bool) ServerOption {
    return func(c *ServerConfig) {
        c.Debug = debug
    }
}

// 应用选项
func NewServerConfig(opts ...ServerOption) *ServerConfig {
    config := DefaultServerConfig()
    for _, opt := range opts {
        opt(config)
    }
    return config
}

func main() {
    // 默认配置
    c1 := NewServerConfig()
    fmt.Printf("默认配置: %+v\n", c1)

    // 自定义配置
    c2 := NewServerConfig(
        WithHost("0.0.0.0"),
        WithPort(9000),
        WithDebug(true),
    )
    fmt.Printf("自定义配置: %+v\n", c2)
}
```

### 练习 4：分析项目方法

查看 `pkg/milvus/collection.go`，回答：

1. `CreateCollection` 方法的接收者是什么类型？
2. 为什么所有方法都返回 `interface{}`？
3. `start := time.Now()` 在每个方法开始处的作用是什么？

---

## ❓ 自测问题

1. 值接收者和指针接收者的主要区别是什么？
   <details>
   <summary>查看答案</summary>
   值接收者操作的是结构体的副本，不能修改原始值；指针接收者操作原始值，可以修改
   </details>

2. 可变参数在函数内部是什么类型？
   <details>
   <summary>查看答案</summary>
   切片类型。例如 numbers ...int 在函数内部是 []int
   </details>

3. defer 语句的执行顺序是什么？
   <details>
   <summary>查看答案</summary>
   LIFO（后进先出），最后声明的 defer 最先执行
   </details>

4. 什么是闭包？
   <details>
   <summary>查看答案</summary>
   闭包是一个函数值，它引用了其函数体之外的变量。闭包可以访问并修改这些变量。
   </details>

5. 函数选项模式的优点是什么？
   <details>
   <summary>查看答案</summary>
   提供灵活的配置方式，可选参数清晰，易于扩展，API 向后兼容
   </details>

---

## 💡 本章要点

1. **函数定义**：`func name(params) returnType`
2. **多返回值**：Go 支持返回多个值，常用于返回 `(result, error)`
3. **方法**：绑定到类型的函数，通过接收者定义
4. **接收者选择**：
   - 值接收者：不修改、小结构体
   - 指针接收者：需要修改、大结构体、保持一致性
5. **函数是一等公民**：可以赋值、传递、返回
6. **defer**：延迟执行，用于清理资源

---

## 下一步

在下一章，我们将深入学习接口：

- 接口定义和实现
- 隐式实现
- 空接口 interface{}
- 类型断言和类型开关

[继续第6章：接口 →](./06-interfaces.md)

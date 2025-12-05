# 第7章：错误处理

## 学习目标

完成本章后，你将能够：

- ✅ 理解 Go 的错误处理哲学
- ✅ 掌握 error 接口
- ✅ 创建自定义错误类型
- ✅ 使用错误包装（Error Wrapping）
- ✅ 理解项目中的错误处理模式

---

## 📖 7.1 Go 错误处理哲学

Go 使用显式的错误返回值，而不是异常机制：

```go
// 其他语言（伪代码）
try {
    file = openFile("data.txt")
    data = file.read()
} catch (FileNotFoundError e) {
    // 处理错误
}

// Go 方式
file, err := os.Open("data.txt")
if err != nil {
    // 处理错误
    return err
}
data, err := io.ReadAll(file)
if err != nil {
    return err
}
```

### 为什么这样设计？

1. **显式** - 错误不会被意外忽略
2. **简单** - 控制流清晰
3. **高效** - 没有异常处理的开销

---

## 📖 7.2 error 接口

`error` 是 Go 内置的接口：

```go
type error interface {
    Error() string
}
```

任何实现了 `Error()` 方法的类型都是 error。

### 创建简单错误

```go
import "errors"

// 方式1：errors.New
err := errors.New("something went wrong")

// 方式2：fmt.Errorf
err := fmt.Errorf("file %s not found", filename)

// 检查错误
if err != nil {
    fmt.Println("Error:", err.Error())
    // 或者直接
    fmt.Println("Error:", err)  // 自动调用 Error()
}
```

---

## 📖 7.3 函数返回错误

### 标准模式

```go
func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

// 调用
result, err := divide(10, 0)
if err != nil {
    fmt.Println("Error:", err)
    return
}
fmt.Println("Result:", result)
```

### 🔍 项目中的错误返回

```go
// pkg/milvus/client.go

func (m *Milvus) createClient(address, collectionName string, token ...string) (*Client, error) {
    ctx := m.vu.Context()

    // ... 配置创建 ...

    c, err := milvusclient.New(ctx, milvusConfig)
    if err != nil {
        // 包装错误并返回
        return nil, fmt.Errorf("failed to create milvus client: %v", err)
    }

    return &Client{
        client:            c,
        ctx:               ctx,
        // ...
    }, nil  // 成功时错误为 nil
}
```

---

## 📖 7.4 哨兵错误（Sentinel Errors）

预定义的错误值，用于比较：

```go
// 定义哨兵错误
var ErrNotFound = errors.New("not found")
var ErrPermissionDenied = errors.New("permission denied")

func findUser(id int) (*User, error) {
    // ...
    return nil, ErrNotFound
}

// 检查特定错误
user, err := findUser(123)
if err == ErrNotFound {
    fmt.Println("用户不存在")
} else if err != nil {
    fmt.Println("其他错误:", err)
}
```

### 🔍 项目中的哨兵错误

```go
// pkg/milvus/errors.go

var (
    ErrCollectionNameRequired = errors.New("collection name required")
    ErrEmptyData              = errors.New("no valid columns provided")
    ErrEmptyVectorArray       = errors.New("empty vector array")
    ErrNoSearchRequests       = errors.New("at least one search request required")
    ErrInvalidDataType        = errors.New("invalid data type")
    ErrUnsupportedType        = errors.New("unsupported type")
    ErrSchemaParseError       = errors.New("failed to parse schema")
)

// 使用
// pkg/milvus/collection.go
if name == "" {
    return toMap(&OperationResult{
        Success:      false,
        ResponseTime: float64(time.Since(start).Milliseconds()),
        Error:        ErrCollectionNameRequired.Error(),
    })
}
```

---

## 📖 7.5 自定义错误类型

当需要携带更多错误信息时：

```go
// 自定义错误类型
type ValidationError struct {
    Field   string
    Message string
}

// 实现 error 接口
func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

// 使用
func validateUser(u *User) error {
    if u.Name == "" {
        return &ValidationError{Field: "name", Message: "cannot be empty"}
    }
    if u.Age < 0 {
        return &ValidationError{Field: "age", Message: "must be non-negative"}
    }
    return nil
}
```

### 🔍 项目中的自定义错误

```go
// pkg/milvus/errors.go

// MilvusError 包含额外的上下文信息
type MilvusError struct {
    Op      string  // 失败的操作
    Err     error   // 底层错误
    Context string  // 额外上下文
}

// 实现 error 接口
func (e *MilvusError) Error() string {
    if e.Context != "" {
        return fmt.Sprintf("%s: %s: %v", e.Op, e.Context, e.Err)
    }
    return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// 实现 Unwrap 方法（支持 errors.Is 和 errors.As）
func (e *MilvusError) Unwrap() error {
    return e.Err
}

// 创建新错误
func newError(op string, err error, context string) error {
    return &MilvusError{
        Op:      op,
        Err:     err,
        Context: context,
    }
}

// 包装现有错误
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

## 📖 7.6 错误包装（Go 1.13+）

### fmt.Errorf 和 %w

```go
// 包装错误（保留原始错误链）
originalErr := errors.New("connection refused")
wrappedErr := fmt.Errorf("failed to connect to database: %w", originalErr)

// 输出：failed to connect to database: connection refused
fmt.Println(wrappedErr)
```

### errors.Is - 检查错误链

```go
var ErrNotFound = errors.New("not found")

func getUser(id int) error {
    return fmt.Errorf("getUser failed: %w", ErrNotFound)
}

err := getUser(123)

// 检查错误链中是否包含 ErrNotFound
if errors.Is(err, ErrNotFound) {
    fmt.Println("用户不存在")
}
```

### errors.As - 提取特定错误类型

```go
type QueryError struct {
    Query   string
    Message string
}

func (e *QueryError) Error() string {
    return fmt.Sprintf("query error: %s - %s", e.Query, e.Message)
}

// 检查并提取
var queryErr *QueryError
if errors.As(err, &queryErr) {
    fmt.Printf("失败的查询: %s\n", queryErr.Query)
}
```

### 🔍 项目中的 Unwrap

```go
// pkg/milvus/errors.go

// MilvusError 实现 Unwrap 方法
func (e *MilvusError) Unwrap() error {
    return e.Err
}

// 这使得可以使用 errors.Is 和 errors.As 检查底层错误
// 例如：
// err := wrapError("Insert", ErrEmptyData)
// errors.Is(err, ErrEmptyData)  // true
```

---

## 📖 7.7 项目的错误处理模式

xk6-milvus 使用统一的 `OperationResult` 结构返回结果，而不是返回 error：

```go
// pkg/milvus/types.go

type OperationResult struct {
    Success      bool        `json:"success"`           // 操作是否成功
    ResponseTime float64     `json:"response_time_ms"`  // 响应时间
    Result       interface{} `json:"result,omitempty"`  // 成功时的结果
    Error        string      `json:"error,omitempty"`   // 失败时的错误信息
    Empty        bool        `json:"empty"`             // 结果是否为空
    Recall       float32     `json:"recall"`            // 召回率
}
```

### 为什么这样设计？

1. **JavaScript 兼容性** - JavaScript 不支持 Go 的多返回值错误模式
2. **统一指标** - 每个操作都有响应时间
3. **无异常** - 错误不会中断 k6 测试执行

### 错误处理示例

```go
// pkg/milvus/collection.go

func (c *Client) LoadCollection(collectionName ...string) interface{} {
    start := time.Now()

    name := c.defaultCollection
    if len(collectionName) > 0 && collectionName[0] != "" {
        name = collectionName[0]
    }

    // 参数验证错误
    if name == "" {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        ErrCollectionNameRequired.Error(),
        })
    }

    option := milvusclient.NewLoadCollectionOption(name)
    task, err := c.client.LoadCollection(c.ctx, option)

    // SDK 调用错误
    if err != nil {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        fmt.Sprintf("failed to load collection: %v", err),
        })
    }

    // 等待任务完成
    err = task.Await(c.ctx)
    if err != nil {
        return toMap(&OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        fmt.Sprintf("failed to wait for collection load: %v", err),
        })
    }

    // 成功
    return toMap(&OperationResult{
        Success:      true,
        ResponseTime: float64(time.Since(start).Milliseconds()),
        Result:       map[string]interface{}{"collection": name},
    })
}
```

### JavaScript 中的错误检查

```javascript
import milvus from "k6/x/milvus";
import { check } from "k6";

export default function() {
    const client = milvus.client("localhost:19530");

    const result = client.loadCollection("my_collection");

    // 检查结果
    check(result, {
        "operation successful": (r) => r.success === true,
        "no error": (r) => !r.error,
    });

    if (!result.success) {
        console.error(`Error: ${result.error}`);
    }
}
```

---

## ✏️ 动手练习

### 练习 1：基本错误处理

```go
package main

import (
    "errors"
    "fmt"
)

// 定义哨兵错误
var (
    ErrDivideByZero = errors.New("division by zero")
    ErrNegativeNumber = errors.New("negative number not allowed")
)

// 安全除法
func safeDivide(a, b int) (int, error) {
    if b == 0 {
        return 0, ErrDivideByZero
    }
    return a / b, nil
}

// 平方根（不允许负数）
func safeSqrt(n float64) (float64, error) {
    if n < 0 {
        return 0, ErrNegativeNumber
    }
    // 简单实现（实际应使用 math.Sqrt）
    return n * 0.5, nil
}

func main() {
    // 测试除法
    result, err := safeDivide(10, 2)
    if err != nil {
        fmt.Println("Error:", err)
    } else {
        fmt.Println("10 / 2 =", result)
    }

    result, err = safeDivide(10, 0)
    if err == ErrDivideByZero {
        fmt.Println("不能除以零！")
    }

    // 测试平方根
    _, err = safeSqrt(-4)
    if errors.Is(err, ErrNegativeNumber) {
        fmt.Println("不能计算负数的平方根！")
    }
}
```

### 练习 2：自定义错误类型

```go
package main

import (
    "fmt"
)

// 自定义错误类型
type APIError struct {
    StatusCode int
    Message    string
    Endpoint   string
}

func (e *APIError) Error() string {
    return fmt.Sprintf("API error %d at %s: %s",
        e.StatusCode, e.Endpoint, e.Message)
}

// 模拟 API 调用
func callAPI(endpoint string) error {
    // 模拟不同的错误情况
    switch endpoint {
    case "/users":
        return nil
    case "/admin":
        return &APIError{
            StatusCode: 403,
            Message:    "Access denied",
            Endpoint:   endpoint,
        }
    default:
        return &APIError{
            StatusCode: 404,
            Message:    "Not found",
            Endpoint:   endpoint,
        }
    }
}

func main() {
    endpoints := []string{"/users", "/admin", "/unknown"}

    for _, ep := range endpoints {
        err := callAPI(ep)
        if err != nil {
            // 类型断言提取详细信息
            if apiErr, ok := err.(*APIError); ok {
                fmt.Printf("Endpoint: %s\n", apiErr.Endpoint)
                fmt.Printf("Status: %d\n", apiErr.StatusCode)
                fmt.Printf("Message: %s\n\n", apiErr.Message)
            }
        } else {
            fmt.Printf("%s: OK\n\n", ep)
        }
    }
}
```

### 练习 3：模拟项目的 OperationResult 模式

```go
package main

import (
    "encoding/json"
    "fmt"
    "time"
)

// 统一返回结构
type OperationResult struct {
    Success      bool        `json:"success"`
    ResponseTime float64     `json:"response_time_ms"`
    Result       interface{} `json:"result,omitempty"`
    Error        string      `json:"error,omitempty"`
}

// 模拟数据库操作
func queryDatabase(query string) OperationResult {
    start := time.Now()

    // 模拟验证
    if query == "" {
        return OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        "query cannot be empty",
        }
    }

    // 模拟执行
    time.Sleep(50 * time.Millisecond)  // 模拟延迟

    // 模拟错误
    if query == "SELECT * FROM invalid" {
        return OperationResult{
            Success:      false,
            ResponseTime: float64(time.Since(start).Milliseconds()),
            Error:        "table 'invalid' does not exist",
        }
    }

    // 成功
    return OperationResult{
        Success:      true,
        ResponseTime: float64(time.Since(start).Milliseconds()),
        Result: map[string]interface{}{
            "rows": []map[string]interface{}{
                {"id": 1, "name": "Alice"},
                {"id": 2, "name": "Bob"},
            },
        },
    }
}

func main() {
    queries := []string{
        "SELECT * FROM users",
        "",
        "SELECT * FROM invalid",
    }

    for _, q := range queries {
        result := queryDatabase(q)

        // 转为 JSON 显示
        jsonBytes, _ := json.MarshalIndent(result, "", "  ")
        fmt.Printf("Query: %q\n%s\n\n", q, string(jsonBytes))
    }
}
```

### 练习 4：分析项目错误处理

阅读 `pkg/milvus/converters.go`，找出：

1. 哪些地方使用了 `wrapError`？
2. 哪些地方使用了 `newError`？
3. 这两个函数的区别是什么？

---

## ❓ 自测问题

1. Go 为什么不使用异常机制？
   <details>
   <summary>查看答案</summary>
   Go 设计者认为异常机制会导致控制流不清晰，错误容易被忽略。显式错误返回使错误处理更明确。
   </details>

2. `errors.Is(err, target)` 的作用是什么？
   <details>
   <summary>查看答案</summary>
   检查 err 的错误链中是否包含 target 错误。支持嵌套/包装的错误。
   </details>

3. 为什么项目使用 `OperationResult` 而不是返回 `error`？
   <details>
   <summary>查看答案</summary>
   因为这是 k6 扩展，需要与 JavaScript 交互。JavaScript 不支持 Go 的多返回值模式，使用统一结构更方便。
   </details>

4. `Unwrap()` 方法的作用是什么？
   <details>
   <summary>查看答案</summary>
   返回被包装的底层错误，使 errors.Is 和 errors.As 能够遍历错误链。
   </details>

5. 什么是哨兵错误？
   <details>
   <summary>查看答案</summary>
   预定义的全局错误变量，用于与返回的错误进行比较，如 io.EOF、sql.ErrNoRows 等。
   </details>

---

## 💡 本章要点

1. **Go 使用显式错误返回** - 不是异常
2. **error 是接口** - 只需实现 `Error() string`
3. **哨兵错误** - 预定义的可比较错误值
4. **自定义错误** - 携带更多上下文信息
5. **错误包装** - `fmt.Errorf("%w", err)`
6. **错误检查**：
   - `errors.Is()` - 检查错误链
   - `errors.As()` - 提取特定类型
7. **项目模式** - `OperationResult` 统一返回结构

---

## 下一步

在下一章，我们将深入学习 JSON 处理和类型转换：

- encoding/json 包
- 结构体标签
- JavaScript 与 Go 类型映射
- 项目中的转换器实现

[继续第8章：JSON 与类型转换 →](./08-json-and-conversion.md)

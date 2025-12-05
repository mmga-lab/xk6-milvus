# 第9章：k6 扩展架构

## 学习目标

完成本章后，你将能够：

- ✅ 理解 k6 扩展的工作原理
- ✅ 掌握 RootModule/ModuleInstance 模式
- ✅ 理解 VU（Virtual User）概念
- ✅ 了解上下文管理
- ✅ 理解项目如何与 JavaScript 交互

---

## 📖 9.1 k6 简介

[k6](https://k6.io/) 是一个现代化的负载测试工具：

- 使用 **JavaScript** 编写测试脚本
- 底层使用 **Go** 实现高性能
- 支持 **扩展** 来添加新功能

### k6 测试脚本示例

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    vus: 10,        // 10 个虚拟用户
    duration: '30s', // 运行 30 秒
};

export default function() {
    const res = http.get('https://test-api.k6.io');
    check(res, { 'status is 200': (r) => r.status === 200 });
    sleep(1);
}
```

### 扩展的作用

k6 内置了 HTTP、WebSocket 等模块，但可以通过扩展添加：

- 数据库客户端（如 xk6-milvus）
- 消息队列
- 自定义协议
- 等等

---

## 📖 9.2 k6 模块系统

### 模块注册

每个扩展都需要向 k6 注册自己：

```go
// pkg/milvus/module.go

import "go.k6.io/k6/js/modules"

func init() {
    // 注册模块，路径是 JavaScript 中的导入路径
    modules.Register("k6/x/milvus", new(RootModule))
}
```

JavaScript 中使用：

```javascript
import milvus from 'k6/x/milvus';
// 或
import { client } from 'k6/x/milvus';
```

### 模块接口

k6 定义了两个核心接口：

```go
// k6 的模块接口（简化版）
type Module interface {
    NewModuleInstance(vu VU) Instance
}

type Instance interface {
    Exports() Exports
}

type Exports struct {
    Default interface{}
    Named   map[string]interface{}
}
```

---

## 📖 9.3 RootModule / ModuleInstance 模式

### 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        k6 Runtime                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────┐                                       │
│  │   RootModule    │ ← 全局单例，程序启动时创建              │
│  │   (单例)        │                                       │
│  └────────┬────────┘                                       │
│           │                                                 │
│           │ NewModuleInstance(vu)                          │
│           │                                                 │
│  ┌────────┴────────┬────────────────┬────────────────┐    │
│  │                 │                │                │    │
│  ▼                 ▼                ▼                ▼    │
│ ┌─────┐         ┌─────┐         ┌─────┐         ┌─────┐  │
│ │Milvus│         │Milvus│         │Milvus│         │Milvus│  │
│ │(VU 1)│         │(VU 2)│         │(VU 3)│         │(VU N)│  │
│ └──┬───┘         └──┬───┘         └──┬───┘         └──┬───┘  │
│    │                │                │                │      │
│    ▼                ▼                ▼                ▼      │
│ ┌─────┐         ┌─────┐         ┌─────┐         ┌─────┐     │
│ │Client│         │Client│         │Client│         │Client│     │
│ └─────┘         └─────┘         └─────┘         └─────┘     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 为什么需要这种模式？

1. **VU 隔离** - 每个虚拟用户有独立的模块实例
2. **并发安全** - 避免共享状态导致的竞态条件
3. **资源管理** - 每个 VU 可以有独立的连接、上下文等

---

## 📖 9.4 项目实现详解

### RootModule

```go
// pkg/milvus/module.go

// RootModule 是全局模块实例
// 整个 k6 进程中只有一个
type RootModule struct{}

// NewModuleInstance 为每个 VU 创建新的模块实例
// k6 会为每个 VU 调用这个方法
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
    return &Milvus{vu: vu}  // 创建 VU 专属的实例
}
```

### Milvus（ModuleInstance）

```go
// pkg/milvus/module.go

// Milvus 是每个 VU 的模块实例
type Milvus struct {
    vu modules.VU  // 保存 VU 引用，用于获取上下文等
}

// Exports 返回模块的导出内容
func (m *Milvus) Exports() modules.Exports {
    return modules.Exports{
        // 默认导出：import milvus from "k6/x/milvus"
        Default: m,

        // 命名导出：import { client } from "k6/x/milvus"
        Named: map[string]interface{}{
            "client":               m.Client,
            "clientWithCollection": m.ClientWithCollection,
        },
    }
}
```

### 接口验证

```go
// pkg/milvus/module.go

// 编译时验证接口实现
var (
    _ modules.Module   = &RootModule{}  // 确保 RootModule 实现 Module
    _ modules.Instance = &Milvus{}      // 确保 Milvus 实现 Instance
)
```

---

## 📖 9.5 VU（Virtual User）

### 什么是 VU？

VU（Virtual User）是 k6 中的虚拟用户：

- 每个 VU 独立执行测试脚本
- VU 之间并行运行
- 每个 VU 有自己的状态和资源

### VU 接口

```go
// k6 的 VU 接口（简化版）
type VU interface {
    // 获取执行上下文（用于取消、超时等）
    Context() context.Context

    // 获取 VU 状态
    State() *lib.State

    // 获取 JavaScript 运行时
    Runtime() *goja.Runtime
}
```

### 使用 VU 上下文

```go
// pkg/milvus/client.go

func (m *Milvus) createClient(address, collectionName string, token ...string) (*Client, error) {
    // 从 VU 获取上下文
    ctx := m.vu.Context()

    // 使用 VU 上下文创建 Milvus 客户端
    // 这样当 k6 测试停止时，可以正确取消进行中的操作
    c, err := milvusclient.New(ctx, milvusConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create milvus client: %v", err)
    }

    return &Client{
        client: c,
        ctx:    ctx,  // 保存上下文供后续操作使用
        vu:     m.vu, // 保存 VU 引用
        // ...
    }, nil
}
```

### 上下文的作用

```go
// pkg/milvus/collection.go

func (c *Client) LoadCollection(collectionName ...string) interface{} {
    // ...

    task, err := c.client.LoadCollection(c.ctx, option)
    // c.ctx 是 VU 的上下文

    // 等待操作完成
    err = task.Await(c.ctx)
    // 如果 k6 测试停止，上下文会被取消
    // Await 会立即返回错误，而不是无限等待
}
```

---

## 📖 9.6 JavaScript 交互

### 导出函数

Go 函数可以直接导出给 JavaScript 调用：

```go
// pkg/milvus/client.go

// Client 方法会被导出
func (m *Milvus) Client(address string, token ...string) (*Client, error) {
    // ...
}
```

```javascript
// JavaScript 中调用
import milvus from 'k6/x/milvus';

const client = milvus.client('localhost:19530');
// 或带认证
const client = milvus.client('localhost:19530', 'user:password');
```

### 导出方法

Client 的方法也会被导出：

```go
// pkg/milvus/collection.go

func (c *Client) CreateCollection(schemaInput interface{}) interface{} {
    // ...
}

func (c *Client) LoadCollection(collectionName ...string) interface{} {
    // ...
}
```

```javascript
// JavaScript 中调用
const result = client.createCollection({
    name: 'test',
    fields: [...]
});

client.loadCollection('test');
```

### 参数转换

| JavaScript | Go 函数参数 |
|-----------|-------------|
| `'localhost:19530'` | `address string` |
| `{name: 'test', ...}` | `schemaInput interface{}` |
| `['field1', 'field2']` | `outputFields []interface{}` |
| `10` | `topK int` |

### 返回值转换

```go
// 返回 map[string]interface{} 给 JavaScript
func (c *Client) Search(...) interface{} {
    // ...
    return toMap(&OperationResult{
        Success:      true,
        ResponseTime: 123.45,
        Result:       results,
    })
}
```

```javascript
// JavaScript 中接收
const result = client.search(vectors, 10, params);
console.log(result.success);        // true
console.log(result.response_time_ms); // 123.45
console.log(result.result);         // [...]
```

---

## 📖 9.7 完整流程

### 从注册到调用

```
1. k6 启动
   ↓
2. 导入 xk6-milvus 包
   ↓
3. pkg/milvus/module.go 的 init() 执行
   ↓
4. modules.Register("k6/x/milvus", new(RootModule))
   ↓
5. JavaScript 执行 import milvus from "k6/x/milvus"
   ↓
6. k6 为当前 VU 调用 RootModule.NewModuleInstance(vu)
   ↓
7. 创建 Milvus{vu: vu} 实例
   ↓
8. 调用 Milvus.Exports() 获取导出内容
   ↓
9. JavaScript 可以访问 milvus.client() 等方法
   ↓
10. 调用 milvus.client('localhost:19530')
    ↓
11. 执行 Go 函数 (m *Milvus) Client(...)
    ↓
12. 创建并返回 *Client
    ↓
13. JavaScript 调用 client.search(...)
    ↓
14. 执行 Go 方法 (c *Client) Search(...)
    ↓
15. 返回 map[string]interface{} 给 JavaScript
```

---

## 📖 9.8 最佳实践

### 1. 使用 VU 上下文

```go
// ✅ 好：使用 VU 上下文
ctx := m.vu.Context()
c, err := milvusclient.New(ctx, config)

// ❌ 差：使用 background 上下文
ctx := context.Background()
c, err := milvusclient.New(ctx, config)
```

### 2. 每个 VU 独立实例

```go
// ✅ 好：每个 VU 创建自己的客户端
func (m *Milvus) Client(...) (*Client, error) {
    return &Client{
        ctx: m.vu.Context(),
        vu:  m.vu,
        // ...
    }, nil
}

// ❌ 差：全局共享客户端
var globalClient *Client  // 不要这样做！
```

### 3. 返回友好的结构

```go
// ✅ 好：返回 map，JavaScript 可以直接访问
return toMap(&OperationResult{
    Success: true,
    Result:  data,
})

// ❌ 差：返回 Go 结构体（JavaScript 访问可能有问题）
return &OperationResult{
    Success: true,
    Result:  data,
}
```

---

## ✏️ 动手练习

### 练习 1：理解模块注册

阅读 `pkg/milvus/module.go`，回答：

1. `init()` 函数什么时候执行？
2. `"k6/x/milvus"` 这个路径在哪里使用？
3. 为什么 `NewModuleInstance` 需要接收 `vu` 参数？

### 练习 2：追踪调用流程

在 JavaScript 中执行：

```javascript
import milvus from 'k6/x/milvus';

const client = milvus.client('localhost:19530');
const result = client.createCollection({
    name: 'test',
    fields: [
        {name: 'id', dataType: 'Int64', isPrimaryKey: true},
        {name: 'vector', dataType: 'FloatVector', dimension: 128}
    ]
});
```

追踪调用链，写出涉及的 Go 函数：

1. `milvus.client(...)` → ?
2. `client.createCollection(...)` → ?
3. schema 对象如何转换为 Go 类型？

### 练习 3：创建简单的 k6 扩展

```go
// ~/go-learning/simple-ext/module.go
package simpleext

import "go.k6.io/k6/js/modules"

func init() {
    modules.Register("k6/x/simple", new(RootModule))
}

type RootModule struct{}

type SimpleModule struct {
    vu modules.VU
}

func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
    return &SimpleModule{vu: vu}
}

func (m *SimpleModule) Exports() modules.Exports {
    return modules.Exports{
        Named: map[string]interface{}{
            "hello": m.Hello,
            "add":   m.Add,
        },
    }
}

func (m *SimpleModule) Hello(name string) string {
    return "Hello, " + name + "!"
}

func (m *SimpleModule) Add(a, b int) int {
    return a + b
}
```

思考：
1. 这个扩展导出了什么？
2. JavaScript 如何使用这个扩展？
3. 如何添加一个返回当前时间的方法？

---

## ❓ 自测问题

1. 为什么需要 RootModule 和 ModuleInstance 两层结构？
   <details>
   <summary>查看答案</summary>
   RootModule 是全局单例，ModuleInstance 是每个 VU 独立的实例。这种设计确保 VU 之间的隔离，避免共享状态导致的并发问题。
   </details>

2. VU 上下文的作用是什么？
   <details>
   <summary>查看答案</summary>
   提供请求取消能力。当 k6 测试停止时，上下文会被取消，所有使用该上下文的操作都会立即终止。
   </details>

3. `modules.Register` 的第一个参数是什么？
   <details>
   <summary>查看答案</summary>
   JavaScript 中的导入路径。如 "k6/x/milvus" 对应 `import milvus from 'k6/x/milvus'`。
   </details>

4. 为什么 Client 方法返回 `interface{}` 而不是具体类型？
   <details>
   <summary>查看答案</summary>
   因为返回值需要传递给 JavaScript。使用 interface{} 配合 map[string]interface{} 可以让 JavaScript 直接访问字段。
   </details>

5. 编译时接口验证 `var _ Module = &RootModule{}` 的作用是什么？
   <details>
   <summary>查看答案</summary>
   在编译时检查 RootModule 是否实现了 Module 接口。如果没有实现，编译会失败，避免运行时错误。
   </details>

---

## 💡 本章要点

1. **k6 扩展机制** - 通过 `modules.Register` 注册模块
2. **两层结构**：
   - `RootModule` - 全局单例
   - `ModuleInstance` - 每个 VU 独立
3. **VU 隔离** - 每个虚拟用户有独立的状态
4. **上下文管理** - 使用 VU 上下文进行请求控制
5. **JS 交互** - Go 函数和方法可以直接导出给 JavaScript

---

## 下一步

在最后一章，我们将进行动手实践：

- 添加新功能
- 编写测试
- 运行完整示例

[继续第10章：动手实践 →](./10-hands-on-exercises.md)

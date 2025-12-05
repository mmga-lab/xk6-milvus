# 第2章：包与模块系统

## 学习目标

完成本章后，你将能够：

- ✅ 理解 Go 包的概念和作用
- ✅ 掌握 import 的各种用法
- ✅ 理解 go.mod 文件结构
- ✅ 明白 xk6-milvus 项目的包组织方式

---

## 📖 2.1 什么是包（Package）

包是 Go 语言中代码组织的基本单位。每个 `.go` 文件都必须属于一个包。

### 包的作用

1. **组织代码** - 将相关功能放在一起
2. **命名空间** - 避免命名冲突
3. **封装** - 控制可见性（大写字母开头 = 导出）
4. **复用** - 包可以被其他代码导入使用

### 包声明

每个 Go 文件的第一行（非注释）必须是包声明：

```go
package milvus  // 声明此文件属于 milvus 包
```

---

## 🔍 2.2 项目中的包结构

让我们看看 xk6-milvus 项目的包组织：

```
xk6-milvus/
├── register.go              # package milvus（根包）
├── go.mod                   # module github.com/mmga-lab/xk6-milvus
│
└── pkg/
    └── milvus/              # package milvus（子包）
        ├── module.go        # package milvus
        ├── client.go        # package milvus
        ├── types.go         # package milvus
        ├── collection.go    # package milvus
        ├── data.go          # package milvus
        ├── search.go        # package milvus
        ├── converters.go    # package milvus
        ├── errors.go        # package milvus
        ├── config.go        # package milvus
        └── helpers.go       # package milvus
```

💡 **注意**：虽然有两个目录层级，但它们声明的包名都是 `milvus`。包名不必与目录名相同，但通常保持一致是好习惯。

### 💻 查看实际代码

**register.go（根目录）**：
```go
package milvus

import (
    _ "github.com/mmga-lab/xk6-milvus/pkg/milvus"
)
```

**pkg/milvus/module.go**：
```go
package milvus

import (
    "go.k6.io/k6/js/modules"
)

func init() {
    modules.Register("k6/x/milvus", new(RootModule))
}
// ...
```

---

## 📖 2.3 import 详解

### 基本导入

```go
import "fmt"  // 导入标准库的 fmt 包
```

### 多包导入（推荐格式）

```go
import (
    "fmt"
    "time"
    "encoding/json"
)
```

### 导入路径

```go
import (
    // 标准库 - 短路径
    "fmt"
    "time"
    "encoding/json"

    // 第三方包 - 完整模块路径
    "github.com/milvus-io/milvus/client/v2/milvusclient"
    "go.k6.io/k6/js/modules"

    // 本项目的包
    "github.com/mmga-lab/xk6-milvus/pkg/milvus"
)
```

### 特殊导入方式

```go
import (
    // 1. 普通导入 - 使用包名访问
    "fmt"                    // fmt.Println()

    // 2. 别名导入 - 解决命名冲突或简化
    milvus "github.com/milvus-io/milvus/client/v2/milvusclient"
    // 使用：milvus.New()

    // 3. 点导入 - 直接使用导出的标识符（不推荐）
    . "fmt"
    // 使用：Println() 而不是 fmt.Println()

    // 4. 空白导入 - 只执行 init()，不使用包
    _ "github.com/mmga-lab/xk6-milvus/pkg/milvus"
)
```

---

## 🔍 2.4 理解空白导入

这是 k6 扩展的核心机制！

### register.go 的作用

```go
package milvus

import (
    _ "github.com/mmga-lab/xk6-milvus/pkg/milvus"
)
```

**为什么用 `_`？**

当你导入一个包但不使用它的任何导出内容时，Go 编译器会报错。使用 `_` 告诉编译器："我知道我没有直接使用这个包，但我需要它的副作用。"

**副作用是什么？**

导入一个包会执行它的 `init()` 函数：

```go
// pkg/milvus/module.go
func init() {
    // 这行代码在包被导入时自动执行！
    modules.Register("k6/x/milvus", new(RootModule))
}
```

### 执行流程

```
xk6 build 编译时
    ↓
导入 register.go
    ↓
register.go 导入 pkg/milvus（空白导入）
    ↓
pkg/milvus 的 init() 执行
    ↓
模块注册到 k6："k6/x/milvus"
    ↓
JavaScript 可以 import milvus from "k6/x/milvus"
```

---

## 📖 2.5 go.mod 文件详解

### 项目的 go.mod

```go
module github.com/mmga-lab/xk6-milvus

go 1.24

require (
    github.com/milvus-io/milvus/client/v2 v2.6.1
    github.com/stretchr/testify v1.10.0
    go.k6.io/k6 v0.57.0
)

require (
    // indirect 依赖（被直接依赖引入的）
    github.com/grafana/sobek v0.0.0-20250122152117-46e70c39dba3 // indirect
    // ...更多
)
```

### 各部分含义

| 部分 | 说明 |
|------|------|
| `module` | 模块路径，也是别人导入你的包时使用的路径 |
| `go 1.24` | 最低 Go 版本要求 |
| `require` | 直接依赖列表 |
| `// indirect` | 间接依赖（被依赖的依赖） |

### 常用命令

```bash
# 添加依赖
go get github.com/some/package

# 添加特定版本
go get github.com/some/package@v1.2.3

# 更新依赖
go get -u github.com/some/package

# 整理依赖（删除未使用，添加缺失）
go mod tidy

# 下载依赖到本地缓存
go mod download

# 查看依赖图
go mod graph
```

---

## 📖 2.6 包的可见性

Go 使用大小写来控制可见性：

### 导出（公开）- 大写开头

```go
// 这些可以被其他包访问
type Client struct { ... }        // 导出的类型
func NewClient() *Client { ... }  // 导出的函数
const MaxRetries = 3              // 导出的常量
var DefaultTimeout = 30           // 导出的变量
```

### 未导出（私有）- 小写开头

```go
// 这些只能在同一个包内访问
type client struct { ... }           // 未导出的类型
func createClient() *client { ... }  // 未导出的函数
const maxConnections = 100           // 未导出的常量
var defaultConfig = Config{}         // 未导出的变量
```

### 🔍 项目中的例子

```go
// pkg/milvus/types.go

// OperationResult - 大写，导出，JavaScript 可以访问
type OperationResult struct {
    Success      bool        `json:"success"`
    ResponseTime float64     `json:"response_time_ms"`
    // ...
}

// pkg/milvus/helpers.go

// toMap - 小写，未导出，只在 milvus 包内使用
func toMap(result *OperationResult) map[string]interface{} {
    // ...
}

// getCollectionName - 小写，未导出
func (c *Client) getCollectionName(collectionName ...string) string {
    // ...
}
```

---

## 📖 2.7 init 函数

每个包可以有多个 `init()` 函数，它们在包被导入时自动执行。

### init 的特点

1. 没有参数，没有返回值
2. 不能被显式调用
3. 一个文件可以有多个 init
4. 执行顺序：依赖包的 init → 当前包变量初始化 → 当前包的 init

### 🔍 项目中的 init

```go
// pkg/milvus/module.go

func init() {
    // 在 k6 模块系统中注册扩展
    modules.Register("k6/x/milvus", new(RootModule))
}
```

**执行时机**：当 k6 构建时包含这个扩展，init 函数会在程序启动时自动执行，将扩展注册到 k6 中。

---

## ✏️ 动手练习

### 练习 1：分析导入

打开 `pkg/milvus/collection.go`，列出所有导入的包：

```bash
head -15 pkg/milvus/collection.go
```

思考：
1. 哪些是标准库？
2. 哪些是第三方包？
3. 每个包的作用是什么？

### 练习 2：创建自己的包

```bash
mkdir -p ~/go-learning/myproject
cd ~/go-learning/myproject
go mod init myproject
```

创建 `greet/greet.go`：
```go
package greet

import "fmt"

// Hello 是导出的函数
func Hello(name string) {
    fmt.Printf("Hello, %s!\n", name)
}

// goodbye 是未导出的函数
func goodbye(name string) {
    fmt.Printf("Goodbye, %s!\n", name)
}
```

创建 `main.go`：
```go
package main

import "myproject/greet"

func main() {
    greet.Hello("Gopher")  // 可以调用
    // greet.goodbye("Gopher")  // 编译错误！
}
```

运行并观察：
```bash
go run main.go
```

### 练习 3：探索 init 执行顺序

创建 `~/go-learning/init-order/main.go`：

```go
package main

import "fmt"

var x = initVar()

func initVar() int {
    fmt.Println("1. 变量初始化")
    return 1
}

func init() {
    fmt.Println("2. 第一个 init")
}

func init() {
    fmt.Println("3. 第二个 init")
}

func main() {
    fmt.Println("4. main 函数")
}
```

运行观察输出顺序：
```bash
cd ~/go-learning/init-order
go mod init init-order
go run main.go
```

---

## ❓ 自测问题

1. 如何让一个函数可以被其他包访问？
   <details>
   <summary>查看答案</summary>
   函数名首字母大写
   </details>

2. `import _ "some/package"` 的作用是什么？
   <details>
   <summary>查看答案</summary>
   空白导入，只执行包的 init 函数，不使用包的导出内容
   </details>

3. go.mod 中的 `// indirect` 是什么意思？
   <details>
   <summary>查看答案</summary>
   间接依赖，即不是直接 import 的，而是被直接依赖所依赖的
   </details>

4. 同一个包的代码必须在同一个文件吗？
   <details>
   <summary>查看答案</summary>
   不需要。同一个目录下所有声明相同 package 的 .go 文件都属于同一个包
   </details>

5. xk6-milvus 的模块注册是在哪里执行的？
   <details>
   <summary>查看答案</summary>
   pkg/milvus/module.go 的 init() 函数中
   </details>

---

## 💡 本章要点

1. **包是组织单位** - 相关代码放在同一个包
2. **大写导出，小写私有** - Go 的可见性规则
3. **四种导入方式**：
   - 普通导入
   - 别名导入
   - 点导入（不推荐）
   - 空白导入（执行 init）
4. **go.mod 管理依赖** - 模块路径、版本、依赖列表
5. **init 自动执行** - 包被导入时触发

---

## 下一步

在下一章，我们将学习 Go 的基础类型和变量，包括：

- 基本数据类型
- 变量声明方式
- 类型推断
- 常量

[继续第3章：基础类型与变量 →](./03-basic-types.md)

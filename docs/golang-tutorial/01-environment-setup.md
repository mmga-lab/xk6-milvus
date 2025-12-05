# 第1章：环境搭建与第一个程序

## 学习目标

完成本章后，你将能够：

- ✅ 安装和配置 Go 开发环境
- ✅ 理解 Go 工作区结构
- ✅ 成功构建和运行 xk6-milvus 项目
- ✅ 使用基本的 Go 命令

---

## 📖 1.1 安装 Go

### macOS

```bash
# 使用 Homebrew
brew install go

# 验证安装
go version
# 输出类似：go version go1.24.0 darwin/arm64
```

### Linux (Ubuntu/Debian)

```bash
# 下载最新版本（以 1.24 为例）
wget https://go.dev/dl/go1.24.linux-amd64.tar.gz

# 解压到 /usr/local
sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz

# 添加到 PATH（编辑 ~/.bashrc 或 ~/.zshrc）
export PATH=$PATH:/usr/local/go/bin

# 验证
go version
```

### Windows

1. 下载安装包：https://go.dev/dl/
2. 运行安装程序
3. 打开命令提示符验证：`go version`

---

## 📖 1.2 配置开发环境

### 环境变量

Go 有几个重要的环境变量：

```bash
# 查看当前配置
go env

# 重要的环境变量：
# GOROOT - Go 安装目录
# GOPATH - 工作目录（默认 ~/go）
# GOBIN  - 可执行文件目录
# GOPROXY - 模块代理（国内建议设置）
```

### 国内用户设置代理

```bash
# 设置国内代理加速下载
go env -w GOPROXY=https://goproxy.cn,direct

# 开启模块支持
go env -w GO111MODULE=on
```

### IDE 配置

推荐使用 **VS Code**：

1. 安装 VS Code
2. 安装 "Go" 扩展（作者：Go Team at Google）
3. 打开命令面板（Ctrl+Shift+P），运行 "Go: Install/Update Tools"
4. 选择全部工具并安装

💡 **小贴士**：VS Code 的 Go 扩展提供了代码补全、跳转定义、自动格式化等功能。

---

## 📖 1.3 理解 Go 工作区

```
~/go/                          # GOPATH 目录
├── bin/                       # 编译后的可执行文件
├── pkg/                       # 编译后的包文件
└── src/                       # 源代码（旧模式）

~/projects/                    # 你的项目目录（推荐）
└── xk6-milvus/               # 项目根目录
    ├── go.mod                # 模块定义文件
    ├── go.sum                # 依赖校验文件
    └── ...                   # 源代码
```

💡 **现代 Go 开发**：从 Go 1.11 开始，推荐使用 Go Modules，项目可以放在任意目录。

---

## 🔍 1.4 克隆并探索项目

### 克隆项目

```bash
# 创建工作目录
mkdir -p ~/projects
cd ~/projects

# 克隆项目
git clone https://github.com/mmga-lab/xk6-milvus.git
cd xk6-milvus

# 查看项目结构
ls -la
```

### 💻 项目结构一览

```
xk6-milvus/
├── register.go          # 入口文件 ← 从这里开始
├── go.mod               # 模块定义
├── go.sum               # 依赖锁定
├── Makefile             # 构建脚本
├── pkg/
│   └── milvus/          # 核心代码
│       ├── module.go    # 模块注册
│       ├── client.go    # 客户端
│       ├── types.go     # 类型定义
│       └── ...
├── examples/            # 使用示例
└── docs/                # 文档
```

---

## 💻 1.5 第一个 Go 命令

### 查看模块信息

```bash
# 进入项目目录
cd ~/projects/xk6-milvus

# 查看模块名
cat go.mod
```

输出：
```
module github.com/mmga-lab/xk6-milvus

go 1.24

require (
    github.com/milvus-io/milvus/client/v2 v2.6.1
    go.k6.io/k6 v0.57.0
    ...
)
```

🔍 **代码解读**：
- `module github.com/mmga-lab/xk6-milvus` - 模块路径，也是导入路径
- `go 1.24` - 最低 Go 版本要求
- `require (...)` - 项目依赖

### 下载依赖

```bash
# 下载所有依赖
go mod download

# 或者整理依赖（推荐）
go mod tidy
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./pkg/milvus/

# 详细输出
go test -v ./pkg/milvus/
```

---

## 💻 1.6 构建项目

这个项目是 k6 扩展，需要使用 xk6 工具构建：

### 安装 xk6

```bash
go install go.k6.io/xk6/cmd/xk6@latest
```

### 构建 k6 + 扩展

```bash
# 使用 Makefile（推荐）
make build

# 或手动构建
xk6 build --with github.com/mmga-lab/xk6-milvus=.
```

### 验证构建

```bash
# 检查 k6 版本
./k6 version

# 应该显示包含 xk6-milvus 的构建信息
```

---

## 💻 1.7 编写你的第一个 Go 程序

让我们在项目外创建一个简单的 Go 程序来熟悉语法：

```bash
# 创建测试目录
mkdir -p ~/go-learning
cd ~/go-learning

# 初始化模块
go mod init hello
```

创建 `main.go`：

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, Golang!")
    fmt.Println("我正在学习 xk6-milvus 项目")
}
```

运行：

```bash
go run main.go
```

---

## ✏️ 动手练习

### 练习 1：探索 go 命令

运行以下命令并观察输出：

```bash
go help           # 查看所有命令
go env            # 查看环境配置
go version        # 查看版本
go list -m all    # 查看所有依赖（在项目目录运行）
```

### 练习 2：阅读入口文件

打开 `register.go` 文件，尝试理解：

```go
package milvus

import (
    _ "github.com/mmga-lab/xk6-milvus/pkg/milvus"
)
```

思考：
1. `package milvus` 是什么意思？
2. `import` 前面的 `_` 是什么？
3. 为什么文件里没有 `func main()`？

### 练习 3：运行测试

```bash
cd ~/projects/xk6-milvus

# 运行单元测试并查看覆盖率
go test -v -cover ./pkg/milvus/
```

记录：哪些测试通过了？覆盖率是多少？

---

## ❓ 自测问题

1. Go 程序的入口函数是什么？
   <details>
   <summary>查看答案</summary>
   main 包中的 main() 函数
   </details>

2. `go mod tidy` 命令的作用是什么？
   <details>
   <summary>查看答案</summary>
   整理 go.mod 文件，添加缺失的依赖，删除不需要的依赖
   </details>

3. 为什么 xk6-milvus 项目没有 main() 函数？
   <details>
   <summary>查看答案</summary>
   因为它是一个库/扩展，不是独立的可执行程序。它会被 k6 加载和调用。
   </details>

4. `_` 导入是什么意思？
   <details>
   <summary>查看答案</summary>
   空白导入，只执行包的 init() 函数，不使用包中的任何导出内容
   </details>

---

## 💡 本章要点

1. **Go 安装简单** - 下载、解压、设置 PATH
2. **Go Modules** - 现代 Go 使用 go.mod 管理依赖
3. **常用命令**：
   - `go run` - 编译并运行
   - `go build` - 编译
   - `go test` - 运行测试
   - `go mod tidy` - 整理依赖
4. **xk6-milvus 是库** - 没有 main 函数，被 k6 加载

---

## 下一步

在下一章，我们将深入学习 Go 的包和模块系统，理解：

- package 声明的含义
- import 的各种方式
- go.mod 文件的结构
- 项目中的包如何组织

[继续第2章：包与模块系统 →](./02-packages-and-modules.md)

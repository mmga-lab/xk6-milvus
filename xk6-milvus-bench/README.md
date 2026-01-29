# xk6-milvus-bench

Milvus 性能/稳定性测试框架 - 基于 xk6-milvus 的 Python CLI 工具

## 概述

xk6-milvus-bench 是一个用于 Milvus 向量数据库性能测试的框架，采用 **Python + Jinja2 + k6** 架构：

- **Python CLI** 作为用户入口，提供简洁的命令行界面
- **Jinja2 模板** 动态生成测试脚本和 K8s 资源
- **k6 (xk6-milvus)** 作为执行引擎
- **k6 operator** 在 Kubernetes 中执行分布式测试

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Python CLI (xk6-milvus-bench)            │
│  - 配置管理 (YAML)                                          │
│  - 测试编排                                                 │
│  - 报告生成                                                 │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Jinja2 模板引擎                          │
│  - k6 测试脚本模板 (.js.j2)                                 │
│  - TestRun CRD 模板 (.yaml.j2)                             │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    k6 执行引擎                              │
│  本地模式: ./k6 run generated_script.js                    │
│  K8s 模式: kubectl apply -f generated_testrun.yaml         │
└─────────────────────────────────────────────────────────────┘
```

## 安装

### 从 PyPI 安装（推荐）

```bash
# 安装（包含预编译的 k6 二进制）
pip install xk6-milvus-bench

# 或使用 uv
uv pip install xk6-milvus-bench

# 安装数据集支持（可选）
pip install xk6-milvus-bench[datasets]
```

### 从源码安装

```bash
# 克隆仓库
git clone https://github.com/mmga-lab/xk6-milvus.git
cd xk6-milvus/xk6-milvus-bench

# 使用 uv（推荐）
uv venv -p 3.10
source .venv/bin/activate
uv pip install -e .

# 或使用 pip
pip install -e .
```

### 验证安装

```bash
# 查看 k6 信息
xk6-milvus-bench k6 info
```

## 快速开始

### 查看可用场景

```bash
xk6-milvus-bench list
```

输出：
```
Available scenarios:
  search        - 向量搜索性能测试
  insert        - 数据插入性能测试
  mixed         - 混合负载测试 (读写混合)
  concurrent    - 高并发测试
  soak          - 稳定性/持久性测试
  recall        - 召回率验证测试
```

### 运行测试

```bash
# 基本搜索测试
xk6-milvus-bench run search --host localhost:19530

# 使用配置文件
xk6-milvus-bench run search -c configs/stress.yaml

# 指定参数
xk6-milvus-bench run search \
  --host localhost:19530 \
  --vus 20 \
  --duration 10m

# 测试矩阵
xk6-milvus-bench run search \
  --matrix "topk=[10,50,100]" \
  --matrix "vus=[5,10,20]"
```

### 生成脚本（不运行）

```bash
xk6-milvus-bench generate search -o ./output/
```

### K8s 模式

```bash
# 部署 k6 operator
xk6-milvus-bench deploy --install-operator

# 运行分布式测试
xk6-milvus-bench run search \
  --mode k8s \
  --namespace k6-tests \
  --parallelism 4
```

## 命令参考

### `run` - 运行测试

```bash
xk6-milvus-bench run <scenario> [options]

参数:
  scenario              测试场景名称

选项:
  -c, --config PATH     配置文件路径
  --mode TEXT           运行模式: local/k8s [默认: local]
  --host TEXT           Milvus 地址
  --token TEXT          Milvus 认证 token
  --vus INTEGER         虚拟用户数
  --duration TEXT       测试时长 (如: 5m, 1h)
  --matrix TEXT         测试矩阵参数 (可多次使用)
  --parallelism INTEGER K8s 并行度
  --namespace TEXT      K8s 命名空间
  -o, --output PATH     输出目录
  --k6 TEXT             k6 二进制文件路径
  --dry-run             仅生成脚本，不运行
```

### `generate` - 生成测试脚本

```bash
xk6-milvus-bench generate <scenario> [options]

参数:
  scenario              测试场景名称

选项:
  -c, --config PATH     配置文件路径
  -o, --output PATH     输出目录 [默认: ./generated]
  --k8s                 生成 K8s 资源 [默认: True]
```

### `list` - 列出可用场景

```bash
xk6-milvus-bench list
```

### `report` - 生成报告

```bash
xk6-milvus-bench report <result-dir> [options]

参数:
  result-dir            结果目录

选项:
  -o, --output PATH     报告输出路径 [默认: ./report.html]
  -f, --format TEXT     报告格式: html/json [默认: html]
```

### `deploy` - 部署到 K8s

```bash
xk6-milvus-bench deploy [options]

选项:
  -n, --namespace TEXT  K8s 命名空间 [默认: k6-tests]
  --install-operator    安装 k6 operator
```

### `dataset` - 管理 benchmark 数据集

```bash
# 列出可用数据集
xk6-milvus-bench dataset list

# 下载数据集
xk6-milvus-bench dataset download sift-128-euclidean

# 查看数据集详情
xk6-milvus-bench dataset info sift-128-euclidean

# 删除数据集
xk6-milvus-bench dataset delete sift-128-euclidean
```

支持的标准数据集：
- `sift-128-euclidean` - SIFT1M (128维, 1M向量, L2)
- `gist-960-euclidean` - GIST1M (960维, 1M向量, L2)
- `glove-25-angular` - GloVe-25 (25维, 1.2M向量, COSINE)
- `glove-100-angular` - GloVe-100 (100维, 1.2M向量, COSINE)
- `fashion-mnist-784-euclidean` - Fashion-MNIST (784维, 60K向量, L2)
- `mnist-784-euclidean` - MNIST (784维, 60K向量, L2)

**注意**：下载数据集需要额外依赖：

```bash
pip install xk6-milvus-bench[datasets]
# 或
pip install h5py pyarrow numpy
```

## 配置文件

### 默认配置 (configs/default.yaml)

```yaml
# Milvus 连接配置
milvus:
  host: ${MILVUS_HOST:-localhost:19530}
  token: ${MILVUS_TOKEN:-}
  database: default

# 集合配置
collection:
  name_prefix: perf_test_
  vector_dim: 128
  index:
    type: HNSW
    metric: L2
    params:
      M: 16
      efConstruction: 200

# k6 执行配置
execution:
  vus: 10
  duration: 5m

# 性能阈值
thresholds:
  search_p95_ms: 100
  search_p99_ms: 200
  insert_p95_ms: 500
  min_recall: 0.95
  max_error_rate: 0.01

# K8s 配置
kubernetes:
  namespace: k6-tests
  image: harbor.milvus.io/milvus/k6-milvus:latest
  parallelism: 1
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: "2"
      memory: 2Gi

# 场景配置
scenario:
  name: default
  type: search
  setup:
    data_size: 10000
    batch_size: 1000
  search:
    topk: 10
    output_fields: [category, price]
```

### 场景配置继承

```yaml
# configs/scenarios/search.yaml
extends: default

scenario:
  name: search-performance
  type: search
  variants:
    - name: basic
      filter: null
    - name: filtered
      filter: "price > 50"

execution:
  vus: 20
  duration: 10m
```

## 测试矩阵

测试矩阵功能可以自动生成所有参数组合并依次执行：

```bash
# 生成 27 个测试组合
xk6-milvus-bench run search \
  --matrix "topk=[10,50,100]" \
  --matrix "ef=[64,128,256]" \
  --matrix "vus=[5,10,20]"
```

或在配置文件中定义：

```yaml
matrix:
  topk: [10, 50, 100]
  ef: [64, 128, 256]
  vus: [5, 10, 20]
```

## 测试场景

### search - 向量搜索测试

测试向量相似性搜索性能，包括：
- 基本搜索
- 带过滤条件的搜索
- 不同 topK 值的搜索

### insert - 数据插入测试

测试数据插入性能，包括：
- 小批量插入 (10 条)
- 中批量插入 (100 条)
- 大批量插入 (1000 条)

### mixed - 混合负载测试

模拟真实场景的读写混合负载：
- 56% 搜索操作
- 24% 查询操作
- 16% 插入操作
- 4% 删除操作

### concurrent - 高并发测试

测试系统在高并发下的表现：
- 快速 VU 增长
- 峰值负载
- 超时检测

### soak - 稳定性测试

长时间运行测试，检测：
- 内存泄漏
- 性能退化
- 延迟方差

### recall - 召回率验证

验证搜索结果质量：
- 不同 topK 值的召回率
- 召回率分布
- 低召回率检测

### benchmark - 标准数据集性能评估

使用标准向量数据库 benchmark 数据集进行精确性能评估：
- 真实召回率计算（与 ground truth 对比）
- 支持 SIFT1M, GIST1M, GloVe 等标准数据集
- 需要先下载数据集：`xk6-milvus-bench dataset download <name>`
- 使用 xk6-parquet 扩展读取 Parquet 格式数据

```bash
# 下载数据集
xk6-milvus-bench dataset download sift-128-euclidean

# 运行 benchmark 测试
xk6-milvus-bench run benchmark \
  --host localhost:19530 \
  --dataset sift-128-euclidean \
  --vus 10 \
  --duration 5m
```

## Docker 镜像

构建 xk6-milvus Docker 镜像：

```bash
# 在 xk6-milvus 项目根目录
docker build -t harbor.milvus.io/milvus/k6-milvus:latest -f xk6-milvus-bench/docker/Dockerfile .
docker push harbor.milvus.io/milvus/k6-milvus:latest
```

## Grafana 监控

导入 `k8s/monitoring/grafana-dashboard.json` 到 Grafana，可视化以下指标：

- Search QPS 和延迟分布
- Insert 吞吐量和延迟
- 召回率趋势
- 错误率

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `MILVUS_HOST` | Milvus 服务地址 | `localhost:19530` |
| `MILVUS_TOKEN` | Milvus 认证 token | 空 |
| `TESTRUN_NAME` | 测试运行名称 | `local` |

## 输出示例

```
$ xk6-milvus-bench run search --host localhost:19530 --vus 10 --duration 2m

╭─────────────────────────────────────────╮
│  xk6-milvus-bench v0.1.0                │
│  Scenario: search                       │
╰─────────────────────────────────────────╯

Configuration:
  Target: localhost:19530
  VUs: 10
  Duration: 2m
  Mode: local

Running combination 1/1:
  Generated script: ./output/search.js

──────────────────────────────────────────
Results: search-performance
──────────────────────────────────────────
  Status: ✅ success
  Duration: 123.45s
  Thresholds: ✅ All passed

  Metrics:
    Latency:
      milvus_search_latency: avg=23.45, p95=67.80, p99=98.20
    Throughput:
      milvus_search_ops: 15234
    Quality:
      milvus_recall: avg=97.23, p95=99.10

JSON report: ./output/report.json
HTML report: ./output/report.html

✅ All tests passed
```

## 开发

```bash
# 安装开发依赖
uv pip install -e ".[dev]"

# 运行测试
pytest tests/ -v

# 代码检查
ruff check .

# 代码格式化
ruff format .
```

## 许可证

MIT License

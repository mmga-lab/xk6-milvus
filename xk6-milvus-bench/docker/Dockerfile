# xk6-milvus Docker 镜像
# 包含 xk6-milvus, xk6-parquet, xk6-faker 扩展
# 用于在 Kubernetes 中运行 Milvus 性能测试

# 构建阶段：编译带扩展的 k6
FROM golang:1.25-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git make

# 安装 xk6
RUN go install go.k6.io/xk6/cmd/xk6@latest

# 设置工作目录
WORKDIR /build

# 使用 xk6 构建带多个扩展的 k6
# - xk6-milvus: Milvus 客户端扩展
# - xk6-parquet: Parquet 文件读取扩展（用于加载 benchmark 数据集）
# - xk6-faker: 随机数据生成扩展
RUN xk6 build \
    --with github.com/mmga-lab/xk6-milvus@latest \
    --with github.com/mmga-lab/xk6-parquet@latest \
    --with github.com/grafana/xk6-faker@latest \
    --output /build/k6

# 运行阶段：最小化镜像
FROM alpine:3.19

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    libc6-compat

# 创建非 root 用户
RUN adduser -D -u 1000 k6user

# 创建数据目录
RUN mkdir -p /data /scripts && chown -R k6user:k6user /data /scripts

# 复制 k6 二进制文件
COPY --from=builder /build/k6 /usr/local/bin/k6

# 设置用户
USER k6user

# 设置工作目录
WORKDIR /scripts

# 数据集目录
VOLUME ["/data"]

# 默认命令
ENTRYPOINT ["k6"]
CMD ["--help"]

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD k6 version || exit 1

# 标签
LABEL org.opencontainers.image.title="xk6-milvus"
LABEL org.opencontainers.image.description="k6 with Milvus, Parquet, and Faker extensions for vector database performance testing"
LABEL org.opencontainers.image.source="https://github.com/mmga-lab/xk6-milvus"
LABEL org.opencontainers.image.vendor="MMGA Lab"
LABEL org.opencontainers.image.version="1.0.0"

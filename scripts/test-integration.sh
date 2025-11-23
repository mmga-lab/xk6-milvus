#!/bin/bash

# 集成测试运行脚本
# 使用方法: ./scripts/test-integration.sh [milvus-host:port]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== xk6-milvus 集成测试 ===${NC}\n"

# 1. 检查 MILVUS_HOST
if [ -z "$1" ]; then
    if [ -z "$MILVUS_HOST" ]; then
        echo -e "${YELLOW}未指定 Milvus 地址，使用默认值: localhost:19530${NC}"
        export MILVUS_HOST="localhost:19530"
    else
        echo -e "${GREEN}使用环境变量 MILVUS_HOST: $MILVUS_HOST${NC}"
    fi
else
    export MILVUS_HOST="$1"
    echo -e "${GREEN}使用指定的 Milvus 地址: $MILVUS_HOST${NC}"
fi

# 2. 检查 Milvus 连接
echo -e "\n${YELLOW}正在检查 Milvus 连接...${NC}"
MILVUS_API_HOST=$(echo $MILVUS_HOST | cut -d':' -f1)
MILVUS_API_PORT=9091

if command -v curl &> /dev/null; then
    if curl -sf "http://${MILVUS_API_HOST}:${MILVUS_API_PORT}/healthz" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Milvus 服务可访问${NC}"
    else
        echo -e "${RED}✗ 无法连接到 Milvus (http://${MILVUS_API_HOST}:${MILVUS_API_PORT}/healthz)${NC}"
        echo -e "${YELLOW}请确保 Milvus 服务正在运行${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠ curl 未安装，跳过连接检查${NC}"
fi

# 3. 运行集成测试
echo -e "\n${YELLOW}正在运行集成测试...${NC}\n"
go test -tags=integration -v -race -coverprofile=integration-coverage.txt -covermode=atomic ./pkg/milvus

# 4. 显示测试结果
if [ $? -eq 0 ]; then
    echo -e "\n${GREEN}=== 集成测试通过 ✓ ===${NC}"

    # 显示覆盖率
    if [ -f integration-coverage.txt ]; then
        COVERAGE=$(go tool cover -func=integration-coverage.txt | grep total | awk '{print $3}')
        echo -e "${GREEN}集成测试覆盖率: $COVERAGE${NC}"

        # 生成 HTML 报告
        go tool cover -html=integration-coverage.txt -o integration-coverage.html
        echo -e "${GREEN}覆盖率报告已生成: integration-coverage.html${NC}"
    fi
else
    echo -e "\n${RED}=== 集成测试失败 ✗ ===${NC}"
    exit 1
fi

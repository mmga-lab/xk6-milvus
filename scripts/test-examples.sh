#!/bin/bash

# k6 Examples 端到端测试脚本
# 使用方法: ./scripts/test-examples.sh [milvus-host:port]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== xk6-milvus Examples 端到端测试 ===${NC}\n"

# 1. 检查 k6 是否存在
if [ ! -f "./k6" ]; then
    echo -e "${YELLOW}k6 二进制文件不存在，正在构建...${NC}"
    make build
    if [ $? -ne 0 ]; then
        echo -e "${RED}✗ k6 构建失败${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ k6 构建成功${NC}\n"
fi

# 2. 检查 Milvus 地址
if [ -z "$1" ]; then
    if [ -z "$MILVUS_HOST" ]; then
        echo -e "${YELLOW}未指定 Milvus 地址，使用默认值: localhost:19530${NC}"
        MILVUS_ADDRESS="localhost:19530"
    else
        MILVUS_ADDRESS="$MILVUS_HOST"
        echo -e "${GREEN}使用环境变量 MILVUS_HOST: $MILVUS_ADDRESS${NC}"
    fi
else
    MILVUS_ADDRESS="$1"
    echo -e "${GREEN}使用指定的 Milvus 地址: $MILVUS_ADDRESS${NC}"
fi

# 3. 定义要测试的 examples
EXAMPLES=(
    "examples/basic-operations.js"
    "examples/collection-management.js"
    "examples/vector-search.js"
)

# 可选的高级示例（如果存在）
OPTIONAL_EXAMPLES=(
    "examples/hybrid-search.js"
    "examples/full-text-search.js"
)

# 4. 测试函数
run_example() {
    local example_file=$1
    local example_name=$(basename "$example_file" .js)

    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}测试: $example_name${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

    # 临时修改文件中的 Milvus 地址
    local temp_file="${example_file}.tmp"
    if grep -q "const MILVUS_ADDRESS" "$example_file"; then
        sed "s|const MILVUS_ADDRESS = .*|const MILVUS_ADDRESS = \"${MILVUS_ADDRESS}\";|" "$example_file" > "$temp_file"
    else
        cp "$example_file" "$temp_file"
    fi

    # 运行测试
    if ./k6 run --quiet "$temp_file"; then
        echo -e "${GREEN}✓ $example_name 测试通过${NC}"
        rm "$temp_file"
        return 0
    else
        echo -e "${RED}✗ $example_name 测试失败${NC}"
        rm "$temp_file"
        return 1
    fi
}

# 5. 运行所有测试
echo -e "\n${YELLOW}开始运行 Examples 测试...${NC}"

PASSED=0
FAILED=0
SKIPPED=0

# 运行必需的示例
for example in "${EXAMPLES[@]}"; do
    if [ -f "$example" ]; then
        if run_example "$example"; then
            ((PASSED++))
        else
            ((FAILED++))
        fi
    else
        echo -e "${YELLOW}⚠ 跳过: $example (文件不存在)${NC}"
        ((SKIPPED++))
    fi
done

# 运行可选的示例
for example in "${OPTIONAL_EXAMPLES[@]}"; do
    if [ -f "$example" ]; then
        echo -e "\n${YELLOW}运行可选示例: $(basename $example)${NC}"
        if run_example "$example"; then
            ((PASSED++))
        else
            echo -e "${YELLOW}⚠ 可选示例失败，但不影响总体结果${NC}"
        fi
    fi
done

# 6. 显示测试总结
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}测试总结${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}通过: $PASSED${NC}"
echo -e "${RED}失败: $FAILED${NC}"
echo -e "${YELLOW}跳过: $SKIPPED${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}=== 所有 Examples 测试通过 ✓ ===${NC}"
    exit 0
else
    echo -e "${RED}=== 有 $FAILED 个 Example 测试失败 ✗ ===${NC}"
    exit 1
fi

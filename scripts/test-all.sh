#!/bin/bash

# 完整测试流程脚本
# 使用方法: ./scripts/test-all.sh [milvus-host:port]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

echo -e "${MAGENTA}"
echo "╔════════════════════════════════════════════════════════════╗"
echo "║          xk6-milvus 完整测试套件                          ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo -e "${NC}\n"

MILVUS_ADDRESS="${1:-${MILVUS_HOST:-localhost:19530}}"
echo -e "${GREEN}Milvus 地址: $MILVUS_ADDRESS${NC}\n"

# 测试结果统计
TOTAL_TESTS=3
PASSED_TESTS=0
FAILED_TESTS=0

# 测试函数
run_test_phase() {
    local phase_name=$1
    local phase_command=$2

    echo -e "\n${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║  $phase_name${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}\n"

    if eval "$phase_command"; then
        echo -e "\n${GREEN}✓ $phase_name 通过${NC}"
        ((PASSED_TESTS++))
        return 0
    else
        echo -e "\n${RED}✗ $phase_name 失败${NC}"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Phase 1: 单元测试
run_test_phase "阶段 1: 单元测试 (无需 Milvus)" "make test"

# Phase 2: Go 集成测试
export MILVUS_HOST="$MILVUS_ADDRESS"
run_test_phase "阶段 2: Go 集成测试 (需要 Milvus)" "./scripts/test-integration.sh $MILVUS_ADDRESS"

# Phase 3: k6 Examples 端到端测试
run_test_phase "阶段 3: k6 Examples 端到端测试" "./scripts/test-examples.sh $MILVUS_ADDRESS"

# 最终报告
echo -e "\n${MAGENTA}"
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                  测试完成 - 总结报告                       ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

echo -e "总测试阶段: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "通过阶段:   ${GREEN}$PASSED_TESTS${NC}"
echo -e "失败阶段:   ${RED}$FAILED_TESTS${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║              ✓✓✓ 所有测试通过！✓✓✓                      ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}\n"

    # 显示覆盖率信息
    if [ -f coverage.txt ]; then
        UNIT_COVERAGE=$(go tool cover -func=coverage.txt | grep total | awk '{print $3}')
        echo -e "${BLUE}单元测试覆盖率: $UNIT_COVERAGE${NC}"
    fi

    if [ -f integration-coverage.txt ]; then
        INT_COVERAGE=$(go tool cover -func=integration-coverage.txt | grep total | awk '{print $3}')
        echo -e "${BLUE}集成测试覆盖率: $INT_COVERAGE${NC}"
    fi

    exit 0
else
    echo -e "${RED}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║              ✗✗✗ 有测试失败！✗✗✗                        ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════════════════╝${NC}\n"
    exit 1
fi

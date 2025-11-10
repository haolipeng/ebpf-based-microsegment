#!/bin/bash
#
# eBPF Microsegment Systemd 健康检查脚本
#

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

FAILED=0

check_service() {
    local service=$1
    if systemctl is-active --quiet $service; then
        echo -e "${GREEN}✓${NC} $service is running"
        return 0
    else
        echo -e "${RED}✗${NC} $service is not running"
        return 1
    fi
}

check_api() {
    local name=$1
    local url=$2
    if curl -sf $url >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} $name API ($url) is accessible"
        return 0
    else
        echo -e "${RED}✗${NC} $name API ($url) is not accessible"
        return 1
    fi
}

echo "========================================"
echo "eBPF Microsegment 健康检查"
echo "========================================"
echo ""

# 检查 PostgreSQL
echo "检查 PostgreSQL 服务..."
if ! check_service postgresql; then
    FAILED=1
fi
echo ""

# 检查 Server
echo "检查 Server 服务..."
if ! check_service microsegment-server; then
    FAILED=1
fi

# 检查 Server API
if ! check_api "Server" "http://localhost:8080/health"; then
    FAILED=1
fi

# 检查 Server gRPC
if nc -z localhost 9090 2>/dev/null; then
    echo -e "${GREEN}✓${NC} Server gRPC port (9090) is open"
else
    echo -e "${RED}✗${NC} Server gRPC port (9090) is not accessible"
    FAILED=1
fi
echo ""

# 检查 Agent
echo "检查 Agent 服务..."
if ! check_service microsegment-agent; then
    FAILED=1
fi

# 检查 Agent API
if ! check_api "Agent" "http://localhost:8081/health"; then
    FAILED=1
fi
echo ""

# 总结
echo "========================================"
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}所有检查通过!${NC}"
    echo "========================================"
    exit 0
else
    echo -e "${RED}部分检查失败!${NC}"
    echo "========================================"
    echo ""
    echo "查看日志:"
    echo "  journalctl -u microsegment-server -n 50"
    echo "  journalctl -u microsegment-agent -n 50"
    exit 1
fi

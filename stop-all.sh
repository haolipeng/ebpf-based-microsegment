#!/bin/bash
# eBPF Microsegmentation - Complete Shutdown Script

PROJECT_ROOT="/home/work/ebpf-based-microsegment"

# Color codes for better readability
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Ensure script runs with root privileges (needed to stop agent process started via sudo)
if [[ $EUID -ne 0 ]]; then
  echo -e "${YELLOW}检测到当前不是 root 权限，正在使用 sudo 重新执行...${NC}"
  exec sudo -E "$0" "$@"
fi

cd "$PROJECT_ROOT"

echo -e "${BOLD}========================================"
echo -e "停止 eBPF 微分段系统"
echo -e "========================================${NC}"
echo

# 1. Stop Web UI
echo -e "${BLUE}[1/4] 停止 Web UI...${NC}"
if pkill -f "vite" > /dev/null 2>&1; then
    echo -e "      ${GREEN}✓ Web UI 已停止${NC}"
else
    echo -e "      ${YELLOW}Web UI 未运行${NC}"
fi
echo

# 2. Stop Agent
echo -e "${BLUE}[2/4] 停止 Agent 组件...${NC}"
if pkill -f microsegment-agent > /dev/null 2>&1; then
    echo -e "      ${GREEN}✓ Agent 已停止${NC}"
else
    echo -e "      ${YELLOW}Agent 未运行${NC}"
fi
echo

# 3. Stop Server
echo -e "${BLUE}[3/4] 停止 Server 组件...${NC}"
if pkill -f microsegment-server > /dev/null 2>&1; then
    echo -e "      ${GREEN}✓ Server 已停止${NC}"
else
    echo -e "      ${YELLOW}Server 未运行${NC}"
fi
echo

# 4. Stop PostgreSQL
echo -e "${BLUE}[4/4] 停止 PostgreSQL 数据库...${NC}"
if docker-compose -f docker-compose.simple.yml down > /dev/null 2>&1; then
    echo -e "      ${GREEN}✓ PostgreSQL 已停止${NC}"
else
    echo -e "      ${YELLOW}PostgreSQL 未运行${NC}"
fi
echo

echo -e "${BOLD}${GREEN}========================================"
echo -e "所有服务已停止"
echo -e "========================================${NC}"

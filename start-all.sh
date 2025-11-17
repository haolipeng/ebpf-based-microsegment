#!/bin/bash
# eBPF Microsegmentation - Complete Startup Script

set -e

PROJECT_ROOT="/home/work/ebpf-based-microsegment"
cd "$PROJECT_ROOT"

# Disable proxy for localhost and local IP connections
export NO_PROXY=localhost,127.0.0.1,10.107.12.201
export no_proxy=localhost,127.0.0.1,10.107.12.201

# Color codes for better readability
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color
BOLD='\033[1m'

echo -e "${BOLD}========================================"
echo -e "启动 eBPF 微分段系统"
echo -e "========================================${NC}"
echo

# 1. Start PostgreSQL
echo -e "${BLUE}[1/4] 启动 PostgreSQL 数据库...${NC}"
if docker ps | grep -q microsegment-postgres; then
    echo -e "      ${YELLOW}PostgreSQL 已在运行${NC}"
else
    docker-compose -f docker-compose.simple.yml up -d > /dev/null 2>&1
    echo -e "      等待 PostgreSQL 就绪..."
    sleep 5
fi
echo -e "      ${GREEN}✓ PostgreSQL 运行正常${NC}"
echo

# 2. Start Server
echo -e "${BLUE}[2/4] 启动 Server 组件...${NC}"
if pgrep -f microsegment-server > /dev/null; then
    SERVER_PID=$(pgrep -f microsegment-server)
    echo -e "      ${YELLOW}Server 已在运行 (PID: ${SERVER_PID})${NC}"
else
    nohup ./bin/microsegment-server -config config/server.yaml > /tmp/server.log 2>&1 &
    sleep 3
    if pgrep -f microsegment-server > /dev/null; then
        SERVER_PID=$(pgrep -f microsegment-server)
        echo -e "      ${GREEN}✓ Server 启动成功 (PID: ${SERVER_PID})${NC}"
    else
        echo -e "      ${RED}✗ Server 启动失败，查看日志: /tmp/server.log${NC}"
        exit 1
    fi
fi
echo

# 3. Start Agent
echo -e "${BLUE}[3/4] 启动 Agent 组件...${NC}"
if pgrep -f microsegment-agent > /dev/null; then
    AGENT_PID=$(pgrep -f microsegment-agent | head -1)
    echo -e "      ${YELLOW}Agent 已在运行 (PID: ${AGENT_PID})${NC}"
else
    nohup sudo -E ./bin/microsegment-agent --config config/agent.yaml > /tmp/agent.log 2>&1 &
    sleep 3
    if pgrep -f microsegment-agent > /dev/null; then
        AGENT_PID=$(pgrep -f microsegment-agent | head -1)
        echo -e "      ${GREEN}✓ Agent 启动成功 (PID: ${AGENT_PID})${NC}"
    else
        echo -e "      ${RED}✗ Agent 启动失败，查看日志: /tmp/agent.log${NC}"
        exit 1
    fi
fi
echo

# 4. Start Web UI
echo -e "${BLUE}[4/4] 启动 Web UI...${NC}"
if pgrep -f "vite" > /dev/null; then
    echo -e "      ${YELLOW}Web UI 已在运行${NC}"
else
    cd web
    # Use setsid to completely detach from terminal
    setsid nohup npm run dev > /tmp/web.log 2>&1 < /dev/null &
    cd ..
    sleep 5
    if pgrep -f "vite" > /dev/null; then
        echo -e "      ${GREEN}✓ Web UI 启动成功${NC}"
    else
        echo -e "      ${RED}✗ Web UI 启动失败，查看日志: /tmp/web.log${NC}"
        exit 1
    fi
fi
echo

echo -e "${BOLD}${GREEN}========================================"
echo -e "所有服务启动完成！"
echo -e "========================================${NC}"
echo
echo -e "${BOLD}访问信息:${NC}"
echo -e "  ${BLUE}Web UI:${NC}     http://10.107.12.201:3000"
echo -e "  ${BLUE}Server API:${NC} http://10.107.12.201:8080"
echo -e "  ${BLUE}Agent API:${NC}  http://10.107.12.201:8081"
echo
echo -e "${BOLD}日志文件:${NC}"
echo -e "  ${BLUE}Server:${NC}  /tmp/server.log"
echo -e "  ${BLUE}Agent:${NC}   /tmp/agent.log"
echo -e "  ${BLUE}Web UI:${NC}  /tmp/web.log"
echo
echo -e "${BOLD}查看服务状态:${NC} ps aux | grep -E '(microsegment|vite)'"
echo -e "${BOLD}========================================${NC}"

#!/bin/bash
# eBPF Microsegmentation - Complete Startup Script
# Fixed version with improved error handling and process management

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

# PID files directory
PID_DIR="${PROJECT_ROOT}/.pids"
mkdir -p "$PID_DIR"

# Cleanup function for error handling
cleanup_on_error() {
    echo -e "\n${YELLOW}检测到启动失败，已启动的服务不会自动停止${NC}"
    echo -e "${YELLOW}请执行以下命令清理所有服务:${NC}"
    echo -e "  ${BLUE}./stop-all.sh${NC}"
    echo -e ""
}

# Set trap for cleanup on error
trap cleanup_on_error EXIT

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

    # Wait for PostgreSQL to be ready (max 30 seconds)
    for i in {1..30}; do
        if docker exec microsegment-postgres pg_isready -U postgres > /dev/null 2>&1; then
            break
        fi
        sleep 1
    done

    if ! docker exec microsegment-postgres pg_isready -U postgres > /dev/null 2>&1; then
        echo -e "      ${RED}✗ PostgreSQL 启动超时${NC}"
        exit 1
    fi
fi
echo -e "      ${GREEN}✓ PostgreSQL 运行正常${NC}"
echo

# 2. Start Server
echo -e "${BLUE}[2/4] 启动 Server 组件...${NC}"
if [ -f "$PID_DIR/server.pid" ] && kill -0 $(cat "$PID_DIR/server.pid") 2>/dev/null; then
    SERVER_PID=$(cat "$PID_DIR/server.pid")
    echo -e "      ${YELLOW}Server 已在运行 (PID: ${SERVER_PID})${NC}"
else
    # Clean up stale PID file
    rm -f "$PID_DIR/server.pid"

    # Start server
    nohup ./bin/microsegment-server -config config/server.yaml > /tmp/server.log 2>&1 &
    SERVER_PID=$!
    echo $SERVER_PID > "$PID_DIR/server.pid"

    # Wait for server to start (max 10 seconds)
    echo -e "      等待 Server 启动..."
    for i in {1..10}; do
        if kill -0 $SERVER_PID 2>/dev/null && curl -s http://localhost:8080/api/v1/health > /dev/null 2>&1; then
            echo -e "      ${GREEN}✓ Server 启动成功 (PID: ${SERVER_PID})${NC}"
            break
        fi

        # Check if process died
        if ! kill -0 $SERVER_PID 2>/dev/null; then
            echo -e "      ${RED}✗ Server 进程意外退出${NC}"
            echo -e "      查看日志: tail -20 /tmp/server.log"
            tail -20 /tmp/server.log
            exit 1
        fi
        sleep 1
    done

    # Final check
    if ! curl -s http://localhost:8080/api/v1/health > /dev/null 2>&1; then
        echo -e "      ${RED}✗ Server 启动失败，查看日志: /tmp/server.log${NC}"
        tail -20 /tmp/server.log
        exit 1
    fi
fi
echo

# 3. Start Agent
echo -e "${BLUE}[3/4] 启动 Agent 组件...${NC}"
if [ -f "$PID_DIR/agent.pid" ] && sudo kill -0 $(cat "$PID_DIR/agent.pid") 2>/dev/null; then
    AGENT_PID=$(cat "$PID_DIR/agent.pid")
    echo -e "      ${YELLOW}Agent 已在运行 (PID: ${AGENT_PID})${NC}"
else
    # Clean up stale PID file
    rm -f "$PID_DIR/agent.pid"

    # Start agent with sudo
    nohup sudo -E ./bin/microsegment-agent --config config/agent.yaml > /tmp/agent.log 2>&1 &
    AGENT_PID=$!
    echo $AGENT_PID > "$PID_DIR/agent.pid"

    # Wait for agent to start (max 10 seconds)
    echo -e "      等待 Agent 启动..."
    for i in {1..10}; do
        if sudo kill -0 $AGENT_PID 2>/dev/null && curl -s http://localhost:8081/api/v1/health > /dev/null 2>&1; then
            echo -e "      ${GREEN}✓ Agent 启动成功 (PID: ${AGENT_PID})${NC}"
            break
        fi

        # Check if process died
        if ! sudo kill -0 $AGENT_PID 2>/dev/null; then
            echo -e "      ${RED}✗ Agent 进程意外退出${NC}"
            echo -e "      查看日志: tail -20 /tmp/agent.log"
            tail -20 /tmp/agent.log
            exit 1
        fi
        sleep 1
    done

    # Final check
    if ! curl -s http://localhost:8081/api/v1/health > /dev/null 2>&1; then
        echo -e "      ${RED}✗ Agent 启动失败，查看日志: /tmp/agent.log${NC}"
        tail -20 /tmp/agent.log
        exit 1
    fi
fi
echo

# 4. Start Web UI
echo -e "${BLUE}[4/4] 启动 Web UI...${NC}"

# Check if Web UI is already running (by port or specific process)
if lsof -i :3000 > /dev/null 2>&1 || \
   ([ -f "$PID_DIR/web.pid" ] && kill -0 $(cat "$PID_DIR/web.pid") 2>/dev/null); then
    if [ -f "$PID_DIR/web.pid" ]; then
        WEB_PID=$(cat "$PID_DIR/web.pid")
        echo -e "      ${YELLOW}Web UI 已在运行 (PID: ${WEB_PID})${NC}"
    else
        echo -e "      ${YELLOW}Web UI 已在运行 (端口 3000 被占用)${NC}"
    fi
else
    # Clean up stale PID file
    rm -f "$PID_DIR/web.pid"

    # Change to web directory using absolute path
    cd "$PROJECT_ROOT/web" || {
        echo -e "      ${RED}✗ 无法进入 web 目录${NC}"
        exit 1
    }

    # Start Web UI (only use setsid, remove redundant nohup)
    setsid npm run dev > /tmp/web.log 2>&1 &
    WEB_PID=$!
    echo $WEB_PID > "$PID_DIR/web.pid"

    # Return to project root
    cd "$PROJECT_ROOT" || exit 1

    # Wait for Web UI to start (max 30 seconds)
    echo -e "      等待 Web UI 启动..."
    WEB_STARTED=false
    for i in {1..30}; do
        # Check if Vite dev server is responding
        if curl -s http://localhost:3000 > /dev/null 2>&1; then
            echo -e "      ${GREEN}✓ Web UI 启动成功 (PID: ${WEB_PID})${NC}"
            WEB_STARTED=true
            break
        fi

        # Check if process is still alive
        if ! kill -0 $WEB_PID 2>/dev/null; then
            echo -e "      ${RED}✗ Web UI 进程意外退出${NC}"
            echo -e "      查看日志: tail -30 /tmp/web.log"
            echo -e ""
            tail -30 /tmp/web.log
            exit 1
        fi

        # Show progress
        if [ $((i % 5)) -eq 0 ]; then
            echo -n "."
        fi
        sleep 1
    done
    echo ""

    # Final check
    if [ "$WEB_STARTED" = false ]; then
        echo -e "      ${RED}✗ Web UI 启动超时 (30秒)${NC}"
        echo -e "      进程仍在运行，但端口未响应"
        echo -e "      查看日志: tail -f /tmp/web.log"
        echo -e ""
        echo -e "      ${YELLOW}可能原因:${NC}"
        echo -e "        1. npm 依赖未安装 (执行 cd web && npm install)"
        echo -e "        2. 端口 3000 被占用"
        echo -e "        3. Vite 配置错误"
        exit 1
    fi
fi
echo

# If we reach here, all services started successfully
# Disable the cleanup trap
trap - EXIT

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
echo -e "  ${BLUE}Server:${NC}  /tmp/server.log (tail -f /tmp/server.log)"
echo -e "  ${BLUE}Agent:${NC}   /tmp/agent.log  (tail -f /tmp/agent.log)"
echo -e "  ${BLUE}Web UI:${NC}  /tmp/web.log    (tail -f /tmp/web.log)"
echo
echo -e "${BOLD}PID 文件:${NC}"
echo -e "  ${BLUE}Server:${NC}  $PID_DIR/server.pid"
echo -e "  ${BLUE}Agent:${NC}   $PID_DIR/agent.pid"
echo -e "  ${BLUE}Web UI:${NC}  $PID_DIR/web.pid"
echo
echo -e "${BOLD}管理命令:${NC}"
echo -e "  ${BLUE}查看状态:${NC}    ps aux | grep -E '(microsegment|vite)'"
echo -e "  ${BLUE}停止服务:${NC}    ./stop-all.sh"
echo -e "  ${BLUE}重启服务:${NC}    ./stop-all.sh && ./start-all.sh"
echo -e "${BOLD}========================================${NC}"

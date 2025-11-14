#!/bin/bash
# eBPF Microsegmentation - Complete Startup Script

set -e

PROJECT_ROOT="/home/work/ebpf-based-microsegment"
cd "$PROJECT_ROOT"

# Disable proxy for localhost and local IP connections
export NO_PROXY=localhost,127.0.0.1,10.107.12.201
export no_proxy=localhost,127.0.0.1,10.107.12.201

echo "========================================"
echo "启动 eBPF 微分段系统"
echo "========================================"
echo

# 1. Start PostgreSQL
echo "[1/4] 启动 PostgreSQL 数据库..."
if docker ps | grep -q microsegment-postgres; then
    echo "  PostgreSQL 已在运行"
else
    docker-compose -f docker-compose.simple.yml up -d
    echo "  等待 PostgreSQL 就绪..."
    sleep 5
fi
echo "  ✓ PostgreSQL 运行正常"
echo

# 2. Start Server
echo "[2/4] 启动 Server 组件..."
if pgrep -f microsegment-server > /dev/null; then
    echo "  Server 已在运行"
else
    nohup ./bin/microsegment-server -config config/server.yaml > /tmp/server.log 2>&1 &
    sleep 3
    if pgrep -f microsegment-server > /dev/null; then
        echo "  ✓ Server 启动成功 (PID: $(pgrep -f microsegment-server))"
    else
        echo "  ✗ Server 启动失败，查看日志: /tmp/server.log"
        exit 1
    fi
fi
echo

# 3. Start Agent
echo "[3/4] 启动 Agent 组件..."
if pgrep -f microsegment-agent > /dev/null; then
    echo "  Agent 已在运行"
else
    nohup sudo -E ./bin/microsegment-agent --config config/agent.yaml > /tmp/agent.log 2>&1 &
    sleep 3
    if pgrep -f microsegment-agent > /dev/null; then
        echo "  ✓ Agent 启动成功 (PID: $(pgrep -f microsegment-agent | head -1))"
    else
        echo "  ✗ Agent 启动失败，查看日志: /tmp/agent.log"
        exit 1
    fi
fi
echo

# 4. Start Web UI
echo "[4/4] 启动 Web UI..."
if pgrep -f "vite" > /dev/null; then
    echo "  Web UI 已在运行"
else
    cd web
    nohup npm run dev > /tmp/web.log 2>&1 &
    cd ..
    sleep 5
    if pgrep -f "vite" > /dev/null; then
        echo "  ✓ Web UI 启动成功"
    else
        echo "  ✗ Web UI 启动失败，查看日志: /tmp/web.log"
        exit 1
    fi
fi
echo

echo "========================================"
echo "所有服务启动完成！"
echo "========================================"
echo
echo "访问信息:"
echo "  Web UI:    http://10.107.12.201:3000"
echo "  Server API: http://10.107.12.201:8080"
echo "  Agent API:  http://10.107.12.201:8081"
echo
echo "日志文件:"
echo "  Server:  /tmp/server.log"
echo "  Agent:   /tmp/agent.log"
echo "  Web UI:  /tmp/web.log"
echo
echo "查看服务状态: ps aux | grep -E '(microsegment|vite)'"
echo "========================================"

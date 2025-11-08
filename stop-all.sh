#!/bin/bash
# eBPF Microsegmentation - Complete Shutdown Script

PROJECT_ROOT="/home/work/ebpf-based-microsegment"
cd "$PROJECT_ROOT"

echo "========================================"
echo "停止 eBPF 微分段系统"
echo "========================================"
echo

# 1. Stop Web UI
echo "[1/4] 停止 Web UI..."
pkill -f "vite" && echo "  ✓ Web UI 已停止" || echo "  Web UI 未运行"
echo

# 2. Stop Agent
echo "[2/4] 停止 Agent 组件..."
sudo pkill -f microsegment-agent && echo "  ✓ Agent 已停止" || echo "  Agent 未运行"
echo

# 3. Stop Server
echo "[3/4] 停止 Server 组件..."
pkill -f microsegment-server && echo "  ✓ Server 已停止" || echo "  Server 未运行"
echo

# 4. Stop PostgreSQL
echo "[4/4] 停止 PostgreSQL 数据库..."
docker-compose -f docker-compose.simple.yml down && echo "  ✓ PostgreSQL 已停止" || echo "  PostgreSQL 未运行"
echo

echo "========================================"
echo "所有服务已停止"
echo "========================================"

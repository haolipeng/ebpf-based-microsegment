#!/bin/bash
# API 测试脚本

set -e

API_BASE="http://localhost:8080/api/v1"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "════════════════════════════════════════════════════════════════"
echo "  eBPF 微隔离 API 测试"
echo "════════════════════════════════════════════════════════════════"
echo

# 检查代理是否运行
check_agent() {
    echo -n "检查代理运行状态... "
    if curl -s -f "${API_BASE}/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ 代理正在运行${NC}"
        return 0
    else
        echo -e "${RED}✗ 代理未运行${NC}"
        echo
        echo "请在另一个终端运行："
        echo "  sudo ./bin/microsegment-agent"
        echo
        exit 1
    fi
}

# 测试 1: 健康检查
test_health() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "测试 1: 健康检查"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    echo -n "GET ${API_BASE}/health ... "
    RESPONSE=$(curl -s "${API_BASE}/health")
    echo -e "${GREEN}✓${NC}"
    echo "响应: $RESPONSE"
    echo
}

# 测试 2: 系统状态
test_status() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "测试 2: 系统状态"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    echo "GET ${API_BASE}/status"
    RESPONSE=$(curl -s "${API_BASE}/status")
    if command -v jq > /dev/null 2>&1; then
        echo "$RESPONSE" | jq .
    else
        echo "$RESPONSE"
    fi
    echo
}

# 测试 3: 列出策略（初始为空）
test_list_policies_empty() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "测试 3: 列出策略（初始）"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    echo "GET ${API_BASE}/policies"
    RESPONSE=$(curl -s "${API_BASE}/policies")
    if command -v jq > /dev/null 2>&1; then
        echo "$RESPONSE" | jq .
    else
        echo "$RESPONSE"
    fi
    echo
}

# 测试 4: 创建策略
test_create_policy() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "测试 4: 创建策略"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    echo "POST ${API_BASE}/policies (允许 SSH)"
    RESPONSE=$(curl -s -X POST "${API_BASE}/policies" \
        -H "Content-Type: application/json" \
        -d '{
            "rule_id": 100,
            "src_ip": "0.0.0.0/0",
            "dst_ip": "0.0.0.0/0",
            "dst_port": 22,
            "protocol": "tcp",
            "action": "allow",
            "priority": 100
        }')
    
    if command -v jq > /dev/null 2>&1; then
        echo "$RESPONSE" | jq .
    else
        echo "$RESPONSE"
    fi
    echo
    
    echo "POST ${API_BASE}/policies (拒绝 HTTPS)"
    RESPONSE=$(curl -s -X POST "${API_BASE}/policies" \
        -H "Content-Type: application/json" \
        -d '{
            "rule_id": 101,
            "src_ip": "0.0.0.0/0",
            "dst_ip": "127.0.0.1",
            "dst_port": 443,
            "protocol": "tcp",
            "action": "deny",
            "priority": 200
        }')
    
    if command -v jq > /dev/null 2>&1; then
        echo "$RESPONSE" | jq .
    else
        echo "$RESPONSE"
    fi
    echo
}

# 测试 5: 列出策略（有数据）
test_list_policies() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "测试 5: 列出策略（有数据）"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    echo "GET ${API_BASE}/policies"
    RESPONSE=$(curl -s "${API_BASE}/policies")
    if command -v jq > /dev/null 2>&1; then
        echo "$RESPONSE" | jq .
    else
        echo "$RESPONSE"
    fi
    echo
}

# 测试 6: 查询特定策略
test_get_policy() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "测试 6: 查询特定策略"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    echo "GET ${API_BASE}/policies/100"
    RESPONSE=$(curl -s "${API_BASE}/policies/100")
    if command -v jq > /dev/null 2>&1; then
        echo "$RESPONSE" | jq .
    else
        echo "$RESPONSE"
    fi
    echo
}

# 测试 7: 生成流量
test_generate_traffic() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "测试 7: 生成流量"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    echo -n "发送 50 个 ICMP 包到 127.0.0.1 ... "
    ping -c 50 -W 1 127.0.0.1 > /dev/null 2>&1 || true
    echo -e "${GREEN}✓${NC}"
    echo
    
    # 等待统计更新
    sleep 1
}

# 测试 8: 统计信息
test_statistics() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "测试 8: 统计信息"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    echo "GET ${API_BASE}/stats"
    RESPONSE=$(curl -s "${API_BASE}/stats")
    if command -v jq > /dev/null 2>&1; then
        echo "$RESPONSE" | jq .
    else
        echo "$RESPONSE"
    fi
    echo
    
    echo "GET ${API_BASE}/stats/packets"
    RESPONSE=$(curl -s "${API_BASE}/stats/packets")
    if command -v jq > /dev/null 2>&1; then
        echo "$RESPONSE" | jq .
    else
        echo "$RESPONSE"
    fi
    echo
}

# 测试 9: 更新策略
test_update_policy() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "测试 9: 更新策略"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    echo "PUT ${API_BASE}/policies/100 (修改为拒绝)"
    RESPONSE=$(curl -s -X PUT "${API_BASE}/policies/100" \
        -H "Content-Type: application/json" \
        -d '{
            "rule_id": 100,
            "src_ip": "0.0.0.0/0",
            "dst_ip": "0.0.0.0/0",
            "dst_port": 22,
            "protocol": "tcp",
            "action": "deny",
            "priority": 100
        }')
    
    if command -v jq > /dev/null 2>&1; then
        echo "$RESPONSE" | jq .
    else
        echo "$RESPONSE"
    fi
    echo
}

# 测试 10: 删除策略
test_delete_policy() {
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "测试 10: 删除策略"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    echo "DELETE ${API_BASE}/policies/101"
    RESPONSE=$(curl -s -w "\nHTTP Status: %{http_code}\n" -X DELETE "${API_BASE}/policies/101")
    echo "$RESPONSE"
    echo
    
    echo "验证删除 - GET ${API_BASE}/policies"
    RESPONSE=$(curl -s "${API_BASE}/policies")
    if command -v jq > /dev/null 2>&1; then
        echo "$RESPONSE" | jq .
    else
        echo "$RESPONSE"
    fi
    echo
}

# 运行所有测试
main() {
    check_agent
    echo
    
    test_health
    test_status
    test_list_policies_empty
    test_create_policy
    test_list_policies
    test_get_policy
    test_generate_traffic
    test_statistics
    test_update_policy
    test_delete_policy
    
    echo "════════════════════════════════════════════════════════════════"
    echo -e "${GREEN}✓ 所有测试完成！${NC}"
    echo "════════════════════════════════════════════════════════════════"
}

main


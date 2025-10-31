#!/bin/bash
# TC 配置清理脚本 - 在遇到 "file exists" 错误时使用

set -e

IFACE="${1:-ens33}"

echo "═══════════════════════════════════════════════"
echo "  清理 TC 配置"
echo "═══════════════════════════════════════════════"
echo

echo "接口: $IFACE"
echo

# 检查接口是否存在
if ! ip link show "$IFACE" > /dev/null 2>&1; then
    echo "❌ 错误: 接口 $IFACE 不存在"
    echo
    echo "可用接口:"
    ip link show | grep '^[0-9]' | awk '{print "  " $2}' | sed 's/:$//'
    exit 1
fi

echo "1. 删除 TC filters..."
if sudo tc filter show dev "$IFACE" ingress 2>/dev/null | grep -q .; then
    sudo tc filter del dev "$IFACE" ingress 2>/dev/null || true
    echo "   ✓ Filters 已删除"
else
    echo "   ℹ 没有 filters 需要删除"
fi

echo
echo "2. 删除 clsact qdisc..."
if sudo tc qdisc show dev "$IFACE" 2>/dev/null | grep -q clsact; then
    sudo tc qdisc del dev "$IFACE" clsact 2>/dev/null || true
    echo "   ✓ Qdisc 已删除"
else
    echo "   ℹ 没有 clsact qdisc 需要删除"
fi

echo
echo "3. 验证清理结果..."
echo "   Qdiscs:"
sudo tc qdisc show dev "$IFACE" | sed 's/^/     /'

echo
echo "   Filters:"
if sudo tc filter show dev "$IFACE" ingress 2>/dev/null | grep -q .; then
    sudo tc filter show dev "$IFACE" ingress | sed 's/^/     /'
else
    echo "     (无)"
fi

echo
echo "═══════════════════════════════════════════════"
echo "✓ 清理完成！"
echo "═══════════════════════════════════════════════"
echo
echo "现在可以重新运行代理："
echo "  sudo ./bin/microsegment-agent -i $IFACE"
echo


#!/bin/bash
# VSCode 调试环境检查脚本

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo "======================================"
echo "VSCode 调试环境检查"
echo "======================================"
echo ""

# 1. 检查 Delve 是否安装
echo -n "检查 Delve 调试器... "
if command -v dlv &> /dev/null; then
    DLV_VERSION=$(dlv version 2>&1 | head -1)
    echo -e "${GREEN}✓${NC} $DLV_VERSION"
else
    echo -e "${RED}✗ 未安装${NC}"
    echo "  安装命令: go install github.com/go-delve/delve/cmd/dlv@latest"
    exit 1
fi

# 2. 检查 Go 环境
echo -n "检查 Go 环境... "
if command -v go &> /dev/null; then
    GO_VERSION=$(go version)
    echo -e "${GREEN}✓${NC} $GO_VERSION"
else
    echo -e "${RED}✗ 未安装${NC}"
    exit 1
fi

# 3. 检查 CGO 支持
echo -n "检查 CGO 支持... "
if go env CGO_ENABLED | grep -q "1"; then
    echo -e "${GREEN}✓ 已启用${NC}"
else
    echo -e "${YELLOW}⚠ 未启用${NC}"
    echo "  运行: export CGO_ENABLED=1"
fi

# 4. 检查配置文件
echo ""
echo "检查配置文件:"
for config in agent.yaml agent-xdp-example.yaml agent-k8s-example.yaml server.yaml; do
    echo -n "  - config/$config... "
    if [ -f "config/$config" ]; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗ 不存在${NC}"
    fi
done

# 5. 检查 eBPF 编译产物
echo ""
echo "检查 eBPF 编译产物:"
echo -n "  - src/agent/pkg/dataplane/bpf_x86_bpfel.o... "
if [ -f "src/agent/pkg/dataplane/bpf_x86_bpfel.o" ]; then
    SIZE=$(du -h src/agent/pkg/dataplane/bpf_x86_bpfel.o | cut -f1)
    echo -e "${GREEN}✓${NC} ($SIZE)"
else
    echo -e "${YELLOW}⚠ 未编译${NC}"
    echo "    运行: make bpf"
fi

echo -n "  - src/agent/pkg/dataplane/xdpbpf_x86_bpfel.o... "
if [ -f "src/agent/pkg/dataplane/xdpbpf_x86_bpfel.o" ]; then
    SIZE=$(du -h src/agent/pkg/dataplane/xdpbpf_x86_bpfel.o | cut -f1)
    echo -e "${GREEN}✓${NC} ($SIZE)"
else
    echo -e "${YELLOW}⚠ 未编译${NC}"
    echo "    运行: make bpf"
fi

# 6. 检查 root 权限
echo ""
echo -n "检查当前用户权限... "
if [ "$EUID" -eq 0 ]; then
    echo -e "${GREEN}✓ root${NC}"
else
    echo -e "${YELLOW}⚠ 非 root${NC}"
    echo "  注意: eBPF 程序需要 root 权限,VSCode 调试时会自动提权"
fi

# 7. 检查网卡
echo ""
echo "检查可用网卡:"
ip link show | grep -E "^[0-9]+:" | awk '{print $2}' | sed 's/:$//' | while read iface; do
    echo "  - $iface"
done

# 8. 检查 VSCode 配置文件
echo ""
echo "检查 VSCode 配置:"
for file in launch.json tasks.json settings.json extensions.json; do
    echo -n "  - .vscode/$file... "
    if [ -f ".vscode/$file" ]; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗ 不存在${NC}"
    fi
done

echo ""
echo "======================================"
echo -e "${GREEN}环境检查完成!${NC}"
echo "======================================"
echo ""
echo "下一步:"
echo "  1. 在 VSCode 中打开项目"
echo "  2. 按 F5 或点击 'Run and Debug'"
echo "  3. 选择调试配置 (如 'Debug Agent (Default)')"
echo "  4. 开始调试!"
echo ""
echo "详细说明请查看: .vscode/DEBUG_GUIDE.md"

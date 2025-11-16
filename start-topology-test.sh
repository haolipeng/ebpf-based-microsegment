#!/bin/bash

# 网络拓扑 MVP 快速验证脚本
# 用途: 一键启动前后端服务并打开浏览器测试

set -e

echo "========================================="
echo "网络拓扑 MVP 快速验证"
echo "========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 步骤 1: 检查环境
echo -e "${YELLOW}[1/5] 检查环境...${NC}"

# 检查 npm
if ! command -v npm &> /dev/null; then
    echo -e "${RED}✗ npm 未安装${NC}"
    exit 1
fi
echo -e "${GREEN}✓ npm 已安装${NC}"

# 检查依赖
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}依赖未安装,正在安装...${NC}"
    npm install
fi
echo -e "${GREEN}✓ 依赖已安装${NC}"

# 检查后端
echo ""
echo -e "${YELLOW}[2/5] 检查后端服务...${NC}"
if curl -s http://localhost:8080/api/v1/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 后端服务运行中${NC}"
else
    echo -e "${YELLOW}⚠ 后端服务未运行${NC}"
    echo "  请先启动后端服务:"
    echo "  - 方式 1: sudo systemctl start microsegment-server"
    echo "  - 方式 2: ./bin/microsegment-server"
    echo ""
    read -p "是否继续?(后端未运行时拓扑图将无数据) [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 步骤 3: 验证文件
echo ""
echo -e "${YELLOW}[3/5] 验证拓扑组件文件...${NC}"

FILES=(
    "src/components/topology/TopologyGraph.tsx"
    "src/pages/Topology/index.tsx"
    "src/types/topology.ts"
    "src/hooks/useTopology.ts"
    "src/styles/topology.css"
)

ALL_EXIST=true
for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} $file"
    else
        echo -e "${RED}✗${NC} $file 缺失"
        ALL_EXIST=false
    fi
done

if [ "$ALL_EXIST" = false ]; then
    echo -e "${RED}错误: 部分文件缺失${NC}"
    exit 1
fi

# 步骤 4: TypeScript 检查
echo ""
echo -e "${YELLOW}[4/5] TypeScript 编译检查...${NC}"
if npx tsc --noEmit 2>&1 | grep -q "error"; then
    echo -e "${RED}✗ TypeScript 编译失败${NC}"
    npx tsc --noEmit | head -20
    exit 1
else
    echo -e "${GREEN}✓ TypeScript 编译通过${NC}"
fi

# 步骤 5: 启动开发服务器
echo ""
echo -e "${YELLOW}[5/5] 启动开发服务器...${NC}"
echo ""
echo "========================================="
echo "  开发服务器即将启动"
echo "========================================="
echo ""
echo "  访问地址: http://localhost:5173/topology"
echo ""
echo "  验证清单:"
echo "  ☐ 页面正常加载"
echo "  ☐ 看到拓扑图或 'No Data'"
echo "  ☐ 可以拖拽节点"
echo "  ☐ 可以缩放画布"
echo "  ☐ Tooltip 正常显示"
echo ""
echo "  按 Ctrl+C 停止服务器"
echo "========================================="
echo ""
sleep 2

# 启动开发服务器
npm run dev

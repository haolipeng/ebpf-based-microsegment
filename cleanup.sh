#!/bin/bash
#
# 项目清理脚本
# 用于清理临时文件、构建产物和可选的大型目录
#
# 使用方法:
#   ./cleanup.sh              # 显示可清理项
#   ./cleanup.sh --temp       # 仅清理临时文件
#   ./cleanup.sh --build      # 清理构建产物
#   ./cleanup.sh --all        # 清理所有（谨慎）
#   ./cleanup.sh --docs       # 列出可能过时的文档

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}\n"
}

# 计算大小
get_size() {
    du -sh "$1" 2>/dev/null | cut -f1
}

# 显示可清理项
show_summary() {
    print_header "项目清理分析"

    echo -e "${YELLOW}1. 临时/日志文件:${NC}"
    find . -type f \( -name "*.log" -o -name "*.tmp" -o -name "*.bak" -o -name "*~" -o -name "*.swp" -o -name "*.out" \) \
           -not -path "./node_modules/*" -not -path "./web/node_modules/*" -not -path "./source-references/*" 2>/dev/null | while read f; do
        echo "  - $f ($(get_size "$f"))"
    done
    echo ""

    echo -e "${YELLOW}2. 构建产物:${NC}"
    [ -d "bin" ] && echo "  - bin/                    $(get_size bin)"
    [ -d "web/dist" ] && echo "  - web/dist/               $(get_size web/dist)"
    [ -d "web/node_modules" ] && echo "  - web/node_modules/       $(get_size web/node_modules)"
    find ./src -name "*.o" 2>/dev/null | head -5 | while read f; do
        echo "  - $f"
    done
    echo ""

    echo -e "${YELLOW}3. 大型参考目录:${NC}"
    [ -d "source-references" ] && echo "  - source-references/      $(get_size source-references)"
    echo ""

    echo -e "${YELLOW}4. 数据目录:${NC}"
    [ -d "data" ] && echo "  - data/                   $(get_size data)"
    [ -d "logs" ] && echo "  - logs/                   $(get_size logs)"
    echo ""

    echo -e "${GREEN}使用 ./cleanup.sh --help 查看清理选项${NC}"
}

# 清理临时文件
clean_temp() {
    print_header "清理临时文件"

    # 日志文件
    echo "删除日志文件..."
    find . -type f -name "*.log" -not -path "./node_modules/*" -not -path "./web/node_modules/*" -not -path "./source-references/*" -delete 2>/dev/null || true

    # 备份文件
    echo "删除备份文件..."
    find . -type f \( -name "*.bak" -o -name "*~" -o -name "*.swp" \) -not -path "./node_modules/*" -not -path "./web/node_modules/*" -not -path "./source-references/*" -delete 2>/dev/null || true

    # 测试输出
    echo "删除测试输出..."
    find . -type f -name "*.out" -name "coverage.*" -not -path "./node_modules/*" -not -path "./web/node_modules/*" -not -path "./source-references/*" -delete 2>/dev/null || true

    # PID 文件
    rm -rf .pids/ 2>/dev/null || true

    echo -e "${GREEN}临时文件清理完成${NC}"
}

# 清理构建产物
clean_build() {
    print_header "清理构建产物"

    read -p "确认清理构建产物? (需要重新编译) [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # 二进制文件
        echo "删除 bin/..."
        rm -rf bin/ 2>/dev/null || true

        # 前端构建
        echo "删除 web/dist/..."
        rm -rf web/dist/ 2>/dev/null || true

        # eBPF 编译产物
        echo "删除 *.o 文件..."
        find ./src -name "*.o" -delete 2>/dev/null || true

        echo -e "${GREEN}构建产物清理完成${NC}"
    else
        echo "已取消"
    fi
}

# 清理 node_modules
clean_node() {
    print_header "清理 node_modules"

    read -p "确认清理 node_modules? (需要重新 npm install) [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "删除 web/node_modules/..."
        rm -rf web/node_modules/ 2>/dev/null || true
        echo -e "${GREEN}node_modules 清理完成${NC}"
    else
        echo "已取消"
    fi
}

# 清理参考代码
clean_references() {
    print_header "清理参考代码"

    echo -e "${YELLOW}警告: source-references/ 包含学习参考代码${NC}"
    read -p "确认删除 source-references/? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf source-references/ 2>/dev/null || true
        echo -e "${GREEN}参考代码清理完成${NC}"
    else
        echo "已取消"
    fi
}

# 列出可能过时的文档
list_old_docs() {
    print_header "可能过时的文档"

    echo -e "${YELLOW}会话/任务记录 (可考虑归档):${NC}"
    ls -la docs/session-*.md docs/task-*.md docs/phase*.md 2>/dev/null | awk '{print "  " $NF " (" $5 " bytes)"}'

    echo ""
    echo -e "${YELLOW}根目录旧文档:${NC}"
    ls -la project_status.md E2E_TESTING_FRAMEWORK.md 2>/dev/null | awk '{print "  " $NF " (" $5 " bytes)"}'

    echo ""
    echo -e "${YELLOW}web/ 规划文档:${NC}"
    ls -la web/DEVELOPMENT_ROADMAP.md web/ROADMAP_VISUAL.md web/GRAPH_DATABASE_IMPLEMENTATION.md 2>/dev/null | awk '{print "  " $NF " (" $5 " bytes)"}'

    echo ""
    echo -e "${BLUE}提示: 这些文档需要人工审核后决定是否删除${NC}"
}

# 清理所有
clean_all() {
    print_header "完整清理"

    echo -e "${RED}警告: 这将清理所有临时文件和构建产物${NC}"
    read -p "确认继续? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        clean_temp
        clean_build
        clean_node
        echo -e "${GREEN}完整清理完成${NC}"
    else
        echo "已取消"
    fi
}

# 帮助
show_help() {
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  (无参数)    显示可清理项目摘要"
    echo "  --temp      清理临时文件 (日志、备份、*.out)"
    echo "  --build     清理构建产物 (bin/, dist/, *.o)"
    echo "  --node      清理 node_modules/"
    echo "  --refs      清理 source-references/"
    echo "  --docs      列出可能过时的文档"
    echo "  --all       清理所有 (临时+构建+node)"
    echo "  --help      显示此帮助"
    echo ""
    echo "示例:"
    echo "  $0                  # 查看摘要"
    echo "  $0 --temp           # 只清理日志等临时文件"
    echo "  $0 --build --node   # 清理构建和 npm 依赖"
}

# 主入口
case "${1:-}" in
    --temp)
        clean_temp
        ;;
    --build)
        clean_build
        ;;
    --node)
        clean_node
        ;;
    --refs)
        clean_references
        ;;
    --docs)
        list_old_docs
        ;;
    --all)
        clean_all
        ;;
    --help|-h)
        show_help
        ;;
    "")
        show_summary
        ;;
    *)
        echo "未知选项: $1"
        show_help
        exit 1
        ;;
esac

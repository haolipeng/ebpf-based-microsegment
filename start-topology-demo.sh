#!/bin/bash
#
# 网络拓扑演示环境一键启动脚本
#
# 功能：
#   1. 启动 Docker 测试容器（生成网络流量）
#   2. 启动 microsegment-server（后端 API）
#   3. 启动 microsegment-agent（流量采集）
#   4. 启动前端开发服务器（拓扑展示）
#
# 使用方法：
#   ./start-topology-demo.sh          # 启动所有服务
#   ./start-topology-demo.sh stop     # 停止所有服务
#   ./start-topology-demo.sh status   # 查看状态
#   ./start-topology-demo.sh logs     # 查看日志

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 配置
DOCKER_TEST_DIR="$SCRIPT_DIR/deploy/docker-test"
WEB_DIR="$SCRIPT_DIR/web"
LOG_DIR="$SCRIPT_DIR/logs"
PID_DIR="$SCRIPT_DIR/.pids"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

print_banner() {
    echo -e "${CYAN}"
    cat << 'EOF'
╔═══════════════════════════════════════════════════════════════╗
║         eBPF Microsegmentation - 网络拓扑演示环境             ║
╠═══════════════════════════════════════════════════════════════╣
║                                                               ║
║   Docker 容器 ──> Agent 采集 ──> Server API ──> Web 展示      ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

print_status() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[OK]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 创建必要目录
init_dirs() {
    mkdir -p "$LOG_DIR" "$PID_DIR"
}

# 检查依赖
check_dependencies() {
    print_status "检查依赖..."

    local missing=()

    # Docker
    if ! command -v docker &> /dev/null; then
        missing+=("docker")
    fi

    # Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        missing+=("docker-compose")
    fi

    # npm (可选，用于前端)
    if ! command -v npm &> /dev/null; then
        print_warning "npm 未安装，将跳过前端启动"
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        print_error "缺少依赖: ${missing[*]}"
        exit 1
    fi

    print_success "依赖检查通过"
}

# 检查二进制文件
check_binaries() {
    print_status "检查二进制文件..."

    if [ ! -f "$SCRIPT_DIR/bin/microsegment-server" ]; then
        print_warning "microsegment-server 未编译"
        print_status "正在编译..."
        make -C "$SCRIPT_DIR" server 2>/dev/null || {
            print_error "编译失败，请先运行 'make server'"
            return 1
        }
    fi

    if [ ! -f "$SCRIPT_DIR/bin/microsegment-agent" ]; then
        print_warning "microsegment-agent 未编译"
        print_status "正在编译..."
        make -C "$SCRIPT_DIR" agent 2>/dev/null || {
            print_error "编译失败，请先运行 'make agent'"
            return 1
        }
    fi

    print_success "二进制文件就绪"
}

# 启动 Docker 测试环境
start_docker() {
    print_status "启动 Docker 测试容器..."

    cd "$DOCKER_TEST_DIR"

    # 使用简化版，无需构建
    if [ -f docker-compose-simple.yml ]; then
        docker-compose -f docker-compose-simple.yml up -d 2>&1 | while read line; do
            echo "  $line"
        done
        print_success "Docker 容器已启动"
    else
        print_error "docker-compose-simple.yml 不存在"
        return 1
    fi

    cd "$SCRIPT_DIR"

    # 等待容器就绪
    print_status "等待容器就绪..."
    sleep 5

    # 验证
    local running=$(docker ps --filter "name=client-" --filter "name=nginx-" -q | wc -l)
    if [ "$running" -gt 0 ]; then
        print_success "$running 个容器正在运行"
    else
        print_warning "容器可能未正常启动，请检查 docker logs"
    fi
}

# 启动 Server
start_server() {
    print_status "启动 microsegment-server..."

    # 检查是否已运行
    if pgrep -f "microsegment-server" > /dev/null; then
        print_warning "Server 已在运行"
        return 0
    fi

    # 检查数据库
    if ! PGPASSWORD=secret psql -h localhost -U microsegment_user -d microsegment -c "SELECT 1" &>/dev/null; then
        print_warning "数据库连接失败，Server 可能无法正常工作"
    fi

    # 启动
    nohup "$SCRIPT_DIR/bin/microsegment-server" -c "$SCRIPT_DIR/config/server.yaml" \
        > "$LOG_DIR/server.log" 2>&1 &
    echo $! > "$PID_DIR/server.pid"

    sleep 2

    # 验证
    if curl -s http://localhost:8080/api/v1/health > /dev/null 2>&1; then
        print_success "Server 启动成功 (PID: $(cat $PID_DIR/server.pid))"
    else
        print_warning "Server 可能未完全启动，请检查日志: $LOG_DIR/server.log"
    fi
}

# 启动 Agent
start_agent() {
    print_status "启动 microsegment-agent..."

    # 需要 root 权限
    if [ "$EUID" -ne 0 ]; then
        print_warning "Agent 需要 root 权限运行"
        print_status "使用 sudo 启动 Agent..."
    fi

    # 检查是否已运行
    if pgrep -f "microsegment-agent" > /dev/null; then
        print_warning "Agent 已在运行"
        return 0
    fi

    # 启动
    sudo nohup "$SCRIPT_DIR/bin/microsegment-agent" -c "$SCRIPT_DIR/config/agent.yaml" \
        > "$LOG_DIR/agent.log" 2>&1 &
    sleep 1
    local agent_pid=$(pgrep -f "microsegment-agent" | head -1)
    if [ -n "$agent_pid" ]; then
        echo "$agent_pid" > "$PID_DIR/agent.pid"
        print_success "Agent 启动成功 (PID: $agent_pid)"
    else
        print_warning "Agent 可能未正常启动，请检查日志: $LOG_DIR/agent.log"
    fi
}

# 启动前端
start_frontend() {
    print_status "启动前端开发服务器..."

    if ! command -v npm &> /dev/null; then
        print_warning "npm 未安装，跳过前端启动"
        return 0
    fi

    cd "$WEB_DIR"

    # 检查依赖
    if [ ! -d "node_modules" ]; then
        print_status "安装前端依赖..."
        npm install 2>&1 | tail -5
    fi

    # 检查是否已运行
    if pgrep -f "vite" > /dev/null; then
        print_warning "前端开发服务器已在运行"
        return 0
    fi

    # 启动
    nohup npm run dev > "$LOG_DIR/frontend.log" 2>&1 &
    echo $! > "$PID_DIR/frontend.pid"

    sleep 3

    if pgrep -f "vite" > /dev/null; then
        print_success "前端启动成功"
    else
        print_warning "前端可能未正常启动，请检查日志: $LOG_DIR/frontend.log"
    fi

    cd "$SCRIPT_DIR"
}

# 停止所有服务
stop_all() {
    print_status "停止所有服务..."

    # 停止前端
    if [ -f "$PID_DIR/frontend.pid" ]; then
        kill $(cat "$PID_DIR/frontend.pid") 2>/dev/null || true
        rm -f "$PID_DIR/frontend.pid"
    fi
    pkill -f "vite" 2>/dev/null || true
    print_success "前端已停止"

    # 停止 Agent
    if [ -f "$PID_DIR/agent.pid" ]; then
        sudo kill $(cat "$PID_DIR/agent.pid") 2>/dev/null || true
        rm -f "$PID_DIR/agent.pid"
    fi
    sudo pkill -f "microsegment-agent" 2>/dev/null || true
    print_success "Agent 已停止"

    # 停止 Server
    if [ -f "$PID_DIR/server.pid" ]; then
        kill $(cat "$PID_DIR/server.pid") 2>/dev/null || true
        rm -f "$PID_DIR/server.pid"
    fi
    pkill -f "microsegment-server" 2>/dev/null || true
    print_success "Server 已停止"

    # 停止 Docker 容器
    cd "$DOCKER_TEST_DIR"
    docker-compose -f docker-compose-simple.yml down 2>/dev/null || true
    print_success "Docker 容器已停止"

    cd "$SCRIPT_DIR"
    print_success "所有服务已停止"
}

# 显示状态
show_status() {
    echo ""
    echo -e "${CYAN}=== 服务状态 ===${NC}"
    echo ""

    # Docker 容器
    echo -e "${BLUE}Docker 容器:${NC}"
    docker ps --filter "name=client-" --filter "name=nginx-" --filter "name=redis" \
              --filter "name=postgres" --filter "name=httpbin" \
              --format "  {{.Names}}: {{.Status}}" 2>/dev/null || echo "  无运行容器"
    echo ""

    # Server
    echo -e "${BLUE}microsegment-server:${NC}"
    if pgrep -f "microsegment-server" > /dev/null; then
        local server_pid=$(pgrep -f "microsegment-server" | head -1)
        echo -e "  ${GREEN}运行中${NC} (PID: $server_pid)"
        if curl -s http://localhost:8080/api/v1/health > /dev/null 2>&1; then
            echo -e "  API: ${GREEN}http://localhost:8080${NC}"
        fi
    else
        echo -e "  ${RED}未运行${NC}"
    fi
    echo ""

    # Agent
    echo -e "${BLUE}microsegment-agent:${NC}"
    if pgrep -f "microsegment-agent" > /dev/null; then
        local agent_pid=$(pgrep -f "microsegment-agent" | head -1)
        echo -e "  ${GREEN}运行中${NC} (PID: $agent_pid)"
    else
        echo -e "  ${RED}未运行${NC}"
    fi
    echo ""

    # Frontend
    echo -e "${BLUE}前端开发服务器:${NC}"
    if pgrep -f "vite" > /dev/null; then
        echo -e "  ${GREEN}运行中${NC}"
        echo -e "  URL: ${GREEN}http://localhost:5173/topology${NC}"
    else
        echo -e "  ${RED}未运行${NC}"
    fi
    echo ""

    # 流量统计
    echo -e "${BLUE}流量采集:${NC}"
    local flow_count=$(curl -s http://localhost:8080/api/v1/flows 2>/dev/null | grep -o '"id"' | wc -l)
    echo "  已采集流量数: $flow_count"
    echo ""
}

# 显示日志
show_logs() {
    local service="${1:-all}"

    case "$service" in
        server)
            tail -f "$LOG_DIR/server.log"
            ;;
        agent)
            tail -f "$LOG_DIR/agent.log"
            ;;
        frontend)
            tail -f "$LOG_DIR/frontend.log"
            ;;
        docker)
            cd "$DOCKER_TEST_DIR"
            docker-compose -f docker-compose-simple.yml logs -f
            ;;
        all)
            echo "使用方式: $0 logs [server|agent|frontend|docker]"
            echo ""
            echo "最近日志:"
            echo ""
            echo "=== Server ===" && tail -5 "$LOG_DIR/server.log" 2>/dev/null || echo "(无日志)"
            echo ""
            echo "=== Agent ===" && tail -5 "$LOG_DIR/agent.log" 2>/dev/null || echo "(无日志)"
            echo ""
            echo "=== Frontend ===" && tail -5 "$LOG_DIR/frontend.log" 2>/dev/null || echo "(无日志)"
            ;;
    esac
}

# 主启动流程
start_all() {
    print_banner
    init_dirs

    echo ""
    print_status "开始启动演示环境..."
    echo ""

    check_dependencies
    check_binaries || true

    echo ""
    echo -e "${CYAN}=== 启动服务 ===${NC}"
    echo ""

    start_docker
    echo ""

    start_server
    echo ""

    start_agent
    echo ""

    start_frontend
    echo ""

    # 等待稳定
    print_status "等待服务稳定..."
    sleep 3

    echo ""
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${GREEN}演示环境启动完成！${NC}"
    echo ""
    echo "  访问地址:"
    echo -e "    ${CYAN}拓扑页面:${NC}  http://localhost:5173/topology"
    echo -e "    ${CYAN}API 接口:${NC}  http://localhost:8080/api/v1/flows"
    echo -e "    ${CYAN}Nginx:${NC}     http://localhost:8880"
    echo ""
    echo "  命令:"
    echo "    ./start-topology-demo.sh status   # 查看状态"
    echo "    ./start-topology-demo.sh logs     # 查看日志"
    echo "    ./start-topology-demo.sh stop     # 停止服务"
    echo ""
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
}

# 使用帮助
show_help() {
    echo "用法: $0 [命令]"
    echo ""
    echo "命令:"
    echo "  start     启动所有服务 (默认)"
    echo "  stop      停止所有服务"
    echo "  restart   重启所有服务"
    echo "  status    查看服务状态"
    echo "  logs      查看日志 [server|agent|frontend|docker]"
    echo "  help      显示帮助"
    echo ""
    echo "示例:"
    echo "  $0              # 启动所有服务"
    echo "  $0 stop         # 停止所有服务"
    echo "  $0 logs server  # 查看 server 日志"
}

# 主入口
case "${1:-start}" in
    start)
        start_all
        ;;
    stop)
        stop_all
        ;;
    restart)
        stop_all
        sleep 2
        start_all
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs "$2"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "未知命令: $1"
        show_help
        exit 1
        ;;
esac

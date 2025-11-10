#!/bin/bash
#
# eBPF Microsegment Systemd 安装脚本
#
set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 root 权限
if [[ $EUID -ne 0 ]]; then
   log_error "必须以 root 用户运行此脚本"
   exit 1
fi

# 检查前置条件
check_prerequisites() {
    log_info "检查前置条件..."

    # 检查 Systemd
    if ! command -v systemctl >/dev/null 2>&1; then
        log_error "Systemd 未安装"
        exit 1
    fi
    log_info "✓ Systemd 已安装"

    # 检查 PostgreSQL
    if ! systemctl is-active --quiet postgresql; then
        log_error "PostgreSQL 服务未运行,请先安装并启动 PostgreSQL"
        log_info "安装方法: sudo apt-get install -y postgresql"
        exit 1
    fi
    log_info "✓ PostgreSQL 正在运行"

    # 检查二进制文件
    if [[ ! -f "bin/microsegment-server" ]]; then
        log_error "Server 二进制文件不存在: bin/microsegment-server"
        exit 1
    fi
    if [[ ! -f "bin/microsegment-agent" ]]; then
        log_error "Agent 二进制文件不存在: bin/microsegment-agent"
        exit 1
    fi
    log_info "✓ 二进制文件已找到"
}

# 创建用户和组
create_user() {
    log_info "创建 microsegment 用户和组..."

    if ! getent group microsegment >/dev/null 2>&1; then
        groupadd --system microsegment
        log_info "✓ 已创建 microsegment 组"
    else
        log_info "✓ microsegment 组已存在"
    fi

    if ! getent passwd microsegment >/dev/null 2>&1; then
        useradd --system --gid microsegment --no-create-home \
            --home-dir /var/lib/microsegment \
            --shell /usr/sbin/nologin \
            --comment "eBPF Microsegment Service User" \
            microsegment
        log_info "✓ 已创建 microsegment 用户"
    else
        log_info "✓ microsegment 用户已存在"
    fi
}

# 安装二进制文件
install_binaries() {
    log_info "安装二进制文件..."

    install -m 0755 -o root -g root \
        bin/microsegment-server /usr/local/bin/
    log_info "✓ 已安装 microsegment-server"

    install -m 0755 -o root -g root \
        bin/microsegment-agent /usr/local/bin/
    log_info "✓ 已安装 microsegment-agent"

    # 安装 eBPF 字节码
    if [[ -f "src/agent/ebpf/flow_tracker_core.o" ]]; then
        mkdir -p /usr/local/lib/ebpf
        install -m 0644 -o root -g root \
            src/agent/ebpf/flow_tracker_core.o /usr/local/lib/ebpf/
        log_info "✓ 已安装 eBPF 字节码"
    else
        log_warn "eBPF 字节码文件不存在,Agent 可能无法正常工作"
    fi
}

# 安装配置文件
install_configs() {
    log_info "安装配置文件..."

    mkdir -p /etc/microsegment

    install -m 0640 -o microsegment -g microsegment \
        deploy/config/systemd/server.yaml /etc/microsegment/
    log_info "✓ 已安装 server.yaml"

    install -m 0640 -o root -g root \
        deploy/config/systemd/agent.yaml /etc/microsegment/
    log_info "✓ 已安装 agent.yaml"

    # 复制环境变量示例文件
    if [[ ! -f "/etc/microsegment/server.env" ]]; then
        install -m 0600 -o root -g root \
            deploy/config/systemd/server.env.example /etc/microsegment/server.env.example
        log_info "✓ 已安装 server.env.example (请根据需要创建 server.env)"
    fi

    if [[ ! -f "/etc/microsegment/agent.env" ]]; then
        install -m 0600 -o root -g root \
            deploy/config/systemd/agent.env.example /etc/microsegment/agent.env.example
        log_info "✓ 已安装 agent.env.example (请根据需要创建 agent.env)"
    fi
}

# 创建数据目录
create_directories() {
    log_info "创建数据目录..."

    mkdir -p /var/lib/microsegment/{server,agent}
    chown -R microsegment:microsegment /var/lib/microsegment/server
    chown -R root:root /var/lib/microsegment/agent
    chmod 0750 /var/lib/microsegment/{server,agent}

    log_info "✓ 已创建数据目录"
}

# 安装 Service 文件
install_services() {
    log_info "安装 Systemd service 文件..."

    install -m 0644 -o root -g root \
        deploy/systemd/microsegment-server.service \
        /etc/systemd/system/
    log_info "✓ 已安装 microsegment-server.service"

    install -m 0644 -o root -g root \
        deploy/systemd/microsegment-agent.service \
        /etc/systemd/system/
    log_info "✓ 已安装 microsegment-agent.service"

    systemctl daemon-reload
    log_info "✓ 已重新加载 systemd daemon"
}

# 启用和启动服务
enable_services() {
    log_info "启用和启动服务..."

    systemctl enable microsegment-server
    log_info "✓ 已启用 microsegment-server"

    systemctl enable microsegment-agent
    log_info "✓ 已启用 microsegment-agent"

    log_info "启动 Server..."
    if systemctl start microsegment-server; then
        log_info "✓ Server 启动成功"
    else
        log_error "Server 启动失败,查看日志: journalctl -u microsegment-server -n 50"
        exit 1
    fi

    sleep 3

    log_info "启动 Agent..."
    if systemctl start microsegment-agent; then
        log_info "✓ Agent 启动成功"
    else
        log_error "Agent 启动失败,查看日志: journalctl -u microsegment-agent -n 50"
        exit 1
    fi
}

# 验证安装
verify_installation() {
    log_info "验证安装..."

    sleep 5

    if systemctl is-active --quiet microsegment-server; then
        log_info "✓ Server 正在运行"
    else
        log_error "Server 未运行"
        journalctl -u microsegment-server -n 50
        exit 1
    fi

    if systemctl is-active --quiet microsegment-agent; then
        log_info "✓ Agent 正在运行"
    else
        log_error "Agent 未运行"
        journalctl -u microsegment-agent -n 50
        exit 1
    fi

    log_info ""
    log_info "========================================="
    log_info "安装成功!"
    log_info "========================================="
    log_info ""
    log_info "服务状态:"
    systemctl status microsegment-server --no-pager -l
    echo ""
    systemctl status microsegment-agent --no-pager -l
    echo ""
    log_info "管理命令:"
    log_info "  查看 Server 状态: systemctl status microsegment-server"
    log_info "  查看 Agent 状态:  systemctl status microsegment-agent"
    log_info "  查看 Server 日志: journalctl -u microsegment-server -f"
    log_info "  查看 Agent 日志:  journalctl -u microsegment-agent -f"
    log_info "  停止服务:        systemctl stop microsegment-agent microsegment-server"
    log_info "  重启服务:        systemctl restart microsegment-server microsegment-agent"
}

# 主函数
main() {
    log_info "========================================="
    log_info "eBPF Microsegment Systemd 安装程序"
    log_info "========================================="
    echo ""

    check_prerequisites
    create_user
    install_binaries
    install_configs
    create_directories
    install_services
    enable_services
    verify_installation
}

main "$@"

#!/bin/bash
#
# eBPF Microsegment Systemd 卸载脚本
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

# 停止服务
stop_services() {
    log_info "停止服务..."

    if systemctl is-active --quiet microsegment-agent; then
        systemctl stop microsegment-agent
        log_info "✓ 已停止 microsegment-agent"
    fi

    if systemctl is-active --quiet microsegment-server; then
        systemctl stop microsegment-server
        log_info "✓ 已停止 microsegment-server"
    fi
}

# 禁用服务
disable_services() {
    log_info "禁用服务..."

    if systemctl is-enabled --quiet microsegment-agent 2>/dev/null; then
        systemctl disable microsegment-agent
        log_info "✓ 已禁用 microsegment-agent"
    fi

    if systemctl is-enabled --quiet microsegment-server 2>/dev/null; then
        systemctl disable microsegment-server
        log_info "✓ 已禁用 microsegment-server"
    fi
}

# 删除 service 文件
remove_services() {
    log_info "删除 service 文件..."

    if [[ -f "/etc/systemd/system/microsegment-server.service" ]]; then
        rm -f /etc/systemd/system/microsegment-server.service
        log_info "✓ 已删除 microsegment-server.service"
    fi

    if [[ -f "/etc/systemd/system/microsegment-agent.service" ]]; then
        rm -f /etc/systemd/system/microsegment-agent.service
        log_info "✓ 已删除 microsegment-agent.service"
    fi

    systemctl daemon-reload
    log_info "✓ 已重新加载 systemd daemon"
}

# 删除二进制文件
remove_binaries() {
    log_info "删除二进制文件..."

    if [[ -f "/usr/local/bin/microsegment-server" ]]; then
        rm -f /usr/local/bin/microsegment-server
        log_info "✓ 已删除 microsegment-server"
    fi

    if [[ -f "/usr/local/bin/microsegment-agent" ]]; then
        rm -f /usr/local/bin/microsegment-agent
        log_info "✓ 已删除 microsegment-agent"
    fi

    if [[ -d "/usr/local/lib/ebpf" ]]; then
        rm -rf /usr/local/lib/ebpf
        log_info "✓ 已删除 eBPF 字节码"
    fi
}

# 删除配置文件
remove_configs() {
    read -p "是否删除配置文件? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "删除配置文件..."
        if [[ -d "/etc/microsegment" ]]; then
            rm -rf /etc/microsegment
            log_info "✓ 已删除配置目录"
        fi
    else
        log_info "保留配置文件"
    fi
}

# 删除数据目录
remove_data() {
    read -p "是否删除数据目录? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "删除数据目录..."
        if [[ -d "/var/lib/microsegment" ]]; then
            rm -rf /var/lib/microsegment
            log_info "✓ 已删除数据目录"
        fi
    else
        log_info "保留数据目录"
    fi
}

# 删除用户和组
remove_user() {
    read -p "是否删除 microsegment 用户和组? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "删除用户和组..."

        if getent passwd microsegment >/dev/null 2>&1; then
            userdel microsegment
            log_info "✓ 已删除 microsegment 用户"
        fi

        if getent group microsegment >/dev/null 2>&1; then
            groupdel microsegment
            log_info "✓ 已删除 microsegment 组"
        fi
    else
        log_info "保留用户和组"
    fi
}

# 主函数
main() {
    log_info "========================================="
    log_info "eBPF Microsegment Systemd 卸载程序"
    log_info "========================================="
    echo ""

    log_warn "此操作将卸载 eBPF Microsegment 服务"
    read -p "是否继续? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "取消卸载"
        exit 0
    fi

    stop_services
    disable_services
    remove_services
    remove_binaries
    remove_configs
    remove_data
    remove_user

    log_info ""
    log_info "========================================="
    log_info "卸载完成!"
    log_info "========================================="
}

main "$@"

# 设计文档: Systemd 服务集成

## 架构概览

本文档描述 eBPF 微分段系统的 Systemd 服务集成架构,涵盖 service 文件设计、依赖管理、安全加固和运维工具。

## 系统组件

### 1. 服务架构

```
┌─────────────────────────────────────────────────────────┐
│                      Systemd                             │
│                                                           │
│  ┌────────────────────────────────────────────────────┐ │
│  │  postgresql.service (系统 PostgreSQL)              │ │
│  └─────────────┬──────────────────────────────────────┘ │
│                │ After/Requires                          │
│                ↓                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │  microsegment-server.service                        │ │
│  │  - User: microsegment (非 root)                     │ │
│  │  - Restart: on-failure                              │ │
│  │  - Security: ProtectSystem, NoNewPrivileges         │ │
│  └─────────────┬──────────────────────────────────────┘ │
│                │ After/Requires                          │
│                ↓                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │  microsegment-agent.service                         │ │
│  │  - User: root (eBPF 需要)                           │ │
│  │  - Capabilities: CAP_SYS_ADMIN, CAP_NET_ADMIN       │ │
│  │  - Restart: on-failure                              │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### 2. 组件职责

#### postgresql.service
- **职责**: 系统 PostgreSQL 数据库服务
- **管理**: 系统包管理器安装
- **状态**: Server 启动的前置条件

#### microsegment-server.service
- **职责**: 控制平面服务
- **用户**: microsegment (uid/gid 由安装脚本创建)
- **依赖**: PostgreSQL
- **端口**: 8080 (HTTP), 9090 (gRPC)
- **配置**: /etc/microsegment/server.yaml
- **日志**: journald (identifier: microsegment-server)

#### microsegment-agent.service
- **职责**: 数据平面,eBPF 程序加载
- **用户**: root (eBPF 加载需要)
- **依赖**: Server
- **配置**: /etc/microsegment/agent.yaml
- **日志**: journald (identifier: microsegment-agent)

## Service 文件详细设计

### 1. Server Service 完整配置

文件路径: `/etc/systemd/system/microsegment-server.service`

```ini
[Unit]
# 服务描述
Description=eBPF Microsegment Control Plane Server
Documentation=https://github.com/xxx/ebpf-based-microsegment

# 依赖关系
After=network-online.target postgresql.service
Wants=network-online.target
Requires=postgresql.service

# 启动条件
ConditionPathExists=/usr/local/bin/microsegment-server
ConditionPathExists=/etc/microsegment/server.yaml

[Service]
# 服务类型
Type=simple

# 运行用户和组
User=microsegment
Group=microsegment

# 工作目录
WorkingDirectory=/var/lib/microsegment

# 启动命令
ExecStart=/usr/local/bin/microsegment-server --config /etc/microsegment/server.yaml

# 启动前检查 (可选)
ExecStartPre=/usr/local/bin/microsegment-server --config /etc/microsegment/server.yaml --validate

# 优雅停止
ExecStop=/bin/kill -SIGTERM $MAINPID
TimeoutStopSec=30s

# 重启策略
Restart=on-failure
RestartSec=5s
StartLimitBurst=5
StartLimitInterval=60s

# 环境变量
Environment="MICROSEGMENT_ENV=production"
EnvironmentFile=-/etc/microsegment/server.env

# 安全加固
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

# 文件系统权限
ReadWritePaths=/var/lib/microsegment
ReadOnlyPaths=/etc/microsegment

# 资源限制
LimitNOFILE=65536
LimitNPROC=4096

# 日志配置
StandardOutput=journal
StandardError=journal
SyslogIdentifier=microsegment-server
SyslogLevel=info

[Install]
WantedBy=multi-user.target
```

**设计说明**:

1. **[Unit] 段**:
   - `After`: 在网络和 PostgreSQL 之后启动
   - `Wants`: 软依赖网络 (网络不可用时仍尝试启动)
   - `Requires`: 硬依赖 PostgreSQL (PostgreSQL 失败则 Server 也失败)
   - `ConditionPathExists`: 检查二进制和配置文件存在

2. **[Service] 段**:
   - `Type=simple`: 前台运行,不 fork
   - `User/Group=microsegment`: 非 root 用户,安全隔离
   - `ExecStartPre`: 启动前验证配置
   - `Restart=on-failure`: 仅在异常退出时重启
   - `RestartSec=5s`: 重启间隔 5 秒
   - `StartLimitBurst/Interval`: 60 秒内最多重启 5 次,防止疯狂重启

3. **安全加固**:
   - `NoNewPrivileges=true`: 防止提权
   - `ProtectSystem=strict`: 只读 /usr, /boot, /efi
   - `ProtectHome=true`: 隐藏 home 目录
   - `PrivateTmp=true`: 私有 /tmp
   - `ProtectKernelTunables/Modules/ControlGroups`: 保护内核资源

4. **资源限制**:
   - `LimitNOFILE`: 最大打开文件数 65536
   - `LimitNPROC`: 最大进程数 4096

### 2. Agent Service 完整配置

文件路径: `/etc/systemd/system/microsegment-agent.service`

```ini
[Unit]
# 服务描述
Description=eBPF Microsegment Data Plane Agent
Documentation=https://github.com/xxx/ebpf-based-microsegment

# 依赖关系
After=network-online.target microsegment-server.service
Wants=network-online.target
Requires=microsegment-server.service

# 启动条件
ConditionPathExists=/usr/local/bin/microsegment-agent
ConditionPathExists=/etc/microsegment/agent.yaml
ConditionPathExists=/sys/fs/bpf

[Service]
# 服务类型
Type=simple

# 运行用户 (eBPF 需要 root)
User=root

# 工作目录
WorkingDirectory=/var/lib/microsegment

# 启动命令
ExecStart=/usr/local/bin/microsegment-agent --config /etc/microsegment/agent.yaml

# 启动前检查
ExecStartPre=/usr/local/bin/microsegment-agent --config /etc/microsegment/agent.yaml --validate
ExecStartPre=/usr/bin/test -d /sys/fs/bpf

# 优雅停止
ExecStop=/bin/kill -SIGTERM $MAINPID
TimeoutStopSec=30s

# 停止后清理 eBPF 程序
ExecStopPost=/usr/local/bin/microsegment-agent --cleanup

# 重启策略
Restart=on-failure
RestartSec=5s
StartLimitBurst=5
StartLimitInterval=60s

# 环境变量
Environment="MICROSEGMENT_ENV=production"
EnvironmentFile=-/etc/microsegment/agent.env

# eBPF 所需权限
AmbientCapabilities=CAP_SYS_ADMIN CAP_NET_ADMIN CAP_BPF CAP_PERFMON CAP_NET_RAW
SecureBits=keep-caps

# 最小化安全限制 (eBPF 需要较多权限)
ProtectHome=true
PrivateTmp=true

# 资源限制
LimitNOFILE=65536
LimitMEMLOCK=infinity

# 日志配置
StandardOutput=journal
StandardError=journal
SyslogIdentifier=microsegment-agent
SyslogLevel=info

[Install]
WantedBy=multi-user.target
```

**设计说明**:

1. **[Unit] 段**:
   - `Requires=microsegment-server.service`: 依赖 Server 运行
   - `ConditionPathExists=/sys/fs/bpf`: 检查 BPF 文件系统挂载

2. **[Service] 段**:
   - `User=root`: eBPF 加载需要 root 权限
   - `ExecStartPre`: 检查配置和 BPF 文件系统
   - `ExecStopPost`: 停止后清理 eBPF 程序

3. **权限配置**:
   - `AmbientCapabilities`: 授予 eBPF 所需的最小 Capabilities:
     - `CAP_SYS_ADMIN`: 加载 eBPF 程序
     - `CAP_NET_ADMIN`: 网络管理
     - `CAP_BPF`: eBPF 操作 (内核 5.8+)
     - `CAP_PERFMON`: 性能监控 (内核 5.8+)
     - `CAP_NET_RAW`: 原始 socket 访问
   - `SecureBits=keep-caps`: 保持 Capabilities

4. **资源限制**:
   - `LimitMEMLOCK=infinity`: eBPF maps 需要锁定内存

## 文件系统布局

### 1. 安装目录结构

```
/usr/local/bin/
├── microsegment-server         # Server 二进制
└── microsegment-agent          # Agent 二进制

/etc/microsegment/
├── server.yaml                 # Server 配置
├── agent.yaml                  # Agent 配置
├── server.env                  # Server 环境变量 (可选)
└── agent.env                   # Agent 环境变量 (可选)

/var/lib/microsegment/
├── server/                     # Server 数据目录
│   └── data/
└── agent/                      # Agent 数据目录
    └── logs/

/var/log/
└── microsegment/               # 日志目录 (如果不使用 journald)
    ├── server.log
    └── agent.log

/etc/systemd/system/
├── microsegment-server.service
└── microsegment-agent.service

/usr/share/doc/microsegment/    # 文档目录
├── README.md
├── LICENSE
└── examples/
```

### 2. 用户和组

安装脚本创建:
```bash
# 创建 microsegment 系统用户和组
sudo groupadd --system microsegment
sudo useradd --system --gid microsegment --no-create-home \
    --home-dir /var/lib/microsegment \
    --shell /usr/sbin/nologin \
    --comment "eBPF Microsegment Service User" \
    microsegment
```

### 3. 文件权限

```bash
# 二进制文件
/usr/local/bin/microsegment-server: root:root 0755
/usr/local/bin/microsegment-agent: root:root 0755

# 配置文件
/etc/microsegment/server.yaml: microsegment:microsegment 0640
/etc/microsegment/agent.yaml: root:root 0640
/etc/microsegment/*.env: root:root 0600 (敏感信息)

# 数据目录
/var/lib/microsegment/server: microsegment:microsegment 0750
/var/lib/microsegment/agent: root:root 0750

# Service 文件
/etc/systemd/system/*.service: root:root 0644
```

## 安装和部署流程

### 1. 安装脚本设计

脚本: `deploy/scripts/install-systemd.sh`

**执行流程**:

```bash
#!/bin/bash
set -e

# 1. 检查权限
if [[ $EUID -ne 0 ]]; then
   echo "必须以 root 运行"
   exit 1
fi

# 2. 检查前置条件
check_prerequisites() {
    # 检查 Systemd
    command -v systemctl >/dev/null || { echo "Systemd 未安装"; exit 1; }

    # 检查 PostgreSQL
    systemctl is-active postgresql >/dev/null || { echo "PostgreSQL 未运行"; exit 1; }
}

# 3. 创建用户和组
create_user() {
    if ! getent group microsegment >/dev/null; then
        groupadd --system microsegment
    fi

    if ! getent passwd microsegment >/dev/null; then
        useradd --system --gid microsegment --no-create-home \
            --home-dir /var/lib/microsegment \
            --shell /usr/sbin/nologin \
            microsegment
    fi
}

# 4. 安装二进制文件
install_binaries() {
    install -m 0755 -o root -g root \
        bin/microsegment-server /usr/local/bin/
    install -m 0755 -o root -g root \
        bin/microsegment-agent /usr/local/bin/
}

# 5. 安装配置文件
install_configs() {
    mkdir -p /etc/microsegment
    install -m 0640 -o microsegment -g microsegment \
        deploy/config/systemd/server.yaml /etc/microsegment/
    install -m 0640 -o root -g root \
        deploy/config/systemd/agent.yaml /etc/microsegment/
}

# 6. 创建数据目录
create_directories() {
    mkdir -p /var/lib/microsegment/{server,agent}
    chown -R microsegment:microsegment /var/lib/microsegment/server
    chown -R root:root /var/lib/microsegment/agent
    chmod 0750 /var/lib/microsegment/{server,agent}
}

# 7. 安装 Service 文件
install_services() {
    install -m 0644 -o root -g root \
        deploy/systemd/microsegment-server.service \
        /etc/systemd/system/
    install -m 0644 -o root -g root \
        deploy/systemd/microsegment-agent.service \
        /etc/systemd/system/

    systemctl daemon-reload
}

# 8. 启用和启动服务
enable_services() {
    systemctl enable microsegment-server
    systemctl enable microsegment-agent

    systemctl start microsegment-server
    systemctl start microsegment-agent
}

# 9. 验证安装
verify_installation() {
    sleep 5  # 等待服务启动

    systemctl is-active --quiet microsegment-server || {
        echo "Server 启动失败"
        journalctl -u microsegment-server -n 50
        exit 1
    }

    systemctl is-active --quiet microsegment-agent || {
        echo "Agent 启动失败"
        journalctl -u microsegment-agent -n 50
        exit 1
    }

    echo "安装成功!"
}

# 执行安装
main() {
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
```

### 2. 卸载脚本设计

脚本: `deploy/scripts/uninstall-systemd.sh`

**执行流程**:
1. 停止服务
2. 禁用服务
3. 删除 service 文件
4. 删除二进制文件
5. 删除配置文件 (可选)
6. 删除数据目录 (可选)
7. 删除用户和组 (可选)

## 日志管理

### 1. Journald 集成

所有日志输出到 systemd journal:

```bash
# 查看 Server 日志
sudo journalctl -u microsegment-server

# 实时查看
sudo journalctl -u microsegment-server -f

# 查看最近 100 行
sudo journalctl -u microsegment-server -n 100

# 查看指定时间范围
sudo journalctl -u microsegment-server --since "2024-01-01" --until "2024-01-02"

# 查看优先级为 error 的日志
sudo journalctl -u microsegment-server -p err

# 组合查看 Server 和 Agent
sudo journalctl -u microsegment-server -u microsegment-agent -f
```

### 2. 日志配置

在 `/etc/systemd/journald.conf` 中配置:

```ini
[Journal]
Storage=persistent
Compress=yes
MaxRetentionSec=7d
MaxFileSec=1d
```

## 健康检查和监控

### 1. 服务状态检查

```bash
# 检查服务状态
systemctl status microsegment-server
systemctl status microsegment-agent

# 检查是否运行
systemctl is-active microsegment-server
systemctl is-active microsegment-agent

# 检查是否启用
systemctl is-enabled microsegment-server
systemctl is-enabled microsegment-agent
```

### 2. 健康检查脚本

脚本: `deploy/scripts/health-check-systemd.sh`

```bash
#!/bin/bash

check_service() {
    local service=$1
    if systemctl is-active --quiet $service; then
        echo "✓ $service is running"
        return 0
    else
        echo "✗ $service is not running"
        return 1
    fi
}

check_api() {
    local url=$1
    if curl -sf $url >/dev/null; then
        echo "✓ API $url is accessible"
        return 0
    else
        echo "✗ API $url is not accessible"
        return 1
    fi
}

# 检查服务
check_service microsegment-server || exit 1
check_service microsegment-agent || exit 1

# 检查 API
check_api http://localhost:8080/health || exit 1

echo "All checks passed!"
```

## 故障排查

### 1. 常见问题

**问题 1: Server 启动失败**
```bash
# 查看详细日志
sudo journalctl -u microsegment-server -xe

# 检查配置文件
sudo /usr/local/bin/microsegment-server --config /etc/microsegment/server.yaml --validate

# 检查 PostgreSQL
sudo systemctl status postgresql
```

**问题 2: Agent 无法加载 eBPF 程序**
```bash
# 检查内核版本
uname -r  # 需要 >= 5.8

# 检查 BPF 文件系统
mount | grep bpf

# 检查 Capabilities
sudo systemctl show microsegment-agent | grep Capabilities
```

**问题 3: 服务频繁重启**
```bash
# 查看重启历史
systemctl status microsegment-server

# 检查启动限制
systemctl show microsegment-server | grep StartLimit
```

## 安全考虑

### 1. 最小权限原则

- **Server**: 非 root 用户,严格的文件系统保护
- **Agent**: Root 用户,但限制 Capabilities

### 2. 安全加固检查清单

- [ ] Server 使用非 root 用户
- [ ] 启用 NoNewPrivileges
- [ ] 启用 ProtectSystem
- [ ] 启用 ProtectHome
- [ ] 启用 PrivateTmp
- [ ] Agent 仅授予必要的 Capabilities
- [ ] 配置文件权限正确 (640/600)
- [ ] 二进制文件权限正确 (755)

## 性能考虑

### 1. 资源限制

```ini
# 内存限制
MemoryMax=1G
MemoryHigh=800M

# CPU 限制
CPUQuota=200%

# IO 限制
IOWeight=500
```

### 2. 优先级

```ini
# 提高 Agent 优先级 (eBPF 实时性要求)
Nice=-5
IOSchedulingClass=realtime
IOSchedulingPriority=0
```

## 总结

本设计提供完整的 Systemd 服务集成方案:
1. **标准化服务**: 符合 Systemd 规范的 service 文件
2. **依赖管理**: 正确的服务依赖关系
3. **安全加固**: 最小权限和安全选项
4. **自动化**: 安装、卸载和健康检查脚本
5. **日志集成**: Journald 集成和查询
6. **故障排查**: 详细的排查指南

适用场景:
- 传统裸机/虚拟机部署
- 企业内部私有云
- 需要系统级服务管理的环境
- 长期运行的生产系统

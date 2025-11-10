# eBPF Microsegment Systemd 部署指南

本指南介绍如何在 Linux 系统上使用 Systemd 部署 eBPF 微分段系统。

## 目录

- [前置要求](#前置要求)
- [快速开始](#快速开始)
- [手工安装](#手工安装)
- [服务管理](#服务管理)
- [配置说明](#配置说明)
- [日志查看](#日志查看)
- [故障排查](#故障排查)
- [安全说明](#安全说明)
- [性能调优](#性能调优)

## 前置要求

### 系统要求

- **操作系统**: Linux 发行版 (推荐 Ubuntu 22.04, CentOS 8, Debian 11)
- **Systemd 版本**: >= 229
- **内核版本**: >= 5.8 (支持 eBPF)
- **权限**: root 用户或 sudo 权限

### 软件依赖

- **PostgreSQL**: >= 12 (必须已安装并运行)
- **Go**: >= 1.21 (仅构建时需要)

检查前置条件:

```bash
# 检查 Systemd 版本
systemctl --version

# 检查内核版本
uname -r

# 检查 PostgreSQL
systemctl status postgresql
```

## 快速开始

### 使用安装脚本 (推荐)

```bash
# 1. 克隆仓库
git clone https://github.com/haolipeng/ebpf-based-microsegment.git
cd ebpf-based-microsegment

# 2. 构建二进制文件
make build

# 3. 运行安装脚本
sudo ./deploy/scripts/install-systemd.sh
```

安装脚本将自动完成以下操作:
- 创建 microsegment 用户和组
- 安装二进制文件到 /usr/local/bin/
- 安装配置文件到 /etc/microsegment/
- 创建数据目录 /var/lib/microsegment/
- 安装 service 文件
- 启用并启动服务

### 验证安装

```bash
# 检查服务状态
systemctl status microsegment-server
systemctl status microsegment-agent

# 运行健康检查
sudo ./deploy/scripts/health-check-systemd.sh
```

## 手工安装

如果不使用安装脚本,可以手工安装:

### 1. 创建用户和组

```bash
sudo groupadd --system microsegment
sudo useradd --system --gid microsegment --no-create-home \
    --home-dir /var/lib/microsegment \
    --shell /usr/sbin/nologin \
    microsegment
```

### 2. 安装二进制文件

```bash
sudo install -m 0755 bin/microsegment-server /usr/local/bin/
sudo install -m 0755 bin/microsegment-agent /usr/local/bin/
sudo mkdir -p /usr/local/lib/ebpf
sudo install -m 0644 src/agent/ebpf/flow_tracker_core.o /usr/local/lib/ebpf/
```

### 3. 安装配置文件

```bash
sudo mkdir -p /etc/microsegment
sudo install -m 0640 -o microsegment -g microsegment \
    deploy/config/systemd/server.yaml /etc/microsegment/
sudo install -m 0640 -o root -g root \
    deploy/config/systemd/agent.yaml /etc/microsegment/
```

### 4. 创建数据目录

```bash
sudo mkdir -p /var/lib/microsegment/{server,agent}
sudo chown -R microsegment:microsegment /var/lib/microsegment/server
sudo chown -R root:root /var/lib/microsegment/agent
sudo chmod 0750 /var/lib/microsegment/{server,agent}
```

### 5. 安装 Service 文件

```bash
sudo cp deploy/systemd/microsegment-server.service /etc/systemd/system/
sudo cp deploy/systemd/microsegment-agent.service /etc/systemd/system/
sudo systemctl daemon-reload
```

### 6. 启用并启动服务

```bash
sudo systemctl enable microsegment-server microsegment-agent
sudo systemctl start microsegment-server
sleep 3
sudo systemctl start microsegment-agent
```

## 服务管理

### 启动服务

```bash
# 启动 Server
sudo systemctl start microsegment-server

# 启动 Agent
sudo systemctl start microsegment-agent
```

**注意**: Agent 依赖 Server,建议先启动 Server,等待几秒后再启动 Agent。

### 停止服务

```bash
# 停止 Agent (先停止 Agent)
sudo systemctl stop microsegment-agent

# 停止 Server
sudo systemctl stop microsegment-server
```

### 重启服务

```bash
# 重启 Server
sudo systemctl restart microsegment-server

# 重启 Agent
sudo systemctl restart microsegment-agent
```

### 查看状态

```bash
# 查看 Server 状态
sudo systemctl status microsegment-server

# 查看 Agent 状态
sudo systemctl status microsegment-agent
```

### 开机自启

```bash
# 启用开机自启
sudo systemctl enable microsegment-server microsegment-agent

# 禁用开机自启
sudo systemctl disable microsegment-server microsegment-agent

# 检查是否已启用
systemctl is-enabled microsegment-server
systemctl is-enabled microsegment-agent
```

### 重新加载配置

修改配置文件后:

```bash
# 重启服务应用新配置
sudo systemctl restart microsegment-server
sudo systemctl restart microsegment-agent
```

## 配置说明

### 配置文件位置

- **Server 配置**: `/etc/microsegment/server.yaml`
- **Agent 配置**: `/etc/microsegment/agent.yaml`
- **环境变量**: `/etc/microsegment/server.env` (可选)
- **环境变量**: `/etc/microsegment/agent.env` (可选)

### 配置优先级

1. 环境变量 (最高优先级)
2. 配置文件
3. 默认值 (代码中定义)

### Server 配置示例

编辑 `/etc/microsegment/server.yaml`:

```yaml
server:
  http:
    host: 0.0.0.0
    port: 8080
  grpc:
    host: 0.0.0.0
    port: 9090

database:
  host: localhost
  port: 5432
  name: microsegment
  user: microsegment_user
  password: your_password_here
  sslmode: require

log:
  level: info
  format: json
```

### Agent 配置示例

编辑 `/etc/microsegment/agent.yaml`:

```yaml
agent:
  api:
    port: 8081
  server:
    url: http://localhost:8080
  ebpf:
    program_path: /usr/local/lib/ebpf/flow_tracker_core.o

log:
  level: info
  format: json
```

### 环境变量使用

创建 `/etc/microsegment/server.env`:

```bash
# 数据库密码 (推荐使用环境变量)
MICROSEGMENT_DB_PASSWORD=your_secure_password

# 日志级别
MICROSEGMENT_LOG_LEVEL=debug
```

修改后重启服务:

```bash
sudo systemctl restart microsegment-server
```

## 日志查看

### 使用 journalctl

```bash
# 查看 Server 日志
sudo journalctl -u microsegment-server

# 查看 Agent 日志
sudo journalctl -u microsegment-agent

# 实时查看日志 (-f)
sudo journalctl -u microsegment-server -f

# 查看最近 100 行
sudo journalctl -u microsegment-server -n 100

# 查看指定时间范围
sudo journalctl -u microsegment-server --since "2024-01-01" --until "2024-01-02"

# 查看 error 级别日志
sudo journalctl -u microsegment-server -p err

# 组合查看 Server 和 Agent
sudo journalctl -u microsegment-server -u microsegment-agent -f
```

### 日志配置

编辑 `/etc/systemd/journald.conf`:

```ini
[Journal]
Storage=persistent
Compress=yes
MaxRetentionSec=7d
MaxFileSec=1d
```

重启 journald:

```bash
sudo systemctl restart systemd-journald
```

## 故障排查

### 常见问题

#### 1. Server 启动失败

**症状**: `systemctl status microsegment-server` 显示失败

**排查步骤**:

```bash
# 1. 查看详细日志
sudo journalctl -u microsegment-server -xe

# 2. 检查配置文件
sudo /usr/local/bin/microsegment-server --config /etc/microsegment/server.yaml --validate

# 3. 检查 PostgreSQL
sudo systemctl status postgresql
sudo -u postgres psql -c "\l"

# 4. 检查端口占用
sudo netstat -tulpn | grep -E '8080|9090'

# 5. 检查文件权限
ls -l /etc/microsegment/server.yaml
ls -l /var/lib/microsegment/server
```

#### 2. Agent 无法加载 eBPF 程序

**症状**: Agent 启动失败,日志显示 eBPF 加载错误

**排查步骤**:

```bash
# 1. 检查内核版本
uname -r  # 需要 >= 5.8

# 2. 检查 BPF 文件系统
mount | grep bpf

# 3. 如果未挂载,手动挂载
sudo mount -t bpf bpf /sys/fs/bpf

# 4. 检查 eBPF 字节码文件
ls -l /usr/local/lib/ebpf/flow_tracker_core.o

# 5. 检查 Agent Capabilities
sudo systemctl show microsegment-agent | grep Capabilities

# 6. 查看详细错误
sudo journalctl -u microsegment-agent -xe
```

#### 3. 服务频繁重启

**症状**: 服务不断重启

**排查步骤**:

```bash
# 1. 查看重启历史
systemctl status microsegment-server

# 2. 检查启动限制
systemctl show microsegment-server | grep StartLimit

# 3. 查看崩溃日志
sudo journalctl -u microsegment-server --since "1 hour ago"

# 4. 临时禁用自动重启调试
sudo systemctl edit microsegment-server
# 添加: [Service]
#       Restart=no
```

#### 4. Agent 无法连接 Server

**症状**: Agent 日志显示连接 Server 失败

**排查步骤**:

```bash
# 1. 检查 Server 是否运行
systemctl is-active microsegment-server

# 2. 测试 Server API
curl http://localhost:8080/health

# 3. 检查防火墙
sudo iptables -L -n | grep 8080

# 4. 检查 Agent 配置
cat /etc/microsegment/agent.yaml | grep server_url
```

## 安全说明

### Server 安全

Server 使用非 root 用户 (`microsegment`) 运行,启用了多项安全加固选项:

- **NoNewPrivileges**: 防止提权
- **ProtectSystem=strict**: 只读系统目录
- **ProtectHome**: 隐藏 home 目录
- **PrivateTmp**: 私有 /tmp 目录
- **ProtectKernelTunables/Modules/ControlGroups**: 保护内核资源

### Agent 安全

Agent 需要 root 权限运行以加载 eBPF 程序,但使用 Capabilities 限制权限:

**所需 Capabilities**:
- **CAP_SYS_ADMIN**: 加载 eBPF 程序
- **CAP_NET_ADMIN**: 网络管理
- **CAP_BPF**: eBPF 操作 (内核 5.8+)
- **CAP_PERFMON**: 性能监控 (内核 5.8+)
- **CAP_NET_RAW**: 原始 socket 访问

**安全风险**: Agent 以 root 身份运行并具有 CAP_SYS_ADMIN 权限,存在一定安全风险。

**缓解措施**:
- 仅在可信环境运行
- 定期更新和审计代码
- 监控 Agent 行为
- 使用最小权限原则

### 文件权限

```
/usr/local/bin/microsegment-server: root:root 0755
/usr/local/bin/microsegment-agent: root:root 0755
/etc/microsegment/server.yaml: microsegment:microsegment 0640
/etc/microsegment/agent.yaml: root:root 0640
/etc/microsegment/*.env: root:root 0600
/var/lib/microsegment/server: microsegment:microsegment 0750
/var/lib/microsegment/agent: root:root 0750
```

## 性能调优

### 资源限制调整

编辑 service 文件:

```bash
sudo systemctl edit microsegment-server
```

添加:

```ini
[Service]
# 内存限制
MemoryMax=2G
MemoryHigh=1.5G

# CPU 限制
CPUQuota=200%

# 文件描述符
LimitNOFILE=131072

# 进程数
LimitNPROC=8192
```

重新加载并重启:

```bash
sudo systemctl daemon-reload
sudo systemctl restart microsegment-server
```

### 优先级调整

对于 Agent (实时性要求高):

```bash
sudo systemctl edit microsegment-agent
```

添加:

```ini
[Service]
Nice=-5
IOSchedulingClass=realtime
IOSchedulingPriority=0
```

### 监控资源使用

```bash
# 查看服务资源使用
systemctl status microsegment-server
systemctl status microsegment-agent

# 详细资源统计
systemd-cgtop

# 查看特定服务
systemd-cgtop microsegment-server.service
```

## 卸载

### 使用卸载脚本

```bash
sudo ./deploy/scripts/uninstall-systemd.sh
```

### 手工卸载

```bash
# 1. 停止并禁用服务
sudo systemctl stop microsegment-agent microsegment-server
sudo systemctl disable microsegment-agent microsegment-server

# 2. 删除 service 文件
sudo rm -f /etc/systemd/system/microsegment-{server,agent}.service
sudo systemctl daemon-reload

# 3. 删除二进制文件
sudo rm -f /usr/local/bin/microsegment-{server,agent}
sudo rm -rf /usr/local/lib/ebpf

# 4. (可选) 删除配置和数据
sudo rm -rf /etc/microsegment
sudo rm -rf /var/lib/microsegment

# 5. (可选) 删除用户和组
sudo userdel microsegment
sudo groupdel microsegment
```

## 附录

### 服务依赖关系

```
PostgreSQL → Server → Agent
```

- Agent 依赖 Server
- Server 依赖 PostgreSQL

### 文件清单

```
/usr/local/bin/microsegment-server          # Server 二进制
/usr/local/bin/microsegment-agent           # Agent 二进制
/usr/local/lib/ebpf/flow_tracker_core.o     # eBPF 字节码
/etc/microsegment/server.yaml               # Server 配置
/etc/microsegment/agent.yaml                # Agent 配置
/etc/microsegment/server.env                # Server 环境变量
/etc/microsegment/agent.env                 # Agent 环境变量
/etc/systemd/system/microsegment-server.service  # Server service
/etc/systemd/system/microsegment-agent.service   # Agent service
/var/lib/microsegment/server/               # Server 数据目录
/var/lib/microsegment/agent/                # Agent 数据目录
```

### 相关命令参考

```bash
# Systemd 命令
systemctl start/stop/restart/status <service>
systemctl enable/disable <service>
systemctl is-active/is-enabled <service>
systemctl daemon-reload
systemctl list-units --type=service

# 日志命令
journalctl -u <service>
journalctl -u <service> -f
journalctl -u <service> -n <lines>
journalctl -u <service> --since "<time>"
journalctl -u <service> -p <priority>

# 调试命令
systemd-analyze verify <service>
systemd-analyze critical-chain <service>
systemd-cgtop
```

## 支持

如有问题,请:
1. 查看本文档的故障排查章节
2. 查看日志: `journalctl -u microsegment-server -u microsegment-agent -n 100`
3. 提交 Issue: https://github.com/haolipeng/ebpf-based-microsegment.git/issues

# 提案: 添加 Systemd 服务集成

## 概述

本提案旨在为 eBPF 微分段系统添加 Systemd 服务集成支持,包括生产级别的 service 文件、部署脚本和相关文档,以支持在传统 Linux 基础设施上作为系统服务运行。

## Why

当前 eBPF 微分段系统的 Server 和 Agent 组件已经实现并可运行,但缺少 Systemd 系统服务集成,导致:

**问题 1: 传统环境部署困难**
- 没有标准的 systemd service 文件
- 无法使用 systemctl 管理服务生命周期
- 系统启动时无法自动启动服务
- 缺少服务依赖管理 (Server 依赖 PostgreSQL, Agent 依赖 Server)

**问题 2: 运维管理不便**
- 无法使用系统标准工具 (systemctl, journalctl)
- 服务崩溃后无自动重启机制
- 日志管理混乱,未集成到 journald
- 资源限制和安全加固配置缺失

**问题 3: 企业环境兼容性差**
- 很多企业仍使用传统的裸机或虚拟机部署
- 不支持 Systemd 导致运维团队无法接受
- 缺少符合企业标准的部署方式
- 难以集成到现有的运维流程和监控系统

**业务影响**:
- 阻碍在传统基础设施环境的推广
- 增加运维团队的学习和管理成本
- 降低系统可靠性(缺少自动重启和监控)
- 难以满足企业级部署要求

本提案通过提供完整的 Systemd 服务集成,解决上述问题,使系统能够在传统 Linux 环境中稳定运行。

## 动机

当前项目已经具备:
- 可运行的 Server 和 Agent 二进制文件
- 基本的启动脚本 (`start-all.sh`, `stop-all.sh`)
- 配置文件支持 (YAML 格式)

但是缺少:
1. **Systemd Service 文件**: Server 和 Agent 的标准 systemd unit 文件
2. **服务依赖管理**: Server 依赖 PostgreSQL, Agent 依赖 Server 的依赖关系定义
3. **安全加固配置**: NoNewPrivileges, ProtectSystem, PrivateTmp 等安全选项
4. **自动重启机制**: 服务崩溃后的自动恢复配置
5. **部署工具**: 安装、卸载和管理 systemd 服务的脚本
6. **部署文档**: 详细的 systemd 部署指南

这些缺失导致:
- 无法在传统 Linux 环境标准化部署
- 服务管理依赖手工脚本,易出错
- 缺少系统级的监控和日志集成
- 安全性配置不足

## 目标

### 主要目标

1. **创建 Server Systemd Service**
   - 标准的 systemd unit 文件
   - 依赖 PostgreSQL (After/Wants/Requires)
   - 非 root 用户运行
   - 自动重启配置 (Restart=on-failure)
   - 安全加固选项 (NoNewPrivileges, ProtectSystem 等)
   - 日志集成到 journald

2. **创建 Agent Systemd Service**
   - 标准的 systemd unit 文件
   - 依赖 Server (After/Requires)
   - Root 用户运行 (eBPF 需要)
   - Capabilities 配置 (CAP_SYS_ADMIN, CAP_NET_ADMIN)
   - 自动重启配置
   - 日志集成到 journald

3. **部署自动化**
   - Systemd 服务安装脚本
   - 服务卸载脚本
   - 健康检查脚本
   - 日志查看脚本

4. **文档和测试**
   - Systemd 部署文档
   - 服务管理指南
   - 故障排查指南
   - 部署流程测试验证

### 非目标

- Docker 容器化配置 (在单独的提案中处理)
- Kubernetes 部署配置 (在单独的提案中处理)
- 其他 init 系统支持 (SysVinit, Upstart)
- 完整的系统监控配置 (应在单独的提案中处理)

## 设计概要

### 1. Server Service 文件设计

```ini
[Unit]
Description=eBPF Microsegment Server
Documentation=https://github.com/xxx/ebpf-based-microsegment
After=network-online.target postgresql.service
Wants=network-online.target
Requires=postgresql.service

[Service]
Type=simple
User=microsegment
Group=microsegment
ExecStart=/usr/local/bin/microsegment-server --config /etc/microsegment/server.yaml
Restart=on-failure
RestartSec=5s

# 安全加固
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/lib/microsegment

# 日志
StandardOutput=journal
StandardError=journal
SyslogIdentifier=microsegment-server

[Install]
WantedBy=multi-user.target
```

**关键设计点**:
- **依赖管理**: After/Requires postgresql.service 确保数据库先启动
- **非 root 用户**: 使用专门的 microsegment 用户运行
- **自动重启**: Restart=on-failure, RestartSec=5s
- **安全加固**: NoNewPrivileges, ProtectSystem, ProtectHome 等
- **日志集成**: 输出到 systemd journal

### 2. Agent Service 文件设计

```ini
[Unit]
Description=eBPF Microsegment Agent
Documentation=https://github.com/xxx/ebpf-based-microsegment
After=network-online.target microsegment-server.service
Wants=network-online.target
Requires=microsegment-server.service

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/microsegment-agent --config /etc/microsegment/agent.yaml
Restart=on-failure
RestartSec=5s

# eBPF 所需权限
AmbientCapabilities=CAP_SYS_ADMIN CAP_NET_ADMIN CAP_BPF CAP_PERFMON
SecureBits=keep-caps

# 日志
StandardOutput=journal
StandardError=journal
SyslogIdentifier=microsegment-agent

[Install]
WantedBy=multi-user.target
```

**关键设计点**:
- **依赖管理**: Requires microsegment-server.service 确保 Server 先启动
- **Root 运行**: eBPF 加载需要 root 权限
- **Capabilities**: 明确授予 CAP_SYS_ADMIN, CAP_NET_ADMIN 等
- **自动重启**: 与 Server 相同的重启策略

### 3. 目录结构

```
deploy/systemd/
├── microsegment-server.service
├── microsegment-agent.service
└── README.md

deploy/scripts/
├── install-systemd.sh
├── uninstall-systemd.sh
└── health-check-systemd.sh

deploy/config/systemd/
├── server.yaml
└── agent.yaml
```

### 4. 部署流程

**安装步骤**:
1. 创建 microsegment 用户和组
2. 复制二进制文件到 /usr/local/bin/
3. 复制配置文件到 /etc/microsegment/
4. 复制 service 文件到 /etc/systemd/system/
5. 重新加载 systemd daemon
6. 启用并启动服务
7. 验证服务状态

**管理命令**:
```bash
# 启动服务
sudo systemctl start microsegment-server
sudo systemctl start microsegment-agent

# 停止服务
sudo systemctl stop microsegment-agent
sudo systemctl stop microsegment-server

# 查看状态
sudo systemctl status microsegment-server
sudo systemctl status microsegment-agent

# 查看日志
sudo journalctl -u microsegment-server -f
sudo journalctl -u microsegment-agent -f

# 开机自启
sudo systemctl enable microsegment-server
sudo systemctl enable microsegment-agent
```

## 影响的规范

本提案将添加新的 Systemd 部署规范:

1. **新规范 systemd-deployment**: 定义 Systemd 服务集成要求

## 依赖关系

### 前置条件
- Server 和 Agent 组件已实现并可运行
- 配置管理已实现 (config.go)
- 目标系统使用 Systemd (大多数现代 Linux 发行版)

### 阻塞项
- 无

## 成功标准

1. **功能完整性**
   - ✅ 可通过 systemctl 管理 Server 和 Agent 服务
   - ✅ 服务依赖关系正确 (启动顺序: PostgreSQL → Server → Agent)
   - ✅ 服务自动重启机制工作正常

2. **可用性**
   - ✅ 部署文档清晰,运维人员可按指南完成部署
   - ✅ 提供安装和卸载脚本
   - ✅ 日志通过 journalctl 查看方便

3. **安全性**
   - ✅ Server 使用非 root 用户运行
   - ✅ Agent 仅授予必要的 Capabilities
   - ✅ 启用安全加固选项 (ProtectSystem, NoNewPrivileges 等)

4. **可靠性**
   - ✅ 服务崩溃后自动重启
   - ✅ 系统启动时服务自动启动
   - ✅ 依赖服务失败时正确处理

## 风险与缓解

### 风险 1: Agent 需要 root 权限
- **描述**: Agent 需要 root 权限加载 eBPF 程序,存在安全风险
- **缓解**:
  - 使用 AmbientCapabilities 精确授予所需权限
  - 避免不必要的特权
  - 文档中明确说明安全风险和最佳实践

### 风险 2: 不同发行版的 Systemd 版本差异
- **描述**: 不同 Linux 发行版的 Systemd 版本和配置可能有差异
- **缓解**:
  - 使用 Systemd 的通用特性,避免新版本特有功能
  - 在多个主流发行版上测试 (Ubuntu, CentOS, Debian)
  - 文档中说明兼容的 Systemd 版本要求

### 风险 3: 服务依赖配置错误
- **描述**: 服务依赖关系配置错误可能导致启动失败
- **缓解**:
  - 使用 After + Requires 明确依赖关系
  - 实现服务健康检查和重试机制
  - 提供详细的故障排查文档

## 后续工作

本提案完成后,可以考虑:

1. **监控集成**: 集成到 Prometheus Node Exporter
2. **日志轮转**: 配置 journald 日志轮转策略
3. **性能调优**: 优化 systemd 配置以提高性能
4. **多实例支持**: 支持在同一主机运行多个 Agent 实例

## 时间估算

- **Service 文件创建**: 0.5 天
- **部署脚本**: 1 天
- **文档编写**: 0.5 天
- **测试验证**: 1 天

**总计**: 约 3 天

## 参考

- [Systemd Service 文件规范](https://www.freedesktop.org/software/systemd/man/systemd.service.html)
- [Systemd 安全加固](https://www.freedesktop.org/software/systemd/man/systemd.exec.html#Security)
- [Linux Capabilities](https://man7.org/linux/man-pages/man7/capabilities.7.html)

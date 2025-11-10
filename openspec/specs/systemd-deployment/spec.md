# systemd-deployment Specification

## Purpose
TBD - created by archiving change add-systemd-deployment. Update Purpose after archive.
## Requirements
### Requirement: Server Systemd Service 规范

系统必须(SHALL)提供 Server 组件的 Systemd service 文件,支持系统服务管理。

#### Scenario: Server Service 安装和启动

**Given** Server 二进制文件已安装到 /usr/local/bin/
**And** Service 文件已安装到 /etc/systemd/system/
**When** 执行 `systemctl start microsegment-server` 命令
**Then** 服务必须(SHALL)成功启动
**And** 服务必须(SHALL)监听 8080 端口
**And** 服务必须(SHALL)监听 9090 端口
**And** 服务必须(SHALL)以非 root 用户运行

#### Scenario: Server Service 依赖管理

**Given** PostgreSQL 服务未启动
**When** 尝试启动 Server 服务
**Then** Server 服务必须(SHALL)等待 PostgreSQL 启动
**And** 如果 PostgreSQL 启动失败,Server 必须(SHALL)也失败

#### Scenario: Server Service 自动重启

**Given** Server 服务正在运行
**When** Server 进程异常退出
**Then** Systemd 必须(SHALL)在 5 秒后自动重启服务
**And** 如果 60 秒内重启超过 5 次,必须(SHALL)停止自动重启

### Requirement: Agent Systemd Service 规范

系统必须(SHALL)提供 Agent 组件的 Systemd service 文件,支持 eBPF 程序加载。

#### Scenario: Agent Service 启动

**Given** Agent 二进制文件已安装
**And** Server 服务正在运行
**When** 执行 `systemctl start microsegment-agent` 命令
**Then** 服务必须(SHALL)成功启动
**And** 服务必须(SHALL)以 root 用户运行
**And** 服务必须(SHALL)具有 CAP_SYS_ADMIN capability
**And** 服务必须(SHALL)具有 CAP_NET_ADMIN capability
**And** eBPF 程序必须(SHALL)成功加载

#### Scenario: Agent Service 依赖管理

**Given** Server 服务未启动
**When** 尝试启动 Agent 服务
**Then** Agent 服务必须(SHALL)等待 Server 启动
**And** 如果 Server 启动失败,Agent 必须(SHALL)也失败

### Requirement: 安全加固配置

系统必须(SHALL)在 Systemd service 文件中配置安全加固选项。

#### Scenario: Server 安全加固

**Given** Server service 文件已配置
**Then** 服务必须(SHALL)配置 NoNewPrivileges=true
**And** 服务必须(SHALL)配置 ProtectSystem=strict
**And** 服务必须(SHALL)配置 ProtectHome=true
**And** 服务必须(SHALL)配置 PrivateTmp=true
**And** 服务必须(SHALL)配置 ProtectKernelTunables=true
**And** 服务必须(SHALL)配置 ReadWritePaths 限制可写路径

#### Scenario: Agent 最小权限

**Given** Agent 需要 root 权限运行
**Then** 服务必须(SHALL)使用 AmbientCapabilities 授予权限
**And** 必须(SHALL)仅授予必要的 Capabilities
**And** 必须(SHALL)配置 SecureBits=keep-caps

### Requirement: 日志集成

系统必须(SHALL)将服务日志输出到 systemd journal。

#### Scenario: 日志输出到 journald

**Given** Server 或 Agent 服务正在运行
**When** 服务输出日志
**Then** 日志必须(SHALL)发送到 systemd journal
**And** 日志必须(SHALL)包含 SyslogIdentifier (microsegment-server 或 microsegment-agent)
**And** 日志必须(SHALL)可通过 `journalctl -u <service>` 查看

#### Scenario: 日志级别控制

**Given** 服务配置了日志级别
**When** 查看 journald 日志
**Then** 日志必须(SHALL)包含正确的优先级标记
**And** 必须(SHALL)支持通过 `journalctl -p` 过滤日志级别

### Requirement: 配置管理

系统必须(SHALL)提供 Systemd 环境的配置文件模板。

#### Scenario: 配置文件加载

**Given** 配置文件位于 /etc/microsegment/
**When** 服务启动
**Then** 服务必须(SHALL)加载配置文件
**And** 配置文件必须(SHALL)使用系统标准路径
**And** 配置文件必须(SHALL)具有正确的权限 (640 for server, 640 for agent)

#### Scenario: 环境变量支持

**Given** EnvironmentFile 配置了环境变量
**When** 服务启动
**Then** 环境变量必须(SHALL)覆盖配置文件中的值
**And** 敏感信息必须(SHALL)通过环境变量提供

### Requirement: 安装自动化

系统必须(SHALL)提供 Systemd 服务安装脚本。

#### Scenario: 自动化安装

**Given** 存在 install-systemd.sh 脚本
**When** 以 root 用户执行该脚本
**Then** 脚本必须(SHALL)检查前置条件 (Systemd, PostgreSQL)
**And** 脚本必须(SHALL)创建 microsegment 用户和组
**And** 脚本必须(SHALL)安装二进制文件到 /usr/local/bin/
**And** 脚本必须(SHALL)安装配置文件到 /etc/microsegment/
**And** 脚本必须(SHALL)创建数据目录 /var/lib/microsegment/
**And** 脚本必须(SHALL)安装 service 文件到 /etc/systemd/system/
**And** 脚本必须(SHALL)执行 systemctl daemon-reload
**And** 脚本必须(SHALL)启用并启动服务
**And** 脚本必须(SHALL)验证安装成功
**And** 脚本失败时必须(SHALL)返回非零退出码

### Requirement: 卸载工具

系统必须(SHALL)提供 Systemd 服务卸载脚本。

#### Scenario: 自动化卸载

**Given** 服务已通过脚本安装
**When** 以 root 用户执行卸载脚本
**Then** 脚本必须(SHALL)停止所有服务
**And** 脚本必须(SHALL)禁用服务
**And** 脚本必须(SHALL)删除 service 文件
**And** 脚本必须(SHALL)删除二进制文件
**And** 脚本必须(SHALL)执行 systemctl daemon-reload
**And** 脚本必须(SHALL)支持可选删除配置和数据 (带确认)

### Requirement: 健康检查

系统必须(SHALL)提供健康检查脚本验证 Systemd 服务状态。

#### Scenario: 服务健康检查

**Given** 服务已通过 Systemd 部署
**When** 执行健康检查脚本
**Then** 脚本必须(SHALL)检查 Server 服务运行状态
**And** 脚本必须(SHALL)检查 Agent 服务运行状态
**And** 脚本必须(SHALL)检查 PostgreSQL 服务运行状态
**And** 脚本必须(SHALL)检查 Server API 可用性
**And** 脚本必须(SHALL)检查 Agent API 可用性
**And** 所有检查通过时必须(SHALL)返回退出码 0
**And** 任何检查失败时必须(SHALL)返回退出码 1

### Requirement: 文件权限

系统必须(SHALL)设置正确的文件和目录权限。

#### Scenario: 文件权限检查

**Given** 服务已安装
**Then** 二进制文件必须(SHALL)属于 root:root 权限 755
**And** Server 配置文件必须(SHALL)属于 microsegment:microsegment 权限 640
**And** Agent 配置文件必须(SHALL)属于 root:root 权限 640
**And** Server 数据目录必须(SHALL)属于 microsegment:microsegment 权限 750
**And** Agent 数据目录必须(SHALL)属于 root:root 权限 750
**And** Service 文件必须(SHALL)属于 root:root 权限 644

### Requirement: 服务管理

系统必须(SHALL)支持标准的 Systemd 服务管理命令。

#### Scenario: 服务启动和停止

**Given** 服务已安装
**When** 执行 systemctl start/stop/restart 命令
**Then** 服务必须(SHALL)正确响应
**And** Agent 停止必须(SHALL)在 Server 之前
**And** Server 启动必须(SHALL)在 Agent 之前

#### Scenario: 开机自启

**Given** 服务已启用 (systemctl enable)
**When** 系统重启
**Then** 服务必须(SHALL)自动启动
**And** 启动顺序必须(SHALL)遵循依赖关系

### Requirement: 部署文档

系统必须(SHALL)提供详细的 Systemd 部署文档。

#### Scenario: Systemd 部署指南

**Given** 存在 deploy/systemd/README.md 文档
**Then** 文档必须(SHALL)包含前置要求
**And** 文档必须(SHALL)包含快速开始步骤
**And** 文档必须(SHALL)包含服务管理命令
**And** 文档必须(SHALL)包含配置说明
**And** 文档必须(SHALL)包含日志查看方法
**And** 文档必须(SHALL)包含安全说明
**And** 文档必须(SHALL)包含故障排查指南
**And** 文档必须(SHALL)包含性能调优建议


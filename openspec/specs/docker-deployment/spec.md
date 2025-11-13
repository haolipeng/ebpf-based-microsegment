# docker-deployment Specification

## Purpose
TBD - created by archiving change add-docker-deployment. Update Purpose after archive.
## Requirements
### Requirement: Server Dockerfile 规范

系统必须(SHALL)提供 Server 组件的 Dockerfile,支持生产级别的容器化部署。

#### Scenario: Server 镜像构建

**Given** Server 源代码位于 src/server/ 目录
**When** 执行 `docker build -f deploy/docker/Dockerfile.server` 命令
**Then** 构建必须(SHALL)成功完成
**And** 镜像大小必须(SHALL) < 50MB
**And** 镜像必须(SHALL)使用非 root 用户运行
**And** 镜像必须(SHALL)暴露端口 8080 和 9090
**And** 镜像必须(SHALL)包含健康检查指令

#### Scenario: Server 容器运行

**Given** Server 镜像已构建
**When** 使用 docker run 启动容器
**Then** 容器必须(SHALL)成功启动
**And** 必须(SHALL)监听 8080 端口
**And** 必须(SHALL)监听 9090 端口
**And** 健康检查端点 /health 必须(SHALL)返回 200 状态码

### Requirement: Agent Dockerfile 规范

系统必须(SHALL)提供 Agent 组件的 Dockerfile,支持 eBPF 程序加载。

#### Scenario: Agent 镜像构建

**Given** Agent 源代码位于 src/agent/ 目录
**When** 执行 `docker build -f deploy/docker/Dockerfile.agent` 命令
**Then** 构建必须(SHALL)成功完成
**And** 镜像大小必须(SHALL) < 200MB
**And** 镜像必须(SHALL)包含 libbpf 和 bpftool 工具
**And** 镜像必须(SHALL)包含编译后的 eBPF 字节码文件

#### Scenario: Agent 容器运行

**Given** Agent 镜像已构建
**And** 容器以特权模式运行 (privileged: true)
**And** 容器使用 host 网络模式
**When** 使用 docker run 启动容器
**Then** 容器必须(SHALL)成功启动
**And** 必须(SHALL)成功加载 eBPF 程序
**And** 必须(SHALL)可以访问 /sys/fs/bpf

### Requirement: Docker Compose 编排

系统必须(SHALL)提供 Docker Compose 配置文件,编排所有组件。

#### Scenario: Docker Compose 完整部署

**Given** 存在 docker-compose.yml 文件
**When** 执行 `docker-compose up -d` 命令
**Then** 必须(SHALL)启动 postgres, server, agent 服务
**And** 所有服务必须(SHALL)通过健康检查
**And** Server 必须(SHALL)在 PostgreSQL 健康后启动
**And** Agent 必须(SHALL)在 Server 健康后启动
**And** Agent 容器必须(SHALL)配置 privileged: true
**And** Agent 容器必须(SHALL)配置 network_mode: host
**And** Agent 容器必须(SHALL)配置 pid: host

#### Scenario: 数据持久化

**Given** Docker Compose 已启动
**When** 向系统写入数据
**And** 执行 `docker-compose restart` 重启服务
**Then** 数据必须(SHALL)保留
**And** PostgreSQL 数据必须(SHALL)存储在 named volume 中

### Requirement: 开发环境支持

系统必须(SHALL)提供开发环境的 Docker Compose override 配置。

#### Scenario: 开发环境启动

**Given** 存在 docker-compose.dev.yml 文件
**When** 执行 `docker-compose -f docker-compose.yml -f docker-compose.dev.yml up` 命令
**Then** 必须(SHALL)启动开发环境
**And** 源代码必须(SHALL)挂载到容器中
**And** 日志级别必须(SHALL)设置为 debug
**And** 代码修改必须(SHALL)触发服务重载

### Requirement: 构建优化

系统必须(SHALL)提供 .dockerignore 文件优化构建上下文。

#### Scenario: 构建上下文优化

**Given** 存在 .dockerignore 文件
**When** 执行 docker build 命令
**Then** 构建上下文必须(SHALL)排除 .git 目录
**And** 构建上下文必须(SHALL)排除测试文件
**And** 构建上下文必须(SHALL)排除文档文件
**And** 构建时间必须(SHALL)相比无 .dockerignore 减少 > 30%

### Requirement: 配置管理

系统必须(SHALL)提供 Docker 环境的配置文件模板。

#### Scenario: 环境变量配置

**Given** Docker Compose 配置了环境变量
**When** 启动服务
**Then** 环境变量必须(SHALL)覆盖配置文件中的值
**And** 敏感信息(如数据库密码)必须(SHALL)通过环境变量提供

#### Scenario: 配置文件加载

**Given** 存在 deploy/config/docker/server.yaml
**When** Server 容器启动
**Then** 配置文件必须(SHALL)被正确加载
**And** 配置中的环境变量引用必须(SHALL)被解析

### Requirement: 部署自动化

系统必须(SHALL)提供 Docker 自动化部署脚本。

#### Scenario: 一键部署

**Given** 存在 deploy/scripts/deploy-docker.sh 脚本
**When** 执行该脚本
**Then** 脚本必须(SHALL)检查 Docker 环境
**And** 脚本必须(SHALL)构建所有镜像
**And** 脚本必须(SHALL)启动所有服务
**And** 脚本必须(SHALL)运行健康检查
**And** 脚本失败时必须(SHALL)返回非零退出码

### Requirement: 健康检查

系统必须(SHALL)提供健康检查脚本验证 Docker 部署状态。

#### Scenario: 健康检查脚本

**Given** 系统已通过 Docker Compose 部署
**When** 执行健康检查脚本
**Then** 脚本必须(SHALL)检查所有容器运行状态
**And** 脚本必须(SHALL)检查 PostgreSQL 连接
**And** 脚本必须(SHALL)检查 Server API 可用性
**And** 脚本必须(SHALL)检查 Agent eBPF 程序加载状态
**And** 所有检查通过时必须(SHALL)返回退出码 0
**And** 任何检查失败时必须(SHALL)返回退出码 1

### Requirement: 安全配置

系统必须(SHALL)遵循 Docker 安全最佳实践。

#### Scenario: Server 镜像安全

**Given** Server 镜像已构建
**When** 使用 Trivy 扫描镜像漏洞
**Then** 必须(SHALL)无高危漏洞
**And** Server 容器必须(SHALL)使用非 root 用户运行
**And** Server 容器必须(SHALL)不包含构建工具

#### Scenario: Agent 特权最小化

**Given** Agent 需要特权运行
**Then** 文档必须(SHALL)明确说明所需特权及原因
**And** 文档必须(SHALL)说明安全风险
**And** 部署配置必须(SHALL)仅授予必要的最小特权

### Requirement: 部署文档

系统必须(SHALL)提供详细的 Docker 部署文档。

#### Scenario: Docker 部署指南

**Given** 存在 deploy/docker/README.md 文档
**Then** 文档必须(SHALL)包含前置要求
**And** 文档必须(SHALL)包含快速开始步骤
**And** 文档必须(SHALL)包含配置说明
**And** 文档必须(SHALL)包含服务管理命令
**And** 文档必须(SHALL)包含日志查看方法
**And** 文档必须(SHALL)包含常见问题排查
**And** 文档必须(SHALL)包含安全说明


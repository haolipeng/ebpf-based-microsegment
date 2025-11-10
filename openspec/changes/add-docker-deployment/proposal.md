# 提案: 添加 Docker 容器化部署配置

## 概述

本提案旨在为 eBPF 微分段系统添加 Docker 容器化部署支持,包括优化的 Dockerfile、Docker Compose 编排配置和相关部署脚本,以支持开发环境和小规模生产部署场景。

## Why

当前 eBPF 微分段系统的 Server 和 Agent 组件已经实现并可运行,但缺少标准化的 Docker 容器化配置,导致:

**问题 1: 缺少标准容器镜像**
- 没有生产级别的 Dockerfile,用户需要自行构建
- 镜像构建缺乏优化,体积大、构建慢
- 没有多阶段构建,包含不必要的构建工具
- 安全性配置不完善 (root 用户、权限过大)

**问题 2: Docker Compose 配置不完整**
- 当前只有简单的 PostgreSQL 配置
- 缺少 Server 和 Agent 的容器编排
- 没有健康检查和依赖管理
- Agent 特权配置 (eBPF 需要) 未定义

**问题 3: 开发体验差**
- 无法一键启动完整的开发环境
- 缺少开发环境特定配置 (热重载、调试端口)
- 配置管理混乱,环境变量和配置文件混用
- 没有部署脚本,手工操作易出错

**业务影响**:
- 延缓开发人员快速搭建本地环境
- 增加小规模部署的运维成本
- 降低系统可移植性和可复现性
- 团队协作时环境不一致问题频发

本提案通过提供完整的 Docker 容器化配置,解决上述问题,使系统能够快速部署和迁移。

## 动机

当前项目已经具备:
- 可运行的 Server 和 Agent 组件
- 基本的启动脚本 (`start-all.sh`, `stop-all.sh`)
- 简单的 Docker Compose 配置用于 PostgreSQL
- 配置文件支持 (YAML 格式)

但是缺少:
1. **优化的 Dockerfile**: Server 和 Agent 的生产级别 Dockerfile,包含多阶段构建、安全配置
2. **完整的 Docker Compose**: 编排所有组件 (PostgreSQL, Server, Agent, Web UI),配置健康检查和依赖
3. **开发环境配置**: 支持热重载和调试的开发环境 override 配置
4. **部署脚本**: 自动化 Docker 部署流程的脚本
5. **配置模板**: Docker 环境特定的配置模板

这些缺失导致:
- Docker 环境部署复杂且不标准
- 开发效率低下
- 配置管理混乱
- 安全风险 (权限过大、镜像未优化)

## 目标

### 主要目标

1. **创建优化的 Dockerfile**
   - Server Dockerfile: 多阶段构建,Alpine 基础镜像,非 root 用户,镜像 < 50MB
   - Agent Dockerfile: Ubuntu 基础镜像 (eBPF 兼容),包含 libbpf/bpftool,镜像 < 200MB
   - 配置健康检查指令
   - 使用 .dockerignore 优化构建上下文

2. **创建完整的 Docker Compose 配置**
   - 编排 PostgreSQL, Server, Agent, Web UI 所有组件
   - 配置健康检查和服务依赖 (depends_on + healthcheck)
   - Agent 特权配置 (privileged, host network/pid)
   - 数据持久化 (named volumes)
   - 网络隔离 (bridge network)

3. **开发环境支持**
   - Docker Compose override 配置用于开发
   - 源代码挂载实现热重载
   - 调试端口暴露
   - 开发配置文件使用

4. **部署自动化**
   - Docker 一键部署脚本
   - 健康检查脚本
   - 配置模板

5. **文档和测试**
   - Docker 部署文档
   - 部署流程测试验证

### 非目标

- Systemd 服务配置 (在单独的提案中处理)
- Kubernetes 部署配置 (在单独的提案中处理)
- Helm Charts
- 云厂商特定的容器服务配置
- 生产环境监控和告警配置

## 设计概要

### 1. Dockerfile 设计

#### Server Dockerfile
```dockerfile
# 阶段 1: 构建
FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o microsegment-server ./src/server/cmd

# 阶段 2: 运行
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /build/microsegment-server /usr/local/bin/
RUN adduser -D -u 1000 microsegment
USER microsegment
EXPOSE 8080 9090
HEALTHCHECK --interval=30s --timeout=3s CMD wget -q --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["/usr/local/bin/microsegment-server"]
```

**设计决策**:
- 多阶段构建减小最终镜像体积 (~20MB)
- Alpine Linux 基础镜像,最小化攻击面
- 非 root 用户运行,提升安全性
- 内置健康检查

#### Agent Dockerfile
```dockerfile
# 阶段 1: 构建
FROM golang:1.24 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o microsegment-agent ./src/agent/cmd

# 阶段 2: 运行
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y \
    libbpf1 bpftool ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /build/microsegment-agent /usr/local/bin/
COPY --from=builder /build/src/agent/ebpf/flow_tracker_core.o /usr/local/lib/ebpf/
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/microsegment-agent"]
```

**设计决策**:
- Ubuntu 而非 Alpine (eBPF 工具兼容性)
- 包含 libbpf 和 bpftool
- 运行需要 root 权限 (eBPF 加载)

### 2. Docker Compose 架构

```yaml
services:
  postgres:
    image: postgres:15-alpine
    healthcheck:
      test: ["CMD-SHELL", "pg_isready"]

  server:
    build:
      context: .
      dockerfile: deploy/docker/Dockerfile.server
    depends_on:
      postgres:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]

  agent:
    build:
      context: .
      dockerfile: deploy/docker/Dockerfile.agent
    depends_on:
      server:
        condition: service_healthy
    privileged: true
    network_mode: host
    pid: host
```

### 3. 目录结构

```
deploy/docker/
├── Dockerfile.server
├── Dockerfile.agent
├── docker-compose.yml
├── docker-compose.dev.yml
├── .dockerignore
└── README.md

deploy/config/
├── docker/
│   ├── server.yaml
│   └── agent.yaml
└── README.md

deploy/scripts/
├── deploy-docker.sh
└── health-check-docker.sh
```

## 影响的规范

本提案将添加新的 Docker 部署规范:

1. **新规范 docker-deployment**: 定义 Docker 容器化部署要求

## 依赖关系

### 前置条件
- Server 和 Agent 组件已实现并可运行
- 配置管理已实现 (config.go)
- 数据库迁移工具已实现

### 阻塞项
- 无

## 成功标准

1. **功能完整性**
   - ✅ 可通过 `docker build` 成功构建 Server 和 Agent 镜像
   - ✅ 可通过 `docker-compose up` 一键部署完整系统
   - ✅ 所有容器健康检查通过

2. **镜像质量**
   - ✅ Server 镜像大小 < 50MB
   - ✅ Agent 镜像大小 < 200MB
   - ✅ 镜像安全扫描无高危漏洞

3. **可用性**
   - ✅ 部署文档清晰,新用户可按指南完成部署
   - ✅ 提供一键部署脚本
   - ✅ 健康检查可验证部署成功

4. **安全性**
   - ✅ Server 使用非 root 用户
   - ✅ Agent 特权配置有明确文档说明
   - ✅ 密钥通过环境变量管理

## 风险与缓解

### 风险 1: Agent eBPF 程序在容器中的权限问题
- **描述**: Agent 需要加载 eBPF 程序,需要特权容器和主机网络
- **缓解**:
  - 使用 `privileged: true`, `network_mode: host`, `pid: host`
  - 在文档中明确说明安全影响
  - 提供最小权限配置示例

### 风险 2: 镜像构建时间长
- **描述**: Go 依赖下载和编译可能导致构建缓慢
- **缓解**:
  - 利用 Docker 层缓存
  - 分离依赖下载和代码编译步骤
  - 使用 .dockerignore 减小构建上下文

### 风险 3: 数据库初始化顺序
- **描述**: Server 启动时数据库可能未就绪
- **缓解**:
  - 使用 `depends_on` + `healthcheck` 控制启动顺序
  - Server 实现数据库连接重试机制
  - 提供数据库迁移脚本

## 后续工作

本提案完成后,可以考虑:

1. **多架构支持**: 构建 AMD64 和 ARM64 镜像
2. **镜像仓库**: 推送到 Docker Hub 或私有仓库
3. **Docker Swarm**: 支持 Docker Swarm 编排
4. **性能优化**: 进一步优化镜像大小和构建速度

## 时间估算

- **Dockerfile 创建**: 1 天
- **Docker Compose 配置**: 0.5 天
- **配置模板**: 0.5 天
- **部署脚本**: 1 天
- **文档编写**: 0.5 天
- **测试验证**: 0.5 天

**总计**: 约 4 天

## 参考

- [Docker 最佳实践](https://docs.docker.com/develop/dev-best-practices/)
- [Dockerfile 多阶段构建](https://docs.docker.com/build/building/multi-stage/)
- [Docker Compose 健康检查](https://docs.docker.com/compose/compose-file/#healthcheck)

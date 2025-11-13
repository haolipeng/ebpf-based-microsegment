# 提案: 添加最小化 Docker 部署配置

## 概述

为 eBPF 微分段系统添加基础的 Docker 容器化部署支持,实现快速部署和测试。

## Why

**当前问题**:
- 没有 Dockerfile,无法容器化部署 Server 和 Agent
- docker-compose.simple.yml 只包含 PostgreSQL,缺少 Server 和 Agent
- 开发和测试需要手动启动多个进程,容易出错
- 无法一键部署完整的测试环境

**业务影响**:
- 开发效率低,环境搭建复杂
- 测试环境不一致
- 难以快速验证功能

## 目标

### 主要目标

1. **创建基础 Dockerfile**
   - Server Dockerfile: 可构建并运行 Server
   - Agent Dockerfile: 可构建并运行 Agent (支持 eBPF)

2. **完善 Docker Compose**
   - 在现有 docker-compose.simple.yml 基础上添加 Server 和 Agent 服务
   - 配置服务依赖和启动顺序
   - Agent 配置必要的特权权限

3. **一键部署**
   - `docker-compose up` 可启动完整系统 (PostgreSQL + Server + Agent)
   - 可以进行基本的功能测试

### 非目标

- 镜像大小优化 (后续优化)
- 生产级别的安全配置 (后续优化)
- 多架构支持 (后续添加)
- 开发环境配置 (后续添加)
- Web UI 集成 (后续添加)
- 部署脚本和自动化 (后续添加)
- Systemd 服务配置
- Kubernetes 部署配置

## 设计概要

### 1. Dockerfile 设计

#### Server Dockerfile (deploy/docker/Dockerfile.server)
```dockerfile
FROM golang:1.24 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o microsegment-server ./src/server/cmd

FROM ubuntu:22.04
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /build/microsegment-server /usr/local/bin/
EXPOSE 8080 9090
CMD ["/usr/local/bin/microsegment-server", "--config", "/etc/microsegment/server.yaml"]
```

#### Agent Dockerfile (deploy/docker/Dockerfile.agent)
```dockerfile
FROM golang:1.24 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o microsegment-agent ./src/agent/cmd

FROM ubuntu:22.04
RUN apt-get update && apt-get install -y \
    iproute2 iptables ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /build/microsegment-agent /usr/local/bin/
EXPOSE 8081
CMD ["/usr/local/bin/microsegment-agent", "--config", "/etc/microsegment/agent.yaml"]
```

### 2. Docker Compose 配置

扩展现有的 docker-compose.simple.yml:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: microsegment-postgres
    environment:
      POSTGRES_DB: microsegment
      POSTGRES_USER: microsegment_user
      POSTGRES_PASSWORD: secret
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U microsegment_user -d microsegment"]
      interval: 5s
      timeout: 5s
      retries: 5

  server:
    build:
      context: .
      dockerfile: deploy/docker/Dockerfile.server
    container_name: microsegment-server
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "8080:8080"
      - "9090:9090"
    volumes:
      - ./config:/etc/microsegment
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=microsegment_user
      - DB_PASSWORD=secret
      - DB_NAME=microsegment

  agent:
    build:
      context: .
      dockerfile: deploy/docker/Dockerfile.agent
    container_name: microsegment-agent
    depends_on:
      - server
    privileged: true
    network_mode: host
    pid: host
    volumes:
      - ./config:/etc/microsegment
      - /sys/fs/bpf:/sys/fs/bpf:rw
    environment:
      - SERVER_ADDR=localhost:9090

volumes:
  postgres_data:
```

### 3. 配置文件

创建 Docker 专用的配置文件:

- `config/docker-server.yaml`: Server 配置
- `config/docker-agent.yaml`: Agent 配置

### 4. 目录结构

```
deploy/docker/
├── Dockerfile.server       # Server Dockerfile
├── Dockerfile.agent        # Agent Dockerfile
└── README.md              # 部署说明

docker-compose.yml         # 完整的 Docker Compose 配置
config/
├── docker-server.yaml    # Server Docker 配置
└── docker-agent.yaml     # Agent Docker 配置
```

## 影响的规范

本提案将添加新的 Docker 部署规范:

1. **新规范 docker-deployment-minimal**: 定义最小化 Docker 部署要求

## 依赖关系

### 前置条件
- Server 和 Agent 已实现
- 配置文件支持已实现
- PostgreSQL 数据库迁移已实现

### 阻塞项
- 无

## 成功标准

1. **可构建**
   - ✅ `docker build -f deploy/docker/Dockerfile.server` 成功
   - ✅ `docker build -f deploy/docker/Dockerfile.agent` 成功

2. **可部署**
   - ✅ `docker-compose up` 成功启动所有服务
   - ✅ Server 可正常连接 PostgreSQL
   - ✅ Agent 可正常连接 Server

3. **可测试**
   - ✅ 可通过 Server API 创建策略
   - ✅ Agent 可正常上报流量
   - ✅ 策略可正常执行

## 风险与缓解

### 风险 1: Agent eBPF 需要特权
- **缓解**: 使用 `privileged: true` 和 `network_mode: host`

### 风险 2: 镜像可能较大
- **缓解**: 初期可接受,后续优化

### 风险 3: 配置管理
- **缓解**: 使用卷挂载配置文件

## 后续工作

精简版完成后,可以考虑:
1. 镜像大小优化 (多阶段构建优化)
2. 安全性增强 (非 root 用户、最小权限)
3. 开发环境支持 (热重载)
4. 部署脚本
5. 健康检查优化

## 时间估算

- **Dockerfile 创建**: 2 小时
- **Docker Compose 配置**: 1 小时
- **配置文件**: 1 小时
- **测试验证**: 1 小时

**总计**: 约 0.5 天 (5 小时)

# 设计文档: Docker 容器化部署

## 架构概览

本文档描述 eBPF 微分段系统的 Docker 容器化部署架构,涵盖镜像构建、容器编排和配置管理。

## 系统组件

### 1. 容器架构

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Compose                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │              bridge network: microsegment        │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐ │  │
│  │  │ PostgreSQL │  │   Server   │  │   Web UI   │ │  │
│  │  │  (5432)    │◄─┤ (8080/9090)│  │   (3000)   │ │  │
│  │  └────────────┘  └────────────┘  └────────────┘ │  │
│  └──────────────────────────────────────────────────┘  │
│                                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │              host network mode                    │  │
│  │  ┌────────────┐                                   │  │
│  │  │   Agent    │  (privileged: true)              │  │
│  │  │  (eBPF)    │  (pid: host, network: host)      │  │
│  │  └────────────┘                                   │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### 2. 组件职责

#### PostgreSQL 容器
- **镜像**: postgres:15-alpine
- **职责**: 数据持久化
- **网络**: bridge network
- **存储**: named volume (postgres_data)
- **健康检查**: pg_isready

#### Server 容器
- **构建**: 自定义 Dockerfile.server
- **职责**: 控制平面 API 和 gRPC 服务
- **网络**: bridge network
- **端口**: 8080 (HTTP), 9090 (gRPC)
- **依赖**: PostgreSQL (健康后启动)
- **用户**: 非 root (uid 1000)

#### Agent 容器
- **构建**: 自定义 Dockerfile.agent
- **职责**: eBPF 程序加载和流数据上报
- **网络**: host network (访问主机网络栈)
- **特权**: privileged: true (eBPF 加载需要)
- **挂载**: /sys/fs/bpf
- **依赖**: Server (健康后启动)

## Dockerfile 设计

### 1. Server Dockerfile 详细设计

```dockerfile
# ============================================
# 阶段 1: 构建阶段
# ============================================
FROM golang:1.24-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git make

# 设置工作目录
WORKDIR /build

# 复制依赖文件 (利用层缓存)
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建二进制文件 (静态链接)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o microsegment-server \
    ./src/server/cmd

# ============================================
# 阶段 2: 运行阶段
# ============================================
FROM alpine:3.19

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    wget

# 创建非 root 用户
RUN adduser -D -u 1000 -g microsegment microsegment

# 创建必要目录
RUN mkdir -p /etc/microsegment /var/lib/microsegment && \
    chown -R microsegment:microsegment /etc/microsegment /var/lib/microsegment

# 复制二进制文件
COPY --from=builder /build/microsegment-server /usr/local/bin/
RUN chmod +x /usr/local/bin/microsegment-server

# 复制默认配置
COPY deploy/config/docker/server.yaml /etc/microsegment/config.yaml
RUN chown microsegment:microsegment /etc/microsegment/config.yaml

# 切换到非 root 用户
USER microsegment

# 暴露端口
EXPOSE 8080 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# 设置入口点
ENTRYPOINT ["/usr/local/bin/microsegment-server"]
CMD ["--config", "/etc/microsegment/config.yaml"]
```

**关键设计点**:
1. **多阶段构建**: 构建阶段使用完整的 golang 镜像,运行阶段使用最小的 alpine
2. **静态链接**: CGO_ENABLED=0 避免运行时依赖
3. **版本注入**: 使用 -ldflags 注入版本信息
4. **层缓存优化**: 先复制 go.mod/go.sum,再复制源代码
5. **非 root 用户**: 使用 uid 1000 的普通用户运行
6. **健康检查**: 内置健康检查,支持容器编排

### 2. Agent Dockerfile 详细设计

```dockerfile
# ============================================
# 阶段 1: 构建阶段
# ============================================
FROM golang:1.24 AS builder

# 安装构建依赖
RUN apt-get update && apt-get install -y \
    clang \
    llvm \
    libbpf-dev \
    linux-headers-generic \
    make

# 设置工作目录
WORKDIR /build

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译 eBPF 程序
RUN make -C src/agent/ebpf

# 构建 Agent 二进制 (需要 CGO)
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" \
    -o microsegment-agent \
    ./src/agent/cmd

# ============================================
# 阶段 2: 运行阶段
# ============================================
FROM ubuntu:22.04

# 安装运行时依赖
RUN apt-get update && apt-get install -y \
    libbpf1 \
    bpftool \
    ca-certificates \
    iproute2 \
    && rm -rf /var/lib/apt/lists/*

# 创建目录
RUN mkdir -p /usr/local/lib/ebpf /etc/microsegment

# 复制二进制和 eBPF 程序
COPY --from=builder /build/microsegment-agent /usr/local/bin/
COPY --from=builder /build/src/agent/ebpf/*.o /usr/local/lib/ebpf/
RUN chmod +x /usr/local/bin/microsegment-agent

# 复制默认配置
COPY deploy/config/docker/agent.yaml /etc/microsegment/config.yaml

# 暴露 API 端口
EXPOSE 8081

# 设置入口点
ENTRYPOINT ["/usr/local/bin/microsegment-agent"]
CMD ["--config", "/etc/microsegment/config.yaml"]
```

**关键设计点**:
1. **Ubuntu 基础镜像**: eBPF 工具在 Ubuntu 上兼容性更好
2. **包含 eBPF 工具**: libbpf1, bpftool 用于 eBPF 操作
3. **CGO 启用**: Agent 需要 CGO 支持
4. **eBPF 字节码**: 预编译的 .o 文件打包到镜像
5. **Root 运行**: Agent 需要 root 权限加载 eBPF 程序

## Docker Compose 设计

### 1. 主配置文件 (docker-compose.yml)

```yaml
version: '3.8'

services:
  # PostgreSQL 数据库
  postgres:
    image: postgres:15-alpine
    container_name: microsegment-postgres
    environment:
      POSTGRES_DB: microsegment
      POSTGRES_USER: microsegment_user
      POSTGRES_PASSWORD: ${DB_PASSWORD:-changeme}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - microsegment
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U microsegment_user"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    restart: unless-stopped

  # Server 控制平面
  server:
    build:
      context: ../..
      dockerfile: deploy/docker/Dockerfile.server
    container_name: microsegment-server
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      MICROSEGMENT_DB_HOST: postgres
      MICROSEGMENT_DB_PORT: 5432
      MICROSEGMENT_DB_NAME: microsegment
      MICROSEGMENT_DB_USER: microsegment_user
      MICROSEGMENT_DB_PASSWORD: ${DB_PASSWORD:-changeme}
      MICROSEGMENT_SERVER_HTTP_PORT: 8080
      MICROSEGMENT_SERVER_GRPC_PORT: 9090
      MICROSEGMENT_LOG_LEVEL: ${LOG_LEVEL:-info}
    ports:
      - "8080:8080"
      - "9090:9090"
    networks:
      - microsegment
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 20s
    restart: unless-stopped

  # Agent 数据平面
  agent:
    build:
      context: ../..
      dockerfile: deploy/docker/Dockerfile.agent
    container_name: microsegment-agent
    depends_on:
      server:
        condition: service_healthy
    environment:
      MICROSEGMENT_SERVER_URL: http://host.docker.internal:8080
      MICROSEGMENT_AGENT_API_PORT: 8081
      MICROSEGMENT_LOG_LEVEL: ${LOG_LEVEL:-info}
    privileged: true
    network_mode: host
    pid: host
    volumes:
      - /sys/fs/bpf:/sys/fs/bpf:rw
      - /sys/kernel/debug:/sys/kernel/debug:ro
    restart: unless-stopped

  # Web UI (可选)
  webui:
    image: node:18-alpine
    container_name: microsegment-webui
    working_dir: /app
    command: sh -c "npm install && npm start"
    volumes:
      - ../../src/webui:/app
    ports:
      - "3000:3000"
    networks:
      - microsegment
    environment:
      REACT_APP_API_URL: http://localhost:8080
    depends_on:
      - server

networks:
  microsegment:
    driver: bridge

volumes:
  postgres_data:
    driver: local
```

**关键设计点**:
1. **健康检查链**: postgres → server → agent (确保启动顺序)
2. **网络隔离**: Server/PostgreSQL 使用 bridge 网络,Agent 使用 host 网络
3. **环境变量**: 支持通过 .env 文件或环境变量自定义配置
4. **数据持久化**: PostgreSQL 数据使用 named volume
5. **重启策略**: unless-stopped 确保服务自动恢复

### 2. 开发环境 Override (docker-compose.dev.yml)

```yaml
version: '3.8'

services:
  server:
    build:
      context: ../..
      dockerfile: deploy/docker/Dockerfile.server
      target: builder  # 使用构建阶段,包含开发工具
    volumes:
      - ../../src/server:/build/src/server:ro  # 源代码挂载
      - ../../go.mod:/build/go.mod:ro
      - ../../go.sum:/build/go.sum:ro
    environment:
      MICROSEGMENT_LOG_LEVEL: debug
      MICROSEGMENT_DB_SSLMODE: disable
    command: ["go", "run", "./src/server/cmd", "--config", "/etc/microsegment/config.yaml"]

  agent:
    build:
      target: builder
    volumes:
      - ../../src/agent:/build/src/agent:ro
    environment:
      MICROSEGMENT_LOG_LEVEL: debug
    command: ["go", "run", "./src/agent/cmd", "--config", "/etc/microsegment/config.yaml"]

  webui:
    volumes:
      - ../../src/webui:/app
      - /app/node_modules  # 匿名卷,避免覆盖 node_modules
```

**使用方式**:
```bash
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up
```

## 配置管理

### 1. 配置文件结构

```
deploy/config/docker/
├── server.yaml      # Server 默认配置
└── agent.yaml       # Agent 默认配置
```

### 2. 配置优先级

1. 环境变量 (最高优先级)
2. Docker Compose 环境变量
3. 配置文件
4. 默认值 (代码中定义)

### 3. Server 配置示例 (server.yaml)

```yaml
server:
  http:
    host: 0.0.0.0
    port: 8080
  grpc:
    host: 0.0.0.0
    port: 9090

database:
  host: ${MICROSEGMENT_DB_HOST}
  port: ${MICROSEGMENT_DB_PORT}
  name: ${MICROSEGMENT_DB_NAME}
  user: ${MICROSEGMENT_DB_USER}
  password: ${MICROSEGMENT_DB_PASSWORD}
  sslmode: ${MICROSEGMENT_DB_SSLMODE:-require}

log:
  level: ${MICROSEGMENT_LOG_LEVEL:-info}
  format: json
```

## 构建优化

### 1. .dockerignore 设计

```
# Git
.git
.gitignore

# 文档
docs/
*.md
LICENSE

# 测试
*_test.go
testdata/
coverage*.out

# 构建产物
bin/
*.out
*.o

# IDE
.vscode/
.idea/

# 临时文件
tmp/
*.tmp

# 部署文件 (避免循环)
deploy/
```

### 2. 构建缓存策略

1. **依赖层缓存**: 先复制 go.mod/go.sum,后复制源代码
2. **多阶段构建**: 分离构建和运行环境
3. **BuildKit**: 使用 Docker BuildKit 加速构建

## 部署流程

### 1. 快速开始

```bash
# 1. 克隆仓库
git clone https://github.com/haolipeng/ebpf-based-microsegment.git
cd ebpf-based-microsegment

# 2. 配置环境变量 (可选)
cp deploy/docker/.env.example deploy/docker/.env
# 编辑 .env 文件

# 3. 启动所有服务
cd deploy/docker
docker-compose up -d

# 4. 检查健康状态
docker-compose ps
./scripts/health-check-docker.sh
```

### 2. 自动化部署脚本

脚本 `deploy/scripts/deploy-docker.sh` 执行以下步骤:

1. 检查 Docker 和 Docker Compose 环境
2. 验证配置文件
3. 构建镜像
4. 启动 PostgreSQL
5. 等待数据库就绪
6. 运行数据库迁移
7. 启动 Server 和 Agent
8. 运行健康检查
9. 输出访问信息

## 健康检查

### 1. Server 健康检查

- **端点**: GET /health
- **响应**:
```json
{
  "status": "healthy",
  "checks": {
    "database": true,
    "grpc": true
  },
  "version": "v0.1.0"
}
```

### 2. Agent 健康检查

- 检查 eBPF 程序加载状态
- 检查与 Server 的连接
- 检查 BPF maps 可访问性

## 安全考虑

### 1. 镜像安全

- **最小化**: 使用 Alpine/Ubuntu 最小镜像
- **非 root**: Server 使用非 root 用户
- **扫描**: 使用 Trivy 扫描镜像漏洞
- **签名**: 生产镜像进行签名

### 2. 运行时安全

- **网络隔离**: Server/DB 使用独立 bridge 网络
- **特权说明**: Agent 特权使用文档化
- **密钥管理**: 敏感信息通过环境变量或 Secret

### 3. Agent 特权风险

Agent 需要以下特权:
- `privileged: true`: 加载 eBPF 程序
- `network_mode: host`: 访问主机网络栈
- `pid: host`: 访问主机进程信息

**风险**: 容器逃逸,主机网络访问
**缓解**: 仅在可信环境运行,监控 Agent 行为

## 监控和日志

### 1. 日志输出

- 所有容器日志输出到 stdout/stderr
- Docker 日志驱动配置
- 查看日志: `docker-compose logs -f [service]`

### 2. 指标暴露

- Server: /metrics 端点 (Prometheus 格式)
- Agent: /metrics 端点

## 总结

本设计提供完整的 Docker 容器化方案:
1. **优化的镜像**: 多阶段构建,安全配置,体积小
2. **完整的编排**: Docker Compose 管理所有组件
3. **开发支持**: 开发环境 override,热重载
4. **自动化**: 一键部署脚本,健康检查
5. **文档**: 详细的部署和使用文档

适用场景:
- 开发环境
- 小规模生产部署
- 快速原型验证
- CI/CD 测试环境

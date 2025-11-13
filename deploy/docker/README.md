# Docker 部署指南

本指南介绍如何使用 Docker 和 Docker Compose 快速部署 eBPF 微分段系统。

## 前提条件

- Docker Engine 20.10 或更高版本
- Docker Compose v2.0 或更高版本
- Linux 内核 5.10 或更高版本 (支持 eBPF)
- Root 权限 (Agent 需要加载 eBPF 程序)

## 快速开始

### 1. 构建镜像

在项目根目录执行:

```bash
# 构建 Server 镜像
docker build -f deploy/docker/Dockerfile.server -t microsegment-server:latest .

# 构建 Agent 镜像
docker build -f deploy/docker/Dockerfile.agent -t microsegment-agent:latest .
```

### 2. 启动服务

使用 Docker Compose 一键启动所有服务:

```bash
docker-compose up -d
```

这将启动以下服务:
- **PostgreSQL**: 数据库服务 (端口 5432)
- **Server**: 控制平面服务 (端口 8080 HTTP, 9090 gRPC)
- **Agent**: 数据平面服务 (使用 host 网络模式)

### 3. 验证部署

检查服务状态:

```bash
# 查看所有服务状态
docker-compose ps

# 查看 Server 日志
docker-compose logs -f server

# 查看 Agent 日志
docker-compose logs -f agent

# 查看 PostgreSQL 日志
docker-compose logs -f postgres
```

验证 Server API:

```bash
# 检查 Server 健康状态
curl http://localhost:8080/health

# 查看 Agent 列表
curl http://localhost:8080/api/v1/agents
```

验证 Agent API:

```bash
# 检查 Agent 健康状态 (需要使用主机网络)
curl http://localhost:8081/health
```

### 4. 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止并删除数据卷
docker-compose down -v
```

## 配置文件

配置文件位于 `config/` 目录:

- **config/docker-server.yaml**: Server 配置文件
- **config/docker-agent.yaml**: Agent 配置文件

你可以根据需要修改这些配置文件,修改后重启服务生效:

```bash
docker-compose restart server
docker-compose restart agent
```

## 环境变量

### Server 环境变量

在 `docker-compose.yml` 中可以配置以下环境变量:

- `DB_HOST`: PostgreSQL 主机地址 (默认: postgres)
- `DB_PORT`: PostgreSQL 端口 (默认: 5432)
- `DB_USER`: 数据库用户名 (默认: microsegment_user)
- `DB_PASSWORD`: 数据库密码 (默认: secret)
- `DB_NAME`: 数据库名称 (默认: microsegment)

### Agent 环境变量

- `SERVER_ADDR`: Server gRPC 地址 (默认: localhost:9090)

## 网络架构

```
┌─────────────────────────────────────────────────────────┐
│                      Host Network                        │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  ┌──────────────┐    ┌──────────────┐                   │
│  │  PostgreSQL  │◄───│    Server    │                   │
│  │   :5432      │    │  :8080 :9090 │                   │
│  └──────────────┘    └───────▲──────┘                   │
│                              │                            │
│                              │ gRPC                       │
│                              │                            │
│                      ┌───────┴──────┐                    │
│                      │     Agent     │                    │
│                      │  (host mode)  │                    │
│                      └───────────────┘                    │
│                              │                            │
│                              │ eBPF                       │
│                              ▼                            │
│                      ┌───────────────┐                    │
│                      │  eth0/网络接口 │                   │
│                      └───────────────┘                    │
└─────────────────────────────────────────────────────────┘
```

## 注意事项

### Agent 特权模式

Agent 容器运行时需要以下特权:

- **privileged: true**: 需要加载 eBPF 程序和管理网络接口
- **network_mode: host**: 需要访问主机网络接口
- **pid: host**: 需要访问主机进程信息
- **/sys/fs/bpf 挂载**: 需要读写 BPF 文件系统

这些权限是 eBPF 程序正常工作所必需的。

### 数据持久化

PostgreSQL 数据存储在 Docker 卷 `postgres_data` 中。如果需要清除所有数据:

```bash
docker-compose down -v
```

### 端口冲突

如果端口已被占用,可以在 `docker-compose.yml` 中修改端口映射:

```yaml
ports:
  - "8080:8080"  # 修改为 "18080:8080" 等
```

## 故障排查

### Server 无法连接到 PostgreSQL

检查 PostgreSQL 是否启动成功:

```bash
docker-compose logs postgres
```

检查 Server 日志:

```bash
docker-compose logs server
```

### Agent 无法连接到 Server

检查 Agent 日志:

```bash
docker-compose logs agent
```

确认 Server 服务已启动:

```bash
curl http://localhost:8080/health
```

### eBPF 程序加载失败

确认内核版本:

```bash
uname -r
```

确认 BPF 文件系统已挂载:

```bash
mount | grep bpf
```

检查 Agent 日志中的错误信息:

```bash
docker-compose logs agent | grep -i error
```

## 下一步

部署成功后,你可以:

1. 通过 Server API 创建策略
2. 查看流量数据
3. 监控 Agent 状态
4. 配置自定义策略规则

详细的 API 文档请参考主项目 README。

# eBPF Microsegmentation - 部署指南

## 快速启动

### 启动所有服务

```bash
./start-all.sh
```

此脚本将依次启动：
1. PostgreSQL 数据库 (Docker)
2. Server 组件 (控制平面)
3. Agent 组件 (数据平面)
4. Web UI 前端

### 停止所有服务

```bash
./stop-all.sh
```

## 访问地址

从宿主机 Windows 浏览器访问：

- **Web UI 界面**: http://10.107.12.201:3000
- **Server HTTP API**: http://10.107.12.201:8080
- **Agent API**: http://10.107.12.201:8081

## 服务架构

```
┌─────────────────────────────────────────────────────┐
│                  Web UI (React)                      │
│               http://0.0.0.0:3000                    │
└───────────────────┬─────────────────────────────────┘
                    │ HTTP/REST API
┌───────────────────▼─────────────────────────────────┐
│             Server 组件 (Go)                         │
│   HTTP API: 0.0.0.0:8080                            │
│   gRPC: 0.0.0.0:9090                                │
└───────────┬────────────────────┬────────────────────┘
            │                    │
            │ gRPC               │ PostgreSQL
┌───────────▼────────┐    ┌─────▼──────────────────┐
│  Agent 组件 (Go)   │    │   PostgreSQL 数据库    │
│  API: 0.0.0.0:8081 │    │   Port: 5432           │
│  eBPF 数据平面     │    │   (Docker Container)   │
└────────────────────┘    └────────────────────────┘
```

## 组件说明

### 1. PostgreSQL 数据库
- 运行方式: Docker 容器
- 端口: 5432
- 数据库名: microsegment
- 用户: microsegment_user
- 密码: secret
- 数据卷: Docker volume (持久化存储)

### 2. Server 组件
- 二进制: `./bin/microsegment-server`
- 配置文件: `src/server/config.yaml`
- 日志文件: `/tmp/server.log`
- 功能:
  - Agent 注册和管理
  - Flow 数据聚合
  - 策略分发
  - WebSocket 实时流
  - HTTP REST API
  - gRPC 服务

### 3. Agent 组件
- 二进制: `./bin/microsegment-agent`
- 配置文件: `agent-config.yaml`
- 日志文件: `/tmp/agent.log`
- 运行模式: agent-server (连接到控制平面)
- 监听接口: ens33
- 功能:
  - eBPF 数据包过滤
  - Flow 收集和上报
  - 策略执行
  - 本地 SQLite 存储
  - 健康检查

### 4. Web UI
- 技术栈: React 19 + TypeScript + Vite
- 目录: `web/`
- 日志文件: `/tmp/web.log`
- 功能:
  - Dashboard 仪表盘
  - Agent 管理
  - Flow 查询和可视化
  - 策略管理
  - 实时图表 (ECharts)

## 配置文件

### Server 配置 (`src/server/config.yaml`)
```yaml
server:
  http_addr: "0.0.0.0:8080"
  grpc_addr: "0.0.0.0:9090"

database:
  host: localhost
  port: 5432
  user: microsegment_user
  password: secret
  dbname: microsegment
  sslmode: disable
```

### Agent 配置 (`agent-config.yaml`)
```yaml
mode: agent-server
interface: ens33
log_level: info

server:
  server_addr: localhost:9090
  agent_id: agent-001
  batch_size: 100
  batch_timeout: 5s

flow:
  enabled: true
  storage_path: /home/work/ebpf-based-microsegment/data/flows.db
```

### Web UI 配置 (`web/vite.config.ts`)
```typescript
server: {
  host: '0.0.0.0',  // 监听所有接口
  port: 3000,
  proxy: {
    '/api': {
      target: 'http://localhost:8080',  // 代理到 Server API
      changeOrigin: true,
    },
  },
}
```

## 日志位置

- Server: `/tmp/server.log`
- Agent: `/tmp/agent.log`
- Web UI: `/tmp/web.log`
- PostgreSQL: `docker logs microsegment-postgres`

## 故障排查

### 查看服务状态

```bash
# 检查进程
ps aux | grep -E '(microsegment|vite)'

# 检查端口
netstat -tlnp | grep -E '(3000|8080|8081|9090|5432)'

# 检查 Docker
docker ps
```

### 查看实时日志

```bash
# Server
tail -f /tmp/server.log

# Agent
tail -f /tmp/agent.log

# Web UI
tail -f /tmp/web.log
```

### 常见问题

**Q: Web UI 无法访问**
- 检查 Vite 是否在运行: `ps aux | grep vite`
- 检查端口 3000 是否被占用: `netstat -tlnp | grep 3000`
- 查看日志: `cat /tmp/web.log`

**Q: Agent 无法连接到 Server**
- 检查 Server 是否在运行: `ps aux | grep microsegment-server`
- 检查 gRPC 端口 9090: `netstat -tlnp | grep 9090`
- 查看 Agent 日志: `tail -f /tmp/agent.log`

**Q: 数据库连接失败**
- 检查 PostgreSQL 容器: `docker ps | grep postgres`
- 检查端口: `netstat -tlnp | grep 5432`
- 测试连接: `psql -h localhost -U microsegment_user -d microsegment`

**Q: Agent 报错 "interface not found"**
- 查看可用网络接口: `ip link show`
- 修改 `agent-config.yaml` 中的 `interface` 字段

## 手动操作

### 数据库迁移

```bash
cd src/server
DB_HOST=localhost DB_PORT=5432 DB_USER=microsegment_user \
DB_PASSWORD=secret DB_NAME=microsegment \
./scripts/migrate.sh up
```

### 重新编译

```bash
# Server
cd src/server
go build -o ../../bin/microsegment-server ./cmd

# Agent
cd src/agent
go build -o ../../bin/microsegment-agent ./cmd

# Web UI
cd web
npm run build
```

## 生产部署建议

1. **使用环境变量管理敏感信息**
   - 数据库密码
   - API 密钥
   - TLS 证书

2. **启用 HTTPS**
   - 配置反向代理 (Nginx/Caddy)
   - 使用 Let's Encrypt 证书

3. **配置日志轮转**
   - 使用 logrotate 管理日志文件
   - 或者使用 systemd journal

4. **配置为系统服务**
   - 创建 systemd service 文件
   - 开机自启动

5. **监控和告警**
   - 集成 Prometheus + Grafana
   - 配置健康检查端点
   - 设置告警规则

6. **备份策略**
   - PostgreSQL 定期备份
   - Flow 数据归档
   - 配置文件版本控制

# Agent 运行模式说明

Microsegment Agent 支持两种运行模式，满足不同的使用场景。

## 模式概览

### 1. Agent-Server 模式 (默认)

Agent 连接到集中式控制平面 Server，实现：
- 流量数据上报和集中分析
- 集中式策略管理和分发
- Agent 健康状态监控
- 全局流量可视化

**适用场景**：
- 生产环境部署
- 需要集中管理多个节点
- 需要全局流量分析和策略编排

### 2. Standalone 模式

Agent 独立运行，不连接 Server，提供：
- 本地 API 接口用于调试
- 节点级流量查询和监控
- 本地策略管理
- 独立的性能监控

**适用场景**：
- 开发和测试环境
- 单节点部署
- 故障排查和调试
- 无需集中管理的场景

## 配置方式

### Agent-Server 模式

使用配置文件 `config/agent-server.yaml`：

```yaml
mode: agent-server

server:
  server_addr: localhost:9090  # Server 地址
  agent_id: ""                 # Agent ID (留空自动生成)
  batch_size: 100
  batch_timeout: 5s
  reconnect_interval: 30s

api:
  enabled: true      # 可选：启用本地 API 用于调试
  host: 127.0.0.1    # 仅本地访问
  port: 8080
```

启动命令：
```bash
sudo ./bin/microsegment-agent -c src/agent/config/agent-server.yaml
```

### Standalone 模式

使用配置文件 `config/standalone.yaml`：

```yaml
mode: standalone

# server 配置可省略或设为 null
server: null

api:
  enabled: true      # 必须启用 API
  host: 0.0.0.0      # 允许远程访问
  port: 8080
```

启动命令：
```bash
sudo ./bin/microsegment-agent -c src/agent/config/standalone.yaml
```

## 功能对比

| 功能 | Agent-Server 模式 | Standalone 模式 |
|------|------------------|-----------------|
| eBPF 包过滤 | ✅ | ✅ |
| 本地策略执行 | ✅ | ✅ |
| 本地流量收集 | ✅ | ✅ |
| 本地 API 访问 | ✅ (可选) | ✅ (必需) |
| 流量上报到 Server | ✅ | ❌ |
| Server 心跳和监控 | ✅ | ❌ |
| 集中策略同步 | ✅ | ❌ |
| 全局流量分析 | ✅ | ❌ |

## API 端点

两种模式都支持以下本地 API 端点（当 `api.enabled=true` 时）：

### 健康检查
- `GET /api/v1/health` - 健康状态
- `GET /api/v1/status` - 详细状态信息

### 统计信息
- `GET /api/v1/stats` - 所有统计信息
- `GET /api/v1/stats/packets` - 包统计
- `GET /api/v1/stats/sessions` - 会话统计
- `GET /api/v1/stats/policies` - 策略统计

### 策略管理
- `GET /api/v1/policies` - 列出所有策略
- `GET /api/v1/policies/:id` - 获取单个策略
- `POST /api/v1/policies` - 创建策略 (standalone)
- `PUT /api/v1/policies/:id` - 更新策略 (standalone)
- `DELETE /api/v1/policies/:id` - 删除策略 (standalone)

### 流量查询 (需启用 flow collection)
- `GET /api/v1/flows` - 查询流量记录
- `GET /api/v1/flows/:id` - 获取单条流量
- `GET /api/v1/flows/active` - 活跃流量
- `GET /api/v1/flows/stream` - WebSocket 实时流量推送
- `GET /api/v1/flows/dependencies` - 服务依赖关系
- `GET /api/v1/flows/top-talkers` - Top talkers 分析

### 配置管理
- `GET /api/v1/config` - 获取当前配置
- `PUT /api/v1/config` - 更新配置

## 模式切换

要在两种模式之间切换，只需：

1. 修改配置文件中的 `mode` 字段
2. 根据需要调整相关配置（server、api 等）
3. 重启 Agent

示例：
```bash
# 从 agent-server 切换到 standalone
sudo systemctl stop microsegment-agent
# 修改配置文件 mode: standalone
sudo systemctl start microsegment-agent
```

## 注意事项

### Agent-Server 模式
- 确保 Server 已启动并可访问
- 检查网络防火墙配置，允许 gRPC 端口 (默认 9090)
- Agent ID 建议在首次运行后保持不变

### Standalone 模式
- 必须启用 API (`api.enabled: true`)
- 建议设置 `api.host: 0.0.0.0` 以允许远程访问
- 策略管理完全由本地 API 控制
- 无法获得全局视图和集中分析能力

## 故障排查

### Agent-Server 模式连接失败
```bash
# 检查 Server 是否运行
curl http://<server-host>:8081/health

# 检查网络连接
telnet <server-host> 9090

# 查看 Agent 日志
sudo journalctl -u microsegment-agent -f
```

### Standalone 模式 API 无法访问
```bash
# 检查 API 是否启动
curl http://localhost:8080/api/v1/health

# 检查防火墙规则
sudo iptables -L -n | grep 8080

# 确认配置正确
cat config/standalone.yaml | grep -A 5 "api:"
```

## 推荐使用方式

### 开发阶段
使用 **Standalone 模式**：
- 快速启动，无需 Server 依赖
- 便于调试和测试
- 直接通过 API 管理策略

### 生产部署
使用 **Agent-Server 模式**：
- 集中管理和监控
- 全局流量分析
- 统一策略编排
- 高可用和扩展性

### 故障排查
临时切换到 **Standalone 模式**：
- 隔离节点问题
- 直接查看节点状态
- 本地测试策略效果

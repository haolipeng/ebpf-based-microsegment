# 变更提案：添加控制平面 API 服务

**Change ID**: `add-control-plane-api`  
**Status**: Proposed  
**Created**: 2025-10-30

## Why

数据平面（eBPF 数据包处理）已经完成并可运行，但没有用户友好的方式来管理策略、监控系统状态或配置微隔离系统。运维人员需要一个控制平面 API 来：

1. **动态管理策略** 而无需重新编译或重新加载 eBPF 程序
2. **查询实时统计信息**（数据包、会话、策略命中/未命中）
3. **监控系统健康状况** 和数据平面状态
4. **配置系统参数**（会话超时、日志级别）
5. **与外部系统集成**（编排器、SIEM、仪表板）

像 Cilium 和 Illumio 这样的商业解决方案提供了丰富的控制平面 API。我们的系统需要类似的功能才能做好生产准备。

## What Changes

### 核心 API 服务
- 使用标准库或轻量级框架（Gin/Echo）的 Go RESTful API 服务器
- 在可配置端口（默认 :8080）上的 HTTP 服务器
- 结构化 JSON 请求/响应
- 使用适当 HTTP 状态码的错误处理
- 请求日志记录和指标

### 策略管理端点
```
POST   /api/v1/policies          - 创建新策略
GET    /api/v1/policies          - 列出所有策略
GET    /api/v1/policies/:id      - 获取特定策略
PUT    /api/v1/policies/:id      - 更新策略
DELETE /api/v1/policies/:id      - 删除策略
```

### 统计和监控端点
```
GET /api/v1/stats                - 获取数据平面统计信息
GET /api/v1/stats/sessions       - 获取会话计数
GET /api/v1/stats/policies       - 获取策略统计信息
GET /api/v1/health               - 健康检查端点
```

### 配置端点
```
GET /api/v1/config               - 获取当前配置
PUT /api/v1/config               - 更新配置
```

### 数据模型
- **Policy**: `{id, src_ip, dst_ip, src_port, dst_port, protocol, action, priority, rule_id}`
- **Statistics**: `{total_packets, allowed, denied, sessions_new, sessions_active, ...}`
- **Config**: `{log_level, session_timeout, interface, ...}`

### 集成
- 连接到现有的 `dataplane.DataPlane` 进行 eBPF 操作
- 使用现有的 `policy.PolicyManager` 进行策略 CRUD
- 添加包装现有 Go API 的 HTTP 处理器

## Impact

**新能力：**
- `control-plane-api`（新增）- HTTP API 服务
- `policy-management`（修改）- 通过 HTTP 端点公开

**受影响的代码：**
- `src/agent/cmd/main.go` - 与数据平面一起启动 API 服务器
- `src/agent/pkg/api/`（新增）- HTTP 处理器和路由
- `src/agent/pkg/api/handlers/`（新增）- 端点处理器
- `src/agent/pkg/api/models/`（新增）- 请求/响应模型
- `src/agent/pkg/dataplane/dataplane.go` - 导出必要的方法
- `src/agent/pkg/policy/policy.go` - 确保线程安全访问

**依赖项：**
- 如果使用 `net/http`，则无需新的外部依赖项
- 可选：Gin（`github.com/gin-gonic/gin`）或 Echo 用于路由/中间件

**破坏性更改：**
- 无 - 纯粹是附加的

**性能影响：**
- 最小 - API 在单独的 goroutine 中运行
- 数据平面性能不受影响
- 策略更新可能有短暂的同步开销

**安全考虑：**
- MVP 中无身份验证（未来添加）
- 默认在 localhost:8080 上运行
- 添加 CORS 支持以进行 Web UI 集成
- 对所有 API 端点进行输入验证

## Success Criteria

1. API 服务器与数据平面一起成功启动
2. 所有策略 CRUD 操作通过 HTTP 工作
3. 统计端点返回准确的实时数据
4. API 对策略操作的响应时间在 100ms 内
5. 对数据平面性能无影响（<1% CPU 开销）
6. 优雅关闭（等待正在进行的请求）
7. 生成 OpenAPI/Swagger 文档
8. 所有端点的集成测试通过

## Non-Goals（未来工作）

- 身份验证/授权（未来：JWT、mTLS）
- 分布式部署（未来：多代理协调）
- 数据库持久化（未来：PostgreSQL/etcd）
- GraphQL API（未来增强）
- 用于实时流式传输的 WebSocket（未来）
- 速率限制（未来）

## Timeline

- **第 1 周**：API 框架、路由、基本处理器
- **第 2 周**：策略 CRUD 端点、测试
- **第 3 周**：统计端点、文档
- **总计**：2-3 周

## References

- 当前数据平面：`src/agent/pkg/dataplane/`
- 当前策略管理器：`src/agent/pkg/policy/`
- 设计文档：`changes/add-control-plane-api/design.md`


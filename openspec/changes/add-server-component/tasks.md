# 实施任务: Server 组件实现

**变更 ID**: `add-server-component`
**任务创建日期**: 2025-11-04
**预估总工作量**: 7-10 天

---

## 📋 任务清单

### Phase 1: 项目初始化和数据库准备 (1-2 天)

#### Task 1.1: 创建 Server 项目结构
- [ ] 创建 `src/server/` 目录
- [ ] 初始化 Go 模块
  ```bash
  cd src/server
  go mod init github.com/ebpf-microsegment/src/server
  ```
- [ ] 创建目录结构
  ```bash
  mkdir -p cmd pkg/{grpc,api,storage,manager,aggregator,config} migrations config
  ```
- [ ] 创建 `cmd/main.go` 骨架文件
- [ ] 创建 `.gitignore`

#### Task 1.2: 安装必需依赖
- [ ] 添加 gRPC 依赖
  ```bash
  go get google.golang.org/grpc@latest
  go get google.golang.org/protobuf@latest
  ```
- [ ] 添加 PostgreSQL 驱动
  ```bash
  go get github.com/lib/pq@latest
  ```
- [ ] 添加 Gin 框架
  ```bash
  go get github.com/gin-gonic/gin@latest
  ```
- [ ] 添加配置管理
  ```bash
  go get github.com/spf13/viper@latest
  go get gopkg.in/yaml.v3@latest
  ```
- [ ] 添加日志库
  ```bash
  go get github.com/sirupsen/logrus@latest
  ```
- [ ] 执行 `go mod tidy`

#### Task 1.3: 设置 PostgreSQL 数据库
- [ ] 安装 PostgreSQL (≥14) 和 TimescaleDB 扩展
  ```bash
  # Ubuntu/Debian
  sudo apt-get install postgresql-14 postgresql-14-timescaledb

  # macOS
  brew install postgresql@14 timescaledb
  ```
- [ ] 创建数据库
  ```sql
  CREATE DATABASE microsegment;
  CREATE USER microsegment_user WITH PASSWORD 'secret';
  GRANT ALL PRIVILEGES ON DATABASE microsegment TO microsegment_user;
  ```
- [ ] 启用 TimescaleDB 扩展
  ```sql
  \c microsegment
  CREATE EXTENSION IF NOT EXISTS timescaledb;
  ```

#### Task 1.4: 创建数据库迁移脚本
- [ ] 创建 `migrations/001_initial_schema.up.sql`
  - [ ] flows 表定义 (TimescaleDB Hypertable)
  - [ ] policies 表定义
  - [ ] agents 表定义
  - [ ] events 表定义
  - [ ] 所有索引定义
  - [ ] 触发器定义
- [ ] 创建 `migrations/001_initial_schema.down.sql` (回滚脚本)
- [ ] 安装迁移工具
  ```bash
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  ```
- [ ] 执行迁移
  ```bash
  migrate -path migrations -database "postgres://user:pass@localhost:5432/microsegment?sslmode=disable" up
  ```
- [ ] 验证表创建成功
  ```sql
  \dt
  SELECT * FROM timescaledb_information.hypertables;
  ```

---

### Phase 2: 存储层实现 (2 天)

#### Task 2.1: 实现 PostgreSQL 连接管理
- [ ] 创建 `pkg/storage/postgres.go`
- [ ] 实现 `NewPostgresDB(config *DBConfig) (*sql.DB, error)`
  - [ ] 连接池配置 (max_open_conns, max_idle_conns)
  - [ ] 连接超时和重试机制
  - [ ] 健康检查 (Ping)
- [ ] 实现 `Close()` 优雅关闭
- [ ] 单元测试 (使用 testcontainers)

#### Task 2.2: 实现 FlowStorage
- [ ] 创建 `pkg/storage/flow_storage.go`
- [ ] 定义 `FlowStorage` 接口
  ```go
  type FlowStorage interface {
      BatchSaveFlowEvents(events []*pb.FlowEvent) error
      QueryFlows(ctx context.Context, query *FlowQuery) ([]*Flow, int, error)
      GetFlowSummary(ctx context.Context, req *pb.FlowSummaryRequest) (*pb.FlowSummary, error)
      GetDependencies(ctx context.Context, start, end time.Time) ([]*Dependency, error)
  }
  ```
- [ ] 实现 `PostgresFlowStorage` 结构体
- [ ] 实现 `BatchSaveFlowEvents()` (批量插入优化)
  - [ ] 使用事务
  - [ ] 使用 `COPY` 协议或批量 INSERT
- [ ] 实现 `QueryFlows()` (支持复杂过滤)
  - [ ] 动态 SQL 构建
  - [ ] 时间范围过滤
  - [ ] IP/协议/状态过滤
  - [ ] JSONB 标签查询
  - [ ] 分页和排序
- [ ] 实现 `GetFlowSummary()` (聚合查询)
  - [ ] 使用物化视图加速
- [ ] 实现 `GetDependencies()` (依赖关系分析)
  - [ ] 按标签分组
  - [ ] 聚合流量数据
- [ ] 单元测试 (使用 mock DB)
- [ ] 集成测试 (真实 PostgreSQL)

#### Task 2.3: 实现 PolicyStorage
- [ ] 创建 `pkg/storage/policy_storage.go`
- [ ] 定义 `PolicyStorage` 接口
- [ ] 实现 CRUD 操作
  - [ ] `CreatePolicy(policy *Policy) error`
  - [ ] `GetAllPolicies(ctx context.Context) ([]*Policy, uint64, error)`
  - [ ] `GetPolicy(ruleID int) (*Policy, error)`
  - [ ] `UpdatePolicy(policy *Policy) error`
  - [ ] `DeletePolicy(ruleID int) error`
- [ ] 实现版本管理
  - [ ] 自动递增版本号 (使用触发器)
  - [ ] 获取当前版本 `GetPolicyVersion() (uint64, error)`
- [ ] 实现策略统计存储
  - [ ] `SavePolicyStats(ctx context.Context, report *pb.PolicyStatsReport) error`
- [ ] 单元测试
- [ ] 集成测试

#### Task 2.4: 实现 AgentStorage
- [ ] 创建 `pkg/storage/agent_storage.go`
- [ ] 定义 `AgentStorage` 接口
- [ ] 实现 Agent 管理
  - [ ] `RegisterAgent(ctx context.Context, req *pb.RegisterRequest) error`
  - [ ] `UpdateHeartbeat(ctx context.Context, agentID string, metrics *pb.AgentMetrics) error`
  - [ ] `UpdateAgentStatus(ctx context.Context, req *pb.StatusReport) error`
  - [ ] `UnregisterAgent(ctx context.Context, agentID string) error`
  - [ ] `GetAgent(agentID string) (*Agent, error)`
  - [ ] `ListAgents(status string) ([]*Agent, error)`
- [ ] 实现在线检测
  - [ ] `GetOnlineAgents() ([]*Agent, error)` (last_heartbeat < 30s)
- [ ] 单元测试
- [ ] 集成测试

#### Task 2.5: 实现 EventStorage (审计日志)
- [ ] 创建 `pkg/storage/event_storage.go`
- [ ] 定义 `EventStorage` 接口
- [ ] 实现事件记录
  - [ ] `LogEvent(eventType, source, agentID, message string, metadata map[string]interface{}) error`
- [ ] 实现事件查询
  - [ ] `QueryEvents(query *EventQuery) ([]*Event, error)`
- [ ] 单元测试

---

### Phase 3: gRPC 服务层实现 (2-3 天)

#### Task 3.1: 实现 FlowService
- [ ] 创建 `pkg/grpc/flow_service.go`
- [ ] 实现 `NewFlowService(storage FlowStorage) *FlowService`
- [ ] 实现 `ReportFlowEvents(stream pb.FlowService_ReportFlowEventsServer) error`
  - [ ] 批量接收流事件 (客户端流)
  - [ ] 批次大小控制 (1000 条/批)
  - [ ] 错误处理和日志
  - [ ] 返回接收统计
- [ ] 实现 `QueryFlows(ctx, req) (*pb.FlowQueryResponse, error)`
  - [ ] 参数验证
  - [ ] 调用 FlowStorage.QueryFlows()
  - [ ] 结果转换
- [ ] 实现 `GetFlowSummary(ctx, req) (*pb.FlowSummary, error)`
- [ ] 单元测试 (mock storage)

#### Task 3.2: 实现 PolicyService
- [ ] 创建 `pkg/grpc/policy_service.go`
- [ ] 实现 `NewPolicyService(storage PolicyStorage) *PolicyService`
- [ ] 实现订阅者管理
  - [ ] `subscribers map[string]chan *pb.PolicyUpdate`
  - [ ] 添加/删除订阅者
- [ ] 实现 `SyncPolicies(ctx, req) (*pb.SyncResponse, error)`
  - [ ] 返回完整策略列表
  - [ ] 返回当前版本号
- [ ] 实现 `SubscribePolicies(req, stream) error` (服务器流)
  - [ ] 注册订阅
  - [ ] 首次发送完整策略
  - [ ] 持续监听更新
  - [ ] 断开连接清理
- [ ] 实现 `BroadcastPolicyUpdate(update *pb.PolicyUpdate)`
  - [ ] 向所有订阅者广播
  - [ ] 处理 channel 满的情况
- [ ] 实现 `ReportPolicyStats(ctx, req) (*pb.ReportResponse, error)`
- [ ] 单元测试

#### Task 3.3: 实现 AgentService
- [ ] 创建 `pkg/grpc/agent_service.go`
- [ ] 实现 `NewAgentService(storage AgentStorage, mgr *AgentManager) *AgentService`
- [ ] 实现 `RegisterAgent(ctx, req) (*pb.RegisterResponse, error)`
  - [ ] 保存 Agent 信息
  - [ ] 返回 Server 配置
  - [ ] 记录审计日志
- [ ] 实现 `Heartbeat(ctx, req) (*pb.HeartbeatResponse, error)`
  - [ ] 更新最后心跳时间
  - [ ] 保存 Agent 指标
  - [ ] 返回待下发命令
- [ ] 实现 `ReportStatus(ctx, req) (*pb.StatusResponse, error)`
  - [ ] 更新 Agent 状态
  - [ ] 记录审计日志
- [ ] 实现 `UnregisterAgent(ctx, req) (*pb.UnregisterResponse, error)`
  - [ ] 标记 Agent 已注销
  - [ ] 清理订阅
- [ ] 单元测试

#### Task 3.4: 实现 gRPC Server 入口
- [ ] 创建 `pkg/grpc/server.go`
- [ ] 实现 `NewGRPCServer(config *Config, services ...) *grpc.Server`
  - [ ] 创建 gRPC Server
  - [ ] 注册所有服务
  - [ ] 配置拦截器 (日志、认证、指标)
- [ ] 实现 `Start(addr string) error`
  - [ ] 监听 TCP 端口
  - [ ] 优雅启动
- [ ] 实现 `GracefulStop()`
  - [ ] 优雅关闭
  - [ ] 等待现有连接完成
- [ ] 单元测试

#### Task 3.5: 实现 gRPC 拦截器
- [ ] 创建 `pkg/grpc/interceptor.go`
- [ ] 实现日志拦截器
  - [ ] 记录所有 RPC 调用
  - [ ] 记录耗时
- [ ] 实现错误处理拦截器
  - [ ] 统一错误响应格式
- [ ] 实现指标拦截器 (可选)
  - [ ] 记录请求计数
  - [ ] 记录延迟分布

---

### Phase 4: HTTP API 层实现 (1-2 天)

#### Task 4.1: 实现 HTTP Server 基础
- [ ] 创建 `pkg/api/server.go`
- [ ] 实现 `NewHTTPServer(config *Config) *Server`
  - [ ] 创建 Gin Engine
  - [ ] 配置中间件
- [ ] 实现 `Start() error`
  - [ ] 监听 HTTP 端口
- [ ] 实现 `GracefulStop()`
  - [ ] 优雅关闭
- [ ] 单元测试

#### Task 4.2: 实现路由和中间件
- [ ] 创建 `pkg/api/router.go`
- [ ] 实现 `setupRoutes(engine *gin.Engine, handlers ...)`
  - [ ] 定义所有路由
  - [ ] 分组管理 (v1)
- [ ] 创建 `pkg/api/middleware.go`
- [ ] 实现 CORS 中间件
- [ ] 实现日志中间件
- [ ] 实现错误处理中间件
- [ ] 实现认证中间件 (可选)

#### Task 4.3: 实现 Flow Handlers
- [ ] 创建 `pkg/api/handlers/flow.go`
- [ ] 实现 `NewFlowHandler(storage FlowStorage) *FlowHandler`
- [ ] 实现 `ListFlows(c *gin.Context)`
  - [ ] GET /api/v1/flows
  - [ ] 解析查询参数
  - [ ] 调用 FlowStorage
  - [ ] 返回 JSON
- [ ] 实现 `GetFlow(c *gin.Context)`
  - [ ] GET /api/v1/flows/:id
- [ ] 实现 `GetFlowSummary(c *gin.Context)`
  - [ ] GET /api/v1/flows/summary
- [ ] 实现 `GetDependencies(c *gin.Context)`
  - [ ] GET /api/v1/dependencies
- [ ] 单元测试 (HTTP 测试)

#### Task 4.4: 实现 Policy Handlers
- [ ] 创建 `pkg/api/handlers/policy.go`
- [ ] 实现 `NewPolicyHandler(storage PolicyStorage) *PolicyHandler`
- [ ] 实现 CRUD 端点
  - [ ] POST /api/v1/policies (创建)
  - [ ] GET /api/v1/policies (列表)
  - [ ] GET /api/v1/policies/:id (详情)
  - [ ] PUT /api/v1/policies/:id (更新)
  - [ ] DELETE /api/v1/policies/:id (删除)
- [ ] 实现输入验证
- [ ] 单元测试

#### Task 4.5: 实现 Agent Handlers
- [ ] 创建 `pkg/api/handlers/agent.go`
- [ ] 实现 `NewAgentHandler(storage AgentStorage) *AgentHandler`
- [ ] 实现 `ListAgents(c *gin.Context)`
  - [ ] GET /api/v1/agents
  - [ ] 支持状态过滤
- [ ] 实现 `GetAgent(c *gin.Context)`
  - [ ] GET /api/v1/agents/:id
  - [ ] 返回详细信息和指标
- [ ] 单元测试

#### Task 4.6: 实现健康检查 Handler
- [ ] 创建 `pkg/api/handlers/health.go`
- [ ] 实现 `GetHealth(c *gin.Context)`
  - [ ] GET /api/v1/health
  - [ ] 检查数据库连接
  - [ ] 检查 gRPC Server 状态
  - [ ] 返回健康状态
- [ ] 单元测试

---

### Phase 5: 业务逻辑层实现 (1 天)

#### Task 5.1: 实现 AgentManager
- [ ] 创建 `pkg/manager/agent_manager.go`
- [ ] 实现 `NewAgentManager(storage AgentStorage) *AgentManager`
- [ ] 实现 `OnAgentRegistered(agentID string)`
  - [ ] 记录事件
  - [ ] 通知监控系统
- [ ] 实现 `OnAgentUnregistered(agentID string)`
- [ ] 实现 `GetPendingCommands(agentID string) []*pb.AgentCommand`
  - [ ] 返回待下发命令队列
- [ ] 实现 `SendCommand(agentID string, cmd *pb.AgentCommand)`
  - [ ] 添加命令到队列
- [ ] 单元测试

#### Task 5.2: 实现 PolicyDistributor
- [ ] 创建 `pkg/manager/policy_distributor.go`
- [ ] 实现 `NewPolicyDistributor(policyService *PolicyService)`
- [ ] 实现 `OnPolicyCreated(policy *pb.Policy)`
  - [ ] 广播 POLICY_UPDATE_ADD
- [ ] 实现 `OnPolicyUpdated(policy *pb.Policy)`
  - [ ] 广播 POLICY_UPDATE_MODIFY
- [ ] 实现 `OnPolicyDeleted(ruleID int)`
  - [ ] 广播 POLICY_UPDATE_DELETE
- [ ] 单元测试

#### Task 5.3: 实现 HealthMonitor (可选)
- [ ] 创建 `pkg/manager/health_monitor.go`
- [ ] 实现 `NewHealthMonitor(storage AgentStorage)`
- [ ] 实现 `Start()` 后台监控
  - [ ] 定期检查 Agent 心跳
  - [ ] 标记超时 Agent 为 UNHEALTHY
- [ ] 实现 `Stop()` 停止监控
- [ ] 单元测试

---

### Phase 6: 配置和入口实现 (0.5 天)

#### Task 6.1: 实现配置管理
- [ ] 创建 `pkg/config/config.go`
- [ ] 定义配置结构体
  ```go
  type Config struct {
      Server   ServerConfig
      Database DBConfig
      Logging  LogConfig
      Retention RetentionConfig
  }
  ```
- [ ] 创建 `pkg/config/loader.go`
- [ ] 实现 `LoadConfig(path string) (*Config, error)`
  - [ ] 支持 YAML 文件
  - [ ] 支持环境变量覆盖
  - [ ] 提供默认值
- [ ] 单元测试

#### Task 6.2: 实现 main.go 入口
- [ ] 创建 `cmd/main.go`
- [ ] 实现配置加载
- [ ] 实现组件初始化顺序
  ```go
  1. 加载配置
  2. 初始化日志
  3. 连接数据库
  4. 创建 Storage 层
  5. 创建 Manager 层
  6. 创建 gRPC Server
  7. 创建 HTTP Server
  8. 启动所有服务
  9. 等待信号 (SIGINT/SIGTERM)
  10. 优雅关闭
  ```
- [ ] 实现信号处理
  - [ ] 监听 SIGINT, SIGTERM
  - [ ] 优雅关闭所有组件
- [ ] 实现启动日志
  ```
  2025-11-04 10:00:00 INFO Starting microsegment-server v1.0.0
  2025-11-04 10:00:01 INFO Database connected: postgres://localhost:5432/microsegment
  2025-11-04 10:00:02 INFO gRPC Server listening on :9090
  2025-11-04 10:00:02 INFO HTTP Server listening on :8080
  2025-11-04 10:00:02 INFO Server ready
  ```

#### Task 6.3: 创建配置文件示例
- [ ] 创建 `config/server.yaml.example`
  - [ ] 完整的配置示例
  - [ ] 注释说明每个字段
- [ ] 创建 `config/database.yaml.example`
- [ ] 创建环境变量示例 `.env.example`

---

### Phase 7: 测试和验证 (1-2 天)

#### Task 7.1: 集成测试
- [ ] 创建 `pkg/integration_test.go`
- [ ] 测试 gRPC Server 启动
- [ ] 测试 HTTP Server 启动
- [ ] 测试数据库连接
- [ ] 测试 Agent 注册流程
- [ ] 测试流事件上报
- [ ] 测试策略同步
- [ ] 测试全局流量查询

#### Task 7.2: 端到端测试
- [ ] 启动 Server
- [ ] 使用 grpcurl 测试 gRPC 端点
  ```bash
  grpcurl -plaintext localhost:9090 list
  grpcurl -plaintext localhost:9090 microsegment.agent.AgentService/RegisterAgent
  ```
- [ ] 使用 curl 测试 HTTP 端点
  ```bash
  curl http://localhost:8080/api/v1/health
  curl http://localhost:8080/api/v1/agents
  curl http://localhost:8080/api/v1/flows
  ```
- [ ] 验证数据库数据
  ```sql
  SELECT COUNT(*) FROM flows;
  SELECT * FROM agents;
  SELECT * FROM policies;
  ```

#### Task 7.3: 性能测试
- [ ] 创建 `benchmark/flow_ingestion_test.go`
- [ ] 测试流事件批量写入性能
  - [ ] 目标: 10,000 events/s
  - [ ] 工具: ghz (gRPC benchmarking)
- [ ] 测试全局查询性能
  - [ ] 目标: <100ms for 1000 records
  - [ ] 工具: ab (Apache Bench)
- [ ] 测试并发 Agent 连接
  - [ ] 目标: 1000+ concurrent agents
- [ ] 记录性能基准

#### Task 7.4: 负载测试 (可选)
- [ ] 使用 k6 进行 HTTP 负载测试
- [ ] 使用 ghz 进行 gRPC 负载测试
- [ ] 监控资源使用 (CPU、内存、磁盘 I/O)
- [ ] 调优数据库配置

---

### Phase 8: 文档和打包 (0.5 天)

#### Task 8.1: 编写 README
- [ ] 创建 `src/server/README.md`
- [ ] 简介和功能说明
- [ ] 快速开始指南
- [ ] 配置说明
- [ ] API 文档链接
- [ ] 故障排查

#### Task 8.2: 创建 Makefile
- [ ] 创建 `src/server/Makefile`
- [ ] 实现 `make build` (编译二进制)
- [ ] 实现 `make test` (运行测试)
- [ ] 实现 `make docker` (构建 Docker 镜像)
- [ ] 实现 `make migrate-up` (执行数据库迁移)
- [ ] 实现 `make migrate-down` (回滚迁移)
- [ ] 实现 `make clean` (清理构建产物)

#### Task 8.3: 构建和打包
- [ ] 编译二进制文件
  ```bash
  go build -o microsegment-server ./cmd
  ```
- [ ] 测试二进制运行
  ```bash
  ./microsegment-server --config config/server.yaml
  ```
- [ ] 创建 Dockerfile (将在提案 5 中详细实现)

#### Task 8.4: 提交代码
- [ ] Git 提交所有代码
  ```bash
  git add src/server/
  git commit -m "feat: Implement Server component

  - Add gRPC services (FlowService, PolicyService, AgentService)
  - Add PostgreSQL storage layer
  - Add HTTP API endpoints
  - Add database migrations
  - Add integration tests

  🤖 Generated with Claude Code
  Co-Authored-By: Claude <noreply@anthropic.com>"
  ```
- [ ] 更新 CHANGELOG.md
- [ ] 更新项目 README.md

---

## 📊 任务依赖关系

```
Phase 1 (项目初始化)
    ↓
Phase 2 (存储层) ← 必须先完成
    ↓
    ├─→ Phase 3 (gRPC 服务层)
    └─→ Phase 4 (HTTP API 层)
         ↓
    Phase 5 (业务逻辑层)
         ↓
    Phase 6 (配置和入口)
         ↓
    Phase 7 (测试验证)
         ↓
    Phase 8 (文档打包)
```

**可并行任务**:
- Phase 3 和 Phase 4 (在 Phase 2 完成后)
- Task 7.1, 7.2, 7.3 (可同时进行)

---

## ✅ 验收标准

### 功能验收
- [ ] Server 成功编译为 `microsegment-server` 可执行文件
- [ ] gRPC Server 监听 :9090,可接受 Agent 连接
- [ ] HTTP Server 监听 :8080,可接受 API 请求
- [ ] PostgreSQL 数据库初始化成功,所有表创建
- [ ] Agent 可成功注册和发送心跳
- [ ] 流事件可成功上报并存储
- [ ] 策略可成功同步和订阅
- [ ] 全局流量查询 API 正常工作
- [ ] 依赖关系分析 API 正常工作

### 质量验收
- [ ] 单元测试覆盖率 > 70%
- [ ] 集成测试通过
- [ ] 性能测试达标 (10K events/s, <100ms 查询)
- [ ] 代码通过 golint 检查
- [ ] 无明显内存泄漏

### 文档验收
- [ ] README 完整
- [ ] API 文档完整
- [ ] 配置示例文件完整
- [ ] 数据库 schema 文档完整

---

## 🚨 风险和缓解

### 风险 1: PostgreSQL 性能瓶颈
**影响**: 高
**概率**: 中
**缓解**:
- 使用 TimescaleDB 时序优化
- 创建合理索引
- 使用批量插入
- 使用物化视图

### 风险 2: gRPC 流式传输复杂度
**影响**: 中
**概率**: 中
**缓解**:
- 参考 gRPC 官方示例
- 充分测试错误场景
- 实现重连机制

### 风险 3: 数据库迁移失败
**影响**: 高
**概率**: 低
**缓解**:
- 提供回滚脚本
- 在测试环境充分测试
- 备份数据

---

## 📝 实施笔记

### 性能优化建议
1. **批量插入**: 使用 PostgreSQL COPY 或批量 INSERT
2. **连接池**: 合理配置数据库连接池大小
3. **索引优化**: 为高频查询字段创建索引
4. **查询缓存**: 对热点查询使用 Redis 缓存

### 安全建议
1. **TLS 支持**: gRPC 和 HTTP 都支持 TLS
2. **认证**: 实现 Agent 认证机制 (Token/mTLS)
3. **SQL 注入防护**: 使用参数化查询
4. **输入验证**: 严格验证所有输入

---

**任务清单创建**: ✅
**预计完成时间**: 7-10 天
**下一步**: 开始实施 Phase 1

# Flow 数据收集 API - 实施任务清单

## Phase 1: eBPF 数据平面 Flow 收集（Week 1）

### Task 1.1: 定义 Flow 事件数据结构
- [ ] 创建 `src/bpf/headers/flow_types.h`
- [ ] 定义 `struct flow_event` 结构体
- [ ] 定义 `enum flow_event_type` (NEW/UPDATE/CLOSED)
- [ ] 编写单元测试验证结构体大小和对齐

**验收标准**：
- 结构体大小为 48 字节（packed）
- 字段对齐正确，无填充问题

### Task 1.2: 实现 Ring Buffer Map
- [ ] 在 `tc_microsegment.bpf.c` 中定义 `flow_events` Ring Buffer
- [ ] 设置容量为 256KB
- [ ] 实现 `push_flow_event()` 辅助函数
- [ ] 处理 Ring Buffer 满的情况（丢弃事件并计数）

**验收标准**：
- Ring Buffer 成功创建并可通过 `bpftool map show` 查看
- 能够成功 reserve 和 submit 事件

### Task 1.3: 集成 Flow 事件推送到 TC 程序
- [ ] 修改 `tc_microsegment.bpf.c` 主函数
- [ ] 在新连接建立时推送 FLOW_NEW 事件
- [ ] 在连接关闭时（FIN/RST）推送 FLOW_CLOSED 事件
- [ ] 可选：周期性推送 FLOW_UPDATE 事件（每 10 秒）

**验收标准**：
- 测试环境中能够通过 `bpftool prog tracelog` 看到事件推送日志
- eBPF 程序仍能通过 verifier 验证

### Task 1.4: 编译和加载测试
- [ ] 更新 Makefile 编译新的 eBPF 代码
- [ ] 加载 eBPF 程序到测试接口
- [ ] 使用 `bpftool map dump` 验证 Ring Buffer
- [ ] 发送测试流量验证事件生成

**验收标准**：
- eBPF 程序加载成功
- 能够看到 Ring Buffer 中有数据

---

## Phase 2: Go 控制平面 Flow Collector（Week 2）

### Task 2.1: 创建 Flow 包结构
- [ ] 创建 `src/agent/pkg/flow/` 目录
- [ ] 创建 `types.go` 定义 Flow 数据结构
- [ ] 创建 `collector.go` 实现 Collector
- [ ] 创建 `storage.go` 实现 Storage 接口

**验收标准**：
- 包结构符合 Go 命名约定
- 所有类型定义有完整的 godoc 注释

### Task 2.2: 实现 Flow Collector
- [ ] 实现 `NewCollector()` 构造函数
- [ ] 实现 `Start()` 启动收集循环
- [ ] 实现 `collectLoop()` 从 Ring Buffer 读取
- [ ] 实现 `parseFlowEvent()` 解析二进制数据
- [ ] 实现 `enrichWithLabels()` 标签增强

**验收标准**：
- Collector 能够成功读取 Ring Buffer 事件
- 事件解析正确，所有字段值符合预期
- 能够正确关联工作负载标签

### Task 2.3: 实现 SQLite 存储
- [ ] 实现 `NewSQLiteStorage()` 构造函数
- [ ] 实现 `initSchema()` 创建 flows 表
- [ ] 实现 `SaveFlow()` 插入流记录
- [ ] 实现 `GetFlow()` 查询单条记录
- [ ] 创建必要的索引（timestamp, source_ip, protocol）

**验收标准**：
- 数据库表创建成功
- Flow 数据能够正确持久化
- 索引创建正确（通过 `EXPLAIN QUERY PLAN` 验证）

### Task 2.4: 集成到 Agent 主程序
- [ ] 修改 `cmd/main.go` 初始化 Flow Collector
- [ ] 打开 eBPF Ring Buffer Reader
- [ ] 启动 Collector goroutine
- [ ] 实现优雅关闭（context cancel）

**验收标准**：
- Agent 启动时自动启动 Flow Collector
- Agent 关闭时 Collector 正确清理资源

### Task 2.5: 单元测试
- [ ] 编写 `collector_test.go` 测试事件解析
- [ ] 编写 `storage_test.go` 测试 CRUD 操作
- [ ] 编写集成测试验证端到端流程
- [ ] 覆盖率达到 80%+

**验收标准**：
- 所有测试通过：`go test ./pkg/flow/...`
- 覆盖率：`go test -cover ./pkg/flow/...`

---

## Phase 3: Flow 查询 API（Week 3）

### Task 3.1: 实现 Flow 查询逻辑
- [ ] 在 `storage.go` 中实现 `ListFlows(query *FlowQuery)`
- [ ] 支持时间范围过滤
- [ ] 支持 IP/端口/协议过滤
- [ ] 支持标签选择器过滤（JSON 查询）
- [ ] 实现分页（LIMIT/OFFSET）
- [ ] 实现排序（按时间、字节数、包数）

**验收标准**：
- 查询 1000 条记录响应时间 < 100ms
- 所有过滤条件正确工作

### Task 3.2: 实现 Flow API 端点
- [ ] 创建 `src/agent/pkg/api/handlers/flow_handler.go`
- [ ] 实现 `GET /api/v1/flows` 端点
  - 解析查询参数
  - 调用 Storage.ListFlows()
  - 返回分页结果
- [ ] 实现 `GET /api/v1/flows/:id` 端点
- [ ] 添加输入验证和错误处理

**验收标准**：
- API 返回正确的 JSON 格式
- 错误情况返回适当的 HTTP 状态码

### Task 3.3: 实现 Flow Summary API
- [ ] 在 `storage.go` 中实现 `GetSummary()`
- [ ] 聚合查询：总流数、总包数、总字节数
- [ ] 按状态分组：ACTIVE/CLOSED
- [ ] 按动作分组：ALLOW/DENY
- [ ] 实现 `GET /api/v1/flows/summary` 端点

**验收标准**：
- Summary 统计数据准确
- 查询响应时间 < 50ms

### Task 3.4: 实现应用依赖关系 API
- [ ] 创建 `aggregator.go` 实现聚合逻辑
- [ ] 实现 `GetDependencies()` 按标签分组 Flow
- [ ] 计算工作负载间的通信关系
- [ ] 实现 `GET /api/v1/flows/dependencies` 端点

**验收标准**：
- 依赖关系数据正确反映工作负载通信
- 前端能够基于数据渲染 ADM 图

### Task 3.5: 实现 Top Talkers API
- [ ] 在 `storage.go` 中实现 `GetTopTalkers()`
- [ ] 按字节数排序查询 Top N 源 IP
- [ ] 按字节数排序查询 Top N 目标 IP
- [ ] 实现 `GET /api/v1/flows/top-talkers` 端点

**验收标准**：
- Top Talkers 数据准确
- 支持可配置的 N 值（默认 10）

### Task 3.6: API 文档和测试
- [ ] 编写 API 文档（OpenAPI/Swagger）
- [ ] 编写 API 集成测试（httptest）
- [ ] 使用 curl/Postman 进行手工测试

**验收标准**：
- API 文档完整且准确
- 所有端点有对应的集成测试

---

## Phase 4: 实时 Flow 推送（Week 4）

### Task 4.1: 实现 WebSocket Hub
- [ ] 创建 `websocket.go` 实现 WebSocket 管理
- [ ] 实现 `Hub` 结构体管理所有客户端连接
- [ ] 实现 `Register()` 注册新客户端
- [ ] 实现 `Unregister()` 移除断开的客户端
- [ ] 实现 `Broadcast()` 广播消息到所有客户端

**验收标准**：
- Hub 能够管理多个并发客户端
- 客户端断开时正确清理资源

### Task 4.2: 实现 WebSocket 端点
- [ ] 在 `api/handlers/` 中创建 `flow_stream_handler.go`
- [ ] 实现 `GET /api/v1/flows/stream` WebSocket 升级
- [ ] 处理客户端连接/断开事件
- [ ] 实现心跳机制（ping/pong）

**验收标准**：
- 前端能够成功建立 WebSocket 连接
- 连接断开后能够自动重连

### Task 4.3: 集成实时推送到 Collector
- [ ] 修改 `collector.go` 在收集到 Flow 后调用 `wsHub.Broadcast()`
- [ ] 实现消息序列化（JSON）
- [ ] 添加消息缓冲机制（避免阻塞 Collector）

**验收标准**：
- 新 Flow 事件在 500ms 内推送到前端
- Collector 不会因 WebSocket 慢客户端而阻塞

### Task 4.4: 实现流事件过滤
- [ ] 支持客户端订阅特定类型的 Flow（按标签、协议等）
- [ ] 在 Hub 中实现过滤逻辑
- [ ] 只向匹配的客户端推送事件

**验收标准**：
- 客户端只接收感兴趣的 Flow 事件
- 过滤不影响推送性能

### Task 4.5: 前端集成测试
- [ ] 编写 WebSocket 客户端测试代码（JavaScript）
- [ ] 验证实时接收 Flow 事件
- [ ] 验证重连逻辑
- [ ] 性能测试：1000 flows/s 推送

**验收标准**：
- 前端能够实时显示 Flow 数据
- 推送延迟 < 500ms

---

## Phase 5: 优化和生产准备（Week 5）

### Task 5.1: 性能优化
- [ ] 数据库查询优化（EXPLAIN ANALYZE）
- [ ] 添加查询结果缓存（LRU Cache）
- [ ] 实现批量插入 Flow（减少 DB 写入频率）
- [ ] 优化 Ring Buffer 大小（基于负载测试）

**验收标准**：
- 查询响应时间降低 50%
- Collector 能够处理 10,000 flows/s

### Task 5.2: 数据生命周期管理
- [ ] 实现自动清理旧 Flow 数据（默认保留 7 天）
- [ ] 添加定时任务（cron job）执行清理
- [ ] 实现数据归档功能（可选）
- [ ] 添加磁盘空间监控告警

**验收标准**：
- 旧数据自动删除
- 数据库大小保持在可控范围

### Task 5.3: 可观测性
- [ ] 添加 Prometheus 指标
  - flow_events_total（按事件类型）
  - flow_collector_errors_total
  - flow_storage_latency_seconds
  - websocket_clients_connected
- [ ] 添加结构化日志
- [ ] 实现健康检查端点

**验收标准**：
- 所有关键指标可通过 Prometheus 查询
- 日志包含足够的上下文信息

### Task 5.4: 错误处理和恢复
- [ ] Ring Buffer 读取失败重试机制
- [ ] 数据库连接池配置和重连
- [ ] WebSocket 客户端异常处理
- [ ] 添加熔断机制（Circuit Breaker）

**验收标准**：
- 组件失败时能够自动恢复
- 错误日志清晰可追溯

### Task 5.5: 文档和示例
- [ ] 编写 Flow API 使用文档
- [ ] 编写前端集成指南
- [ ] 提供 curl 示例脚本
- [ ] 录制演示视频

**验收标准**：
- 文档完整且易于理解
- 示例代码可直接运行

---

## Phase 6: 可选增强功能

### Task 6.1: 迁移到 InfluxDB（可选）
- [ ] 评估 InfluxDB vs SQLite 性能对比
- [ ] 实现 `InfluxDBStorage` 实现 Storage 接口
- [ ] 数据迁移脚本
- [ ] 性能基准测试

### Task 6.2: 流量异常检测（可选）
- [ ] 实现基线学习算法
- [ ] 检测异常流量模式（突发流量、DDoS 等）
- [ ] 添加告警机制

### Task 6.3: Flow 导出功能（可选）
- [ ] 支持导出 Flow 数据为 CSV/JSON
- [ ] 实现 NetFlow/IPFIX 格式导出
- [ ] 与 SIEM 系统集成

---

## 验收标准汇总

### 功能验收
- [ ] 能够从 eBPF 收集 Flow 事件并持久化到 SQLite
- [ ] 能够通过 REST API 查询和过滤 Flow 数据
- [ ] 能够通过 WebSocket 实时推送新 Flow 事件
- [ ] 前端能够基于 Flow API 渲染应用依赖地图
- [ ] 支持标签过滤查询（如："app=nginx"）

### 性能验收
- [ ] Flow Collector 处理能力：≥ 10,000 flows/s
- [ ] API 查询响应时间：< 100ms（1000 条记录）
- [ ] WebSocket 推送延迟：< 500ms
- [ ] 数据库查询优化：使用索引，EXPLAIN 验证

### 可靠性验收
- [ ] Ring Buffer 满时能够优雅降级（丢弃事件并记录）
- [ ] 数据库连接失败时能够自动重连
- [ ] WebSocket 客户端断开后能够自动清理
- [ ] Agent 重启后能够恢复 Flow 收集

### 可观测性验收
- [ ] 所有关键操作有 Prometheus 指标
- [ ] 所有错误有结构化日志
- [ ] 提供健康检查端点

---

## 依赖关系

```
Phase 1 (eBPF Flow Collection)
    └──> Task 1.1 → Task 1.2 → Task 1.3 → Task 1.4

Phase 2 (Go Collector)
    └──> Task 2.1 → Task 2.2 → Task 2.3 → Task 2.4 → Task 2.5
    (依赖 Phase 1 完成)

Phase 3 (Query API)
    └──> Task 3.1 → Task 3.2 → Task 3.3 → Task 3.4 → Task 3.5 → Task 3.6
    (依赖 Phase 2 完成)

Phase 4 (Real-time Push)
    └──> Task 4.1 → Task 4.2 → Task 4.3 → Task 4.4 → Task 4.5
    (依赖 Phase 2 完成，与 Phase 3 并行)

Phase 5 (Optimization)
    └──> Task 5.1 → Task 5.2 → Task 5.3 → Task 5.4 → Task 5.5
    (依赖 Phase 3 和 Phase 4 完成)

Phase 6 (Optional)
    └──> 独立任务，可随时开始
```

---

## 时间估算

| Phase | 任务数 | 预计工作量 | 依赖关系 |
|-------|-------|-----------|---------|
| Phase 1: eBPF Flow Collection | 4 | 5-7 天 | 无依赖 |
| Phase 2: Go Collector | 5 | 5-7 天 | 依赖 Phase 1 |
| Phase 3: Query API | 6 | 7-10 天 | 依赖 Phase 2 |
| Phase 4: Real-time Push | 5 | 5-7 天 | 依赖 Phase 2 |
| Phase 5: Optimization | 5 | 5-7 天 | 依赖 Phase 3+4 |
| Phase 6: Optional | 3 | 按需 | 独立 |

**总计**：约 4-5 周（不含可选功能）

---

## 测试策略

### 单元测试
- 每个 Go 包必须有 `*_test.go` 文件
- 覆盖率目标：≥ 80%
- 命令：`go test ./pkg/flow/... -cover`

### 集成测试
- 测试 eBPF → Go → Storage 端到端流程
- 测试 API 端点正确性
- 命令：`go test ./test/integration/... -tags=integration`

### 性能测试
- 使用 `go test -bench` 测试关键路径
- 使用 `ab` 或 `wrk` 测试 API 吞吐量
- 模拟 10,000 flows/s 负载

### 手工测试
- 使用 `curl` 测试 REST API
- 使用浏览器测试 WebSocket 连接
- 使用 `bpftool` 验证 eBPF 程序行为

---

## 风险和缓解

### 风险 1: eBPF 程序复杂度导致验证器失败
**缓解**：
- 将复杂逻辑拆分为多个小函数（内联）
- 避免循环，使用 `#pragma unroll`
- 使用 `bpftool prog load` 的 `-d` 参数调试

### 风险 2: Ring Buffer 事件丢失
**缓解**：
- 监控 Ring Buffer 丢弃计数
- 根据负载测试调整 Ring Buffer 大小
- 添加告警机制

### 风险 3: 数据库性能瓶颈
**缓解**：
- 创建适当的索引
- 实现批量插入
- 长期考虑迁移到 InfluxDB

### 风险 4: WebSocket 连接不稳定
**缓解**：
- 实现心跳机制
- 客户端自动重连
- 消息序号去重

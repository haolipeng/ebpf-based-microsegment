# 实施任务: gRPC 协议定义

**变更 ID**: `add-grpc-protocol-definitions`
**任务创建日期**: 2025-11-04
**预估总工作量**: 3-5 天

---

## 📋 任务清单

### Phase 1: 环境准备和工具安装 (0.5 天)

#### Task 1.1: 安装 Protocol Buffers 编译器
- [ ] 安装 protoc (≥ 3.20)
  ```bash
  # Ubuntu/Debian
  sudo apt-get install -y protobuf-compiler

  # macOS
  brew install protobuf

  # 验证安装
  protoc --version
  ```
- [ ] 安装 Go 插件
  ```bash
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

  # 验证安装
  which protoc-gen-go
  which protoc-gen-go-grpc
  ```

#### Task 1.2: 更新 Go 依赖
- [ ] 修改 `go.mod` 添加 gRPC 依赖
  ```go
  require (
      google.golang.org/grpc v1.60.0
      google.golang.org/protobuf v1.32.0
  )
  ```
- [ ] 执行 `go mod tidy`
- [ ] 验证依赖下载成功

#### Task 1.3: 创建项目目录结构
- [ ] 创建 `proto/` 目录
- [ ] 创建 `src/proto/` 输出目录
- [ ] 创建 `scripts/` 脚本目录

---

### Phase 2: Protocol Buffers 定义 (2 天)

#### Task 2.1: 定义 common.proto (0.5 天)
- [ ] 创建 `proto/common.proto`
- [ ] 定义枚举类型
  - [ ] Protocol (UNKNOWN=0, TCP=6, UDP=17, ICMP=1, ANY=255)
  - [ ] PolicyAction (UNKNOWN=0, ALLOW=1, DENY=2, LOG=3)
  - [ ] FlowEventType (UNKNOWN=0, NEW=1, UPDATE=2, CLOSED=3, TIMEOUT=4)
  - [ ] FlowDirection (UNKNOWN=0, INGRESS=1, EGRESS=2)
  - [ ] FlowState (UNKNOWN=0, ACTIVE=1, CLOSED=2, TIMEOUT=3)
- [ ] 定义通用消息
  - [ ] ReportResponse (success, message, accepted_count, rejected_count)
  - [ ] TimeRange (start_time, end_time)
- [ ] 添加完整注释和文档
- [ ] 编译验证: `protoc --proto_path=proto --go_out=src/proto proto/common.proto`

#### Task 2.2: 定义 flow.proto (0.75 天)
- [ ] 创建 `proto/flow.proto`
- [ ] 导入 `common.proto`
- [ ] 定义 FlowService 服务
  - [ ] ReportFlowEvents (stream FlowEvent → ReportResponse)
  - [ ] QueryFlows (FlowQuery → FlowQueryResponse)
  - [ ] GetFlowSummary (FlowSummaryRequest → FlowSummary)
- [ ] 定义消息类型
  - [ ] FlowEvent (48 字节优化)
    - fixed32 src_ip, dst_ip
    - uint32 src_port, dst_port
    - Protocol protocol
    - FlowEventType event_type
    - FlowDirection direction
    - uint64 packet_count, byte_count
    - fixed64 timestamp_ns
    - uint32 policy_id
    - PolicyAction policy_action
    - FlowState state
    - string agent_id
    - map<string, string> source_labels, dest_labels
  - [ ] FlowQuery (过滤参数)
  - [ ] FlowQueryResponse
  - [ ] Flow (完整流记录)
  - [ ] FlowSummaryRequest
  - [ ] FlowSummary
  - [ ] ProtocolStats
  - [ ] IPStats
- [ ] 添加注释
- [ ] 编译验证

#### Task 2.3: 定义 policy.proto (0.5 天)
- [ ] 创建 `proto/policy.proto`
- [ ] 导入 `common.proto`
- [ ] 定义 PolicyService 服务
  - [ ] SyncPolicies (SyncRequest → SyncResponse)
  - [ ] SubscribePolicies (SubscribeRequest → stream PolicyUpdate)
  - [ ] ReportPolicyStats (PolicyStatsReport → ReportResponse)
- [ ] 定义消息类型
  - [ ] SyncRequest (agent_id, last_sync_time, policy_version)
  - [ ] SyncResponse (policies, policy_version, server_time)
  - [ ] SubscribeRequest (agent_id, current_version)
  - [ ] PolicyUpdate (update_type, policy, policy_version)
  - [ ] PolicyUpdateType 枚举 (ADD=1, MODIFY=2, DELETE=3)
  - [ ] Policy (rule_id, src_ip, dst_ip, ports, protocol, action, priority, timestamps)
  - [ ] PolicyStatsReport
  - [ ] PolicyStats
- [ ] 添加注释
- [ ] 编译验证

#### Task 2.4: 定义 agent.proto (0.5 天)
- [ ] 创建 `proto/agent.proto`
- [ ] 定义 AgentService 服务
  - [ ] RegisterAgent (RegisterRequest → RegisterResponse)
  - [ ] Heartbeat (HeartbeatRequest → HeartbeatResponse)
  - [ ] ReportStatus (StatusReport → StatusResponse)
  - [ ] UnregisterAgent (UnregisterRequest → UnregisterResponse)
- [ ] 定义消息类型
  - [ ] RegisterRequest (agent_id, hostname, version, interface, ip_addresses, os, kernel_version, start_time, capabilities)
  - [ ] AgentCapabilities
  - [ ] RegisterResponse (success, message, server_version, server_time, config)
  - [ ] AgentConfig (heartbeat_interval, stats_interval, flow_batch_size, flow_batch_timeout)
  - [ ] HeartbeatRequest (agent_id, timestamp, metrics)
  - [ ] AgentMetrics (cpu_usage, memory_usage, packets, sessions, flows, policies)
  - [ ] HeartbeatResponse (healthy, message, commands)
  - [ ] AgentCommand (command_type, command_id, parameters)
  - [ ] AgentCommandType 枚举
  - [ ] StatusReport
  - [ ] AgentStatus 枚举
  - [ ] StatusResponse
  - [ ] UnregisterRequest/Response
- [ ] 添加注释
- [ ] 编译验证

---

### Phase 3: 代码生成和工具 (0.5 天)

#### Task 3.1: 创建 Makefile
- [ ] 创建 `Makefile` (在项目根目录)
- [ ] 实现 `make generate` 目标
- [ ] 实现 `make clean` 目标
- [ ] 实现 `make install-tools` 目标
- [ ] 实现 `make lint` 目标 (可选)
- [ ] 实现 `make help` 目标
- [ ] 测试所有 make 目标

#### Task 3.2: 创建代码生成脚本
- [ ] 创建 `scripts/generate-proto.sh`
- [ ] 添加工具检查逻辑
- [ ] 添加目录创建逻辑
- [ ] 添加 protoc 调用
- [ ] 添加错误处理
- [ ] 添加成功提示
- [ ] 赋予执行权限: `chmod +x scripts/generate-proto.sh`
- [ ] 测试脚本执行

#### Task 3.3: 生成 Go 代码
- [ ] 执行 `make generate` 或 `./scripts/generate-proto.sh`
- [ ] 验证生成的文件
  - [ ] `src/proto/common/common.pb.go`
  - [ ] `src/proto/flow/flow.pb.go`
  - [ ] `src/proto/flow/flow_grpc.pb.go`
  - [ ] `src/proto/policy/policy.pb.go`
  - [ ] `src/proto/policy/policy_grpc.pb.go`
  - [ ] `src/proto/agent/agent.pb.go`
  - [ ] `src/proto/agent/agent_grpc.pb.go`
- [ ] 检查生成的代码无编译错误
  ```bash
  cd src/proto
  go build ./...
  ```

---

### Phase 4: 测试和验证 (1 天)

#### Task 4.1: 创建单元测试 (0.5 天)
- [ ] 创建 `src/proto/common/common_test.go`
  - [ ] 测试枚举值正确性
  - [ ] 测试消息序列化/反序列化
- [ ] 创建 `src/proto/flow/flow_test.go`
  - [ ] 测试 FlowEvent 序列化
  - [ ] 测试 FlowEvent 大小 (应 ≤ 200 bytes)
  - [ ] 测试批量序列化性能
- [ ] 创建 `src/proto/policy/policy_test.go`
  - [ ] 测试 Policy 序列化
- [ ] 创建 `src/proto/agent/agent_test.go`
  - [ ] 测试 AgentInfo 序列化
- [ ] 运行所有测试: `go test ./src/proto/...`

#### Task 4.2: 创建集成测试示例 (0.25 天)
- [ ] 创建 `examples/grpc_client/main.go`
  - [ ] 连接 Server
  - [ ] 发送 FlowEvent 流
  - [ ] 接收 ReportResponse
- [ ] 创建 `examples/grpc_server/main.go`
  - [ ] 启动 gRPC 服务器
  - [ ] 实现 FlowService
  - [ ] 接收并打印事件
- [ ] 测试客户端-服务端通信
  ```bash
  # Terminal 1
  go run examples/grpc_server/main.go

  # Terminal 2
  go run examples/grpc_client/main.go
  ```

#### Task 4.3: 性能基准测试 (0.25 天)
- [ ] 创建 `src/proto/flow/flow_bench_test.go`
  - [ ] 基准测试: FlowEvent 序列化
  - [ ] 基准测试: FlowEvent 反序列化
  - [ ] 基准测试: 批量序列化 (1000 events)
- [ ] 运行基准测试
  ```bash
  go test -bench=. -benchmem ./src/proto/flow/
  ```
- [ ] 记录性能指标
  - 目标: 序列化 > 100K ops/s
  - 目标: 内存分配 < 1KB per event

---

### Phase 5: 文档和示例 (0.5 天)

#### Task 5.1: 创建 README
- [ ] 创建 `proto/README.md`
  - [ ] 简介
  - [ ] 目录结构说明
  - [ ] 代码生成指南
  - [ ] 使用示例
  - [ ] 贡献指南

#### Task 5.2: 创建 API 文档
- [ ] 创建 `docs/grpc-api.md`
  - [ ] FlowService 接口文档
  - [ ] PolicyService 接口文档
  - [ ] AgentService 接口文档
  - [ ] 消息格式说明
  - [ ] 错误码定义

#### Task 5.3: 创建使用示例
- [ ] 创建 `examples/README.md`
- [ ] 完善客户端示例代码
- [ ] 完善服务端示例代码
- [ ] 添加配置文件示例

---

### Phase 6: 规范增量和归档 (0.5 天)

#### Task 6.1: 创建规范增量
- [ ] 创建 `openspec/changes/add-grpc-protocol-definitions/spec/grpc-protocol.md`
  - [ ] ADDED 章节: gRPC 服务定义
  - [ ] ADDED 章节: Protocol Buffers 消息定义
  - [ ] ADDED 章节: 代码生成规范
  - [ ] 添加场景示例

#### Task 6.2: 更新项目文档
- [ ] 更新 `openspec/project.md`
  - [ ] 添加 gRPC 技术栈
  - [ ] 添加 proto 目录说明
  - [ ] 更新架构图

#### Task 6.3: 提交代码
- [ ] Git 提交所有 proto 文件
- [ ] Git 提交生成的代码
- [ ] Git 提交脚本和 Makefile
- [ ] Git 提交测试和示例
- [ ] Git 提交文档

---

## 📊 任务依赖关系

```
Task 1.1 → Task 1.2 → Task 1.3
                ↓
       Task 2.1 (common.proto)
                ↓
     ┌──────────┼──────────┐
     ↓          ↓          ↓
Task 2.2   Task 2.3   Task 2.4
(flow)     (policy)    (agent)
     └──────────┬──────────┘
                ↓
       Task 3.1 & 3.2 & 3.3
                ↓
       Task 4.1 & 4.2 & 4.3
                ↓
       Task 5.1 & 5.2 & 5.3
                ↓
       Task 6.1 & 6.2 & 6.3
```

**可并行任务**:
- Task 2.2, 2.3, 2.4 (在 common.proto 完成后)
- Task 3.1, 3.2 (可同时创建)
- Task 4.1, 4.2, 4.3 (可同时进行)
- Task 5.1, 5.2, 5.3 (可同时编写)

---

## ✅ 验收标准

### 代码质量
- [ ] 所有 proto 文件编译通过,无语法错误
- [ ] 生成的 Go 代码无编译错误
- [ ] 所有单元测试通过
- [ ] 代码覆盖率 > 80%

### 功能完整性
- [ ] 定义了 FlowService, PolicyService, AgentService 三个服务
- [ ] 所有关键消息类型已定义
- [ ] 枚举类型包含 UNKNOWN = 0 值

### 文档完整性
- [ ] proto 文件包含注释
- [ ] 提供 README 和 API 文档
- [ ] 提供可运行的示例代码

### 性能要求
- [ ] FlowEvent 序列化 > 100K ops/s
- [ ] FlowEvent 大小 < 200 bytes
- [ ] 内存分配 < 1KB per event

---

## 🚨 风险和缓解

### 风险 1: protoc 版本不兼容
**缓解**: 明确要求 protoc >= 3.20,在 README 中说明

### 风险 2: 字段编号冲突
**缓解**: 使用字段编号范围规划 (1-15 高频, 16+ 低频)

### 风险 3: 性能不达标
**缓解**: 使用 fixed32/fixed64 优化,benchmark 验证

---

## 📝 实施笔记

### 字段编号规划
- **1-15**: 高频字段 (1 字节 tag)
- **16-2047**: 中频字段 (2 字节 tag)
- **2048+**: 低频字段 (3+ 字节 tag)

### 保留字段编号
- 每个消息预留字段编号范围用于未来扩展
- 已删除字段使用 `reserved` 关键字

---

**任务清单创建**: ✅
**下一步**: 开始实施 Phase 1

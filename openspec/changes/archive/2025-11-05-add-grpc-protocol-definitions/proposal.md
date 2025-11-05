# 提案: 添加 gRPC 协议定义

**变更 ID**: `add-grpc-protocol-definitions`
**提案日期**: 2025-11-04
**状态**: 提案中
**优先级**: P0 (关键路径)
**预估工作量**: 3-5 天

---

## 📋 概述

定义 Agent-Server 架构所需的 gRPC 接口协议,作为从单体架构迁移到分布式架构的基础设施。本提案将创建完整的 Protocol Buffers 定义,用于 Agent 和 Server 之间的高效通信。

## 🎯 目标

1. **定义统一的通信协议** - 为 Agent 和 Server 建立标准化的数据交换格式
2. **支持高性能数据传输** - 使用 gRPC 和 Protocol Buffers 实现高吞吐量、低延迟通信
3. **保证类型安全** - 通过强类型定义避免序列化错误
4. **支持流式传输** - 为实时流事件上报和策略订阅提供 gRPC Stream 支持
5. **为未来扩展打基础** - 设计可扩展的接口,支持未来功能增强

## 💡 动机

### 当前架构限制

当前系统采用 **单体架构**:
- 所有组件运行在一个进程中
- 数据存储在本地 SQLite
- 无法跨节点查询流量数据
- 策略配置需要逐节点执行

### Agent-Server 架构需求

为支持 **10-10000 节点的大规模部署**,需要:
- Agent (节点上) ↔ Server (中心) 高效通信
- 流事件实时上报 (10K+ events/s)
- 策略集中下发和订阅
- Agent 健康监控和管理

### 为什么选择 gRPC?

| 特性 | gRPC | REST/HTTP | MQTT |
|-----|------|-----------|------|
| 性能 | ✅ 高 (Protobuf) | ⚠️ 中 (JSON) | ✅ 高 |
| 类型安全 | ✅ 强类型 | ❌ 弱类型 | ❌ 无类型 |
| 流式传输 | ✅ 双向流 | ⏸️ SSE 单向 | ✅ Pub/Sub |
| 工具链 | ✅ 丰富 | ✅ 丰富 | ⚠️ 较少 |
| 学习曲线 | ⚠️ 中等 | ✅ 低 | ⚠️ 中等 |

**结论**: gRPC 在性能、类型安全和流式传输方面最优,是 Agent-Server 通信的最佳选择。

## 🏗️ 核心设计

### 服务划分

定义 **3 个核心 gRPC 服务**:

#### 1. FlowService (流事件上报)
```protobuf
service FlowService {
  // 批量上报流事件 (Agent → Server)
  rpc ReportFlowEvents(stream FlowEvent) returns (ReportResponse);

  // 查询历史流量 (Server → Client, 用于同步)
  rpc QueryFlows(FlowQuery) returns (stream Flow);
}
```

#### 2. PolicyService (策略同步)
```protobuf
service PolicyService {
  // 拉取完整策略列表 (Agent → Server)
  rpc SyncPolicies(SyncRequest) returns (SyncResponse);

  // 订阅策略更新 (Server → Agent 推送)
  rpc SubscribePolicies(SubscribeRequest) returns (stream PolicyUpdate);

  // 上报策略执行统计 (Agent → Server)
  rpc ReportPolicyStats(PolicyStatsReport) returns (ReportResponse);
}
```

#### 3. AgentService (Agent 管理)
```protobuf
service AgentService {
  // Agent 注册 (启动时)
  rpc RegisterAgent(RegisterRequest) returns (RegisterResponse);

  // 心跳 (每10秒)
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);

  // Agent 状态上报
  rpc ReportStatus(StatusReport) returns (ReportResponse);
}
```

### 数据结构设计

**核心消息类型**:

```protobuf
// 流事件 (对应 eBPF struct flow_event)
message FlowEvent {
  // 5-tuple
  fixed32 src_ip = 1;
  fixed32 dst_ip = 2;
  uint32 src_port = 3;
  uint32 dst_port = 4;
  Protocol protocol = 5;

  // 事件元数据
  FlowEventType event_type = 6;
  FlowDirection direction = 7;

  // 流量统计
  uint64 packet_count = 8;
  uint64 byte_count = 9;
  uint64 timestamp_ns = 10;

  // 策略上下文
  uint32 policy_id = 11;
  PolicyAction policy_action = 12;
  FlowState state = 13;

  // 标签 (用于依赖分析)
  map<string, string> source_labels = 14;
  map<string, string> dest_labels = 15;
}

// 策略定义
message Policy {
  uint32 rule_id = 1;
  string src_ip = 2;
  string dst_ip = 3;
  uint32 src_port = 4;
  uint32 dst_port = 5;
  Protocol protocol = 6;
  PolicyAction action = 7;
  uint32 priority = 8;
  int64 created_at = 9;
  int64 updated_at = 10;
}

// Agent 信息
message AgentInfo {
  string agent_id = 1;
  string hostname = 2;
  string version = 3;
  string interface = 4;
  repeated string ip_addresses = 5;
  int64 start_time = 6;
}
```

### 通信模式

#### 模式 1: 批量上报 (Batch Reporting)
```
Agent                           Server
  │                               │
  ├─ FlowEvent 1 ────────────────>│
  ├─ FlowEvent 2 ────────────────>│
  ├─ FlowEvent 3 ────────────────>│
  │  ... (1000 条或 1 秒)           │
  ├─ CloseStream() ──────────────>│
  │<────── ReportResponse ─────────┤
  │  (ack: 1000 events received)  │
```

#### 模式 2: 策略订阅 (Policy Subscription)
```
Agent                           Server
  │                               │
  ├─ SubscribePolicies() ────────>│
  │<────── Policy 1 ───────────────┤ (初始策略)
  │<────── Policy 2 ───────────────┤
  │<────── Policy 3 ───────────────┤
  │                               │
  │  ... (策略更新时) ...          │
  │<────── PolicyUpdate ───────────┤ (新增/修改/删除)
  │                               │
```

#### 模式 3: 心跳 (Heartbeat)
```
Agent                           Server
  │                               │
  ├─ Heartbeat() ────────────────>│ (每10秒)
  │  {agent_id, status, metrics}  │
  │<────── HeartbeatResponse ─────┤
  │  {healthy: true}              │
```

## 📂 文件结构

```
proto/                          # 新增 protobuf 定义目录
├── flow.proto                  # 流事件相关定义
├── policy.proto                # 策略相关定义
├── agent.proto                 # Agent 管理定义
├── common.proto                # 通用类型和枚举
└── Makefile                    # 代码生成脚本

src/proto/                      # 生成的 Go 代码 (自动生成)
├── flow/
│   ├── flow.pb.go
│   └── flow_grpc.pb.go
├── policy/
│   ├── policy.pb.go
│   └── policy_grpc.pb.go
└── agent/
    ├── agent.pb.go
    └── agent_grpc.pb.go

scripts/
└── generate-proto.sh           # 代码生成脚本
```

## 🔧 技术要求

### 工具依赖

```bash
# 必需工具
- protoc (Protocol Buffers 编译器) >= 3.20
- protoc-gen-go (Go 代码生成器) >= 1.28
- protoc-gen-go-grpc (gRPC Go 代码生成器) >= 1.2

# 安装命令
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Go 依赖包

```go
// go.mod 新增依赖
google.golang.org/grpc v1.60.0
google.golang.org/protobuf v1.32.0
```

## ✅ 验收标准

### 功能验收
- [ ] 所有 .proto 文件编译通过,无语法错误
- [ ] 生成的 Go 代码可正常导入和使用
- [ ] 提供完整的代码生成脚本 (Makefile 或 shell 脚本)
- [ ] proto 文件包含完整的注释和文档

### 质量验收
- [ ] 所有消息类型使用正确的字段编号 (1-15 为常用字段)
- [ ] 枚举类型定义完整,包含未知值 (value 0)
- [ ] 使用 fixed32/fixed64 优化已知 4/8 字节字段
- [ ] 定义合理的 RPC 超时和重试策略

### 文档验收
- [ ] 提供 proto 文件使用文档
- [ ] 提供代码生成指南
- [ ] 提供接口调用示例

## 🔗 依赖

**前置依赖**: 无

**后续依赖**:
- add-server-component (需要实现 gRPC 服务端)
- refactor-agent-for-remote-reporting (需要实现 gRPC 客户端)

## 📊 风险评估

| 风险 | 影响 | 概率 | 缓解措施 |
|-----|------|------|---------|
| proto 格式版本不兼容 | 高 | 低 | 使用 proto3 语法,向后兼容 |
| 字段编号冲突 | 中 | 低 | 严格遵循字段编号规范 |
| 性能不达标 | 高 | 低 | 使用 fixed 类型优化,benchmark 验证 |
| 学习曲线陡峭 | 中 | 中 | 提供完整示例和文档 |

## 🚀 未来扩展

### 可选功能
- [ ] 支持 gRPC-Web (浏览器客户端)
- [ ] 添加 OpenAPI 文档生成 (grpc-gateway)
- [ ] 支持消息压缩 (gzip)
- [ ] 添加认证和 TLS 支持

---

**提案人**: Claude Code
**审批**: 待审批
**下一步**: 创建 design.md 详细设计文档

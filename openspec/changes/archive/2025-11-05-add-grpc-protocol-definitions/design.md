# 设计文档: gRPC 协议定义

**变更 ID**: `add-grpc-protocol-definitions`
**设计日期**: 2025-11-04
**版本**: 1.0

---

## 📐 详细设计

### 1. Protocol Buffers 定义

#### 1.1 common.proto - 通用类型定义

```protobuf
syntax = "proto3";

package microsegment.common;

option go_package = "github.com/ebpf-microsegment/src/proto/common";

// 协议类型
enum Protocol {
  PROTOCOL_UNKNOWN = 0;
  PROTOCOL_TCP = 6;
  PROTOCOL_UDP = 17;
  PROTOCOL_ICMP = 1;
  PROTOCOL_ANY = 255;
}

// 策略动作
enum PolicyAction {
  POLICY_ACTION_UNKNOWN = 0;
  POLICY_ACTION_ALLOW = 1;
  POLICY_ACTION_DENY = 2;
  POLICY_ACTION_LOG = 3;
}

// 流事件类型
enum FlowEventType {
  FLOW_EVENT_UNKNOWN = 0;
  FLOW_EVENT_NEW = 1;      // 新建连接
  FLOW_EVENT_UPDATE = 2;   // 连接更新
  FLOW_EVENT_CLOSED = 3;   // 连接关闭
  FLOW_EVENT_TIMEOUT = 4;  // 连接超时
}

// 流方向
enum FlowDirection {
  FLOW_DIRECTION_UNKNOWN = 0;
  FLOW_DIRECTION_INGRESS = 1;  // 入站
  FLOW_DIRECTION_EGRESS = 2;   // 出站
}

// 流状态
enum FlowState {
  FLOW_STATE_UNKNOWN = 0;
  FLOW_STATE_ACTIVE = 1;   // 活跃
  FLOW_STATE_CLOSED = 2;   // 已关闭
  FLOW_STATE_TIMEOUT = 3;  // 超时
}

// 通用响应
message ReportResponse {
  bool success = 1;
  string message = 2;
  uint64 accepted_count = 3;  // 接受的记录数
  uint64 rejected_count = 4;  // 拒绝的记录数
}

// 时间范围
message TimeRange {
  int64 start_time = 1;  // Unix timestamp (秒)
  int64 end_time = 2;
}
```

#### 1.2 flow.proto - 流事件定义

```protobuf
syntax = "proto3";

package microsegment.flow;

import "common.proto";

option go_package = "github.com/ebpf-microsegment/src/proto/flow";

// FlowService - 流事件管理服务
service FlowService {
  // 批量上报流事件 (Agent → Server, 客户端流)
  rpc ReportFlowEvents(stream FlowEvent) returns (common.ReportResponse);

  // 查询流量 (Server → Client)
  rpc QueryFlows(FlowQuery) returns (FlowQueryResponse);

  // 获取流量汇总 (Server → Client)
  rpc GetFlowSummary(FlowSummaryRequest) returns (FlowSummary);
}

// 流事件 (对应 eBPF struct flow_event)
message FlowEvent {
  // 5-tuple 标识 (20 bytes)
  fixed32 src_ip = 1;      // 源 IP (网络字节序)
  fixed32 dst_ip = 2;      // 目标 IP
  uint32 src_port = 3;     // 源端口
  uint32 dst_port = 4;     // 目标端口
  common.Protocol protocol = 5;

  // 事件元数据
  common.FlowEventType event_type = 6;
  common.FlowDirection direction = 7;

  // 流量统计
  uint64 packet_count = 8;
  uint64 byte_count = 9;
  fixed64 timestamp_ns = 10;  // 纳秒时间戳

  // 策略上下文
  uint32 policy_id = 11;
  common.PolicyAction policy_action = 12;
  common.FlowState state = 13;

  // Agent 信息
  string agent_id = 14;

  // 标签 (用于依赖分析)
  map<string, string> source_labels = 15;
  map<string, string> dest_labels = 16;
}

// 流查询请求
message FlowQuery {
  common.TimeRange time_range = 1;
  string source_ip = 2;
  string dest_ip = 3;
  common.Protocol protocol = 4;
  common.FlowState state = 5;
  common.PolicyAction policy_action = 6;

  // 分页
  uint32 limit = 7;   // 默认 100
  uint32 offset = 8;  // 默认 0

  // 排序
  string sort_by = 9;     // start_time, byte_count, packet_count
  string sort_order = 10; // asc, desc
}

// 流查询响应
message FlowQueryResponse {
  repeated Flow flows = 1;
  uint32 total_count = 2;
  FlowQueryInfo query_info = 3;
}

// 流记录 (完整信息)
message Flow {
  string id = 1;  // 流 ID (5-tuple 哈希)

  // 5-tuple
  string source_ip = 2;
  uint32 source_port = 3;
  string dest_ip = 4;
  uint32 dest_port = 5;
  string protocol = 6;  // TCP/UDP/ICMP

  // 统计
  uint64 packet_count = 7;
  uint64 byte_count = 8;
  int64 duration_ms = 9;

  // 时间
  int64 start_time = 10;  // Unix timestamp
  int64 end_time = 11;
  int64 last_seen = 12;

  // 标签
  map<string, string> source_labels = 13;
  map<string, string> dest_labels = 14;

  // 策略
  uint32 policy_id = 15;
  string policy_action = 16;
  string state = 17;
  string direction = 18;

  // Agent
  string agent_id = 19;
}

// 查询元数据
message FlowQueryInfo {
  uint32 limit = 1;
  uint32 offset = 2;
  string sort_by = 3;
  string sort_order = 4;
}

// 流量汇总请求
message FlowSummaryRequest {
  common.TimeRange time_range = 1;
  string agent_id = 2;  // 可选,特定 Agent 的汇总
}

// 流量汇总响应
message FlowSummary {
  int64 total_flows = 1;
  int64 active_flows = 2;
  int64 closed_flows = 3;
  uint64 total_packets = 4;
  uint64 total_bytes = 5;
  int64 allowed_flows = 6;
  int64 denied_flows = 7;

  repeated ProtocolStats top_protocols = 8;
  repeated IPStats top_sources = 9;
  repeated IPStats top_destinations = 10;
}

// 协议统计
message ProtocolStats {
  string protocol = 1;
  int64 flow_count = 2;
  uint64 packet_count = 3;
  uint64 byte_count = 4;
}

// IP 统计
message IPStats {
  string ip = 1;
  int64 flow_count = 2;
  uint64 packet_count = 3;
  uint64 byte_count = 4;
}
```

#### 1.3 policy.proto - 策略定义

```protobuf
syntax = "proto3";

package microsegment.policy;

import "common.proto";

option go_package = "github.com/ebpf-microsegment/src/proto/policy";

// PolicyService - 策略管理服务
service PolicyService {
  // 同步完整策略列表 (Agent → Server, 拉取模式)
  rpc SyncPolicies(SyncRequest) returns (SyncResponse);

  // 订阅策略更新 (Server → Agent, 推送模式, 服务器流)
  rpc SubscribePolicies(SubscribeRequest) returns (stream PolicyUpdate);

  // 上报策略执行统计 (Agent → Server)
  rpc ReportPolicyStats(PolicyStatsReport) returns (common.ReportResponse);
}

// 同步请求
message SyncRequest {
  string agent_id = 1;
  int64 last_sync_time = 2;  // Unix timestamp
  uint64 policy_version = 3;  // Agent 当前策略版本号
}

// 同步响应
message SyncResponse {
  repeated Policy policies = 1;
  uint64 policy_version = 2;  // Server 策略版本号
  int64 server_time = 3;
}

// 订阅请求
message SubscribeRequest {
  string agent_id = 1;
  uint64 current_version = 2;  // Agent 当前版本
}

// 策略更新 (增量更新)
message PolicyUpdate {
  PolicyUpdateType update_type = 1;
  Policy policy = 2;
  uint64 policy_version = 3;  // 新版本号
}

// 更新类型
enum PolicyUpdateType {
  POLICY_UPDATE_UNKNOWN = 0;
  POLICY_UPDATE_ADD = 1;
  POLICY_UPDATE_MODIFY = 2;
  POLICY_UPDATE_DELETE = 3;
}

// 策略定义
message Policy {
  uint32 rule_id = 1;
  string src_ip = 2;      // CIDR 格式
  string dst_ip = 3;      // CIDR 格式
  uint32 src_port = 4;    // 0 表示任意
  uint32 dst_port = 5;    // 0 表示任意
  common.Protocol protocol = 6;
  common.PolicyAction action = 7;
  uint32 priority = 8;
  int64 created_at = 9;
  int64 updated_at = 10;

  // 扩展字段 (未来支持标签策略)
  map<string, string> metadata = 11;
}

// 策略统计上报
message PolicyStatsReport {
  string agent_id = 1;
  int64 report_time = 2;
  repeated PolicyStats stats = 3;
}

// 单条策略统计
message PolicyStats {
  uint32 rule_id = 1;
  uint64 hit_count = 2;     // 命中次数
  uint64 packet_count = 3;  // 数据包数
  uint64 byte_count = 4;    // 字节数
  int64 last_hit_time = 5;  // 最后命中时间
}
```

#### 1.4 agent.proto - Agent 管理定义

```protobuf
syntax = "proto3";

package microsegment.agent;

option go_package = "github.com/ebpf-microsegment/src/proto/agent";

// AgentService - Agent 管理服务
service AgentService {
  // Agent 注册 (启动时调用一次)
  rpc RegisterAgent(RegisterRequest) returns (RegisterResponse);

  // 心跳 (每10秒调用一次)
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);

  // 状态上报 (异常或关键事件时)
  rpc ReportStatus(StatusReport) returns (StatusResponse);

  // Agent 注销 (关闭时调用)
  rpc UnregisterAgent(UnregisterRequest) returns (UnregisterResponse);
}

// 注册请求
message RegisterRequest {
  string agent_id = 1;      // Agent 唯一标识 (hostname 或 UUID)
  string hostname = 2;
  string version = 3;       // Agent 版本号
  string interface = 4;     // 监听的网络接口
  repeated string ip_addresses = 5;
  string os = 6;            // 操作系统
  string kernel_version = 7;
  int64 start_time = 8;     // Agent 启动时间

  AgentCapabilities capabilities = 9;
}

// Agent 能力
message AgentCapabilities {
  bool supports_flow_collection = 1;
  bool supports_label_based_policy = 2;
  bool supports_workload_discovery = 3;
  string ebpf_version = 4;
}

// 注册响应
message RegisterResponse {
  bool success = 1;
  string message = 2;
  string server_version = 3;
  int64 server_time = 4;

  AgentConfig config = 5;  // Server 下发的配置
}

// Agent 配置
message AgentConfig {
  uint32 heartbeat_interval = 1;  // 心跳间隔(秒)
  uint32 stats_interval = 2;      // 统计上报间隔(秒)
  uint32 flow_batch_size = 3;     // 流事件批量大小
  uint32 flow_batch_timeout = 4;  // 流事件批量超时(秒)
}

// 心跳请求
message HeartbeatRequest {
  string agent_id = 1;
  int64 timestamp = 2;

  AgentMetrics metrics = 3;
}

// Agent 指标
message AgentMetrics {
  // 系统指标
  double cpu_usage = 1;      // CPU 使用率 (%)
  double memory_usage = 2;   // 内存使用率 (%)

  // eBPF 指标
  uint64 total_packets = 3;
  uint64 allowed_packets = 4;
  uint64 denied_packets = 5;
  uint64 active_sessions = 6;

  // Flow 采集指标
  uint64 flows_processed = 7;
  uint64 flows_dropped = 8;

  // 策略指标
  uint32 policy_count = 9;
  uint64 policy_hits = 10;
  uint64 policy_misses = 11;
}

// 心跳响应
message HeartbeatResponse {
  bool healthy = 1;
  string message = 2;

  // Server 可下发命令
  repeated AgentCommand commands = 3;
}

// Agent 命令
message AgentCommand {
  AgentCommandType command_type = 1;
  string command_id = 2;
  map<string, string> parameters = 3;
}

// 命令类型
enum AgentCommandType {
  COMMAND_UNKNOWN = 0;
  COMMAND_RELOAD_POLICIES = 1;  // 重新加载策略
  COMMAND_COLLECT_LOGS = 2;     // 收集日志
  COMMAND_RESTART = 3;          // 重启
}

// 状态上报请求
message StatusReport {
  string agent_id = 1;
  int64 timestamp = 2;
  AgentStatus status = 3;
  string message = 4;
  map<string, string> metadata = 5;
}

// Agent 状态
enum AgentStatus {
  STATUS_UNKNOWN = 0;
  STATUS_HEALTHY = 1;
  STATUS_DEGRADED = 2;   // 降级(部分功能不可用)
  STATUS_UNHEALTHY = 3;  // 不健康
  STATUS_STOPPING = 4;   // 停止中
}

// 状态上报响应
message StatusResponse {
  bool acknowledged = 1;
  string message = 2;
}

// 注销请求
message UnregisterRequest {
  string agent_id = 1;
  string reason = 2;  // 注销原因
}

// 注销响应
message UnregisterResponse {
  bool success = 1;
  string message = 2;
}
```

### 2. 代码生成配置

#### 2.1 Makefile

```makefile
# Makefile for Protocol Buffers code generation

PROTO_DIR := proto
OUT_DIR := src/proto
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)

# 工具
PROTOC := protoc
PROTOC_GEN_GO := protoc-gen-go
PROTOC_GEN_GO_GRPC := protoc-gen-go-grpc

# 生成选项
PROTO_OPTS := --proto_path=$(PROTO_DIR)
GO_OUT := --go_out=$(OUT_DIR) --go_opt=paths=source_relative
GRPC_OUT := --go-grpc_out=$(OUT_DIR) --go-grpc_opt=paths=source_relative

.PHONY: all clean install-tools generate

all: generate

# 生成所有 protobuf 代码
generate:
	@echo "Generating Protocol Buffers code..."
	@mkdir -p $(OUT_DIR)/common
	@mkdir -p $(OUT_DIR)/flow
	@mkdir -p $(OUT_DIR)/policy
	@mkdir -p $(OUT_DIR)/agent
	$(PROTOC) $(PROTO_OPTS) $(GO_OUT) $(GRPC_OUT) $(PROTO_FILES)
	@echo "✓ Code generation complete"

# 清理生成的代码
clean:
	@echo "Cleaning generated files..."
	@rm -rf $(OUT_DIR)/*
	@echo "✓ Clean complete"

# 安装必需工具
install-tools:
	@echo "Installing protobuf tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "✓ Tools installed"

# 验证 proto 文件
lint:
	@echo "Linting proto files..."
	@for file in $(PROTO_FILES); do \
		echo "Checking $$file..."; \
		$(PROTOC) $(PROTO_OPTS) --lint_out=. $$file || true; \
	done

# 查看帮助
help:
	@echo "Available targets:"
	@echo "  make generate      - Generate Go code from proto files"
	@echo "  make clean         - Remove generated files"
	@echo "  make install-tools - Install required tools"
	@echo "  make lint          - Lint proto files"
```

#### 2.2 生成脚本 (scripts/generate-proto.sh)

```bash
#!/bin/bash
# Protocol Buffers 代码生成脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="$PROJECT_ROOT/proto"
OUT_DIR="$PROJECT_ROOT/src/proto"

echo "🔧 Generating Protocol Buffers code..."

# 检查工具
if ! command -v protoc &> /dev/null; then
    echo "❌ protoc not found. Please install protobuf compiler."
    exit 1
fi

if ! command -v protoc-gen-go &> /dev/null; then
    echo "❌ protoc-gen-go not found. Installing..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "❌ protoc-gen-go-grpc not found. Installing..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# 创建输出目录
mkdir -p "$OUT_DIR/common"
mkdir -p "$OUT_DIR/flow"
mkdir -p "$OUT_DIR/policy"
mkdir -p "$OUT_DIR/agent"

# 生成代码
cd "$PROJECT_ROOT"
protoc \
  --proto_path=proto \
  --go_out=src/proto --go_opt=paths=source_relative \
  --go-grpc_out=src/proto --go-grpc_opt=paths=source_relative \
  proto/*.proto

echo "✅ Code generation complete!"
echo "📂 Generated files in: $OUT_DIR"
```

### 3. Go 依赖配置

#### 3.1 go.mod 更新

```go
module github.com/ebpf-microsegment

go 1.24

require (
    // ... 现有依赖 ...

    // gRPC 和 Protobuf (新增)
    google.golang.org/grpc v1.60.0
    google.golang.org/protobuf v1.32.0
)
```

### 4. 使用示例

#### 4.1 客户端示例 (Agent)

```go
package main

import (
    "context"
    "io"
    "log"

    pb "github.com/ebpf-microsegment/src/proto/flow"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func reportFlowEvents(serverAddr string, events []*pb.FlowEvent) error {
    // 连接 Server
    conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return err
    }
    defer conn.Close()

    client := pb.NewFlowServiceClient(conn)

    // 创建流式请求
    stream, err := client.ReportFlowEvents(context.Background())
    if err != nil {
        return err
    }

    // 发送所有事件
    for _, event := range events {
        if err := stream.Send(event); err != nil {
            return err
        }
    }

    // 接收响应
    resp, err := stream.CloseAndRecv()
    if err != nil {
        return err
    }

    log.Printf("✓ Reported %d events, accepted: %d, rejected: %d",
        len(events), resp.AcceptedCount, resp.RejectedCount)
    return nil
}
```

#### 4.2 服务端示例 (Server)

```go
package main

import (
    "io"
    "log"

    pb "github.com/ebpf-microsegment/src/proto/flow"
)

type flowServer struct {
    pb.UnimplementedFlowServiceServer
}

func (s *flowServer) ReportFlowEvents(stream pb.FlowService_ReportFlowEventsServer) error {
    var count uint64 = 0

    for {
        event, err := stream.Recv()
        if err == io.EOF {
            // 客户端关闭流
            return stream.SendAndClose(&pb.ReportResponse{
                Success:       true,
                AcceptedCount: count,
            })
        }
        if err != nil {
            return err
        }

        // 处理事件
        log.Printf("Received flow event: %s:%d -> %s:%d",
            event.SrcIp, event.SrcPort,
            event.DstIp, event.DstPort)

        // 保存到数据库
        // storage.SaveFlowEvent(event)

        count++
    }
}
```

### 5. 性能优化

#### 5.1 字段编号优化

```protobuf
// 字段 1-15 使用 1 字节编码,优先分配给高频字段
message FlowEvent {
  fixed32 src_ip = 1;      // ✅ 1 字节 tag
  fixed32 dst_ip = 2;      // ✅ 1 字节 tag
  uint32 src_port = 3;     // ✅ 1 字节 tag
  // ... 高频字段在 1-15 范围

  // 字段 16+ 使用 2 字节编码,分配给低频字段
  map<string, string> metadata = 16;  // ⚠️ 2 字节 tag
}
```

#### 5.2 批量传输

```go
// Agent 侧批量缓冲
const (
    maxBatchSize    = 1000
    maxBatchTimeout = 1 * time.Second
)

func (r *GRPCReporter) batchLoop() {
    ticker := time.NewTicker(maxBatchTimeout)
    batch := make([]*pb.FlowEvent, 0, maxBatchSize)

    for {
        select {
        case event := <-r.eventQueue:
            batch = append(batch, event)
            if len(batch) >= maxBatchSize {
                r.sendBatch(batch)
                batch = batch[:0]
            }
        case <-ticker.C:
            if len(batch) > 0 {
                r.sendBatch(batch)
                batch = batch[:0]
            }
        }
    }
}
```

### 6. 错误处理

#### 6.1 重试策略

```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// 配置重试策略
var retryPolicy = `{
  "methodConfig": [{
    "name": [{"service": "microsegment.flow.FlowService"}],
    "retryPolicy": {
      "maxAttempts": 4,
      "initialBackoff": "0.1s",
      "maxBackoff": "1s",
      "backoffMultiplier": 2,
      "retryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
    }
  }]
}`

// 创建连接时应用
conn, err := grpc.Dial(
    serverAddr,
    grpc.WithDefaultServiceConfig(retryPolicy),
)
```

#### 6.2 超时控制

```go
import "time"

// 设置 RPC 超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.Heartbeat(ctx, &pb.HeartbeatRequest{
    AgentId:   "agent-01",
    Timestamp: time.Now().Unix(),
})
```

---

## 📊 测试计划

### 单元测试
- [ ] proto 消息序列化/反序列化测试
- [ ] 字段编号唯一性测试
- [ ] 枚举值完整性测试

### 集成测试
- [ ] gRPC 客户端-服务端通信测试
- [ ] 流式传输测试
- [ ] 错误处理和重试测试

### 性能测试
- [ ] 序列化性能 (10K+ events/s)
- [ ] 网络传输吞吐量
- [ ] 内存占用

---

**设计完成**: ✅
**下一步**: 创建 tasks.md 实施任务清单

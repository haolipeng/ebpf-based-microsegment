[根目录](../../CLAUDE.md) > **api/proto**

---

# Proto 模块

## 模块职责

Proto 模块定义 Agent 和 Server 之间的 gRPC 通信协议，负责：

1. **接口定义**: 定义所有 gRPC 服务和消息类型
2. **代码生成**: 生成 Go 代码（protoc-gen-go + protoc-gen-go-grpc）
3. **版本管理**: 维护协议兼容性和版本演进
4. **类型共享**: 定义跨模块共享的数据类型

## 入口文件

### Protocol Buffer 定义文件

1. **common/common.proto** - 公共类型
   - 基础数据类型（IP, Port, Protocol）
   - 时间戳定义
   - 错误码定义

2. **policy/policy.proto** - 策略管理
   - PolicyService (gRPC 服务)
   - Policy、PolicyRule 消息类型

3. **flow/flow.proto** - 流事件管理
   - FlowService (gRPC 服务)
   - FlowEvent、FlowStreamRequest 消息类型

4. **agent/agent.proto** - Agent 管理
   - AgentService (gRPC 服务)
   - AgentInfo、HeartbeatRequest 消息类型

5. **alert/alert.proto** - 告警管理
   - Alert、AlertRule 消息类型

## gRPC 服务定义

### PolicyService

```protobuf
service PolicyService {
    // 获取策略列表
    rpc GetPolicies(GetPoliciesRequest) returns (stream Policy);

    // 订阅策略变更
    rpc SubscribePolicies(SubscribeRequest) returns (stream PolicyUpdate);

    // 创建策略
    rpc CreatePolicy(CreatePolicyRequest) returns (CreatePolicyResponse);

    // 更新策略
    rpc UpdatePolicy(UpdatePolicyRequest) returns (UpdatePolicyResponse);

    // 删除策略
    rpc DeletePolicy(DeletePolicyRequest) returns (DeletePolicyResponse);
}
```

### FlowService

```protobuf
service FlowService {
    // Agent 上报流事件 (双向流)
    rpc ReportFlows(stream FlowEvent) returns (FlowResponse);

    // Server 订阅流事件
    rpc StreamFlows(FlowStreamRequest) returns (stream FlowEvent);

    // 批量查询历史流
    rpc QueryFlows(QueryFlowsRequest) returns (QueryFlowsResponse);
}
```

### AgentService

```protobuf
service AgentService {
    // Agent 注册
    rpc RegisterAgent(AgentInfo) returns (RegisterResponse);

    // 心跳
    rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);

    // 上报状态
    rpc ReportStatus(AgentStatus) returns (StatusResponse);
}
```

## 消息类型

### Policy 相关

```protobuf
message Policy {
    int32 rule_id = 1;
    string src_ip = 2;
    string dst_ip = 3;
    int32 dst_port = 4;
    string protocol = 5;          // tcp/udp/icmp
    string action = 6;            // allow/deny
    string direction = 7;         // ingress/egress/bidirectional
    int32 priority = 8;
    map<string, string> labels = 9;
    google.protobuf.Timestamp created_at = 10;
    google.protobuf.Timestamp updated_at = 11;
}

message PolicyUpdate {
    enum UpdateType {
        CREATED = 0;
        UPDATED = 1;
        DELETED = 2;
    }
    UpdateType type = 1;
    Policy policy = 2;
}
```

### Flow 相关

```protobuf
message FlowEvent {
    string agent_id = 1;
    string src_ip = 2;
    string dst_ip = 3;
    int32 src_port = 4;
    int32 dst_port = 5;
    string protocol = 6;
    int64 bytes = 7;
    int64 packets = 8;
    string action = 9;            // allow/deny
    google.protobuf.Timestamp timestamp = 10;
    int32 rule_id = 11;
    string direction = 12;
    map<string, string> metadata = 13;
}

message FlowStreamRequest {
    repeated string agent_ids = 1;
    google.protobuf.Timestamp start_time = 2;
    google.protobuf.Timestamp end_time = 3;
    repeated string filters = 4;   // e.g., "protocol=tcp", "action=deny"
}
```

### Agent 相关

```protobuf
message AgentInfo {
    string agent_id = 1;
    string hostname = 2;
    string ip_address = 3;
    string version = 4;
    map<string, string> metadata = 5;
}

message HeartbeatRequest {
    string agent_id = 1;
    AgentStatus status = 2;
    google.protobuf.Timestamp timestamp = 3;
}

message AgentStatus {
    string status = 1;            // running/stopped/error
    int64 uptime_seconds = 2;
    int64 active_sessions = 3;
    int64 policies_count = 4;
    ResourceUsage resource_usage = 5;
}

message ResourceUsage {
    double cpu_percent = 1;
    int64 memory_bytes = 2;
    int64 network_rx_bytes = 3;
    int64 network_tx_bytes = 4;
}
```

### Alert 相关

```protobuf
message Alert {
    string id = 1;
    string type = 2;              // policy_violation/system_error/...
    string severity = 3;          // critical/high/medium/low
    string message = 4;
    map<string, string> details = 5;
    bool acknowledged = 6;
    google.protobuf.Timestamp created_at = 7;
}
```

## 代码生成

### 生成命令

```bash
# 生成所有 Proto 代码
make proto

# 或手动执行
./scripts/generate-proto.sh
```

### 生成脚本

**scripts/generate-proto.sh**:
```bash
#!/bin/bash
set -e

PROTO_DIR="api/proto"
OUT_DIR="api/proto"

# Generate common types
protoc -I=${PROTO_DIR} \
    --go_out=${OUT_DIR} \
    --go_opt=paths=source_relative \
    ${PROTO_DIR}/common/common.proto

# Generate policy service
protoc -I=${PROTO_DIR} \
    --go_out=${OUT_DIR} \
    --go_opt=paths=source_relative \
    --go-grpc_out=${OUT_DIR} \
    --go-grpc_opt=paths=source_relative \
    ${PROTO_DIR}/policy/policy.proto

# Generate flow service
protoc -I=${PROTO_DIR} \
    --go_out=${OUT_DIR} \
    --go_opt=paths=source_relative \
    --go-grpc_out=${OUT_DIR} \
    --go-grpc_opt=paths=source_relative \
    ${PROTO_DIR}/flow/flow.proto

# ... (similar for agent, alert)
```

### 生成的文件

```
api/proto/
├── common/
│   ├── common.proto
│   └── common.pb.go                 # 生成的 Go 代码
├── policy/
│   ├── policy.proto
│   ├── policy.pb.go                 # 消息类型
│   └── policy_grpc.pb.go            # gRPC 服务
├── flow/
│   ├── flow.proto
│   ├── flow.pb.go
│   └── flow_grpc.pb.go
├── agent/
│   ├── agent.proto
│   ├── agent.pb.go
│   └── agent_grpc.pb.go
└── alert/
    ├── alert.proto
    └── alert.pb.go
```

## 使用示例

### Server 端实现

```go
import (
    flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
    "google.golang.org/grpc"
)

type flowServiceServer struct {
    flowpb.UnimplementedFlowServiceServer
    storage storage.FlowStorage
}

func (s *flowServiceServer) ReportFlows(
    stream flowpb.FlowService_ReportFlowsServer,
) error {
    for {
        event, err := stream.Recv()
        if err == io.EOF {
            return stream.SendAndClose(&flowpb.FlowResponse{Success: true})
        }
        if err != nil {
            return err
        }

        // 存储流事件
        if err := s.storage.SaveFlow(event); err != nil {
            return err
        }
    }
}
```

### Client 端使用

```go
import (
    flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
    "google.golang.org/grpc"
)

// 创建 gRPC 连接
conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := flowpb.NewFlowServiceClient(conn)

// 创建双向流
stream, err := client.ReportFlows(context.Background())
if err != nil {
    log.Fatal(err)
}

// 发送流事件
event := &flowpb.FlowEvent{
    AgentId:   "agent-001",
    SrcIp:     "10.0.0.1",
    DstIp:     "192.168.1.100",
    SrcPort:   12345,
    DstPort:   443,
    Protocol:  "tcp",
    Bytes:     1024,
    Packets:   10,
    Action:    "allow",
    Timestamp: timestamppb.Now(),
}

if err := stream.Send(event); err != nil {
    log.Fatal(err)
}

// 接收响应
resp, err := stream.CloseAndRecv()
if err != nil {
    log.Fatal(err)
}
log.Printf("Response: %v", resp)
```

## 版本管理

### 兼容性原则

1. **向后兼容**: 新版本必须兼容旧版本
2. **字段保留**: 不删除已有字段，使用 `reserved` 标记
3. **枚举扩展**: 可以添加新枚举值，但不改变现有值

### 示例

```protobuf
message Policy {
    // 已废弃的字段
    reserved 100, 101, 102;
    reserved "old_field_name";

    // 当前字段
    int32 rule_id = 1;
    string src_ip = 2;
    // ...

    // 新增字段（向后兼容）
    string tenant_id = 20;  // v1.1.0 新增
}
```

## 测试

### 单元测试

```go
func TestPolicyMessage(t *testing.T) {
    policy := &policypb.Policy{
        RuleId:    1001,
        SrcIp:     "10.0.0.0/24",
        DstIp:     "192.168.1.100",
        DstPort:   443,
        Protocol:  "tcp",
        Action:    "allow",
        Direction: "ingress",
        Priority:  100,
    }

    // 序列化
    data, err := proto.Marshal(policy)
    require.NoError(t, err)

    // 反序列化
    decoded := &policypb.Policy{}
    err = proto.Unmarshal(data, decoded)
    require.NoError(t, err)

    // 验证
    assert.Equal(t, policy.RuleId, decoded.RuleId)
    assert.Equal(t, policy.SrcIp, decoded.SrcIp)
}
```

## 常见问题 (FAQ)

### Q1: 修改 .proto 文件后如何重新生成代码？

```bash
make clean-proto
make proto
```

### Q2: 如何安装 protoc 和插件？

```bash
# 安装 protoc (Ubuntu)
sudo apt-get install -y protobuf-compiler

# 安装 Go 插件
make install-proto-tools
```

### Q3: 如何查看生成的 gRPC 服务？

```bash
# 使用 grpcurl 列出服务
grpcurl -plaintext localhost:50051 list

# 查看服务方法
grpcurl -plaintext localhost:50051 list policy.PolicyService
```

## 变更记录 (Changelog)

### [初始化] - 2025-11-27 00:02:00

- 创建 Proto 模块文档
- 记录所有 Protocol Buffer 定义
- 扫描覆盖：common, policy, flow, agent, alert
- 覆盖率：100%

---

**最后更新**: 2025-11-27 00:02:00
**维护者**: ebpf-based-microsegment team

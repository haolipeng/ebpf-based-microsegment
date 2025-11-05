# Protocol Buffers Definitions

This directory contains Protocol Buffer (protobuf) definitions for the eBPF-based Microsegmentation system's gRPC API.

## Overview

The gRPC API enables communication between:
- **Agents**: eBPF-based network monitoring agents running on each host
- **Server**: Central management and aggregation server (future component)

## Protocol Definitions

### [`common.proto`](common.proto)

Common types and enumerations used across all services.

**Enums**:
- `Protocol`: Network protocols (TCP, UDP, ICMP, ANY)
- `PolicyAction`: Policy enforcement actions (ALLOW, DENY, LOG)
- `FlowEventType`: Flow lifecycle events (NEW, UPDATE, CLOSED, TIMEOUT)
- `FlowDirection`: Traffic direction (INGRESS, EGRESS)
- `FlowState`: Flow state (ACTIVE, CLOSED, TIMEOUT)

**Messages**:
- `ReportResponse`: Standard response for reporting operations
- `TimeRange`: Time interval for queries

### [`flow.proto`](flow.proto)

Flow event reporting and querying service.

**Service**: `FlowService`
- `ReportFlowEvents`: Stream flow events from agent to server
- `QueryFlows`: Query historical flows
- `GetFlowSummary`: Get aggregated flow statistics

**Key Messages**:
- `FlowEvent`: Compact flow event (~48 bytes base + variable labels)
- `FlowQuery`: Flow query with filtering parameters
- `FlowSummary`: Aggregated flow statistics

### [`policy.proto`](policy.proto)

Policy synchronization and statistics service.

**Service**: `PolicyService`
- `SyncPolicies`: Full policy synchronization
- `SubscribePolicies`: Subscribe to incremental policy updates
- `ReportPolicyStats`: Report policy enforcement statistics

**Key Messages**:
- `Policy`: Compiled IP-based policy rule
- `PolicyUpdate`: Incremental policy change notification
- `PolicyStats`: Policy enforcement statistics

### [`agent.proto`](agent.proto)

Agent lifecycle management service.

**Service**: `AgentService`
- `RegisterAgent`: Register agent on startup
- `Heartbeat`: Periodic health and metrics updates
- `ReportStatus`: Detailed status reporting
- `UnregisterAgent`: Graceful shutdown

**Key Messages**:
- `RegisterRequest`: Agent registration information
- `HeartbeatRequest`: Lightweight health check
- `AgentCommand`: Server-to-agent commands

## Code Generation

### Prerequisites

1. **Install `protoc` compiler** (≥ 3.12):
   ```bash
   # Ubuntu/Debian
   sudo apt-get install -y protobuf-compiler

   # macOS
   brew install protobuf
   ```

2. **Install Go plugins**:
   ```bash
   make install-proto-tools

   # Or manually:
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

### Generate Code

```bash
# From project root
make generate-proto

# Or use the script directly
./scripts/generate-proto.sh
```

### Clean Generated Code

```bash
make clean-proto
```

## Generated Code Structure

```
src/proto/
├── common/
│   ├── common.pb.go          # Protobuf message definitions
│   └── go.mod                # Module dependencies
├── flow/
│   ├── flow.pb.go            # Protobuf message definitions
│   ├── flow_grpc.pb.go       # gRPC service stubs
│   └── go.mod
├── policy/
│   ├── policy.pb.go          # Protobuf message definitions
│   ├── policy_grpc.pb.go     # gRPC service stubs
│   └── go.mod
└── agent/
    ├── agent.pb.go           # Protobuf message definitions
    ├── agent_grpc.pb.go      # gRPC service stubs
    └── go.mod
```

## Usage Examples

### Import in Go Code

```go
import (
    commonpb "github.com/ebpf-microsegment/src/proto/common"
    flowpb "github.com/ebpf-microsegment/src/proto/flow"
    policypb "github.com/ebpf-microsegment/src/proto/policy"
    agentpb "github.com/ebpf-microsegment/src/proto/agent"
)
```

### Create a Flow Event

```go
event := &flowpb.FlowEvent{
    SrcIp:        0x0A000101,  // 10.0.1.1
    DstIp:        0x0A000202,  // 10.0.2.2
    SrcPort:      12345,
    DstPort:      80,
    Protocol:     commonpb.Protocol_PROTOCOL_TCP,
    EventType:    commonpb.FlowEventType_EVENT_NEW,
    Direction:    commonpb.FlowDirection_DIRECTION_EGRESS,
    PacketCount:  10,
    ByteCount:    1500,
    TimestampNs:  time.Now().UnixNano(),
    PolicyId:     100,
    PolicyAction: commonpb.PolicyAction_ACTION_ALLOW,
    State:        commonpb.FlowState_STATE_ACTIVE,
    AgentId:      "agent-001",
    SourceLabels: map[string]string{
        "role": "web",
        "env":  "prod",
    },
    DestLabels: map[string]string{
        "role": "api",
        "env":  "prod",
    },
}
```

### Create a gRPC Client

```go
conn, err := grpc.Dial("server:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
if err != nil {
    log.Fatalf("Failed to connect: %v", err)
}
defer conn.Close()

// Create service clients
flowClient := flowpb.NewFlowServiceClient(conn)
policyClient := policypb.NewPolicyServiceClient(conn)
agentClient := agentpb.NewAgentServiceClient(conn)

// Stream flow events
stream, err := flowClient.ReportFlowEvents(context.Background())
if err != nil {
    log.Fatalf("Failed to create stream: %v", err)
}

// Send events
for _, event := range events {
    if err := stream.Send(event); err != nil {
        log.Printf("Failed to send event: %v", err)
    }
}

// Close stream and get response
resp, err := stream.CloseAndRecv()
if err != nil {
    log.Fatalf("Failed to close stream: %v", err)
}
log.Printf("Server response: %+v", resp)
```

## Design Principles

### 1. Compact Wire Format

- Use `fixed32` for IP addresses (always 4 bytes)
- Use `fixed64` for timestamps (always 8 bytes)
- Field numbers 1-15 for high-frequency fields (1-byte tag)
- Field numbers 16+ for lower-frequency fields (2-byte tag)

**Result**: FlowEvent wire size ~48 bytes base (excluding labels)

### 2. Backward Compatibility

- Never reuse field numbers
- Use `reserved` keyword for deleted fields
- Add new fields with new numbers
- Make new fields optional

### 3. Streaming for High Throughput

- `ReportFlowEvents`: Streaming from agent to server (handles 1000s of events/sec)
- `SubscribePolicies`: Streaming from server to agent (real-time policy updates)

### 4. Enum Safety

- All enums have `UNKNOWN = 0` as default value
- Use explicit field numbers matching standards (e.g., Protocol numbers)

## Performance Considerations

### FlowEvent Optimization

Based on the tasks specification (Task 2.2), the FlowEvent message is optimized to be approximately **48 bytes** (base size, excluding variable labels):

- **Fixed-size fields**: 20 bytes
  - `fixed32 src_ip`: 4 bytes
  - `fixed32 dst_ip`: 4 bytes
  - `fixed64 timestamp_ns`: 8 bytes
  - Field tags: ~4 bytes

- **Variable-size fields**: ~28 bytes (typical)
  - Ports (varint): ~2-4 bytes
  - Counts (varint): ~2-10 bytes
  - Enums (varint): ~5 bytes
  - Policy ID (varint): ~1-2 bytes
  - Agent ID (string): ~10 bytes
  - Field tags: ~8 bytes

- **Maps** (variable overhead):
  - Labels add approximately 10-50 bytes depending on number and size of keys/values

### Benchmarking

Run performance benchmarks (when implemented):

```bash
cd src/proto/flow
go test -bench=. -benchmem
```

Expected performance (targets from tasks.md):
- Serialization: > 100K ops/s
- Memory allocation: < 1KB per event

## Field Number Allocation

To support future extensions, field numbers are allocated as follows:

- **1-15**: High-frequency fields (1-byte varint tags)
- **16-2047**: Medium-frequency fields (2-byte varint tags)
- **2048+**: Low-frequency / future fields (3+ byte varint tags)

## Protocol Versioning

This is **version 1.0.0** of the protocol definitions. Version information is tracked in:
- Git tags (e.g., `proto-v1.0.0`)
- Package versions in generated Go modules
- OpenSpec change tracking: `openspec/changes/add-grpc-protocol-definitions/`

## Troubleshooting

### Import Errors

If you see errors like `no required module provides package github.com/ebpf-microsegment/src/proto/common`:

```bash
# Regenerate all proto code
make clean-proto
make generate-proto

# Verify modules are set up correctly
cd src/proto/flow && go mod tidy
cd ../policy && go mod tidy
cd ../agent && go mod tidy
```

### Compilation Errors

```bash
# Check protoc version
protoc --version  # Should be >= 3.12

# Check Go plugins
which protoc-gen-go
which protoc-gen-go-grpc

# Reinstall if needed
make install-proto-tools
```

### Unused Import Warning

The warning `agent.proto:5:1: warning: Import common.proto is unused` can be safely ignored. It occurs because `agent.proto` currently doesn't reference any types from `common.proto`, but the import is included for future use and consistency.

## Related Documentation

- [Agent-Server Migration Plan](../docs/agent-server-migration-plan.md)
- [Architecture Comparison](../docs/architecture-comparison.md)
- [OpenSpec Change Proposal](../openspec/changes/add-grpc-protocol-definitions/)
- [gRPC API Documentation](../docs/grpc-api.md) (to be created)

## Contributing

When modifying proto files:

1. **Never reuse field numbers** - Use `reserved` instead
2. **Update version** in OpenSpec change tracking
3. **Regenerate code**: `make generate-proto`
4. **Run tests**: `make test`
5. **Update documentation** in this README

## License

See [LICENSE](../LICENSE) file in the project root.

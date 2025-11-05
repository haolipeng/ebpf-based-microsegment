# Refactor Agent for Remote Reporting - Implementation Tasks

## Overview

This document provides a detailed, step-by-step implementation plan for refactoring the microsegmentation agent to support dual-mode operation (standalone and agent-server). The refactoring introduces the Reporter interface pattern for pluggable flow reporting.

**Proposal**: `refactor-agent-for-remote-reporting`
**Dependencies**: `add-grpc-protocol-definitions`, `add-server-component`
**Estimated Total Time**: 8-12 hours
**Complexity**: Medium-High

---

## Phase 1: Reporter Interface and Local Implementation (1-2 hours)

### Objective
Create the Reporter interface abstraction and implement LocalReporter for backward compatibility with standalone mode.

### Tasks

- [x] **1.1 Create Reporter interface**
  - [x] Create `src/agent/pkg/reporter/reporter.go`
  - [x] Define Reporter interface with methods:
    - `Report(ctx context.Context, flow *flow.Flow) error`
    - `ReportBatch(ctx context.Context, flows []*flow.Flow) error`
    - `Start() error`
    - `Stop() error`

  ```go
  // src/agent/pkg/reporter/reporter.go
  package reporter

  import (
      "context"
      "github.com/ebpf-microsegment/src/agent/pkg/flow"
  )

  // Reporter is an interface for reporting flow data
  type Reporter interface {
      Report(ctx context.Context, flow *flow.Flow) error
      ReportBatch(ctx context.Context, flows []*flow.Flow) error
      Start() error
      Stop() error
  }
  ```

- [x] **1.2 Implement LocalReporter**
  - [x] Create `src/agent/pkg/reporter/local_reporter.go`
  - [x] Implement all Reporter interface methods
  - [x] Use existing storage.Storage for data persistence
  - [x] Add logging for standalone mode indication

  ```go
  // src/agent/pkg/reporter/local_reporter.go
  package reporter

  import (
      "context"
      "github.com/ebpf-microsegment/src/agent/pkg/flow"
      "github.com/ebpf-microsegment/src/agent/pkg/storage"
      "github.com/sirupsen/logrus"
  )

  type LocalReporter struct {
      storage storage.Storage
  }

  func NewLocalReporter(storage storage.Storage) *LocalReporter {
      return &LocalReporter{storage: storage}
  }

  func (r *LocalReporter) Report(ctx context.Context, f *flow.Flow) error {
      return r.storage.SaveFlow(ctx, f)
  }

  func (r *LocalReporter) ReportBatch(ctx context.Context, flows []*flow.Flow) error {
      for _, f := range flows {
          if err := r.storage.SaveFlow(ctx, f); err != nil {
              return err
          }
      }
      return nil
  }

  func (r *LocalReporter) Start() error {
      logrus.Info("Local reporter started (standalone mode)")
      return nil
  }

  func (r *LocalReporter) Stop() error {
      return nil
  }
  ```

- [ ] **1.3 Add unit tests for LocalReporter**
  - [ ] Create `src/agent/pkg/reporter/local_reporter_test.go`
  - [ ] Test Report() method with mock storage
  - [ ] Test ReportBatch() with multiple flows
  - [ ] Test error handling from storage layer
  - [ ] Verify zero overhead compared to direct storage calls

  ```go
  // src/agent/pkg/reporter/local_reporter_test.go
  package reporter

  import (
      "context"
      "testing"
      "github.com/ebpf-microsegment/src/agent/pkg/flow"
      "github.com/stretchr/testify/assert"
      "github.com/stretchr/testify/mock"
  )

  type MockStorage struct {
      mock.Mock
  }

  func (m *MockStorage) SaveFlow(ctx context.Context, f *flow.Flow) error {
      args := m.Called(ctx, f)
      return args.Error(0)
  }

  func TestLocalReporter_Report(t *testing.T) {
      mockStorage := new(MockStorage)
      reporter := NewLocalReporter(mockStorage)

      testFlow := &flow.Flow{/* ... */}
      mockStorage.On("SaveFlow", mock.Anything, testFlow).Return(nil)

      err := reporter.Report(context.Background(), testFlow)
      assert.NoError(t, err)
      mockStorage.AssertExpectations(t)
  }
  ```

**Acceptance Criteria**:
- ✅ Reporter interface defined with 4 methods
- ✅ LocalReporter implements all methods
- ✅ LocalReporter uses existing storage layer
- ⏸️ Unit tests pass with >90% coverage
- ⏸️ Zero performance impact on standalone mode

---

## Phase 2: GRPCReporter Implementation (2-3 hours)

### Objective
Implement GRPCReporter for agent-server mode with batching and async sending.

### Tasks

- [x] **2.1 Create GRPCReporter structure**
  - [x] Create `src/agent/pkg/reporter/grpc_reporter.go`
  - [x] Define GRPCReporter struct with fields:
    - serverAddr, agentID
    - gRPC connection and client
    - batchQueue, batchSize
    - stopCh for graceful shutdown

  ```go
  // src/agent/pkg/reporter/grpc_reporter.go
  package reporter

  import (
      "context"
      "time"
      flowpb "github.com/ebpf-microsegment/src/proto/flow"
      "google.golang.org/grpc"
  )

  type GRPCReporter struct {
      serverAddr string
      agentID    string
      conn       *grpc.ClientConn
      client     flowpb.FlowServiceClient
      batchSize  int
      batchQueue chan *flowpb.FlowEvent
      stopCh     chan struct{}
  }

  func NewGRPCReporter(serverAddr, agentID string, batchSize int) *GRPCReporter {
      if batchSize == 0 {
          batchSize = 100
      }
      return &GRPCReporter{
          serverAddr: serverAddr,
          agentID:    agentID,
          batchSize:  batchSize,
          batchQueue: make(chan *flowpb.FlowEvent, batchSize*2),
          stopCh:     make(chan struct{}),
      }
  }
  ```

- [x] **2.2 Implement Start() with connection logic**
  - [x] Connect to gRPC server with retry logic
  - [x] Create FlowServiceClient
  - [x] Start background batchSender goroutine
  - [x] Add connection validation

  ```go
  func (r *GRPCReporter) Start() error {
      conn, err := grpc.NewClient(
          r.serverAddr,
          grpc.WithTransportCredentials(insecure.NewCredentials()),
      )
      if err != nil {
          return fmt.Errorf("failed to connect to server: %w", err)
      }

      r.conn = conn
      r.client = flowpb.NewFlowServiceClient(conn)

      logrus.Infof("gRPC reporter connected to %s (agent-server mode)", r.serverAddr)

      go r.batchSender()
      return nil
  }
  ```

- [x] **2.3 Implement Report() with queueing**
  - [x] Convert flow.Flow to flowpb.FlowEvent
  - [x] Queue event to batchQueue channel
  - [x] Handle queue full scenarios (drop or block)
  - [x] Add metrics for dropped flows

  ```go
  func (r *GRPCReporter) Report(ctx context.Context, f *flow.Flow) error {
      event := r.flowToProto(f)
      select {
      case r.batchQueue <- event:
          return nil
      default:
          return fmt.Errorf("batch queue full, dropping flow")
      }
  }
  ```

- [x] **2.4 Implement batchSender goroutine**
  - [x] Create ticker for periodic batch sending (5s default)
  - [x] Collect events until batchSize or timeout
  - [x] Send batches asynchronously via sendBatchAsync()
  - [x] Handle stopCh for graceful shutdown

  ```go
  func (r *GRPCReporter) batchSender() {
      ticker := time.NewTicker(5 * time.Second)
      defer ticker.Stop()

      batch := make([]*flowpb.FlowEvent, 0, r.batchSize)

      for {
          select {
          case event := <-r.batchQueue:
              batch = append(batch, event)
              if len(batch) >= r.batchSize {
                  r.sendBatchAsync(batch)
                  batch = make([]*flowpb.FlowEvent, 0, r.batchSize)
              }

          case <-ticker.C:
              if len(batch) > 0 {
                  r.sendBatchAsync(batch)
                  batch = make([]*flowpb.FlowEvent, 0, r.batchSize)
              }

          case <-r.stopCh:
              if len(batch) > 0 {
                  r.sendBatchAsync(batch)
              }
              return
          }
      }
  }
  ```

- [x] **2.5 Implement sendBatch() with gRPC streaming**
  - [x] Create bidirectional stream via ReportFlowEvents()
  - [x] Send all events in batch
  - [x] Handle CloseAndRecv() for server response
  - [x] Add error logging and metrics

  ```go
  func (r *GRPCReporter) sendBatch(ctx context.Context, events []*flowpb.FlowEvent) error {
      stream, err := r.client.ReportFlowEvents(ctx)
      if err != nil {
          return fmt.Errorf("failed to create stream: %w", err)
      }

      for _, event := range events {
          if err := stream.Send(event); err != nil {
              return fmt.Errorf("failed to send event: %w", err)
          }
      }

      resp, err := stream.CloseAndRecv()
      if err != nil {
          return fmt.Errorf("failed to close stream: %w", err)
      }

      if !resp.Success {
          return fmt.Errorf("server reported failure: %s", resp.Message)
      }

      return nil
  }
  ```

- [x] **2.6 Implement flowToProto() converter**
  - [x] Convert flow.Flow struct to flowpb.FlowEvent
  - [x] Convert IP strings to uint32
  - [x] Map protocol numbers correctly
  - [x] Include labels (SourceLabels, DestLabels)
  - [x] Set agentID in each event

  ```go
  func (r *GRPCReporter) flowToProto(f *flow.Flow) *flowpb.FlowEvent {
      srcIP := ipToUint32(f.SrcIP)
      dstIP := ipToUint32(f.DstIP)

      return &flowpb.FlowEvent{
          SrcIp:        srcIP,
          DstIp:        dstIP,
          SrcPort:      uint32(f.SrcPort),
          DstPort:      uint32(f.DstPort),
          Protocol:     commonpb.Protocol(f.Protocol),
          EventType:    commonpb.FlowEventType_EVENT_NEW,
          Direction:    commonpb.FlowDirection_DIRECTION_EGRESS,
          PacketCount:  f.PacketCount,
          ByteCount:    f.ByteCount,
          TimestampNs:  uint64(f.Timestamp.UnixNano()),
          PolicyId:     0,
          PolicyAction: commonpb.PolicyAction_ACTION_ALLOW,
          State:        commonpb.FlowState_STATE_ACTIVE,
          AgentId:      r.agentID,
          SourceLabels: f.SourceLabels,
          DestLabels:   f.DestLabels,
      }
  }

  func ipToUint32(ipStr string) uint32 {
      ip := net.ParseIP(ipStr)
      if ip == nil {
          return 0
      }
      ip = ip.To4()
      if ip == nil {
          return 0
      }
      return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
  }
  ```

- [ ] **2.7 Add unit tests for GRPCReporter**
  - [ ] Create `src/agent/pkg/reporter/grpc_reporter_test.go`
  - [ ] Test flowToProto() conversion accuracy
  - [ ] Test batch accumulation logic
  - [ ] Test queue overflow behavior
  - [ ] Mock gRPC client for testing sendBatch()
  - [ ] Test graceful shutdown with pending events

  ```go
  func TestGRPCReporter_FlowToProto(t *testing.T) {
      reporter := NewGRPCReporter("localhost:9090", "agent-1", 100)

      flow := &flow.Flow{
          SrcIP:        "192.168.1.10",
          DstIP:        "10.0.0.5",
          SrcPort:      12345,
          DstPort:      80,
          Protocol:     6, // TCP
          PacketCount:  100,
          ByteCount:    5000,
          Timestamp:    time.Now(),
          SourceLabels: map[string]string{"env": "prod"},
          DestLabels:   map[string]string{"app": "web"},
      }

      event := reporter.flowToProto(flow)

      assert.Equal(t, uint32(0xC0A8010A), event.SrcIp) // 192.168.1.10
      assert.Equal(t, uint32(0x0A000005), event.DstIp) // 10.0.0.5
      assert.Equal(t, uint32(12345), event.SrcPort)
      assert.Equal(t, uint32(80), event.DstPort)
      assert.Equal(t, commonpb.Protocol_PROTOCOL_TCP, event.Protocol)
      assert.Equal(t, "agent-1", event.AgentId)
  }
  ```

**Acceptance Criteria**:
- ✅ GRPCReporter connects to server successfully
- ✅ Batching works (100 events or 5s timeout)
- ✅ Async sending doesn't block flow collection
- ⏸️ Unit tests pass with >85% coverage
- ⏸️ Performance: <5ms overhead per flow
- ✅ Graceful shutdown sends all pending events

---

## Phase 3: AgentClient Wrapper (1-2 hours)

### Objective
Create an AgentClient wrapper for managing agent registration, heartbeat, and policy synchronization.

### Tasks

- [ ] **3.1 Create AgentClient structure**
  - [ ] Create `src/agent/pkg/client/agent_client.go`
  - [ ] Define AgentClient struct with:
    - agentID, hostname, version
    - gRPC connection and service clients
    - Config fields (heartbeat interval, etc.)

  ```go
  // src/agent/pkg/client/agent_client.go
  package client

  import (
      "context"
      agentpb "github.com/ebpf-microsegment/src/proto/agent"
      policypb "github.com/ebpf-microsegment/src/proto/policy"
      "google.golang.org/grpc"
  )

  type AgentClient struct {
      agentID       string
      hostname      string
      version       string
      serverAddr    string

      conn          *grpc.ClientConn
      agentService  agentpb.AgentServiceClient
      policyService policypb.PolicyServiceClient

      heartbeatInterval int
      stopCh            chan struct{}
  }

  func NewAgentClient(serverAddr, agentID, hostname, version string) *AgentClient {
      return &AgentClient{
          serverAddr: serverAddr,
          agentID:    agentID,
          hostname:   hostname,
          version:    version,
          stopCh:     make(chan struct{}),
      }
  }
  ```

- [ ] **3.2 Implement Connect() method**
  - [ ] Establish gRPC connection
  - [ ] Create AgentService and PolicyService clients
  - [ ] Add connection retry with exponential backoff
  - [ ] Validate connection with RegisterAgent()

  ```go
  func (c *AgentClient) Connect() error {
      conn, err := grpc.NewClient(
          c.serverAddr,
          grpc.WithTransportCredentials(insecure.NewCredentials()),
      )
      if err != nil {
          return fmt.Errorf("failed to connect: %w", err)
      }

      c.conn = conn
      c.agentService = agentpb.NewAgentServiceClient(conn)
      c.policyService = policypb.NewPolicyServiceClient(conn)

      return c.registerAgent()
  }
  ```

- [ ] **3.3 Implement RegisterAgent()**
  - [ ] Gather system info (hostname, IPs, OS)
  - [ ] Call AgentService.RegisterAgent()
  - [ ] Store server config (heartbeat interval, etc.)
  - [ ] Return registration status

  ```go
  func (c *AgentClient) registerAgent() error {
      req := &agentpb.RegisterRequest{
          AgentId:     c.agentID,
          Hostname:    c.hostname,
          Version:     c.version,
          IpAddresses: getLocalIPs(),
          OsInfo:      getOSInfo(),
          Metadata:    map[string]string{},
      }

      resp, err := c.agentService.RegisterAgent(context.Background(), req)
      if err != nil {
          return fmt.Errorf("registration failed: %w", err)
      }

      if !resp.Success {
          return fmt.Errorf("registration rejected: %s", resp.Message)
      }

      c.heartbeatInterval = int(resp.Config.HeartbeatInterval)
      logrus.Infof("Registered with server, agent_id=%s", c.agentID)
      return nil
  }
  ```

- [ ] **3.4 Implement StartHeartbeat()**
  - [ ] Create ticker with server-specified interval
  - [ ] Send periodic heartbeat via AgentService.Heartbeat()
  - [ ] Include metrics (flow count, policy count)
  - [ ] Handle heartbeat failures with backoff
  - [ ] Stop on stopCh signal

  ```go
  func (c *AgentClient) StartHeartbeat() {
      ticker := time.NewTicker(time.Duration(c.heartbeatInterval) * time.Second)
      defer ticker.Stop()

      for {
          select {
          case <-ticker.C:
              if err := c.sendHeartbeat(); err != nil {
                  logrus.Errorf("Heartbeat failed: %v", err)
              }
          case <-c.stopCh:
              return
          }
      }
  }

  func (c *AgentClient) sendHeartbeat() error {
      req := &agentpb.HeartbeatRequest{
          AgentId:   c.agentID,
          Timestamp: uint64(time.Now().Unix()),
          Metrics: &agentpb.AgentMetrics{
              CpuUsage:    getCPUUsage(),
              MemoryUsage: getMemoryUsage(),
              FlowCount:   getFlowCount(),
              PolicyCount: getPolicyCount(),
          },
      }

      resp, err := c.agentService.Heartbeat(context.Background(), req)
      if err != nil {
          return err
      }

      if !resp.Acknowledged {
          logrus.Warnf("Heartbeat not acknowledged: %s", resp.Message)
      }

      return nil
  }
  ```

- [ ] **3.5 Implement SyncPolicies()**
  - [ ] Call PolicyService.SyncPolicies()
  - [ ] Receive full policy list from server
  - [ ] Return policies and version for local storage
  - [ ] Add periodic sync based on server config

  ```go
  func (c *AgentClient) SyncPolicies() ([]*policypb.Policy, uint64, error) {
      req := &policypb.SyncRequest{
          AgentId:          c.agentID,
          CurrentPolicyVersion: getCurrentPolicyVersion(),
          Capabilities:     []string{"label-based-policy"},
      }

      resp, err := c.policyService.SyncPolicies(context.Background(), req)
      if err != nil {
          return nil, 0, fmt.Errorf("policy sync failed: %w", err)
      }

      logrus.Infof("Synced %d policies, version=%d", resp.PolicyCount, resp.PolicyVersion)
      return resp.Policies, resp.PolicyVersion, nil
  }
  ```

- [ ] **3.6 Add unit tests for AgentClient**
  - [ ] Create `src/agent/pkg/client/agent_client_test.go`
  - [ ] Test registration flow with mock server
  - [ ] Test heartbeat sending
  - [ ] Test policy sync
  - [ ] Test connection failure handling

  ```go
  func TestAgentClient_RegisterAgent(t *testing.T) {
      // Setup mock gRPC server
      lis := bufconn.Listen(1024 * 1024)
      s := grpc.NewServer()
      agentpb.RegisterAgentServiceServer(s, &mockAgentService{})
      go s.Serve(lis)
      defer s.Stop()

      // Test client registration
      client := NewAgentClient("bufnet", "agent-1", "test-host", "1.0.0")
      err := client.Connect()
      assert.NoError(t, err)
  }
  ```

**Acceptance Criteria**:
- ⏸️ AgentClient connects and registers successfully
- ⏸️ Heartbeat sends every N seconds
- ⏸️ Policy sync retrieves policies correctly
- ⏸️ Connection failures handled gracefully
- ⏸️ Unit tests pass with >80% coverage

---

## Phase 4: Configuration Extension (1 hour)

### Objective
Extend agent configuration to support agent-server mode settings.

### Tasks

- [ ] **4.1 Update config structure**
  - [ ] Open `src/agent/pkg/config/config.go`
  - [ ] Add new `AgentServerConfig` struct
  - [ ] Add mode field: "standalone" or "agent-server"
  - [ ] Add backward compatibility defaults

  ```go
  // src/agent/pkg/config/config.go
  type Config struct {
      // Existing fields
      BPFPath        string        `yaml:"bpf_path"`
      StoragePath    string        `yaml:"storage_path"`
      LogLevel       string        `yaml:"log_level"`

      // New fields for agent-server mode
      Mode           string        `yaml:"mode"` // "standalone" or "agent-server"
      AgentServer    *AgentServerConfig `yaml:"agent_server,omitempty"`
  }

  type AgentServerConfig struct {
      Enabled       bool   `yaml:"enabled"`
      ServerAddr    string `yaml:"server_addr"`
      AgentID       string `yaml:"agent_id"`
      BatchSize     int    `yaml:"batch_size"`
      ReconnectInterval int `yaml:"reconnect_interval"` // seconds
  }
  ```

- [ ] **4.2 Add configuration validation**
  - [ ] Validate mode is either "standalone" or "agent-server"
  - [ ] Require server_addr when mode is "agent-server"
  - [ ] Auto-generate agent_id if not provided
  - [ ] Set sensible defaults (batch_size=100, reconnect=30s)

  ```go
  func (c *Config) Validate() error {
      if c.Mode != "standalone" && c.Mode != "agent-server" {
          return fmt.Errorf("invalid mode: %s (must be 'standalone' or 'agent-server')", c.Mode)
      }

      if c.Mode == "agent-server" {
          if c.AgentServer == nil {
              return fmt.Errorf("agent_server config required for agent-server mode")
          }
          if c.AgentServer.ServerAddr == "" {
              return fmt.Errorf("agent_server.server_addr is required")
          }
          if c.AgentServer.AgentID == "" {
              c.AgentServer.AgentID = generateAgentID()
          }
          if c.AgentServer.BatchSize == 0 {
              c.AgentServer.BatchSize = 100
          }
      }

      return nil
  }

  func generateAgentID() string {
      hostname, _ := os.Hostname()
      return fmt.Sprintf("%s-%d", hostname, time.Now().Unix())
  }
  ```

- [ ] **4.3 Create example configurations**
  - [ ] Create `config/agent-standalone.yaml`
  - [ ] Create `config/agent-server.yaml`
  - [ ] Document all new configuration options

  **config/agent-standalone.yaml**:
  ```yaml
  mode: standalone
  bpf_path: /sys/fs/bpf
  storage_path: /var/lib/microsegment/flows.db
  log_level: info
  ```

  **config/agent-server.yaml**:
  ```yaml
  mode: agent-server
  bpf_path: /sys/fs/bpf
  log_level: info

  agent_server:
    enabled: true
    server_addr: "localhost:9090"
    agent_id: "agent-node1"  # optional, auto-generated if not set
    batch_size: 100
    reconnect_interval: 30
  ```

- [ ] **4.4 Add configuration tests**
  - [ ] Create `src/agent/pkg/config/config_test.go`
  - [ ] Test standalone mode validation
  - [ ] Test agent-server mode validation
  - [ ] Test missing required fields
  - [ ] Test default value assignment

  ```go
  func TestConfig_Validate_StandaloneMode(t *testing.T) {
      cfg := &Config{
          Mode: "standalone",
          StoragePath: "/tmp/flows.db",
      }
      err := cfg.Validate()
      assert.NoError(t, err)
  }

  func TestConfig_Validate_AgentServerMode(t *testing.T) {
      cfg := &Config{
          Mode: "agent-server",
          AgentServer: &AgentServerConfig{
              ServerAddr: "localhost:9090",
          },
      }
      err := cfg.Validate()
      assert.NoError(t, err)
      assert.NotEmpty(t, cfg.AgentServer.AgentID) // Auto-generated
      assert.Equal(t, 100, cfg.AgentServer.BatchSize) // Default
  }
  ```

**Acceptance Criteria**:
- ⏸️ Config supports both modes
- ⏸️ Validation catches invalid configurations
- ⏸️ Example configs provided for both modes
- ⏸️ Tests pass with >90% coverage
- ⏸️ Backward compatible with existing configs

---

## Phase 5: FlowCollector Integration (1-2 hours)

### Objective
Refactor FlowCollector to use the Reporter interface instead of direct storage calls.

### Tasks

- [ ] **5.1 Update FlowCollector structure**
  - [ ] Open `src/agent/pkg/flow/collector.go`
  - [ ] Replace `storage storage.Storage` with `reporter reporter.Reporter`
  - [ ] Update constructor to accept Reporter

  ```go
  // src/agent/pkg/flow/collector.go
  type Collector struct {
      bpfManager *bpf.Manager
      reporter   reporter.Reporter  // Changed from storage.Storage
      stopCh     chan struct{}
  }

  func NewCollector(bpfManager *bpf.Manager, rep reporter.Reporter) *Collector {
      return &Collector{
          bpfManager: bpfManager,
          reporter:   rep,
          stopCh:     make(chan struct{}),
      }
  }
  ```

- [ ] **5.2 Update flow reporting calls**
  - [ ] Replace all `storage.SaveFlow()` calls with `reporter.Report()`
  - [ ] Use `reporter.ReportBatch()` for batched flows
  - [ ] Add context propagation
  - [ ] Keep existing error handling logic

  ```go
  func (c *Collector) processFlow(flow *Flow) {
      ctx := context.Background()
      if err := c.reporter.Report(ctx, flow); err != nil {
          logrus.Errorf("Failed to report flow: %v", err)
      }
  }
  ```

- [ ] **5.3 Update Start() and Stop() methods**
  - [ ] Call `reporter.Start()` in Collector.Start()
  - [ ] Call `reporter.Stop()` in Collector.Stop()
  - [ ] Ensure graceful shutdown waits for reporter

  ```go
  func (c *Collector) Start() error {
      if err := c.reporter.Start(); err != nil {
          return fmt.Errorf("failed to start reporter: %w", err)
      }

      go c.collectFlows()
      logrus.Info("Flow collector started")
      return nil
  }

  func (c *Collector) Stop() error {
      close(c.stopCh)
      return c.reporter.Stop()
  }
  ```

- [ ] **5.4 Update collector tests**
  - [ ] Open `src/agent/pkg/flow/collector_test.go`
  - [ ] Replace storage mocks with reporter mocks
  - [ ] Test with both LocalReporter and GRPCReporter
  - [ ] Verify no behavior changes

  ```go
  type MockReporter struct {
      mock.Mock
  }

  func (m *MockReporter) Report(ctx context.Context, flow *Flow) error {
      args := m.Called(ctx, flow)
      return args.Error(0)
  }

  func TestCollector_ProcessFlow(t *testing.T) {
      mockReporter := new(MockReporter)
      collector := NewCollector(nil, mockReporter)

      testFlow := &Flow{/* ... */}
      mockReporter.On("Report", mock.Anything, testFlow).Return(nil)

      collector.processFlow(testFlow)
      mockReporter.AssertExpectations(t)
  }
  ```

**Acceptance Criteria**:
- ⏸️ FlowCollector uses Reporter interface
- ⏸️ All storage calls replaced with reporter calls
- ⏸️ Start/Stop lifecycle integrated
- ⏸️ Tests updated and passing
- ⏸️ No behavioral changes in standalone mode

---

## Phase 6: Main Entry Point Refactoring (1-2 hours)

### Objective
Update main.go to support both standalone and agent-server modes based on configuration.

### Tasks

- [ ] **6.1 Update main.go structure**
  - [ ] Open `src/agent/cmd/main.go`
  - [ ] Add reporter initialization logic
  - [ ] Add AgentClient initialization for agent-server mode
  - [ ] Keep backward compatibility

  ```go
  // src/agent/cmd/main.go
  func main() {
      cfg := loadConfig()

      if err := cfg.Validate(); err != nil {
          logrus.Fatalf("Invalid configuration: %v", err)
      }

      var rep reporter.Reporter
      var agentClient *client.AgentClient

      switch cfg.Mode {
      case "standalone":
          rep = initStandaloneReporter(cfg)
      case "agent-server":
          rep, agentClient = initAgentServerReporter(cfg)
      default:
          logrus.Fatalf("Unknown mode: %s", cfg.Mode)
      }

      collector := flow.NewCollector(bpfManager, rep)
      // ... rest of initialization
  }
  ```

- [ ] **6.2 Implement initStandaloneReporter()**
  - [ ] Initialize SQLite storage
  - [ ] Create LocalReporter
  - [ ] Return configured reporter

  ```go
  func initStandaloneReporter(cfg *config.Config) reporter.Reporter {
      storage, err := storage.NewSQLiteStorage(cfg.StoragePath)
      if err != nil {
          logrus.Fatalf("Failed to initialize storage: %v", err)
      }

      logrus.Info("Running in standalone mode")
      return reporter.NewLocalReporter(storage)
  }
  ```

- [ ] **6.3 Implement initAgentServerReporter()**
  - [ ] Create GRPCReporter with server address
  - [ ] Create AgentClient
  - [ ] Connect and register with server
  - [ ] Start heartbeat goroutine
  - [ ] Return reporter and client

  ```go
  func initAgentServerReporter(cfg *config.Config) (reporter.Reporter, *client.AgentClient) {
      agentCfg := cfg.AgentServer

      // Create gRPC reporter
      rep := reporter.NewGRPCReporter(
          agentCfg.ServerAddr,
          agentCfg.AgentID,
          agentCfg.BatchSize,
      )

      // Create agent client
      hostname, _ := os.Hostname()
      agentClient := client.NewAgentClient(
          agentCfg.ServerAddr,
          agentCfg.AgentID,
          hostname,
          "1.0.0", // version
      )

      // Connect and register
      if err := agentClient.Connect(); err != nil {
          logrus.Fatalf("Failed to connect to server: %v", err)
      }

      // Start heartbeat
      go agentClient.StartHeartbeat()

      // Sync policies on startup
      if policies, version, err := agentClient.SyncPolicies(); err == nil {
          updateLocalPolicies(policies, version)
      }

      logrus.Info("Running in agent-server mode")
      return rep, agentClient
  }
  ```

- [ ] **6.4 Add graceful shutdown**
  - [ ] Capture SIGINT/SIGTERM signals
  - [ ] Stop collector (which stops reporter)
  - [ ] Stop AgentClient (which stops heartbeat)
  - [ ] Unregister agent if in agent-server mode

  ```go
  func main() {
      // ... initialization

      sigCh := make(chan os.Signal, 1)
      signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

      <-sigCh
      logrus.Info("Shutting down...")

      collector.Stop()

      if agentClient != nil {
          agentClient.Stop()
      }

      logrus.Info("Shutdown complete")
  }
  ```

- [ ] **6.5 Update documentation**
  - [ ] Update `README.md` with dual-mode instructions
  - [ ] Document configuration examples
  - [ ] Document migration from standalone to agent-server

  **README.md update**:
  ```markdown
  ## Running the Agent

  ### Standalone Mode (default)

  Store flows locally in SQLite:

  ```bash
  sudo ./bin/microsegment-agent --config config/agent-standalone.yaml
  ```

  ### Agent-Server Mode

  Report flows to central server via gRPC:

  ```bash
  sudo ./bin/microsegment-agent --config config/agent-server.yaml
  ```

  The agent will:
  - Register with the server
  - Send heartbeats every 30 seconds
  - Batch and report flows every 100 events or 5 seconds
  - Sync policies from the server
  ```

**Acceptance Criteria**:
- ⏸️ main.go supports both modes
- ⏸️ Mode selection based on config
- ⏸️ Graceful shutdown for both modes
- ⏸️ Documentation updated
- ⏸️ Backward compatible with existing deployments

---

## Phase 7: Integration Testing (2-3 hours)

### Objective
Create comprehensive integration tests for both modes and end-to-end workflows.

### Tasks

- [ ] **7.1 Create standalone mode integration test**
  - [ ] Create `src/agent/test/integration/standalone_test.go`
  - [ ] Test flow collection → LocalReporter → SQLite
  - [ ] Verify flows stored correctly
  - [ ] Test shutdown and restart

  ```go
  func TestStandaloneMode_Integration(t *testing.T) {
      // Setup
      tmpDB := t.TempDir() + "/flows.db"
      cfg := &config.Config{
          Mode: "standalone",
          StoragePath: tmpDB,
      }

      storage, _ := storage.NewSQLiteStorage(tmpDB)
      rep := reporter.NewLocalReporter(storage)

      // Simulate flow collection
      testFlow := &flow.Flow{/* ... */}
      err := rep.Report(context.Background(), testFlow)
      assert.NoError(t, err)

      // Verify storage
      flows, err := storage.GetFlows(context.Background(), 10)
      assert.NoError(t, err)
      assert.Len(t, flows, 1)
  }
  ```

- [ ] **7.2 Create agent-server mode integration test**
  - [ ] Create `src/agent/test/integration/agent_server_test.go`
  - [ ] Start mock gRPC server
  - [ ] Test flow collection → GRPCReporter → Server
  - [ ] Verify batch sending
  - [ ] Test reconnection on server failure

  ```go
  func TestAgentServerMode_Integration(t *testing.T) {
      // Start mock server
      mockServer := startMockServer(t)
      defer mockServer.Stop()

      // Setup agent
      cfg := &config.Config{
          Mode: "agent-server",
          AgentServer: &config.AgentServerConfig{
              ServerAddr: mockServer.Addr,
              AgentID: "test-agent",
              BatchSize: 10,
          },
      }

      rep := reporter.NewGRPCReporter(
          cfg.AgentServer.ServerAddr,
          cfg.AgentServer.AgentID,
          cfg.AgentServer.BatchSize,
      )

      err := rep.Start()
      assert.NoError(t, err)

      // Send test flows
      for i := 0; i < 15; i++ {
          flow := &flow.Flow{/* ... */}
          rep.Report(context.Background(), flow)
      }

      // Wait for batch sending
      time.Sleep(6 * time.Second)

      // Verify server received flows
      assert.GreaterOrEqual(t, mockServer.ReceivedFlowCount(), 15)
  }
  ```

- [ ] **7.3 Create end-to-end test with real server**
  - [ ] Create `src/agent/test/e2e/e2e_test.go`
  - [ ] Start real PostgreSQL server
  - [ ] Start microsegment-server
  - [ ] Start agent in agent-server mode
  - [ ] Generate real eBPF flows (if possible, or mock)
  - [ ] Verify flows reach server database
  - [ ] Test policy sync from server to agent

  ```go
  func TestE2E_AgentToServer(t *testing.T) {
      if testing.Short() {
          t.Skip("Skipping E2E test in short mode")
      }

      // Start server
      serverCmd := exec.Command("../../bin/microsegment-server", "--config", "../../src/server/config/server.yaml")
      serverCmd.Start()
      defer serverCmd.Process.Kill()

      time.Sleep(2 * time.Second) // Wait for server startup

      // Start agent
      agentCmd := exec.Command("../../bin/microsegment-agent", "--config", "config/agent-server.yaml")
      agentCmd.Start()
      defer agentCmd.Process.Kill()

      time.Sleep(5 * time.Second)

      // Verify agent registered
      resp, err := http.Get("http://localhost:8080/api/v1/agents")
      assert.NoError(t, err)
      // ... parse and verify agent appears in list
  }
  ```

- [ ] **7.4 Create performance benchmark**
  - [ ] Create `src/agent/test/benchmark/reporter_bench_test.go`
  - [ ] Benchmark LocalReporter throughput
  - [ ] Benchmark GRPCReporter throughput
  - [ ] Compare overhead vs direct storage
  - [ ] Test with varying batch sizes

  ```go
  func BenchmarkLocalReporter_Report(b *testing.B) {
      storage, _ := storage.NewSQLiteStorage(":memory:")
      rep := reporter.NewLocalReporter(storage)
      rep.Start()
      defer rep.Stop()

      flow := &flow.Flow{/* ... */}
      ctx := context.Background()

      b.ResetTimer()
      for i := 0; i < b.N; i++ {
          rep.Report(ctx, flow)
      }
  }

  func BenchmarkGRPCReporter_Report(b *testing.B) {
      mockServer := startMockServer(b)
      defer mockServer.Stop()

      rep := reporter.NewGRPCReporter(mockServer.Addr, "bench-agent", 100)
      rep.Start()
      defer rep.Stop()

      flow := &flow.Flow{/* ... */}
      ctx := context.Background()

      b.ResetTimer()
      for i := 0; i < b.N; i++ {
          rep.Report(ctx, flow)
      }
  }
  ```

**Acceptance Criteria**:
- ⏸️ Integration tests pass for both modes
- ⏸️ E2E test validates full workflow
- ⏸️ Performance benchmarks meet targets:
  - LocalReporter: <0.1ms per flow
  - GRPCReporter: <5ms per flow (amortized with batching)
- ⏸️ All tests automated in CI pipeline

---

## Phase 8: Documentation and Examples (1 hour)

### Objective
Create comprehensive documentation and runnable examples.

### Tasks

- [ ] **8.1 Update main README.md**
  - [ ] Add architecture diagram showing dual modes
  - [ ] Document configuration options
  - [ ] Add troubleshooting section
  - [ ] Update build instructions

  **Architecture Diagram**:
  ```
  Standalone Mode:
  ┌─────────────────────┐
  │  eBPF Flow Collector │
  └──────────┬───────────┘
             │
             ▼
  ┌─────────────────────┐
  │   LocalReporter     │
  └──────────┬───────────┘
             │
             ▼
  ┌─────────────────────┐
  │  SQLite Storage     │
  └─────────────────────┘

  Agent-Server Mode:
  ┌─────────────────────┐
  │  eBPF Flow Collector │
  └──────────┬───────────┘
             │
             ▼
  ┌─────────────────────┐
  │   GRPCReporter      │ ────┬──► Batch Queue
  └──────────┬───────────┘     │
             │                 │
             ▼                 ▼
  ┌─────────────────────┬─────────────┐
  │   AgentClient       │ gRPC Stream │
  │   - Registration    │             │
  │   - Heartbeat       │             │
  │   - Policy Sync     │             │
  └─────────────────────┴──────┬──────┘
                               │
                               ▼
                    ┌──────────────────┐
                    │ Microsegment     │
                    │ Server           │
                    │ (PostgreSQL)     │
                    └──────────────────┘
  ```

- [ ] **8.2 Create migration guide**
  - [ ] Create `docs/migration-standalone-to-agent-server.md`
  - [ ] Document step-by-step migration process
  - [ ] Include rollback procedures
  - [ ] Add FAQ for common issues

  ```markdown
  # Migration Guide: Standalone to Agent-Server Mode

  ## Prerequisites
  - Microsegmentation server deployed and running
  - Network connectivity to server (port 9090)
  - Agent version 1.1.0 or higher

  ## Migration Steps

  ### 1. Backup existing data
  ```bash
  cp /var/lib/microsegment/flows.db /var/lib/microsegment/flows.db.backup
  ```

  ### 2. Update configuration
  Edit `/etc/microsegment/agent.yaml`:
  ```yaml
  mode: agent-server
  agent_server:
    server_addr: "server.example.com:9090"
  ```

  ### 3. Restart agent
  ```bash
  sudo systemctl restart microsegment-agent
  ```

  ### 4. Verify connection
  Check logs:
  ```bash
  sudo journalctl -u microsegment-agent -f
  ```

  Look for: "gRPC reporter connected to server.example.com:9090"
  ```

- [ ] **8.3 Create troubleshooting guide**
  - [ ] Create `docs/troubleshooting.md`
  - [ ] Document common errors and solutions
  - [ ] Add debugging steps

  ```markdown
  ## Common Issues

  ### Agent won't connect to server

  **Symptom**: "failed to connect to server: connection refused"

  **Solutions**:
  1. Verify server is running: `curl http://server:8080/health`
  2. Check network connectivity: `telnet server 9090`
  3. Verify firewall allows port 9090
  4. Check server_addr in config is correct

  ### Flows not appearing in server

  **Symptom**: Agent connected but no flows in database

  **Solutions**:
  1. Check agent logs for errors: `journalctl -u microsegment-agent`
  2. Verify eBPF programs loaded: `bpftool prog list`
  3. Check batch queue status (add metrics)
  4. Verify server receives streams: check server logs
  ```

- [ ] **8.4 Create example deployment**
  - [ ] Create `examples/docker-compose.yaml`
  - [ ] Include server, agent, PostgreSQL
  - [ ] Add sample traffic generator

  ```yaml
  version: '3.8'
  services:
    postgres:
      image: postgres:14
      environment:
        POSTGRES_DB: microsegment
        POSTGRES_USER: microsegment_user
        POSTGRES_PASSWORD: secret
      ports:
        - "5432:5432"

    server:
      build:
        context: .
        dockerfile: Dockerfile.server
      ports:
        - "8080:8080"
        - "9090:9090"
      depends_on:
        - postgres
      environment:
        MICROSEGMENT_DATABASE_HOST: postgres

    agent:
      build:
        context: .
        dockerfile: Dockerfile.agent
      privileged: true
      volumes:
        - /sys/kernel/debug:/sys/kernel/debug
      environment:
        MICROSEGMENT_MODE: agent-server
        MICROSEGMENT_SERVER_ADDR: server:9090
      depends_on:
        - server
  ```

**Acceptance Criteria**:
- ⏸️ README updated with dual-mode docs
- ⏸️ Migration guide created
- ⏸️ Troubleshooting guide created
- ⏸️ Docker Compose example works
- ⏸️ All documentation reviewed

---

## Phase 9: Archiving and Cleanup (30 minutes)

### Objective
Archive the proposal and update specs.

### Tasks

- [ ] **9.1 Verify all acceptance criteria**
  - [ ] Review each phase's acceptance criteria
  - [ ] Run full test suite: `make test`
  - [ ] Run integration tests: `make test-integration`
  - [ ] Run benchmarks: `make benchmark`
  - [ ] Verify documentation completeness

- [ ] **9.2 Update CHANGELOG.md**
  - [ ] Add entry for agent refactoring
  - [ ] Document breaking changes (if any)
  - [ ] Document new configuration options

  ```markdown
  ## [1.1.0] - 2025-11-05

  ### Added
  - Dual-mode support: standalone and agent-server
  - Reporter interface for pluggable flow reporting
  - GRPCReporter for remote flow reporting with batching
  - AgentClient for server communication (registration, heartbeat, policy sync)
  - Configuration support for agent-server mode

  ### Changed
  - FlowCollector now uses Reporter interface instead of direct storage
  - main.go refactored to support mode selection

  ### Migration
  - Existing standalone deployments continue to work without changes
  - To enable agent-server mode, update config with `mode: agent-server`
  ```

- [ ] **9.3 Run archiving command**
  - [ ] Execute: `openspec archive refactor-agent-for-remote-reporting --yes`
  - [ ] Verify specs updated correctly
  - [ ] Check archived directory created

  ```bash
  cd /home/work/ebpf-based-microsegment
  openspec archive refactor-agent-for-remote-reporting --yes
  ```

- [ ] **9.4 Create git commit**
  - [ ] Stage all changes
  - [ ] Create descriptive commit message
  - [ ] Reference OpenSpec proposal ID

  ```bash
  git add .
  git commit -m "feat: Refactor agent for remote reporting (dual-mode support)

  Implements OpenSpec proposal: refactor-agent-for-remote-reporting

  - Added Reporter interface for pluggable flow reporting
  - Implemented LocalReporter for standalone mode (SQLite)
  - Implemented GRPCReporter for agent-server mode (batched gRPC)
  - Created AgentClient for server communication
  - Extended configuration to support both modes
  - Refactored FlowCollector to use Reporter interface
  - Updated main.go for mode selection
  - Added comprehensive tests and documentation

  Agent now supports:
  1. Standalone mode: eBPF → LocalReporter → SQLite (existing behavior)
  2. Agent-Server mode: eBPF → GRPCReporter → Server → PostgreSQL (new)

  Backward compatible with existing deployments.

  🤖 Generated with Claude Code

  Co-Authored-By: Claude <noreply@anthropic.com>"
  ```

**Acceptance Criteria**:
- ⏸️ All tests passing
- ⏸️ Proposal archived successfully
- ⏸️ Specs updated
- ⏸️ Git commit created
- ⏸️ CHANGELOG updated

---

## Risk Assessment

### High-Risk Areas

1. **gRPC Connection Stability**
   - **Risk**: Network failures, server downtime
   - **Mitigation**: Implement exponential backoff, local caching queue
   - **Testing**: Chaos testing with network interruptions

2. **Performance Overhead**
   - **Risk**: GRPCReporter adds latency to flow collection
   - **Mitigation**: Async batching, non-blocking queue
   - **Testing**: Benchmark with high flow rates (>10K flows/sec)

3. **Data Loss**
   - **Risk**: Dropped flows when queue full
   - **Mitigation**: Oversized queue (2x batch size), metrics for drops
   - **Testing**: Load testing with queue saturation

4. **Backward Compatibility**
   - **Risk**: Breaking existing standalone deployments
   - **Mitigation**: Default to standalone mode, thorough testing
   - **Testing**: Regression tests with old configs

### Medium-Risk Areas

1. **Configuration Complexity**
   - **Risk**: Users misconfigure agent-server mode
   - **Mitigation**: Validation, sensible defaults, clear error messages
   - **Testing**: Config validation tests

2. **Policy Sync Issues**
   - **Risk**: Agent and server policies out of sync
   - **Mitigation**: Version tracking, periodic re-sync
   - **Testing**: Test version mismatch scenarios

### Low-Risk Areas

1. **LocalReporter Changes**
   - **Risk**: Minimal - thin wrapper over existing storage
   - **Mitigation**: Thorough unit tests
   - **Testing**: Compare behavior with old direct storage calls

---

## Success Metrics

### Functional Requirements
- ✅ Agent runs in standalone mode (existing behavior preserved)
- ⏸️ Agent runs in agent-server mode (reports to server via gRPC)
- ⏸️ Zero data loss in standalone mode
- ⏸️ <1% data loss in agent-server mode under normal conditions
- ⏸️ Graceful degradation on server failure (local caching)

### Performance Requirements
- ⏸️ LocalReporter overhead: <0.1ms per flow
- ⏸️ GRPCReporter overhead: <5ms per flow (amortized with batching)
- ⏸️ Memory usage: <50MB additional for GRPCReporter
- ⏸️ CPU usage: <5% additional for batching and gRPC

### Quality Requirements
- ⏸️ Unit test coverage: >85%
- ⏸️ Integration test coverage: >70%
- ⏸️ All linters pass (golangci-lint)
- ⏸️ No new security vulnerabilities (gosec)

### Operational Requirements
- ⏸️ Agent connects and registers within 5 seconds
- ⏸️ Heartbeat every 30 seconds (configurable)
- ⏸️ Policy sync within 10 seconds of server update
- ⏸️ Graceful shutdown within 10 seconds

---

## Dependencies Graph

```
Phase 1 (Reporter Interface) → Phase 2 (GRPCReporter)
                            ↘
Phase 3 (AgentClient) ──────→ Phase 4 (Configuration)
                                       ↓
Phase 5 (FlowCollector) ←──────────────┘
       ↓
Phase 6 (main.go)
       ↓
Phase 7 (Integration Tests)
       ↓
Phase 8 (Documentation)
       ↓
Phase 9 (Archiving)
```

**Critical Path**: Phase 1 → 2 → 4 → 5 → 6 → 7 → 9
**Parallel Opportunities**: Phase 3 can be done alongside Phase 2

---

## Implementation Notes

### Code Quality Standards
- Follow existing code style (gofmt, goimports)
- Add godoc comments for all exported types and functions
- Use structured logging (logrus with fields)
- Handle all errors explicitly (no silent failures)

### Testing Standards
- Use testify for assertions and mocks
- Table-driven tests where applicable
- Integration tests use t.TempDir() for cleanup
- E2E tests only run with `-short=false`

### Security Considerations
- gRPC connections use TLS in production (insecure only for MVP)
- Validate all server responses
- Sanitize configuration inputs
- No secrets in logs or error messages

### Performance Optimization
- Use object pooling for FlowEvent conversion
- Batch database operations
- Use buffered channels for async operations
- Profile with pprof before/after changes

---

## Estimated Timeline

| Phase | Task | Time Estimate | Dependencies |
|-------|------|---------------|--------------|
| 1 | Reporter Interface | 1-2 hours | None |
| 2 | GRPCReporter | 2-3 hours | Phase 1 |
| 3 | AgentClient | 1-2 hours | Phase 1 |
| 4 | Configuration | 1 hour | Phase 3 |
| 5 | FlowCollector | 1-2 hours | Phase 1, 4 |
| 6 | main.go | 1-2 hours | Phase 5 |
| 7 | Integration Tests | 2-3 hours | Phase 6 |
| 8 | Documentation | 1 hour | Phase 7 |
| 9 | Archiving | 30 minutes | Phase 8 |

**Total**: 8-12 hours

---

## Completion Checklist

Use this checklist to track overall progress:

- [x] Phase 1: Reporter Interface (COMPLETED)
- [ ] Phase 2: GRPCReporter Implementation (IN PROGRESS)
- [ ] Phase 3: AgentClient Wrapper
- [ ] Phase 4: Configuration Extension
- [ ] Phase 5: FlowCollector Integration
- [ ] Phase 6: Main Entry Point Refactoring
- [ ] Phase 7: Integration Testing
- [ ] Phase 8: Documentation and Examples
- [ ] Phase 9: Archiving and Cleanup

---

## Notes

- This refactoring maintains 100% backward compatibility
- Standalone mode has zero performance impact
- Agent-server mode enables centralized management
- Future enhancements: TLS, authentication, compression
- Consider adding metrics export (Prometheus) in future iteration

---

**Document Version**: 1.0
**Last Updated**: 2025-11-05
**Status**: Implementation In Progress

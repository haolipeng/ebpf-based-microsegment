package reporter

import (
	"context"
	"fmt"
	"strings"
	"time"

	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/flow"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/netutil"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCReporter reports flows to remote server via gRPC
type GRPCReporter struct {
	serverAddr string
	agentID    string
	conn       *grpc.ClientConn
	client     flowpb.FlowServiceClient
	batchSize  int
	batchQueue chan *flowpb.FlowEvent
	stopCh     chan struct{}

	// Retry configuration
	maxRetries     int
	retryBaseDelay time.Duration
	retryMaxDelay  time.Duration

	// Metrics
	totalSent    uint64
	totalFailed  uint64
	totalRetried uint64
}

// NewGRPCReporter creates a new GRPCReporter
func NewGRPCReporter(serverAddr, agentID string, batchSize int) *GRPCReporter {
	if batchSize == 0 {
		batchSize = 100 // Default batch size
	}

	return &GRPCReporter{
		serverAddr:     serverAddr,
		agentID:        agentID,
		batchSize:      batchSize,
		batchQueue:     make(chan *flowpb.FlowEvent, batchSize*2),
		stopCh:         make(chan struct{}),
		maxRetries:     3,                // 最多重试3次
		retryBaseDelay: 1 * time.Second,  // 基础延迟1秒
		retryMaxDelay:  30 * time.Second, // 最大延迟30秒
	}
}

// NewGRPCReporterWithRetry creates a new GRPCReporter with custom retry settings
func NewGRPCReporterWithRetry(serverAddr, agentID string, batchSize, maxRetries int, retryBaseDelay, retryMaxDelay time.Duration) *GRPCReporter {
	if batchSize == 0 {
		batchSize = 100
	}
	if maxRetries < 0 {
		maxRetries = 0 // 0 means no retry
	}
	if retryBaseDelay <= 0 {
		retryBaseDelay = 1 * time.Second
	}
	if retryMaxDelay <= 0 {
		retryMaxDelay = 30 * time.Second
	}

	return &GRPCReporter{
		serverAddr:     serverAddr,
		agentID:        agentID,
		batchSize:      batchSize,
		batchQueue:     make(chan *flowpb.FlowEvent, batchSize*2),
		stopCh:         make(chan struct{}),
		maxRetries:     maxRetries,
		retryBaseDelay: retryBaseDelay,
		retryMaxDelay:  retryMaxDelay,
	}
}

// Start connects to server and starts batch sender
func (r *GRPCReporter) Start() error {
	// Connect to server
	conn, err := grpc.NewClient(r.serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	r.conn = conn
	r.client = flowpb.NewFlowServiceClient(conn)

	logrus.Infof("gRPC reporter connected to %s (agent-server mode)", r.serverAddr)

	// Start batch sender goroutine
	go r.batchSender()

	return nil
}

// Stop closes connection and stops sender
func (r *GRPCReporter) Stop() error {
	close(r.stopCh)
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// Report queues a flow for batched sending
func (r *GRPCReporter) Report(ctx context.Context, f *flow.Flow) error {
	event := r.flowToProto(f)
	select {
	case r.batchQueue <- event:
		return nil
	default:
		//The buffer queue is full; data should be discarded.
		return fmt.Errorf("batch queue full, dropping flow")
	}
}

// ReportBatch sends multiple flows immediately
func (r *GRPCReporter) ReportBatch(ctx context.Context, flows []*flow.Flow) error {
	events := make([]*flowpb.FlowEvent, len(flows))
	for i, f := range flows {
		events[i] = r.flowToProto(f)
	}

	return r.sendBatch(ctx, events)
}

// batchSender periodically sends batched flows
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
			//Periodic reporting
			if len(batch) > 0 {
				r.sendBatchAsync(batch)
				batch = make([]*flowpb.FlowEvent, 0, r.batchSize)
			}

		case <-r.stopCh:
			// report remaining events when received stop signal
			if len(batch) > 0 {
				r.sendBatchAsync(batch)
			}
			return
		}
	}
}

// sendBatchAsync sends batch without blocking, with retry mechanism
func (r *GRPCReporter) sendBatchAsync(events []*flowpb.FlowEvent) {
	go func() {
		if err := r.sendBatchWithRetry(events); err != nil {
			logrus.Errorf("Failed to send batch after %d retries: %v", r.maxRetries, err)
			r.totalFailed++
		} else {
			logrus.Debugf("Successfully sent %d flow events to server", len(events))
			r.totalSent++
		}
	}()
}

// sendBatchWithRetry sends batch with exponential backoff retry
func (r *GRPCReporter) sendBatchWithRetry(events []*flowpb.FlowEvent) error {
	var lastErr error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			// exponential backoff：1s, 2s, 4s, 8s, 16s, 30s(max)
			delay := r.retryBaseDelay * time.Duration(1<<uint(attempt-1))
			if delay > r.retryMaxDelay {
				delay = r.retryMaxDelay
			}

			logrus.Warnf("Retry attempt %d/%d after %v delay", attempt, r.maxRetries, delay)
			time.Sleep(delay)
			r.totalRetried++
		}

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := r.sendBatch(ctx, events)
		cancel()

		if err == nil {
			// send successed
			if attempt > 0 {
				logrus.Infof("Batch sent successfully after %d retries", attempt)
			}
			return nil
		}

		lastErr = err
		logrus.Warnf("Send attempt %d failed: %v", attempt+1, err)
	}

	return fmt.Errorf("all retry attempts exhausted: %w", lastErr)
}

// sendBatch sends a batch of events via gRPC streaming
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

// flowToProto converts internal Flow to protobuf FlowEvent
func (r *GRPCReporter) flowToProto(f *flow.Flow) *flowpb.FlowEvent {
	// Convert IP strings to uint32
	srcIP := netutil.StringToUint32LE(f.SourceIP)
	dstIP := netutil.StringToUint32LE(f.DestIP)

	// Map protocol string to protobuf enum
	protocol := protocolStringToEnum(f.Protocol)

	// Map event type string to protobuf enum
	eventType := eventTypeStringToEnum(f.EventType)

	// Map state string to protobuf enum
	state := stateStringToEnum(f.State)

	// Map direction string to protobuf enum
	direction := directionStringToEnum(f.Direction)

	// Map policy action string to protobuf enum
	policyAction := policyActionStringToEnum(f.PolicyAction)

	return &flowpb.FlowEvent{
		SrcIp:        srcIP,
		DstIp:        dstIP,
		SrcPort:      uint32(f.SourcePort),
		DstPort:      uint32(f.DestPort),
		Protocol:     protocol,
		EventType:    eventType,
		Direction:    direction,
		PacketCount:  f.PacketCount,
		ByteCount:    f.ByteCount,
		TimestampNs:  uint64(f.StartTime.UnixNano()),
		PolicyId:     f.PolicyID,
		PolicyAction: policyAction,
		State:        state,
		AgentId:      r.agentID,
		SourceLabels: f.SourceLabels,
		DestLabels:   f.DestLabels,
	}
}

// protocolStringToEnum converts protocol string to protobuf enum
func protocolStringToEnum(protocol string) commonpb.Protocol {
	switch strings.ToUpper(protocol) {
	case "TCP":
		return commonpb.Protocol_PROTOCOL_TCP
	case "UDP":
		return commonpb.Protocol_PROTOCOL_UDP
	case "ICMP":
		return commonpb.Protocol_PROTOCOL_ICMP
	case "ANY":
		return commonpb.Protocol_PROTOCOL_ANY
	default:
		return commonpb.Protocol_PROTOCOL_UNKNOWN
	}
}

// eventTypeStringToEnum converts event type string to protobuf enum
func eventTypeStringToEnum(eventType string) commonpb.FlowEventType {
	switch strings.ToUpper(eventType) {
	case "NEW":
		return commonpb.FlowEventType_EVENT_NEW
	case "UPDATE":
		return commonpb.FlowEventType_EVENT_UPDATE
	case "CLOSED":
		return commonpb.FlowEventType_EVENT_CLOSED
	case "TIMEOUT":
		return commonpb.FlowEventType_EVENT_TIMEOUT
	default:
		return commonpb.FlowEventType_EVENT_UNKNOWN
	}
}

// stateStringToEnum converts state string to protobuf enum
func stateStringToEnum(state string) commonpb.FlowState {
	switch strings.ToUpper(state) {
	case "ACTIVE":
		return commonpb.FlowState_STATE_ACTIVE
	case "CLOSED":
		return commonpb.FlowState_STATE_CLOSED
	case "TIMEOUT":
		return commonpb.FlowState_STATE_TIMEOUT
	default:
		return commonpb.FlowState_STATE_UNKNOWN
	}
}

// directionStringToEnum converts direction string to protobuf enum
func directionStringToEnum(direction string) commonpb.FlowDirection {
	switch strings.ToUpper(direction) {
	case "INGRESS":
		return commonpb.FlowDirection_DIRECTION_INGRESS
	case "EGRESS":
		return commonpb.FlowDirection_DIRECTION_EGRESS
	default:
		return commonpb.FlowDirection_DIRECTION_UNKNOWN
	}
}

// policyActionStringToEnum converts policy action string to protobuf enum
func policyActionStringToEnum(action string) commonpb.PolicyAction {
	switch strings.ToUpper(action) {
	case "ALLOW":
		return commonpb.PolicyAction_ACTION_ALLOW
	case "DENY":
		return commonpb.PolicyAction_ACTION_DENY
	case "LOG":
		return commonpb.PolicyAction_ACTION_LOG
	default:
		return commonpb.PolicyAction_ACTION_UNKNOWN
	}
}

// GetMetrics returns reporter metrics
func (r *GRPCReporter) GetMetrics() (sent, failed, retried uint64) {
	return r.totalSent, r.totalFailed, r.totalRetried
}

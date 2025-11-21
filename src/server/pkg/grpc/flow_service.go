package grpc

import (
	"context"
	"fmt"
	"io"

	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/netutil"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/storage"
	ws "github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/websocket"
	"github.com/sirupsen/logrus"
)

// FlowServiceServer implements flowpb.FlowServiceServer
type FlowServiceServer struct {
	flowpb.UnimplementedFlowServiceServer
	flowStorage *storage.FlowStorage
	wsHub       *ws.Hub
}

// NewFlowServiceServer creates a new FlowServiceServer
func NewFlowServiceServer(flowStorage *storage.FlowStorage, wsHub *ws.Hub) *FlowServiceServer {
	return &FlowServiceServer{
		flowStorage: flowStorage,
		wsHub:       wsHub,
	}
}

// ReportFlowEvents handles streaming flow events from agents
func (s *FlowServiceServer) ReportFlowEvents(stream flowpb.FlowService_ReportFlowEventsServer) error {
	ctx := stream.Context()
	events := []*flowpb.FlowEvent{}
	acceptedCount := 0
	rejectedCount := 0

	// Receive all events from stream
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			// Stream finished, process batch
			break
		}
		if err != nil {
			logrus.Errorf("Failed to receive flow event: %v", err)
			return fmt.Errorf("failed to receive flow event: %w", err)
		}

		// Basic validation
		if event.AgentId == "" {
			logrus.Warnf("Rejected flow event: missing agent_id")
			rejectedCount++
			continue
		}

		events = append(events, event)
		acceptedCount++
	}

	// Save batch to database
	if len(events) > 0 {
		if err := s.flowStorage.BatchSaveFlowEvents(ctx, events); err != nil {
			logrus.Errorf("Failed to save flow events: %v", err)
			return stream.SendAndClose(&commonpb.ReportResponse{
				Success:        false,
				Message:        fmt.Sprintf("Failed to save events: %v", err),
				AcceptedCount:  0,
				RejectedCount:  uint32(acceptedCount + rejectedCount),
			})
		}

		// Broadcast flows to WebSocket clients (real-time streaming)
		// Convert each FlowEvent to Flow and broadcast
		for _, event := range events {
			flow := s.eventToFlow(event)
			if s.wsHub != nil {
				s.wsHub.Broadcast(flow)
			}
		}
	}

	logrus.Infof("Received %d flow events (%d accepted, %d rejected)",
		acceptedCount+rejectedCount, acceptedCount, rejectedCount)

	return stream.SendAndClose(&commonpb.ReportResponse{
		Success:        true,
		Message:        "Flow events received successfully",
		AcceptedCount:  uint32(acceptedCount),
		RejectedCount:  uint32(rejectedCount),
	})
}

// QueryFlows handles flow query requests
func (s *FlowServiceServer) QueryFlows(ctx context.Context, query *flowpb.FlowQuery) (*flowpb.FlowQueryResponse, error) {
	flows, total, err := s.flowStorage.QueryFlows(ctx, query)
	if err != nil {
		logrus.Errorf("Failed to query flows: %v", err)
		return nil, fmt.Errorf("failed to query flows: %w", err)
	}

	hasMore := (query.Offset + query.Limit) < uint32(total)

	return &flowpb.FlowQueryResponse{
		Flows:      flows,
		TotalCount: uint64(total),
		HasMore:    hasMore,
	}, nil
}

// GetFlowSummary returns aggregated flow statistics
func (s *FlowServiceServer) GetFlowSummary(ctx context.Context, req *flowpb.FlowSummaryRequest) (*flowpb.FlowSummary, error) {
	// MVP: Return basic statistics
	// TODO: Implement proper aggregation
	return &flowpb.FlowSummary{
		TotalFlows:   0,
		TotalPackets: 0,
		TotalBytes:   0,
	}, nil
}

// eventToFlow converts a FlowEvent (from agent) to Flow (for storage/WebSocket)
func (s *FlowServiceServer) eventToFlow(event *flowpb.FlowEvent) *flowpb.Flow {
	// Convert IP addresses from uint32 to string
	srcIP := netutil.Uint32ToString(event.SrcIp)
	dstIP := netutil.Uint32ToString(event.DstIp)

	return &flowpb.Flow{
		// Note: id will be 0 (database will assign proper ID on insert)
		AgentId:      event.AgentId,
		SrcIp:        srcIP,
		DstIp:        dstIP,
		SrcPort:      event.SrcPort,
		DstPort:      event.DstPort,
		Protocol:     event.Protocol,
		Direction:    event.Direction,
		PacketCount:  event.PacketCount,
		ByteCount:    event.ByteCount,
		StartTime:    int64(event.TimestampNs), // Convert uint64 to int64
		PolicyId:     event.PolicyId,
		PolicyAction: event.PolicyAction,
		State:        event.State,
		SourceLabels: event.SourceLabels,
		DestLabels:   event.DestLabels,
	}
}

package grpc

import (
	"context"
	"fmt"
	"io"

	commonpb "github.com/ebpf-microsegment/src/proto/common"
	flowpb "github.com/ebpf-microsegment/src/proto/flow"
	"github.com/ebpf-microsegment/src/server/pkg/storage"
	"github.com/sirupsen/logrus"
)

// FlowServiceServer implements flowpb.FlowServiceServer
type FlowServiceServer struct {
	flowpb.UnimplementedFlowServiceServer
	flowStorage *storage.FlowStorage
}

// NewFlowServiceServer creates a new FlowServiceServer
func NewFlowServiceServer(flowStorage *storage.FlowStorage) *FlowServiceServer {
	return &FlowServiceServer{
		flowStorage: flowStorage,
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

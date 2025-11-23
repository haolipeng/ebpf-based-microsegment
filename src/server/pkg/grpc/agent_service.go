package grpc

import (
	"context"
	"fmt"
	"time"

	agentpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/agent"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/storage"
	"github.com/sirupsen/logrus"
)

// AgentServiceServer implements agentpb.AgentServiceServer
type AgentServiceServer struct {
	agentpb.UnimplementedAgentServiceServer
	agentStorage *storage.AgentStorage
}

// NewAgentServiceServer creates a new AgentServiceServer
func NewAgentServiceServer(agentStorage *storage.AgentStorage) *AgentServiceServer {
	return &AgentServiceServer{
		agentStorage: agentStorage,
	}
}

// RegisterAgent handles agent registration requests
func (s *AgentServiceServer) RegisterAgent(ctx context.Context, req *agentpb.RegisterRequest) (*agentpb.RegisterResponse, error) {
	logrus.Infof("Agent registration: %s (hostname: %s, version: %s)",
		req.AgentId, req.Hostname, req.Version)

	if err := s.agentStorage.RegisterAgent(ctx, req); err != nil {
		logrus.Errorf("Failed to register agent: %v", err)
		return &agentpb.RegisterResponse{
			Success: false,
			Message: fmt.Sprintf("Registration failed: %v", err),
		}, nil
	}

	// Return server configuration
	config := &agentpb.AgentConfig{
		HeartbeatInterval:  30,  // 30 seconds
		StatsInterval:      60,  // 60 seconds
		FlowBatchSize:      100, // 100 events per batch
		FlowBatchTimeout:   5,   // 5 seconds timeout
		DebugMode:          false,
	}

	return &agentpb.RegisterResponse{
		Success:       true,
		Message:       "Agent registered successfully",
		ServerVersion: "1.0.0-mvp",
		ServerTime:    time.Now().UnixNano(),
		Config:        config,
	}, nil
}

// Heartbeat handles periodic health check requests from agents
func (s *AgentServiceServer) Heartbeat(ctx context.Context, req *agentpb.HeartbeatRequest) (*agentpb.HeartbeatResponse, error) {
	logrus.Debugf("Heartbeat from agent: %s", req.AgentId)

	if err := s.agentStorage.UpdateHeartbeat(ctx, req.AgentId, req.Metrics); err != nil {
		logrus.Errorf("Failed to update heartbeat: %v", err)
		return &agentpb.HeartbeatResponse{
			Healthy: false,
			Message: fmt.Sprintf("Heartbeat processing failed: %v", err),
		}, nil
	}

	return &agentpb.HeartbeatResponse{
		Healthy:    true,
		Message:    "Heartbeat received",
		ServerTime: time.Now().UnixNano(),
		Commands:   []*agentpb.AgentCommand{}, // No commands in MVP
	}, nil
}

// ReportStatus handles detailed status reports from agents
func (s *AgentServiceServer) ReportStatus(ctx context.Context, report *agentpb.StatusReport) (*agentpb.StatusResponse, error) {
	logrus.Infof("Status report from agent %s: status=%s, uptime=%ds, policy_count=%d, workload_count=%d",
		report.AgentId, report.Status, report.Uptime, report.PolicyCount, report.WorkloadCount)

	// Persist status report to database
	if err := s.agentStorage.UpdateStatusReport(ctx, report); err != nil {
		logrus.Errorf("Failed to persist status report from agent %s: %v", report.AgentId, err)
		return &agentpb.StatusResponse{
			Success:  false,
			Message:  fmt.Sprintf("Failed to persist status report: %v", err),
			Commands: []*agentpb.AgentCommand{},
		}, nil
	}

	// Log errors if present
	if len(report.Errors) > 0 {
		logrus.Warnf("Agent %s reported %d errors: %v", report.AgentId, len(report.Errors), report.Errors)
	}

	return &agentpb.StatusResponse{
		Success:  true,
		Message:  "Status report received and persisted",
		Commands: []*agentpb.AgentCommand{},
	}, nil
}

// UnregisterAgent handles graceful agent shutdown
func (s *AgentServiceServer) UnregisterAgent(ctx context.Context, req *agentpb.UnregisterRequest) (*agentpb.UnregisterResponse, error) {
	logrus.Infof("Agent unregistration: %s (reason: %s)", req.AgentId, req.Reason)

	// Mark agent as offline with reason
	reason := req.Reason
	if reason == "" {
		reason = "graceful shutdown"
	}

	if err := s.agentStorage.MarkAgentOffline(ctx, req.AgentId, reason); err != nil {
		logrus.Errorf("Failed to mark agent %s as offline: %v", req.AgentId, err)
		return &agentpb.UnregisterResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to unregister agent: %v", err),
		}, nil
	}

	logrus.Infof("Agent %s marked as offline (reason: %s)", req.AgentId, reason)

	return &agentpb.UnregisterResponse{
		Success: true,
		Message: "Agent unregistered and marked as offline",
	}, nil
}

package grpc

import (
	"context"
	"fmt"
	"time"

	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/storage"
	"github.com/sirupsen/logrus"
)

// PolicyServiceServer implements policypb.PolicyServiceServer
type PolicyServiceServer struct {
	policypb.UnimplementedPolicyServiceServer
	policyStorage *storage.PolicyStorage
}

// NewPolicyServiceServer creates a new PolicyServiceServer
func NewPolicyServiceServer(policyStorage *storage.PolicyStorage) *PolicyServiceServer {
	return &PolicyServiceServer{
		policyStorage: policyStorage,
	}
}

// SyncPolicies handles full policy synchronization requests
func (s *PolicyServiceServer) SyncPolicies(ctx context.Context, req *policypb.SyncRequest) (*policypb.SyncResponse, error) {
	logrus.Infof("Policy sync request from agent: %s (current version: %d)",
		req.AgentId, req.PolicyVersion)

	policies, version, err := s.policyStorage.GetAllPolicies(ctx)
	if err != nil {
		logrus.Errorf("Failed to get policies: %v", err)
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}

	return &policypb.SyncResponse{
		Policies:      policies,
		PolicyVersion: version,
		ServerTime:    time.Now().UnixNano(),
		PolicyCount:   uint32(len(policies)),
	}, nil
}

// SubscribePolicies handles policy update subscriptions
func (s *PolicyServiceServer) SubscribePolicies(req *policypb.SubscribeRequest, stream policypb.PolicyService_SubscribePoliciesServer) error {
	logrus.Infof("Policy subscription from agent: %s (current version: %d)",
		req.AgentId, req.CurrentVersion)

	// MVP: Send all current policies and keep connection open
	// TODO: Implement proper pub/sub mechanism for incremental updates

	policies, version, err := s.policyStorage.GetAllPolicies(stream.Context())
	if err != nil {
		return fmt.Errorf("failed to get policies: %w", err)
	}

	// Send initial policies
	for _, policy := range policies {
		update := &policypb.PolicyUpdate{
			UpdateType:    policypb.PolicyUpdateType_UPDATE_ADD,
			Policy:        policy,
			PolicyVersion: version,
			Timestamp:     time.Now().UnixNano(),
		}
		if err := stream.Send(update); err != nil {
			return fmt.Errorf("failed to send policy update: %w", err)
		}
	}

	// Keep connection open (in real implementation, would send updates as they occur)
	<-stream.Context().Done()
	return nil
}

// ReportPolicyStats handles policy enforcement statistics from agents
func (s *PolicyServiceServer) ReportPolicyStats(ctx context.Context, report *policypb.PolicyStatsReport) (*commonpb.ReportResponse, error) {
	logrus.Debugf("Received policy stats from agent %s: %d rules", report.AgentId, len(report.PolicyStats))

	// MVP: Just acknowledge receipt
	// TODO: Persist policy statistics for analysis

	return &commonpb.ReportResponse{
		Success:       true,
		Message:       "Policy stats received",
		AcceptedCount: uint32(len(report.PolicyStats)),
	}, nil
}

package grpc

import (
	"context"
	"fmt"
	"time"

	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/pubsub"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/storage"
	"github.com/sirupsen/logrus"
)

// PolicyServiceServer implements policypb.PolicyServiceServer
type PolicyServiceServer struct {
	policypb.UnimplementedPolicyServiceServer
	policyStorage *storage.PolicyStorage
	pubsub        *pubsub.PolicyPubSub
}

// NewPolicyServiceServer creates a new PolicyServiceServer
func NewPolicyServiceServer(policyStorage *storage.PolicyStorage, policyPubSub *pubsub.PolicyPubSub) *PolicyServiceServer {
	return &PolicyServiceServer{
		policyStorage: policyStorage,
		pubsub:        policyPubSub,
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

// SubscribePolicies handles policy update subscriptions with incremental updates
func (s *PolicyServiceServer) SubscribePolicies(req *policypb.SubscribeRequest, stream policypb.PolicyService_SubscribePoliciesServer) error {
	logrus.Infof("Policy subscription from agent: %s (current version: %d)",
		req.AgentId, req.CurrentVersion)

	// Step 1: Send historical updates since agent's current version
	if req.CurrentVersion > 0 {
		updates, err := s.policyStorage.GetPolicyUpdates(stream.Context(), req.CurrentVersion)
		if err != nil {
			return fmt.Errorf("failed to get policy updates: %w", err)
		}

		logrus.Infof("Sending %d historical updates to agent %s", len(updates), req.AgentId)
		for _, update := range updates {
			if err := stream.Send(update); err != nil {
				return fmt.Errorf("failed to send historical update: %w", err)
			}
		}
	} else {
		// Agent has no policies, send all current policies as ADD updates
		policies, version, err := s.policyStorage.GetAllPolicies(stream.Context())
		if err != nil {
			return fmt.Errorf("failed to get policies: %w", err)
		}

		logrus.Infof("Sending %d initial policies to agent %s", len(policies), req.AgentId)
		for _, policy := range policies {
			update := &policypb.PolicyUpdate{
				UpdateType:    policypb.PolicyUpdateType_UPDATE_ADD,
				Policy:        policy,
				PolicyVersion: version,
				Timestamp:     time.Now().UnixNano(),
			}
			if err := stream.Send(update); err != nil {
				return fmt.Errorf("failed to send initial policy: %w", err)
			}
		}
	}

	// Step 2: Subscribe to future updates
	updateChan := s.pubsub.Subscribe(req.AgentId, 100)
	defer s.pubsub.Unsubscribe(req.AgentId)

	logrus.Infof("Agent %s subscribed to real-time policy updates", req.AgentId)

	// Step 3: Stream updates as they arrive
	for {
		select {
		case <-stream.Context().Done():
			logrus.Infof("Agent %s disconnected from policy stream", req.AgentId)
			return stream.Context().Err()
		case update, ok := <-updateChan:
			if !ok {
				logrus.Warnf("Update channel closed for agent %s", req.AgentId)
				return nil
			}
			if err := stream.Send(update); err != nil {
				logrus.Errorf("Failed to send update to agent %s: %v", req.AgentId, err)
				return fmt.Errorf("failed to send update: %w", err)
			}
			logrus.Debugf("Sent real-time update to agent %s: version=%d, type=%s",
				req.AgentId, update.PolicyVersion, update.UpdateType)
		}
	}
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

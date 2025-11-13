package client

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"time"

	agentpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/agent"
	policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AgentClient manages communication with the microsegmentation server
type AgentClient struct {
	agentID       string
	hostname      string
	version       string
	serverAddr    string

	conn          *grpc.ClientConn
	agentService  agentpb.AgentServiceClient
	policyService policypb.PolicyServiceClient

	heartbeatInterval int
	statsInterval     int
	stopCh            chan struct{}

	// Metrics for heartbeat
	flowCount   uint64
	policyCount uint32
}

// NewAgentClient creates a new AgentClient
func NewAgentClient(serverAddr, agentID, hostname, version string) *AgentClient {
	if hostname == "" {
		hostname, _ = os.Hostname()
	}

	return &AgentClient{
		serverAddr: serverAddr,
		agentID:    agentID,
		hostname:   hostname,
		version:    version,
		stopCh:     make(chan struct{}),
	}
}

// Connect establishes connection to server and registers agent
func (c *AgentClient) Connect() error {
	// Connect to server
	conn, err := grpc.NewClient(
		c.serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.conn = conn
	c.agentService = agentpb.NewAgentServiceClient(conn)
	c.policyService = policypb.NewPolicyServiceClient(conn)

	// Register with server
	if err := c.registerAgent(); err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to register with server: %w", err)
	}

	logrus.Infof("AgentClient connected to %s", c.serverAddr)
	return nil
}

// Close closes the connection to the server
func (c *AgentClient) Close() error {
	close(c.stopCh)

	// Unregister from server
	if c.agentService != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &agentpb.UnregisterRequest{
			AgentId: c.agentID,
			Reason:  "graceful shutdown",
		}

		if _, err := c.agentService.UnregisterAgent(ctx, req); err != nil {
			logrus.Warnf("Failed to unregister agent: %v", err)
		} else {
			logrus.Info("Agent unregistered successfully")
		}
	}

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// registerAgent registers this agent with the server
func (c *AgentClient) registerAgent() error {
	req := &agentpb.RegisterRequest{
		AgentId:     c.agentID,
		Hostname:    c.hostname,
		Version:     c.version,
		IpAddresses: getLocalIPs(),
		Os:          getOSInfo(),
		StartTime:   time.Now().UnixNano(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.agentService.RegisterAgent(ctx, req)
	if err != nil {
		return fmt.Errorf("registration RPC failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("registration rejected: %s", resp.Message)
	}

	// Store server configuration
	c.heartbeatInterval = int(resp.Config.HeartbeatInterval)
	c.statsInterval = int(resp.Config.StatsInterval)

	logrus.WithFields(logrus.Fields{
		"agent_id":           c.agentID,
		"server_version":     resp.ServerVersion,
		"heartbeat_interval": c.heartbeatInterval,
	}).Info("Agent registered successfully")

	return nil
}

// StartHeartbeat starts sending periodic heartbeats to the server
func (c *AgentClient) StartHeartbeat() {
	if c.heartbeatInterval == 0 {
		c.heartbeatInterval = 30 // Default 30 seconds
	}

	ticker := time.NewTicker(time.Duration(c.heartbeatInterval) * time.Second)
	defer ticker.Stop()

	logrus.Infof("Heartbeat started with interval %ds", c.heartbeatInterval)

	for {
		select {
		case <-ticker.C:
			if err := c.sendHeartbeat(); err != nil {
				logrus.Errorf("Heartbeat failed: %v", err)
			}
		case <-c.stopCh:
			logrus.Info("Heartbeat stopped")
			return
		}
	}
}

// sendHeartbeat sends a single heartbeat to the server
func (c *AgentClient) sendHeartbeat() error {
	req := &agentpb.HeartbeatRequest{
		AgentId:   c.agentID,
		Timestamp: time.Now().UnixNano(),
		Metrics: &agentpb.AgentMetrics{
			CpuUsage:       getCPUUsage(),
			MemoryUsage:    getMemoryUsage(),
			FlowsReported:  c.flowCount,
			ActivePolicies: c.policyCount,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.agentService.Heartbeat(ctx, req)
	if err != nil {
		return fmt.Errorf("heartbeat RPC failed: %w", err)
	}

	if !resp.Healthy {
		logrus.Warnf("Heartbeat indicates unhealthy: %s", resp.Message)
	} else {
		logrus.Debug("Heartbeat successful")
	}

	return nil
}

// SyncPolicies synchronizes policies from the server
func (c *AgentClient) SyncPolicies(currentVersion uint64) ([]*policypb.Policy, uint64, error) {
	req := &policypb.SyncRequest{
		AgentId:       c.agentID,
		PolicyVersion: currentVersion,
		LastSyncTime:  time.Now().UnixNano(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.policyService.SyncPolicies(ctx, req)
	if err != nil {
		return nil, 0, fmt.Errorf("policy sync failed: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"policy_count":   resp.PolicyCount,
		"policy_version": resp.PolicyVersion,
		"current_version": currentVersion,
	}).Info("Policies synchronized")

	return resp.Policies, resp.PolicyVersion, nil
}

// UpdateMetrics updates flow and policy counts for heartbeat
func (c *AgentClient) UpdateMetrics(flowCount uint64, policyCount uint32) {
	c.flowCount = flowCount
	c.policyCount = policyCount
}

// getLocalIPs returns all non-loopback IPv4 addresses
func getLocalIPs() []string {
	ips := []string{}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logrus.Warnf("Failed to get network interfaces: %v", err)
		return ips
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}

	return ips
}

// getOSInfo returns OS information string
func getOSInfo() string {
	return fmt.Sprintf("%s %s (%s)", runtime.GOOS, runtime.GOARCH, runtime.Version())
}

// getCPUUsage returns current CPU usage percentage (placeholder)
func getCPUUsage() float32 {
	// TODO: Implement actual CPU usage calculation
	// For now, return 0 as placeholder
	return 0.0
}

// getMemoryUsage returns current memory usage in bytes (placeholder)
func getMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

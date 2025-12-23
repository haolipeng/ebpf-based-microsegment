
// input: agent gRPC server address
// output: gRPC agent client connection
// pos: agent gRPC client - if file updated, must sync with this header comment and pkg/client/CLAUDE.md
package client

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	agentpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/agent"
	policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AgentClient manages communication with the microsegmentation server
type AgentClient struct {
	agentID    string
	hostname   string
	version    string
	serverAddr string

	conn          *grpc.ClientConn
	agentService  agentpb.AgentServiceClient
	policyService policypb.PolicyServiceClient

	heartbeatInterval int
	statsInterval     int
	stopCh            chan struct{}

	// Metrics for heartbeat
	flowCount   uint64
	policyCount uint32

	// Policy version tracking
	policyVersion uint64
	policyMutex   sync.RWMutex
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
		"policy_count":    resp.PolicyCount,
		"policy_version":  resp.PolicyVersion,
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

// cpuStats stores CPU statistics for calculating usage
type cpuStats struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	steal   uint64
}

var (
	lastCPUStats cpuStats
	lastCPUTime  time.Time
	cpuMutex     sync.Mutex
)

// getCPUUsage returns current CPU usage percentage
// Calculates CPU usage by reading /proc/stat and comparing with previous reading
func getCPUUsage() float32 {
	cpuMutex.Lock()
	defer cpuMutex.Unlock()

	stats, err := readCPUStats()
	if err != nil {
		logrus.Debugf("Failed to read CPU stats: %v", err)
		return 0.0
	}

	now := time.Now()

	// First call, initialize baseline
	if lastCPUTime.IsZero() {
		lastCPUStats = stats
		lastCPUTime = now
		return 0.0
	}

	// Calculate deltas
	totalDelta := stats.total() - lastCPUStats.total()
	idleDelta := stats.idle - lastCPUStats.idle

	// Update last stats
	lastCPUStats = stats
	lastCPUTime = now

	// Avoid division by zero
	if totalDelta == 0 {
		return 0.0
	}

	// CPU usage = (totalDelta - idleDelta) / totalDelta * 100
	usage := float32(totalDelta-idleDelta) / float32(totalDelta) * 100.0

	return usage
}

// readCPUStats reads CPU statistics from /proc/stat
func readCPUStats() (cpuStats, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuStats{}, fmt.Errorf("failed to read /proc/stat: %w", err)
	}

	// Find the first "cpu" line (aggregate across all CPUs)
	content := string(data)
	cpuPrefix := "cpu "

	idx := strings.Index(content, cpuPrefix)
	if idx == -1 {
		return cpuStats{}, fmt.Errorf("cpu line not found in /proc/stat")
	}

	// Extract the cpu line (from "cpu " to newline)
	lineStart := idx
	lineEnd := strings.Index(content[lineStart:], "\n")
	if lineEnd == -1 {
		lineEnd = len(content)
	} else {
		lineEnd += lineStart
	}

	cpuLine := content[lineStart:lineEnd]
	fields := strings.Fields(cpuLine)

	if len(fields) < 9 {
		return cpuStats{}, fmt.Errorf("invalid /proc/stat format: expected at least 9 fields, got %d", len(fields))
	}

	// Parse CPU time fields
	// Format: cpu user nice system idle iowait irq softirq steal
	stats := cpuStats{
		user:    parseUint64(fields[1]),
		nice:    parseUint64(fields[2]),
		system:  parseUint64(fields[3]),
		idle:    parseUint64(fields[4]),
		iowait:  parseUint64(fields[5]),
		irq:     parseUint64(fields[6]),
		softirq: parseUint64(fields[7]),
		steal:   parseUint64(fields[8]),
	}

	return stats, nil
}

// total returns total CPU time (sum of all fields)
func (s cpuStats) total() uint64 {
	return s.user + s.nice + s.system + s.idle + s.iowait + s.irq + s.softirq + s.steal
}

// parseUint64 parses a string to uint64, returns 0 on error
func parseUint64(s string) uint64 {
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// getMemoryUsage returns current memory usage in bytes (placeholder)
func getMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

// GetPolicyVersion returns the current policy version
func (c *AgentClient) GetPolicyVersion() uint64 {
	c.policyMutex.RLock()
	defer c.policyMutex.RUnlock()
	return c.policyVersion
}

// SetPolicyVersion updates the current policy version
func (c *AgentClient) SetPolicyVersion(version uint64) {
	c.policyMutex.Lock()
	defer c.policyMutex.Unlock()
	c.policyVersion = version
}

// PolicyUpdateHandler is a callback function for policy updates
type PolicyUpdateHandler func(*policypb.PolicyUpdate) error

// SubscribePolicyUpdates subscribes to policy updates from the server
// The handler function is called for each policy update received
// This function blocks until the stream is closed or an error occurs
func (c *AgentClient) SubscribePolicyUpdates(ctx context.Context, handler PolicyUpdateHandler) error {
	c.policyMutex.RLock()
	currentVersion := c.policyVersion
	c.policyMutex.RUnlock()

	req := &policypb.SubscribeRequest{
		AgentId:        c.agentID,
		CurrentVersion: currentVersion,
	}

	stream, err := c.policyService.SubscribePolicies(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to subscribe to policy updates: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"agent_id":        c.agentID,
		"current_version": currentVersion,
	}).Info("Subscribed to policy updates")

	// Receive policy updates from stream
	for {
		update, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("failed to receive policy update: %w", err)
		}

		logrus.WithFields(logrus.Fields{
			"update_type":    update.UpdateType,
			"rule_id":        update.Policy.GetRuleId(),
			"policy_version": update.PolicyVersion,
		}).Debug("Received policy update")

		// Call handler to process the update
		if err := handler(update); err != nil {
			logrus.Errorf("Failed to handle policy update (version %d): %v", update.PolicyVersion, err)
			// Continue receiving updates even if one fails
			continue
		}

		// Update local policy version
		c.SetPolicyVersion(update.PolicyVersion)

		logrus.WithFields(logrus.Fields{
			"update_type":    update.UpdateType,
			"rule_id":        update.Policy.GetRuleId(),
			"policy_version": update.PolicyVersion,
		}).Info("Policy update applied successfully")
	}
}

// SubscribePolicyUpdatesAsync starts subscribing to policy updates in a background goroutine
// The handler function is called for each policy update received
// Returns a channel that will receive any errors from the subscription
func (c *AgentClient) SubscribePolicyUpdatesAsync(handler PolicyUpdateHandler) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		ctx := context.Background()
		if err := c.SubscribePolicyUpdates(ctx, handler); err != nil {
			logrus.Errorf("Policy update subscription error: %v", err)
			errCh <- err
		}
		close(errCh)
	}()

	return errCh
}

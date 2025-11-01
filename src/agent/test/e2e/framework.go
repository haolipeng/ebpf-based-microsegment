// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// Package e2e provides end-to-end testing framework for eBPF microsegmentation.
// It creates isolated network environments and tests the complete data path
// from API to eBPF packet filtering.
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/api/models"
	"github.com/ebpf-microsegment/src/agent/pkg/dataplane"
	"github.com/ebpf-microsegment/src/agent/pkg/policy"
	"github.com/ebpf-microsegment/src/agent/pkg/testutil"
	"github.com/stretchr/testify/require"
)

// E2ETestEnv represents a complete end-to-end test environment.
// It includes isolated network namespaces, eBPF data plane, and API server.
type E2ETestEnv struct {
	T              *testing.T
	Network        *testutil.TestNetwork
	DataPlane      *dataplane.DataPlane
	PolicyManager  *policy.PolicyManager
	Storage        *policy.SQLiteStorage
	StoragePath    string
	HTTPClient     *http.Client
	APIBaseURL     string
	cleanupFuncs   []func()
}

// NewE2ETestEnv creates a new end-to-end test environment.
// It sets up network namespaces, loads eBPF programs, and prepares the test infrastructure.
//
// The environment includes:
//   - Two network namespaces (client and server)
//   - Virtual ethernet pair connecting them
//   - eBPF program attached to the server veth
//   - Policy manager with storage
//   - HTTP client for API testing
//
// Returns:
//   - *E2ETestEnv: The test environment
//   - error: Error if setup fails
func NewE2ETestEnv(t *testing.T) (*E2ETestEnv, error) {
	env := &E2ETestEnv{
		T:            t,
		HTTPClient:   &http.Client{Timeout: 5 * time.Second},
		cleanupFuncs: make([]func(), 0),
	}

	// Check requirements
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		return nil, fmt.Errorf("E2E requirements not met: %s", msg)
	}

	// Create network environment
	network, err := testutil.NewTestNetwork()
	if err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("failed to create test network: %w", err)
	}
	env.Network = network
	env.addCleanup(network.Cleanup)

	// Create temporary storage for policies
	dbPath := fmt.Sprintf("/tmp/e2e_test_%d.db", time.Now().UnixNano())
	env.StoragePath = dbPath

	storage, err := policy.NewSQLiteStorage(dbPath)
	if err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}
	env.Storage = storage
	env.addCleanup(func() {
		storage.Close()
	})

	// Load eBPF program on server veth
	// We attach to the server side to filter incoming traffic
	var dp *dataplane.DataPlane
	err = network.RunInServerNS(func() error {
		var loadErr error
		dp, loadErr = dataplane.New(network.ServerVeth)
		return loadErr
	})
	if err != nil {
		env.Cleanup()
		return nil, fmt.Errorf("failed to load eBPF program: %w", err)
	}
	env.DataPlane = dp
	env.addCleanup(func() {
		dp.Close()
	})

	// Create policy manager
	env.PolicyManager = policy.NewManagerWithStorage(dp, storage)

	return env, nil
}

// addCleanup adds a cleanup function to be called on test teardown.
func (env *E2ETestEnv) addCleanup(fn func()) {
	env.cleanupFuncs = append(env.cleanupFuncs, fn)
}

// Cleanup releases all resources created by the test environment.
// It should be called with defer after creating the environment.
func (env *E2ETestEnv) Cleanup() {
	// Call cleanup functions in reverse order
	for i := len(env.cleanupFuncs) - 1; i >= 0; i-- {
		env.cleanupFuncs[i]()
	}
}

// CreatePolicy creates a policy via the PolicyManager.
func (env *E2ETestEnv) CreatePolicy(p *policy.Policy) error {
	return env.PolicyManager.AddPolicy(p)
}

// DeletePolicy deletes a policy via the PolicyManager.
func (env *E2ETestEnv) DeletePolicy(p *policy.Policy) error {
	return env.PolicyManager.DeletePolicy(p)
}

// ListPolicies lists all policies.
func (env *E2ETestEnv) ListPolicies() ([]policy.Policy, error) {
	return env.PolicyManager.ListPolicies()
}

// GetStatistics retrieves current statistics from the data plane.
func (env *E2ETestEnv) GetStatistics() dataplane.Statistics {
	return env.DataPlane.GetStatistics()
}

// VerifyPolicyInMap verifies that a policy exists in the eBPF map.
func (env *E2ETestEnv) VerifyPolicyInMap(srcIP, dstIP string, srcPort, dstPort uint16, protocol string) (bool, error) {
	policyMap := env.DataPlane.GetPolicyMap()
	return testutil.VerifyPolicyExists(policyMap, srcIP, dstIP, srcPort, dstPort, protocol)
}

// VerifySessionInMap verifies that a session exists in the eBPF map.
func (env *E2ETestEnv) VerifySessionInMap(srcIP, dstIP string, srcPort, dstPort uint16, protocol string) (bool, error) {
	sessionMap := env.DataPlane.GetSessionMap()
	return testutil.VerifySessionExists(sessionMap, srcIP, dstIP, srcPort, dstPort, protocol)
}

// CountSessions counts active sessions in the eBPF map.
func (env *E2ETestEnv) CountSessions() (int, error) {
	sessionMap := env.DataPlane.GetSessionMap()
	return testutil.CountSessions(sessionMap)
}

// CountPolicies counts policies in the eBPF map.
func (env *E2ETestEnv) CountPolicies() (int, error) {
	policyMap := env.DataPlane.GetPolicyMap()
	return testutil.CountPolicies(policyMap)
}

// SendTCPTraffic sends TCP traffic from client to server.
func (env *E2ETestEnv) SendTCPTraffic(port int, data []byte) error {
	serverIP := env.Network.GetServerIP()
	return testutil.SendTCPPacket(env.Network.ClientNS, serverIP, port, data)
}

// SendUDPTraffic sends UDP traffic from client to server.
func (env *E2ETestEnv) SendUDPTraffic(port int, data []byte) error {
	serverIP := env.Network.GetServerIP()
	return testutil.SendUDPPacket(env.Network.ClientNS, serverIP, port, data)
}

// PingServer sends ICMP ping from client to server.
func (env *E2ETestEnv) PingServer() (bool, error) {
	serverIP := env.Network.GetServerIP()
	return testutil.PingHost(env.Network.ClientNS, serverIP)
}

// StartTCPServer starts a TCP echo server on the server namespace.
func (env *E2ETestEnv) StartTCPServer(port int) (*testutil.TestServer, error) {
	server, err := testutil.StartTCPServer(env.Network.ServerNS, port)
	if err != nil {
		return nil, err
	}

	env.addCleanup(server.Stop)

	// Give server a moment to start listening
	// We don't check connectivity from client namespace here because:
	// 1. The eBPF program might be blocking traffic by default
	// 2. We want to test policy enforcement, not baseline connectivity
	time.Sleep(100 * time.Millisecond)

	return server, nil
}

// StartUDPServer starts a UDP echo server on the server namespace.
func (env *E2ETestEnv) StartUDPServer(port int) (*testutil.TestServer, error) {
	server, err := testutil.StartUDPServer(env.Network.ServerNS, port)
	if err != nil {
		return nil, err
	}

	env.addCleanup(server.Stop)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	return server, nil
}

// TryConnect attempts to connect from client to server.
// Returns true if connection succeeds, false if it's blocked.
func (env *E2ETestEnv) TryConnect(port int) bool {
	serverIP := env.Network.GetServerIP()
	return testutil.TryConnect(env.Network.ClientNS, serverIP, port)
}

// TryConnectUDP attempts UDP communication from client to server.
func (env *E2ETestEnv) TryConnectUDP(port int) bool {
	serverIP := env.Network.GetServerIP()
	return testutil.TryConnectUDP(env.Network.ClientNS, serverIP, port)
}

// AssertTrafficAllowed asserts that traffic is allowed (connection succeeds).
func (env *E2ETestEnv) AssertTrafficAllowed(port int) {
	require.True(env.T, env.TryConnect(port),
		"Traffic should be allowed to port %d", port)
}

// AssertTrafficBlocked asserts that traffic is blocked (connection fails).
func (env *E2ETestEnv) AssertTrafficBlocked(port int) {
	require.False(env.T, env.TryConnect(port),
		"Traffic should be blocked to port %d", port)
}

// AssertStatistic asserts that a specific statistic matches the expected value.
func (env *E2ETestEnv) AssertStatistic(name string, expected uint64) {
	stats := env.GetStatistics()

	var actual uint64
	switch name {
	case "total_packets":
		actual = stats.TotalPackets
	case "allowed_packets":
		actual = stats.AllowedPackets
	case "denied_packets":
		actual = stats.DeniedPackets
	case "new_sessions":
		actual = stats.NewSessions
	case "closed_sessions":
		actual = stats.ClosedSessions
	case "active_sessions":
		actual = stats.ActiveSessions
	case "policy_hits":
		actual = stats.PolicyHits
	case "policy_misses":
		actual = stats.PolicyMisses
	default:
		env.T.Fatalf("Unknown statistic: %s", name)
	}

	require.Equal(env.T, expected, actual,
		"Statistic %s should be %d, got %d", name, expected, actual)
}

// WaitForStatistic waits for a statistic to reach the expected value.
func (env *E2ETestEnv) WaitForStatistic(name string, expected uint64, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		stats := env.GetStatistics()

		var actual uint64
		switch name {
		case "total_packets":
			actual = stats.TotalPackets
		case "allowed_packets":
			actual = stats.AllowedPackets
		case "denied_packets":
			actual = stats.DeniedPackets
		case "new_sessions":
			actual = stats.NewSessions
		default:
			env.T.Fatalf("Unknown statistic: %s", name)
		}

		if actual >= expected {
			return true
		}

		time.Sleep(50 * time.Millisecond)
	}

	return false
}

// DoHTTPRequest performs an HTTP request (for API testing if implemented).
func (env *E2ETestEnv) DoHTTPRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, env.APIBaseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return env.HTTPClient.Do(req)
}

// CreatePolicyViaAPI creates a policy via REST API (if API server is running).
func (env *E2ETestEnv) CreatePolicyViaAPI(req *models.PolicyRequest) (*models.PolicyResponse, error) {
	resp, err := env.DoHTTPRequest("POST", "/api/v1/policies", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var policyResp models.PolicyResponse
	if err := json.NewDecoder(resp.Body).Decode(&policyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &policyResp, nil
}

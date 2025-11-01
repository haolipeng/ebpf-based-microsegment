// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package e2e

import (
	"testing"

	"github.com/ebpf-microsegment/src/agent/pkg/policy"
	"github.com/ebpf-microsegment/src/agent/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_AllowPolicy tests that ALLOW policy allows traffic.
func TestE2E_AllowPolicy(t *testing.T) {
	// Skip if not root
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	// Create test environment
	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	// Start TCP server on port 8080
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err, "Failed to start TCP server")
	defer server.Stop()

	// Create ALLOW policy for port 8080
	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	allowPolicy := &policy.Policy{
		RuleID:   100,
		SrcIP:    clientIP,
		DstIP:    serverIP,
		SrcPort:  0, // Any source port
		DstPort:  8080,
		Protocol: "tcp",
		Action:   "allow",
		Priority: 10,
	}

	err = env.CreatePolicy(allowPolicy)
	require.NoError(t, err, "Failed to create policy")

	// Note: Wildcard policies (src_port=0) are stored in wildcard_policy_map.
	// We verify the policy works by testing traffic is allowed.

	// Test traffic - should be allowed by wildcard ALLOW policy
	testData := []byte("Hello, E2E!")
	err = env.SendTCPTraffic(8080, testData)
	assert.NoError(t, err, "Traffic should be allowed")

	// Verify statistics
	stats := env.GetStatistics()
	assert.Greater(t, stats.TotalPackets, uint64(0), "Should have packet count")
	assert.Greater(t, stats.AllowedPackets, uint64(0), "Should have allowed packets")
	assert.Equal(t, uint64(0), stats.DeniedPackets, "Should have no denied packets")
}

// TestE2E_DenyPolicy tests that DENY policy blocks traffic.
func TestE2E_DenyPolicy(t *testing.T) {
	// Skip if not root
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	// Create test environment
	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	// Start TCP server on port 8080
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err, "Failed to start TCP server")
	defer server.Stop()

	// Create DENY policy for port 8080
	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	denyPolicy := &policy.Policy{
		RuleID:   200,
		SrcIP:    clientIP,
		DstIP:    serverIP,
		SrcPort:  0, // Any source port
		DstPort:  8080,
		Protocol: "tcp",
		Action:   "deny",
		Priority: 10,
	}

	err = env.CreatePolicy(denyPolicy)
	require.NoError(t, err, "Failed to create policy")

	// Note: Wildcard policies (src_port=0) are stored in wildcard_policy_map,
	// not the exact match policy_map, so we skip exact map verification.
	// Instead, we verify the policy works by testing traffic blocking.

	// Test traffic - should be blocked by wildcard DENY policy
	env.AssertTrafficBlocked(8080)

	// Verify statistics
	stats := env.GetStatistics()
	assert.Greater(t, stats.DeniedPackets, uint64(0), "Should have denied packets")
}

// TestE2E_NoPolicy tests behavior when no policy exists.
func TestE2E_NoPolicy(t *testing.T) {
	// Skip if not root
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	// Create test environment
	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	// Start TCP server on port 8080
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err, "Failed to start TCP server")
	defer server.Stop()

	// Don't create any policy - test default behavior
	// Based on your eBPF implementation, this might allow or deny by default

	// Test traffic
	testData := []byte("Hello, E2E!")
	err = env.SendTCPTraffic(8080, testData)

	// Document the default behavior
	if err == nil {
		t.Log("Default behavior: ALLOW (no policy = allow)")
	} else {
		t.Log("Default behavior: DENY (no policy = deny)")
	}

	// Verify statistics are updated
	stats := env.GetStatistics()
	assert.Greater(t, stats.TotalPackets, uint64(0), "Should have packet count")
}

// TestE2E_PolicyPriority tests policy priority handling.
func TestE2E_PolicyPriority(t *testing.T) {
	// Skip if not root
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	// Create test environment
	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	// Start TCP server
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err, "Failed to start TCP server")
	defer server.Stop()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	// Create low-priority DENY policy
	denyPolicy := &policy.Policy{
		RuleID:   300,
		SrcIP:    clientIP,
		DstIP:    serverIP,
		SrcPort:  0,
		DstPort:  8080,
		Protocol: "tcp",
		Action:   "deny",
		Priority: 5, // Lower priority
	}

	// Create high-priority ALLOW policy
	allowPolicy := &policy.Policy{
		RuleID:   301,
		SrcIP:    clientIP,
		DstIP:    serverIP,
		SrcPort:  0,
		DstPort:  8080,
		Protocol: "tcp",
		Action:   "allow",
		Priority: 10, // Higher priority
	}

	// Add deny policy first
	err = env.CreatePolicy(denyPolicy)
	require.NoError(t, err)

	// Add allow policy (higher priority)
	err = env.CreatePolicy(allowPolicy)
	require.NoError(t, err)

	// Traffic should be ALLOWED (higher priority wins)
	testData := []byte("Priority test")
	err = env.SendTCPTraffic(8080, testData)
	assert.NoError(t, err, "Traffic should be allowed due to higher priority ALLOW policy")

	// Verify allowed packets
	stats := env.GetStatistics()
	assert.Greater(t, stats.AllowedPackets, uint64(0), "Should have allowed packets")
}

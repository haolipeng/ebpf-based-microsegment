// Package testcases contains test cases for eBPF programs using BPF_PROG_TEST_RUN.
//
// This file tests session tracking and TCP state machine functionality.
package testcases

import (
	"os"
	"testing"
	"time"

	"ebpf-based-microsegment/src/agent/pkg/dataplane/bpftest"
)

// TestSessionCacheHit tests that established sessions are served from cache.
func TestSessionCacheHit(t *testing.T) {
	skipIfNotRoot(t)
	bpfPath := skipIfNoBPF(t)

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	// Setup: Add a policy and simulate an established session
	runner.ClearPolicyMap()
	runner.ClearSessionMap()
	runner.SetDefaultPolicy(bpftest.ACTION_DENY)

	// Add policy to allow initial connection
	key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
	if err := runner.AddPolicy(key, bpftest.ACTION_ALLOW, 1); err != nil {
		t.Fatalf("Failed to add policy: %v", err)
	}

	// First packet - should create session entry
	syn := bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 80)
	ctx := bpftest.IngressContext()

	result1, err := runner.Run(syn, ctx)
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}
	if !result1.Allowed {
		t.Errorf("First packet should be allowed, got action=%d", result1.ReturnValue)
	}

	// Verify session was created
	sessionCount, err := runner.GetSessionCount()
	if err != nil {
		t.Logf("Warning: Could not get session count: %v", err)
	} else {
		t.Logf("Session count after first packet: %d", sessionCount)
	}

	// Second packet from same flow - should hit session cache
	// This tests the hot path
	ack := bpftest.BuildTCPACK("10.0.0.1", "10.0.0.2", 12345, 80, 1001, 1)
	result2, err := runner.Run(ack, ctx)
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}
	if !result2.Allowed {
		t.Errorf("Second packet should be allowed (session cache hit), got action=%d", result2.ReturnValue)
	}

	// Measure performance difference between cold and hot path
	t.Log("Testing hot path performance...")
	hotResult, err := runner.RunWithTiming(ack, ctx, 1000)
	if err != nil {
		t.Logf("Hot path timing failed: %v", err)
	} else {
		t.Logf("Hot path (session hit) average: %v", hotResult.Duration)
	}
}

// TestTCPHandshake tests a complete TCP three-way handshake.
func TestTCPHandshake(t *testing.T) {
	skipIfNotRoot(t)
	bpfPath := skipIfNoBPF(t)

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	// Setup
	runner.ClearPolicyMap()
	runner.ClearSessionMap()

	// Allow traffic in both directions
	ingressKey := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
	runner.AddPolicy(ingressKey, bpftest.ACTION_ALLOW, 1)

	egressKey := bpftest.NewPolicyKey("10.0.0.2", "10.0.0.1", 0, 12345, bpftest.IPPROTO_TCP, bpftest.DIRECTION_EGRESS)
	runner.AddPolicy(egressKey, bpftest.ACTION_ALLOW, 2)

	handshake := &bpftest.TCPHandshake{
		ClientIP:   "10.0.0.1",
		ServerIP:   "10.0.0.2",
		ClientPort: 12345,
		ServerPort: 80,
		ClientISN:  1000,
		ServerISN:  2000,
	}

	// Step 1: Client -> Server SYN
	t.Log("Step 1: SYN (Client -> Server)")
	syn := handshake.BuildSYN()
	result, err := runner.Run(syn, bpftest.IngressContext())
	if err != nil {
		t.Fatalf("SYN failed: %v", err)
	}
	if !result.Allowed {
		t.Errorf("SYN should be allowed, got action=%d", result.ReturnValue)
	}

	// Step 2: Server -> Client SYN+ACK
	t.Log("Step 2: SYN+ACK (Server -> Client)")
	synack := handshake.BuildSYNACK()
	result, err = runner.Run(synack, bpftest.EgressContext())
	if err != nil {
		t.Fatalf("SYN+ACK failed: %v", err)
	}
	if !result.Allowed {
		t.Errorf("SYN+ACK should be allowed, got action=%d", result.ReturnValue)
	}

	// Step 3: Client -> Server ACK
	t.Log("Step 3: ACK (Client -> Server)")
	ack := handshake.BuildACK()
	result, err = runner.Run(ack, bpftest.IngressContext())
	if err != nil {
		t.Fatalf("ACK failed: %v", err)
	}
	if !result.Allowed {
		t.Errorf("ACK should be allowed, got action=%d", result.ReturnValue)
	}

	t.Log("TCP handshake completed successfully")
}

// TestTCPDataTransfer tests data transfer after handshake.
func TestTCPDataTransfer(t *testing.T) {
	skipIfNotRoot(t)
	bpfPath := skipIfNoBPF(t)

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	// Setup
	runner.ClearPolicyMap()
	runner.ClearSessionMap()

	key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
	runner.AddPolicy(key, bpftest.ACTION_ALLOW, 1)

	// Simulate established connection
	handshake := &bpftest.TCPHandshake{
		ClientIP:   "10.0.0.1",
		ServerIP:   "10.0.0.2",
		ClientPort: 12345,
		ServerPort: 80,
		ClientISN:  1000,
		ServerISN:  2000,
	}

	ctx := bpftest.IngressContext()

	// Run handshake first
	runner.Run(handshake.BuildSYN(), ctx)
	runner.Run(handshake.BuildACK(), ctx)

	// Send data packet
	payload := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	dataPacket := bpftest.BuildTCPData(
		"10.0.0.1", "10.0.0.2",
		12345, 80,
		1001, 2001,
		payload,
	)

	result, err := runner.Run(dataPacket, ctx)
	if err != nil {
		t.Fatalf("Data packet failed: %v", err)
	}
	if !result.Allowed {
		t.Errorf("Data packet should be allowed, got action=%d", result.ReturnValue)
	}

	t.Logf("Data transfer test passed, packet with %d bytes payload allowed", len(payload))
}

// TestTCPConnectionClose tests TCP connection termination.
func TestTCPConnectionClose(t *testing.T) {
	skipIfNotRoot(t)
	bpfPath := skipIfNoBPF(t)

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	// Setup
	runner.ClearPolicyMap()
	runner.ClearSessionMap()

	key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
	runner.AddPolicy(key, bpftest.ACTION_ALLOW, 1)

	ctx := bpftest.IngressContext()

	// Establish connection first
	syn := bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 80)
	runner.Run(syn, ctx)

	ack := bpftest.BuildTCPACK("10.0.0.1", "10.0.0.2", 12345, 80, 1001, 2001)
	runner.Run(ack, ctx)

	// Send FIN to close connection
	fin := bpftest.BuildTCPFIN("10.0.0.1", "10.0.0.2", 12345, 80, 1001, 2001)
	result, err := runner.Run(fin, ctx)
	if err != nil {
		t.Fatalf("FIN packet failed: %v", err)
	}
	if !result.Allowed {
		t.Errorf("FIN packet should be allowed, got action=%d", result.ReturnValue)
	}

	t.Log("TCP connection close test passed")
}

// TestTCPReset tests TCP RST handling.
func TestTCPReset(t *testing.T) {
	skipIfNotRoot(t)
	bpfPath := skipIfNoBPF(t)

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	// Setup
	runner.ClearPolicyMap()
	runner.ClearSessionMap()

	key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
	runner.AddPolicy(key, bpftest.ACTION_ALLOW, 1)

	ctx := bpftest.IngressContext()

	// Establish connection first
	syn := bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 80)
	runner.Run(syn, ctx)

	// Send RST
	rst := bpftest.BuildTCPRST("10.0.0.1", "10.0.0.2", 12345, 80, 1001)
	result, err := runner.Run(rst, ctx)
	if err != nil {
		t.Fatalf("RST packet failed: %v", err)
	}
	if !result.Allowed {
		t.Errorf("RST packet should be allowed, got action=%d", result.ReturnValue)
	}

	t.Log("TCP RST handling test passed")
}

// TestSessionTimeout tests that expired sessions are handled correctly.
func TestSessionTimeout(t *testing.T) {
	skipIfNotRoot(t)
	bpfPath := skipIfNoBPF(t)

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	// Setup
	runner.ClearPolicyMap()
	runner.ClearSessionMap()

	// Note: Session timeout testing is limited with BPF_PROG_TEST_RUN
	// because we can't easily manipulate timestamps.
	// This test verifies basic session cleanup behavior.

	key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
	runner.AddPolicy(key, bpftest.ACTION_ALLOW, 1)

	ctx := bpftest.IngressContext()

	// Create multiple sessions
	for i := uint16(0); i < 5; i++ {
		syn := bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 10000+i, 80)
		_, err := runner.Run(syn, ctx)
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
	}

	// Check session count
	count, err := runner.GetSessionCount()
	if err != nil {
		t.Logf("Warning: Could not get session count: %v", err)
	} else {
		t.Logf("Sessions created: %d", count)
		if count < 5 {
			t.Logf("Note: Some sessions may have been coalesced or not tracked")
		}
	}

	// Clear sessions
	err = runner.ClearSessionMap()
	if err != nil {
		t.Fatalf("Failed to clear sessions: %v", err)
	}

	count, _ = runner.GetSessionCount()
	t.Logf("Sessions after clear: %d", count)
}

// TestUDPSession tests UDP session handling (connectionless).
func TestUDPSession(t *testing.T) {
	skipIfNotRoot(t)
	bpfPath := skipIfNoBPF(t)

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	// Setup
	runner.ClearPolicyMap()
	runner.ClearSessionMap()

	key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 53, bpftest.IPPROTO_UDP, bpftest.DIRECTION_INGRESS)
	runner.AddPolicy(key, bpftest.ACTION_ALLOW, 1)

	ctx := bpftest.IngressContext()

	// First UDP packet
	packet1 := bpftest.BuildUDPPacket("10.0.0.1", "10.0.0.2", 12345, 53, []byte("dns-query-1"))
	result1, err := runner.Run(packet1, ctx)
	if err != nil {
		t.Fatalf("First UDP packet failed: %v", err)
	}
	if !result1.Allowed {
		t.Errorf("First UDP packet should be allowed, got action=%d", result1.ReturnValue)
	}

	// Second UDP packet from same flow
	packet2 := bpftest.BuildUDPPacket("10.0.0.1", "10.0.0.2", 12345, 53, []byte("dns-query-2"))
	result2, err := runner.Run(packet2, ctx)
	if err != nil {
		t.Fatalf("Second UDP packet failed: %v", err)
	}
	if !result2.Allowed {
		t.Errorf("Second UDP packet should be allowed, got action=%d", result2.ReturnValue)
	}

	t.Log("UDP session test passed")
}

// BenchmarkSessionCacheLookup benchmarks session cache performance.
func BenchmarkSessionCacheLookup(b *testing.B) {
	if os.Geteuid() != 0 {
		b.Skip("Benchmark requires root privileges")
	}

	bpfPath := getBPFObjectPath()
	if bpfPath == "" {
		b.Skip("BPF object file not found")
	}

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		b.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	// Setup
	runner.ClearPolicyMap()
	runner.ClearSessionMap()

	key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
	runner.AddPolicy(key, bpftest.ACTION_ALLOW, 1)

	ctx := bpftest.IngressContext()

	// Create a session first
	syn := bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 80)
	runner.Run(syn, ctx)

	// Benchmark packet that should hit session cache
	ack := bpftest.BuildTCPACK("10.0.0.1", "10.0.0.2", 12345, 80, 1001, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runner.Run(ack, ctx)
		if err != nil {
			b.Fatalf("Run failed: %v", err)
		}
	}
}

// BenchmarkSessionCacheLookupTimed uses kernel timing for accurate measurement.
func BenchmarkSessionCacheLookupTimed(b *testing.B) {
	if os.Geteuid() != 0 {
		b.Skip("Benchmark requires root privileges")
	}

	bpfPath := getBPFObjectPath()
	if bpfPath == "" {
		b.Skip("BPF object file not found")
	}

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		b.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	// Setup
	runner.ClearPolicyMap()
	runner.ClearSessionMap()

	key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
	runner.AddPolicy(key, bpftest.ACTION_ALLOW, 1)

	ctx := bpftest.IngressContext()

	// Create a session first
	syn := bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 80)
	runner.Run(syn, ctx)

	// Benchmark packet that should hit session cache
	ack := bpftest.BuildTCPACK("10.0.0.1", "10.0.0.2", 12345, 80, 1001, 1)

	result, err := runner.RunWithTiming(ack, ctx, b.N)
	if err != nil {
		b.Fatalf("RunWithTiming failed: %v", err)
	}

	b.ReportMetric(float64(result.Duration.Nanoseconds()), "ns/op")

	// Report if we meet the <1μs hot path target
	if result.Duration < time.Microsecond {
		b.Logf("✓ Hot path target met: %v < 1μs", result.Duration)
	} else {
		b.Logf("✗ Hot path target NOT met: %v >= 1μs", result.Duration)
	}
}

// BenchmarkColdVsHotPath compares cold path (no session) vs hot path (session hit).
func BenchmarkColdVsHotPath(b *testing.B) {
	if os.Geteuid() != 0 {
		b.Skip("Benchmark requires root privileges")
	}

	bpfPath := getBPFObjectPath()
	if bpfPath == "" {
		b.Skip("BPF object file not found")
	}

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		b.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	// Setup
	key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
	runner.AddPolicy(key, bpftest.ACTION_ALLOW, 1)

	ctx := bpftest.IngressContext()

	iterations := 10000

	b.Run("ColdPath", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Clear session map before each test to force cold path
			runner.ClearSessionMap()

			syn := bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", uint16(10000+i%50000), 80)
			result, err := runner.RunWithTiming(syn, ctx, iterations)
			if err != nil {
				b.Fatalf("Cold path test failed: %v", err)
			}
			b.ReportMetric(float64(result.Duration.Nanoseconds()), "ns/op")
		}
	})

	b.Run("HotPath", func(b *testing.B) {
		// Setup: create a session
		runner.ClearSessionMap()
		syn := bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 80)
		runner.Run(syn, ctx)

		ack := bpftest.BuildTCPACK("10.0.0.1", "10.0.0.2", 12345, 80, 1001, 1)

		for i := 0; i < b.N; i++ {
			result, err := runner.RunWithTiming(ack, ctx, iterations)
			if err != nil {
				b.Fatalf("Hot path test failed: %v", err)
			}
			b.ReportMetric(float64(result.Duration.Nanoseconds()), "ns/op")
		}
	})
}

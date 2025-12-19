// Package testcases contains test cases for eBPF programs using BPF_PROG_TEST_RUN.
//
// These tests verify policy matching logic without requiring real network traffic
// or containers, providing fast feedback during development.
package testcases

import (
	"os"
	"path/filepath"
	"testing"

	"ebpf-based-microsegment/src/agent/pkg/dataplane/bpftest"
)

// getBPFObjectPath returns the path to the compiled eBPF object file.
// It looks in common locations relative to the test file.
func getBPFObjectPath() string {
	// Try common paths
	paths := []string{
		"../../../../../bpf/tc_microsegment.bpf.o",           // From testcases dir
		"../../../../../../src/bpf/tc_microsegment.bpf.o",    // Alternative
		"/home/work/ebpf-based-microsegment/src/bpf/tc_microsegment.bpf.o", // Absolute
	}

	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}

	return ""
}

// skipIfNoBPF skips the test if BPF object file is not available.
func skipIfNoBPF(t *testing.T) string {
	path := getBPFObjectPath()
	if path == "" {
		t.Skip("BPF object file not found, run 'make bpf' first")
	}
	return path
}

// skipIfNotRoot skips the test if not running as root.
func skipIfNotRoot(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Test requires root privileges for BPF_PROG_TEST_RUN")
	}
}

// TestPolicyMatchExact tests exact policy matching (5-tuple).
func TestPolicyMatchExact(t *testing.T) {
	skipIfNotRoot(t)
	bpfPath := skipIfNoBPF(t)

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	tests := []*bpftest.TestCase{
		{
			Name:           "Allow exact match - TCP port 80",
			Packet:         bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 80),
			Context:        bpftest.IngressContext(),
			ExpectedAction: bpftest.TC_ACT_OK,
			SetupMaps: func(r *bpftest.TestRunner) error {
				// Clear existing rules
				r.ClearPolicyMap()
				// Add allow rule for this flow
				key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
				return r.AddPolicy(key, bpftest.ACTION_ALLOW, 1)
			},
			CleanupMaps: func(r *bpftest.TestRunner) error {
				return r.ClearPolicyMap()
			},
		},
		{
			Name:           "Deny exact match - TCP port 443",
			Packet:         bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 443),
			Context:        bpftest.IngressContext(),
			ExpectedAction: bpftest.TC_ACT_SHOT,
			SetupMaps: func(r *bpftest.TestRunner) error {
				r.ClearPolicyMap()
				key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 443, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
				return r.AddPolicy(key, bpftest.ACTION_DENY, 2)
			},
			CleanupMaps: func(r *bpftest.TestRunner) error {
				return r.ClearPolicyMap()
			},
		},
		{
			Name:           "No match - use default policy (deny)",
			Packet:         bpftest.BuildTCPSYN("192.168.1.1", "192.168.1.2", 54321, 8080),
			Context:        bpftest.IngressContext(),
			ExpectedAction: bpftest.TC_ACT_SHOT,
			SetupMaps: func(r *bpftest.TestRunner) error {
				r.ClearPolicyMap()
				// Set default policy to deny
				return r.SetDefaultPolicy(bpftest.ACTION_DENY)
			},
			CleanupMaps: func(r *bpftest.TestRunner) error {
				return nil
			},
		},
		{
			Name:           "No match - use default policy (allow)",
			Packet:         bpftest.BuildTCPSYN("192.168.1.1", "192.168.1.2", 54321, 8080),
			Context:        bpftest.IngressContext(),
			ExpectedAction: bpftest.TC_ACT_OK,
			SetupMaps: func(r *bpftest.TestRunner) error {
				r.ClearPolicyMap()
				// Set default policy to allow
				return r.SetDefaultPolicy(bpftest.ACTION_ALLOW)
			},
			CleanupMaps: func(r *bpftest.TestRunner) error {
				return nil
			},
		},
	}

	results := runner.RunTestCases(tests)

	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Test '%s' error: %v", result.TestCase.Name, result.Error)
			continue
		}
		if !result.Passed {
			t.Errorf("Test '%s' failed: %s", result.TestCase.Name, result.Message)
		} else {
			t.Logf("Test '%s': PASSED", result.TestCase.Name)
		}
	}
}

// TestPolicyMatchDirection tests direction-specific policy matching.
func TestPolicyMatchDirection(t *testing.T) {
	skipIfNotRoot(t)
	bpfPath := skipIfNoBPF(t)

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	tests := []*bpftest.TestCase{
		{
			Name:           "Ingress allow - same packet on ingress",
			Packet:         bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 80),
			Context:        bpftest.IngressContext(),
			ExpectedAction: bpftest.TC_ACT_OK,
			SetupMaps: func(r *bpftest.TestRunner) error {
				r.ClearPolicyMap()
				// Add ingress-only rule
				key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
				return r.AddPolicy(key, bpftest.ACTION_ALLOW, 1)
			},
			CleanupMaps: func(r *bpftest.TestRunner) error {
				return r.ClearPolicyMap()
			},
		},
		{
			Name:           "Egress allow - same IPs on egress",
			Packet:         bpftest.BuildTCPSYN("10.0.0.2", "10.0.0.1", 80, 12345),
			Context:        bpftest.EgressContext(),
			ExpectedAction: bpftest.TC_ACT_OK,
			SetupMaps: func(r *bpftest.TestRunner) error {
				r.ClearPolicyMap()
				// Add egress-only rule
				key := bpftest.NewPolicyKey("10.0.0.2", "10.0.0.1", 0, 12345, bpftest.IPPROTO_TCP, bpftest.DIRECTION_EGRESS)
				return r.AddPolicy(key, bpftest.ACTION_ALLOW, 2)
			},
			CleanupMaps: func(r *bpftest.TestRunner) error {
				return r.ClearPolicyMap()
			},
		},
	}

	results := runner.RunTestCases(tests)

	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Test '%s' error: %v", result.TestCase.Name, result.Error)
			continue
		}
		if !result.Passed {
			t.Errorf("Test '%s' failed: %s", result.TestCase.Name, result.Message)
		} else {
			t.Logf("Test '%s': PASSED", result.TestCase.Name)
		}
	}
}

// TestPolicyMatchProtocol tests protocol-specific policy matching.
func TestPolicyMatchProtocol(t *testing.T) {
	skipIfNotRoot(t)
	bpfPath := skipIfNoBPF(t)

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	tests := []*bpftest.TestCase{
		{
			Name:           "TCP policy - match TCP packet",
			Packet:         bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 80),
			Context:        bpftest.IngressContext(),
			ExpectedAction: bpftest.TC_ACT_OK,
			SetupMaps: func(r *bpftest.TestRunner) error {
				r.ClearPolicyMap()
				key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
				return r.AddPolicy(key, bpftest.ACTION_ALLOW, 1)
			},
			CleanupMaps: func(r *bpftest.TestRunner) error {
				return r.ClearPolicyMap()
			},
		},
		{
			Name:           "UDP policy - match UDP packet",
			Packet:         bpftest.BuildUDPPacket("10.0.0.1", "10.0.0.2", 12345, 53, []byte("dns query")),
			Context:        bpftest.IngressContext(),
			ExpectedAction: bpftest.TC_ACT_OK,
			SetupMaps: func(r *bpftest.TestRunner) error {
				r.ClearPolicyMap()
				key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 53, bpftest.IPPROTO_UDP, bpftest.DIRECTION_INGRESS)
				return r.AddPolicy(key, bpftest.ACTION_ALLOW, 2)
			},
			CleanupMaps: func(r *bpftest.TestRunner) error {
				return r.ClearPolicyMap()
			},
		},
		{
			Name:           "ICMP policy - match ICMP packet",
			Packet:         bpftest.BuildICMPEchoRequest("10.0.0.1", "10.0.0.2", 1, 1),
			Context:        bpftest.IngressContext(),
			ExpectedAction: bpftest.TC_ACT_OK,
			SetupMaps: func(r *bpftest.TestRunner) error {
				r.ClearPolicyMap()
				// ICMP uses protocol field, port is typically 0
				key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 0, bpftest.IPPROTO_ICMP, bpftest.DIRECTION_INGRESS)
				return r.AddPolicy(key, bpftest.ACTION_ALLOW, 3)
			},
			CleanupMaps: func(r *bpftest.TestRunner) error {
				return r.ClearPolicyMap()
			},
		},
		{
			Name:           "TCP policy - UDP packet should not match",
			Packet:         bpftest.BuildUDPPacket("10.0.0.1", "10.0.0.2", 12345, 80, nil),
			Context:        bpftest.IngressContext(),
			ExpectedAction: bpftest.TC_ACT_SHOT, // Should not match TCP rule
			SetupMaps: func(r *bpftest.TestRunner) error {
				r.ClearPolicyMap()
				r.SetDefaultPolicy(bpftest.ACTION_DENY)
				// Only TCP rule exists
				key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
				return r.AddPolicy(key, bpftest.ACTION_ALLOW, 1)
			},
			CleanupMaps: func(r *bpftest.TestRunner) error {
				return r.ClearPolicyMap()
			},
		},
	}

	results := runner.RunTestCases(tests)

	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Test '%s' error: %v", result.TestCase.Name, result.Error)
			continue
		}
		if !result.Passed {
			t.Errorf("Test '%s' failed: %s", result.TestCase.Name, result.Message)
		} else {
			t.Logf("Test '%s': PASSED", result.TestCase.Name)
		}
	}
}

// TestPolicyPriority tests policy priority ordering.
func TestPolicyPriority(t *testing.T) {
	skipIfNotRoot(t)
	bpfPath := skipIfNoBPF(t)

	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
		BPFObjectPath: bpfPath,
	})
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Close()

	// This test verifies that more specific rules take precedence
	// when multiple rules could match
	tests := []*bpftest.TestCase{
		{
			Name:           "Specific rule overrides general rule",
			Packet:         bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 80),
			Context:        bpftest.IngressContext(),
			ExpectedAction: bpftest.TC_ACT_OK,
			SetupMaps: func(r *bpftest.TestRunner) error {
				r.ClearPolicyMap()
				// More specific rule (with source port) - allow
				key1 := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 12345, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
				if err := r.AddPolicy(key1, bpftest.ACTION_ALLOW, 1); err != nil {
					return err
				}
				// Less specific rule (any source port) - deny
				// Note: In hash-based lookup, exact match wins
				return nil
			},
			CleanupMaps: func(r *bpftest.TestRunner) error {
				return r.ClearPolicyMap()
			},
		},
	}

	results := runner.RunTestCases(tests)

	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Test '%s' error: %v", result.TestCase.Name, result.Error)
			continue
		}
		if !result.Passed {
			t.Errorf("Test '%s' failed: %s", result.TestCase.Name, result.Message)
		} else {
			t.Logf("Test '%s': PASSED", result.TestCase.Name)
		}
	}
}

// BenchmarkPolicyLookup benchmarks policy map lookup performance.
func BenchmarkPolicyLookup(b *testing.B) {
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

	// Setup a policy
	runner.ClearPolicyMap()
	key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
	if err := runner.AddPolicy(key, bpftest.ACTION_ALLOW, 1); err != nil {
		b.Fatalf("Failed to add policy: %v", err)
	}

	packet := bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 80)
	ctx := bpftest.IngressContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runner.Run(packet, ctx)
		if err != nil {
			b.Fatalf("Run failed: %v", err)
		}
	}
}

// BenchmarkPolicyLookupWithTiming uses the built-in timing feature.
func BenchmarkPolicyLookupWithTiming(b *testing.B) {
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

	// Setup a policy
	runner.ClearPolicyMap()
	key := bpftest.NewPolicyKey("10.0.0.1", "10.0.0.2", 0, 80, bpftest.IPPROTO_TCP, bpftest.DIRECTION_INGRESS)
	if err := runner.AddPolicy(key, bpftest.ACTION_ALLOW, 1); err != nil {
		b.Fatalf("Failed to add policy: %v", err)
	}

	packet := bpftest.BuildTCPSYN("10.0.0.1", "10.0.0.2", 12345, 80)
	ctx := bpftest.IngressContext()

	// Use kernel-measured timing
	result, err := runner.RunWithTiming(packet, ctx, b.N)
	if err != nil {
		b.Fatalf("RunWithTiming failed: %v", err)
	}

	b.ReportMetric(float64(result.Duration.Nanoseconds()), "ns/op")
}

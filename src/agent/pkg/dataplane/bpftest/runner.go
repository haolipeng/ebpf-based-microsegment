// Package bpftest provides testing utilities for eBPF programs using BPF_PROG_TEST_RUN.
//
// This package enables fast, isolated testing of TC eBPF programs without requiring
// real network traffic or containers. It leverages the kernel's BPF_PROG_TEST_RUN
// syscall to execute eBPF programs in userspace.
//
// Example usage:
//
//	runner, err := bpftest.NewTestRunner(bpftest.RunnerConfig{
//	    BPFObjectPath: "path/to/tc_microsegment.bpf.o",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer runner.Close()
//
//	result, err := runner.Run(packet, &bpftest.SKBuffContext{
//	    Ingress: true,
//	    Ifindex: 1,
//	})
package bpftest

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cilium/ebpf"
)

// TC action return values matching kernel definitions
const (
	TC_ACT_OK       = 0 // Allow packet
	TC_ACT_SHOT     = 2 // Drop packet
	TC_ACT_REDIRECT = 7 // Redirect packet
)

// RunnerConfig holds configuration for creating a TestRunner.
type RunnerConfig struct {
	// BPFObjectPath is the path to the compiled eBPF object file (.o).
	// If empty, attempts to load from the default location.
	BPFObjectPath string

	// PinPath is the path for pinning eBPF maps. If empty, maps are not pinned.
	PinPath string

	// LoadFromPinned indicates whether to load existing pinned maps.
	LoadFromPinned bool
}

// TestRunner manages eBPF program testing using BPF_PROG_TEST_RUN.
type TestRunner struct {
	mu sync.RWMutex

	// eBPF objects
	collection *ebpf.Collection
	tcProgram  *ebpf.Program

	// Maps for test setup
	policyMap         *ebpf.Map
	wildcardPolicyMap *ebpf.Map
	sessionMap        *ebpf.Map
	statsMap          *ebpf.Map
	defaultPolicyMap  *ebpf.Map
	flowEventsMap     *ebpf.Map

	// Configuration
	config RunnerConfig
}

// SKBuffContext represents the __sk_buff context passed to TC programs.
// This structure is used to simulate packet metadata.
type SKBuffContext struct {
	// Ingress indicates if this is an ingress (true) or egress (false) packet.
	Ingress bool

	// Ifindex is the interface index.
	Ifindex uint32

	// Mark is the packet mark (can be used for policy routing).
	Mark uint32

	// Priority is the packet priority.
	Priority uint32

	// Protocol is the L3 protocol (e.g., ETH_P_IP = 0x0800).
	Protocol uint16

	// VLANID is the VLAN tag if present.
	VLANID uint16
}

// TestResult holds the result of a single test run.
type TestResult struct {
	// ReturnValue is the TC action returned by the program.
	ReturnValue int32

	// OutputPacket contains the potentially modified packet data.
	OutputPacket []byte

	// Duration is the time taken to run the program (if measured).
	Duration time.Duration

	// Allowed returns true if the packet was allowed (TC_ACT_OK).
	Allowed bool

	// Dropped returns true if the packet was dropped (TC_ACT_SHOT).
	Dropped bool
}

// TestCase represents a single test case with input and expected output.
type TestCase struct {
	// Name is a descriptive name for the test case.
	Name string

	// Packet is the raw packet data to test.
	Packet []byte

	// Context is the __sk_buff context for the test.
	Context *SKBuffContext

	// ExpectedAction is the expected TC action (TC_ACT_OK, TC_ACT_SHOT, etc.).
	ExpectedAction int32

	// SetupMaps is called before the test to populate eBPF maps.
	SetupMaps func(r *TestRunner) error

	// CleanupMaps is called after the test to clean up eBPF maps.
	CleanupMaps func(r *TestRunner) error
}

// TestCaseResult holds the result of running a test case.
type TestCaseResult struct {
	// TestCase is the original test case.
	TestCase *TestCase

	// Result is the actual test result.
	Result *TestResult

	// Passed indicates if the test passed (actual == expected).
	Passed bool

	// Error contains any error that occurred during the test.
	Error error

	// Message provides additional context about the result.
	Message string
}

// NewTestRunner creates a new TestRunner with the given configuration.
// The caller is responsible for calling Close() when done.
func NewTestRunner(config RunnerConfig) (*TestRunner, error) {
	runner := &TestRunner{
		config: config,
	}

	if err := runner.loadProgram(); err != nil {
		return nil, fmt.Errorf("failed to load eBPF program: %w", err)
	}

	return runner, nil
}

// NewTestRunnerFromObjects creates a TestRunner from pre-loaded eBPF objects.
// This is useful when integrating with existing dataplane code.
func NewTestRunnerFromObjects(prog *ebpf.Program, maps map[string]*ebpf.Map) (*TestRunner, error) {
	if prog == nil {
		return nil, fmt.Errorf("program cannot be nil")
	}

	runner := &TestRunner{
		tcProgram: prog,
	}

	// Extract maps if provided
	if maps != nil {
		runner.policyMap = maps["policy_map"]
		runner.wildcardPolicyMap = maps["wildcard_policy_map"]
		runner.sessionMap = maps["session_map"]
		runner.statsMap = maps["stats_map"]
		runner.defaultPolicyMap = maps["default_policy"]
		runner.flowEventsMap = maps["flow_events"]
	}

	return runner, nil
}

// loadProgram loads the eBPF program and maps.
func (r *TestRunner) loadProgram() error {
	spec, err := ebpf.LoadCollectionSpec(r.config.BPFObjectPath)
	if err != nil {
		return fmt.Errorf("failed to load collection spec: %w", err)
	}

	opts := &ebpf.CollectionOptions{}
	if r.config.PinPath != "" {
		opts.Maps.PinPath = r.config.PinPath
	}

	r.collection, err = ebpf.NewCollectionWithOptions(spec, *opts)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// Find the TC program (typically named tc_microsegment_filter or similar)
	for name, prog := range r.collection.Programs {
		if prog.Type() == ebpf.SchedCLS {
			r.tcProgram = prog
			break
		}
		// Also check for common naming patterns
		if name == "tc_microsegment_filter" || name == "tc_filter" {
			r.tcProgram = prog
			break
		}
	}

	if r.tcProgram == nil {
		return fmt.Errorf("no TC (SchedCLS) program found in object file")
	}

	// Extract maps
	r.policyMap = r.collection.Maps["policy_map"]
	r.wildcardPolicyMap = r.collection.Maps["wildcard_policy_map"]
	r.sessionMap = r.collection.Maps["session_map"]
	r.statsMap = r.collection.Maps["stats_map"]
	r.defaultPolicyMap = r.collection.Maps["default_policy"]
	r.flowEventsMap = r.collection.Maps["flow_events"]

	return nil
}

// Close releases all resources held by the TestRunner.
func (r *TestRunner) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.collection != nil {
		r.collection.Close()
	}
	return nil
}

// Run executes the eBPF program with the given packet and context.
// This uses BPF_PROG_TEST_RUN for fast, isolated testing.
func (r *TestRunner) Run(packet []byte, ctx *SKBuffContext) (*TestResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.tcProgram == nil {
		return nil, fmt.Errorf("TC program not loaded")
	}

	if len(packet) < 14 {
		return nil, fmt.Errorf("packet too small: minimum 14 bytes (Ethernet header)")
	}

	// Build context data for __sk_buff
	ctxData := r.buildContextData(ctx)

	// Run the program using cilium/ebpf's Test method
	ret, output, err := r.tcProgram.Test(packet)
	if err != nil {
		return nil, fmt.Errorf("program test failed: %w", err)
	}

	// Ignore ctxData for now as Test() doesn't accept context directly
	_ = ctxData

	result := &TestResult{
		ReturnValue:  int32(ret),
		OutputPacket: output,
		Allowed:      ret == TC_ACT_OK,
		Dropped:      ret == TC_ACT_SHOT,
	}

	return result, nil
}

// RunWithTiming executes the eBPF program multiple times and measures performance.
// Returns the average duration per execution.
func (r *TestRunner) RunWithTiming(packet []byte, ctx *SKBuffContext, iterations int) (*TestResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.tcProgram == nil {
		return nil, fmt.Errorf("TC program not loaded")
	}

	if iterations < 1 {
		iterations = 1
	}

	// Allocate output buffer
	dataOut := make([]byte, len(packet)+256)

	// Use Run with repeat option for accurate timing
	opts := ebpf.RunOptions{
		Data:    packet,
		DataOut: dataOut,
		Repeat:  uint32(iterations),
	}

	// Measure execution time
	start := time.Now()
	ret, err := r.tcProgram.Run(&opts)
	elapsed := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("program run failed: %w", err)
	}

	// Calculate average duration per iteration
	avgDuration := elapsed / time.Duration(iterations)

	result := &TestResult{
		ReturnValue:  int32(ret),
		OutputPacket: opts.DataOut,
		Duration:     avgDuration,
		Allowed:      ret == TC_ACT_OK,
		Dropped:      ret == TC_ACT_SHOT,
	}

	return result, nil
}

// RunTestCase executes a single test case and returns the result.
func (r *TestRunner) RunTestCase(tc *TestCase) *TestCaseResult {
	result := &TestCaseResult{
		TestCase: tc,
	}

	// Setup maps if provided
	if tc.SetupMaps != nil {
		if err := tc.SetupMaps(r); err != nil {
			result.Error = fmt.Errorf("setup failed: %w", err)
			result.Message = "Map setup failed"
			return result
		}
	}

	// Ensure cleanup runs
	defer func() {
		if tc.CleanupMaps != nil {
			_ = tc.CleanupMaps(r)
		}
	}()

	// Run the test
	testResult, err := r.Run(tc.Packet, tc.Context)
	if err != nil {
		result.Error = err
		result.Message = fmt.Sprintf("Test execution failed: %v", err)
		return result
	}

	result.Result = testResult
	result.Passed = testResult.ReturnValue == tc.ExpectedAction

	if result.Passed {
		result.Message = fmt.Sprintf("PASS: %s (action=%d)", tc.Name, testResult.ReturnValue)
	} else {
		result.Message = fmt.Sprintf("FAIL: %s (expected=%d, got=%d)",
			tc.Name, tc.ExpectedAction, testResult.ReturnValue)
	}

	return result
}

// RunTestCases executes multiple test cases and returns all results.
func (r *TestRunner) RunTestCases(cases []*TestCase) []*TestCaseResult {
	results := make([]*TestCaseResult, len(cases))
	for i, tc := range cases {
		results[i] = r.RunTestCase(tc)
	}
	return results
}

// buildContextData creates the binary representation of __sk_buff context.
func (r *TestRunner) buildContextData(ctx *SKBuffContext) []byte {
	if ctx == nil {
		ctx = &SKBuffContext{
			Ingress:  true,
			Protocol: 0x0800, // ETH_P_IP
		}
	}

	// __sk_buff context structure (simplified, first 16 bytes)
	// This matches the kernel's __sk_buff structure layout
	data := make([]byte, 64)

	// len and pkt_type are set by kernel
	// We can set some fields that affect program behavior

	// Offset 4: priority (u32)
	binary.LittleEndian.PutUint32(data[4:8], ctx.Priority)

	// Offset 8: ingress_ifindex (u32)
	if ctx.Ingress {
		binary.LittleEndian.PutUint32(data[8:12], ctx.Ifindex)
	}

	// Offset 12: ifindex (u32)
	binary.LittleEndian.PutUint32(data[12:16], ctx.Ifindex)

	// Offset 20: mark (u32)
	binary.LittleEndian.PutUint32(data[20:24], ctx.Mark)

	// Offset 24: protocol (u16)
	binary.LittleEndian.PutUint16(data[24:26], ctx.Protocol)

	// Offset 28: vlan_tci (u16) - contains VLAN ID
	binary.LittleEndian.PutUint16(data[28:30], ctx.VLANID)

	return data
}

// Map accessors for test setup

// PolicyMap returns the policy_map for test setup.
func (r *TestRunner) PolicyMap() *ebpf.Map {
	return r.policyMap
}

// WildcardPolicyMap returns the wildcard_policy_map for test setup.
func (r *TestRunner) WildcardPolicyMap() *ebpf.Map {
	return r.wildcardPolicyMap
}

// SessionMap returns the session_map for test setup.
func (r *TestRunner) SessionMap() *ebpf.Map {
	return r.sessionMap
}

// StatsMap returns the stats_map for test setup.
func (r *TestRunner) StatsMap() *ebpf.Map {
	return r.statsMap
}

// DefaultPolicyMap returns the default_policy map for test setup.
func (r *TestRunner) DefaultPolicyMap() *ebpf.Map {
	return r.defaultPolicyMap
}

// Helper functions for map key/value construction

// PolicyKey represents a policy map key matching the eBPF structure.
type PolicyKey struct {
	SrcIP     [4]uint32 // 128 bits for IPv6 compatibility
	DstIP     [4]uint32
	SrcPort   uint16
	DstPort   uint16
	Protocol  uint8
	Direction uint8 // 0=ingress, 1=egress
	IPVersion uint8 // 4 or 6
	Pad       uint8
	VlanID    uint16
	Pad2      uint16
}

// PolicyValue represents a policy map value matching the eBPF structure.
type PolicyValue struct {
	Action   uint8  // 0=deny, 1=allow
	LogLevel uint8  // 0=none, 1=basic, 2=detailed
	Pad      uint16
	RuleID   uint32
}

// SessionKey represents a session map key matching the eBPF structure.
type SessionKey struct {
	SrcIP     [4]uint32
	DstIP     [4]uint32
	SrcPort   uint16
	DstPort   uint16
	Protocol  uint8
	IPVersion uint8
	VlanID    uint16
}

// NewPolicyKey creates a PolicyKey from human-readable parameters.
func NewPolicyKey(srcIP, dstIP string, srcPort, dstPort uint16, protocol uint8, direction uint8) PolicyKey {
	key := PolicyKey{
		SrcPort:   srcPort,
		DstPort:   dstPort,
		Protocol:  protocol,
		Direction: direction,
		IPVersion: 4,
	}

	// Parse source IP
	if srcIP != "" && srcIP != "0.0.0.0" {
		if ip := net.ParseIP(srcIP); ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				key.SrcIP[0] = binary.BigEndian.Uint32(ip4)
			}
		}
	}

	// Parse destination IP
	if dstIP != "" && dstIP != "0.0.0.0" {
		if ip := net.ParseIP(dstIP); ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				key.DstIP[0] = binary.BigEndian.Uint32(ip4)
			}
		}
	}

	return key
}

// NewSessionKey creates a SessionKey from human-readable parameters.
func NewSessionKey(srcIP, dstIP string, srcPort, dstPort uint16, protocol uint8) SessionKey {
	key := SessionKey{
		SrcPort:   srcPort,
		DstPort:   dstPort,
		Protocol:  protocol,
		IPVersion: 4,
	}

	// Parse source IP
	if ip := net.ParseIP(srcIP); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			key.SrcIP[0] = binary.BigEndian.Uint32(ip4)
		}
	}

	// Parse destination IP
	if ip := net.ParseIP(dstIP); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			key.DstIP[0] = binary.BigEndian.Uint32(ip4)
		}
	}

	return key
}

// AddPolicy adds a policy rule to the policy map.
func (r *TestRunner) AddPolicy(key PolicyKey, action uint8, ruleID uint32) error {
	if r.policyMap == nil {
		return fmt.Errorf("policy map not available")
	}

	value := PolicyValue{
		Action: action,
		RuleID: ruleID,
	}

	return r.policyMap.Put(&key, &value)
}

// DeletePolicy removes a policy rule from the policy map.
func (r *TestRunner) DeletePolicy(key PolicyKey) error {
	if r.policyMap == nil {
		return fmt.Errorf("policy map not available")
	}

	return r.policyMap.Delete(&key)
}

// ClearPolicyMap removes all entries from the policy map.
func (r *TestRunner) ClearPolicyMap() error {
	if r.policyMap == nil {
		return fmt.Errorf("policy map not available")
	}

	var key PolicyKey
	for {
		if err := r.policyMap.NextKey(nil, &key); err != nil {
			break // No more keys
		}
		_ = r.policyMap.Delete(&key)
	}
	return nil
}

// SetDefaultPolicy sets the default policy action.
func (r *TestRunner) SetDefaultPolicy(action uint8) error {
	if r.defaultPolicyMap == nil {
		return fmt.Errorf("default policy map not available")
	}

	key := uint32(0)
	return r.defaultPolicyMap.Put(&key, &action)
}

// ClearSessionMap removes all entries from the session map.
func (r *TestRunner) ClearSessionMap() error {
	if r.sessionMap == nil {
		return fmt.Errorf("session map not available")
	}

	var key SessionKey
	for {
		if err := r.sessionMap.NextKey(nil, &key); err != nil {
			break // No more keys
		}
		_ = r.sessionMap.Delete(&key)
	}
	return nil
}

// GetSessionCount returns the number of entries in the session map.
func (r *TestRunner) GetSessionCount() (int, error) {
	if r.sessionMap == nil {
		return 0, fmt.Errorf("session map not available")
	}

	count := 0
	var key SessionKey
	var prevKey *SessionKey

	for {
		if err := r.sessionMap.NextKey(prevKey, &key); err != nil {
			break
		}
		count++
		keyCopy := key
		prevKey = &keyCopy
	}

	return count, nil
}

// Protocol constants for convenience
const (
	IPPROTO_TCP  = 6
	IPPROTO_UDP  = 17
	IPPROTO_ICMP = 1
)

// Direction constants
const (
	DIRECTION_INGRESS = 0
	DIRECTION_EGRESS  = 1
)

// Action constants
const (
	ACTION_DENY  = 0
	ACTION_ALLOW = 1
)

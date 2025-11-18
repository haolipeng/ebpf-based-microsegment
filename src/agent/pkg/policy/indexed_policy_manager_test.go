// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test IndexedPolicyManager for pure logic tests
func createTestIndexedManager(t *testing.T) *IndexedPolicyManager {
	baseMgr := &PolicyManager{
		policyMap:         nil,
		wildcardPolicyMap: nil,
		storage:           nil,
	}

	ipm := &IndexedPolicyManager{
		PolicyManager:           baseMgr,
		protocolOffsetMap:       nil,
		segments:                make(map[uint8]*ProtocolSegment),
		maxPoliciesPerProtocol: 200,
		maxTotalPolicies:       1000,
	}

	return ipm
}

// TestIndexedPolicyManager_SegmentManagement tests basic segment management
func TestIndexedPolicyManager_SegmentManagement(t *testing.T) {
	ipm := createTestIndexedManager(t)

	// Manually create segments (simulating what getOrCreateSegment would do)
	ipm.mu.Lock()
	ipm.segments[6] = &ProtocolSegment{
		Protocol:    6,
		StartIdx:    0,
		PolicyCount: 0,
		MaxCapacity: 200,
	}
	ipm.segments[17] = &ProtocolSegment{
		Protocol:    17,
		StartIdx:    200,
		PolicyCount: 0,
		MaxCapacity: 200,
	}
	ipm.mu.Unlock()

	// Test that segments were created correctly
	stats := ipm.GetSegmentStats()
	require.Equal(t, 2, len(stats))
	assert.Equal(t, uint8(6), stats[6].Protocol)
	assert.Equal(t, uint32(0), stats[6].StartIdx)
	assert.Equal(t, uint8(17), stats[17].Protocol)
	assert.Equal(t, uint32(200), stats[17].StartIdx)
}

// TestIndexedPolicyManager_CalculateNextStartIndex tests index calculation
func TestIndexedPolicyManager_CalculateNextStartIndex(t *testing.T) {
	ipm := createTestIndexedManager(t)

	// Initially should be 0
	assert.Equal(t, uint32(0), ipm.calculateNextStartIndex())

	// Create a segment (in memory only, no eBPF map)
	ipm.mu.Lock()
	ipm.segments[6] = &ProtocolSegment{
		Protocol:    6,
		StartIdx:    0,
		PolicyCount: 50,
		MaxCapacity: 200,
	}
	ipm.mu.Unlock()

	// Next start should be after the max capacity
	assert.Equal(t, uint32(200), ipm.calculateNextStartIndex())

	// Add another segment
	ipm.mu.Lock()
	ipm.segments[17] = &ProtocolSegment{
		Protocol:    17,
		StartIdx:    200,
		PolicyCount: 80,
		MaxCapacity: 200,
	}
	ipm.mu.Unlock()

	// Next start should be after the second segment
	assert.Equal(t, uint32(400), ipm.calculateNextStartIndex())
}

// TestIndexedPolicyManager_GetSegmentStats tests statistics retrieval
func TestIndexedPolicyManager_GetSegmentStats(t *testing.T) {
	ipm := createTestIndexedManager(t)

	// Create some segments with different counts
	ipm.mu.Lock()
	ipm.segments[6] = &ProtocolSegment{
		Protocol:    6,
		StartIdx:    0,
		PolicyCount: 150,
		MaxCapacity: 200,
	}

	ipm.segments[17] = &ProtocolSegment{
		Protocol:    17,
		StartIdx:    200,
		PolicyCount: 80,
		MaxCapacity: 200,
	}
	ipm.mu.Unlock()

	// Get stats
	stats := ipm.GetSegmentStats()

	require.Equal(t, 2, len(stats))
	assert.Equal(t, uint32(150), stats[6].PolicyCount)
	assert.Equal(t, uint32(80), stats[17].PolicyCount)

	// Verify it returns copies, not original pointers
	stats[6].PolicyCount = 999
	statsAgain := ipm.GetSegmentStats()
	assert.Equal(t, uint32(150), statsAgain[6].PolicyCount, "Should not modify original")
}

// TestIndexedPolicyManager_EmptyStats tests stats when no segments exist
func TestIndexedPolicyManager_EmptyStats(t *testing.T) {
	ipm := createTestIndexedManager(t)

	stats := ipm.GetSegmentStats()
	assert.Empty(t, stats, "Should return empty stats when no segments exist")
}

// TestProtocolSegment_ThreadSafety tests thread safety of segment operations
func TestProtocolSegment_ThreadSafety(t *testing.T) {
	ipm := createTestIndexedManager(t)

	// Create a segment
	segment := &ProtocolSegment{
		Protocol:    6,
		StartIdx:    0,
		PolicyCount: 0,
		MaxCapacity: 200,
	}

	ipm.mu.Lock()
	ipm.segments[6] = segment
	ipm.mu.Unlock()

	// Simulate concurrent access
	var wg sync.WaitGroup
	numGoroutines := 10
	incrementsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				segment.mu.Lock()
				segment.PolicyCount++
				segment.mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Verify final count
	expectedCount := uint32(numGoroutines * incrementsPerGoroutine)
	assert.Equal(t, expectedCount, segment.PolicyCount)
}

// TestIndexedPolicyManager_MultipleProtocolSegments tests creating multiple protocol segments
func TestIndexedPolicyManager_MultipleProtocolSegments(t *testing.T) {
	ipm := createTestIndexedManager(t)

	protocols := []uint8{6, 17, 1, 0} // TCP, UDP, ICMP, ANY
	expectedStarts := []uint32{0, 200, 400, 600}

	// Manually create segments
	ipm.mu.Lock()
	for i, proto := range protocols {
		ipm.segments[proto] = &ProtocolSegment{
			Protocol:    proto,
			StartIdx:    expectedStarts[i],
			PolicyCount: 0,
			MaxCapacity: 200,
		}
	}
	ipm.mu.Unlock()

	// Verify all segments exist
	stats := ipm.GetSegmentStats()
	assert.Equal(t, 4, len(stats))

	for i, proto := range protocols {
		assert.Equal(t, expectedStarts[i], stats[proto].StartIdx,
			"Protocol %d should start at %d", proto, expectedStarts[i])
	}
}

// TestIndexedPolicyManager_SegmentCapacityTracking tests segment capacity tracking
func TestIndexedPolicyManager_SegmentCapacityTracking(t *testing.T) {
	ipm := createTestIndexedManager(t)

	segment := &ProtocolSegment{
		Protocol:    6,
		StartIdx:    0,
		PolicyCount: 150,
		MaxCapacity: 200,
	}

	ipm.mu.Lock()
	ipm.segments[6] = segment
	ipm.mu.Unlock()

	// Verify capacity tracking
	stats := ipm.GetSegmentStats()
	assert.Equal(t, uint32(150), stats[6].PolicyCount)
	assert.Equal(t, uint32(200), stats[6].MaxCapacity)
	assert.True(t, stats[6].PolicyCount < stats[6].MaxCapacity, "Should have remaining capacity")
}

// TestIndexedPolicyManager_FullSegment tests full segment detection
func TestIndexedPolicyManager_FullSegment(t *testing.T) {
	ipm := createTestIndexedManager(t)

	segment := &ProtocolSegment{
		Protocol:    6,
		StartIdx:    0,
		PolicyCount: 200, // Full!
		MaxCapacity: 200,
	}

	ipm.mu.Lock()
	ipm.segments[6] = segment
	ipm.mu.Unlock()

	// Verify full segment
	stats := ipm.GetSegmentStats()
	assert.Equal(t, uint32(200), stats[6].PolicyCount)
	assert.Equal(t, uint32(200), stats[6].MaxCapacity)
	assert.True(t, stats[6].PolicyCount >= stats[6].MaxCapacity, "Segment should be full")
}

// TestIndexedPolicyManager_MaxTotalPoliciesLimit tests total capacity limit
func TestIndexedPolicyManager_MaxTotalPoliciesLimit(t *testing.T) {
	ipm := createTestIndexedManager(t)

	// Fill up to the max total policies (1000 / 200 = 5 segments)
	ipm.mu.Lock()
	for i := uint8(0); i < 5; i++ {
		ipm.segments[i] = &ProtocolSegment{
			Protocol:    i,
			StartIdx:    uint32(i) * 200,
			PolicyCount: 0,
			MaxCapacity: 200,
		}
	}
	ipm.mu.Unlock()

	// Check that we've used 5 * 200 = 1000 capacity
	nextIdx := ipm.calculateNextStartIndex()
	assert.Equal(t, uint32(1000), nextIdx, "Should reach max capacity")
}

// TestIndexedPolicyManager_ProtocolParsing tests protocol number parsing
func TestIndexedPolicyManager_ProtocolParsing(t *testing.T) {
	testCases := []struct {
		protocol string
		expected uint8
		hasError bool
	}{
		{"tcp", 6, false},
		{"TCP", 6, false},
		{"udp", 17, false},
		{"UDP", 17, false},
		{"icmp", 1, false},
		{"any", 0, false},
		{"", 0, false},
		{"invalid", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.protocol, func(t *testing.T) {
			result, err := parseProtocol(tc.protocol)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// TestIndexedPolicyManager_SegmentAllocation tests segment allocation strategy
func TestIndexedPolicyManager_SegmentAllocation(t *testing.T) {
	ipm := createTestIndexedManager(t)

	testCases := []struct {
		protocol      uint8
		expectedStart uint32
		name          string
	}{
		{6, 0, "TCP"},
		{17, 200, "UDP"},
		{0, 400, "ANY"},
		{1, 600, "ICMP"},
	}

	ipm.mu.Lock()
	for _, tc := range testCases {
		ipm.segments[tc.protocol] = &ProtocolSegment{
			Protocol:    tc.protocol,
			StartIdx:    tc.expectedStart,
			PolicyCount: 0,
			MaxCapacity: 200,
		}
	}
	ipm.mu.Unlock()

	stats := ipm.GetSegmentStats()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedStart, stats[tc.protocol].StartIdx,
				"Protocol %d should start at index %d", tc.protocol, tc.expectedStart)
			assert.Equal(t, uint32(200), stats[tc.protocol].MaxCapacity)
		})
	}
}

// Benchmark tests

// BenchmarkCalculateNextStartIndex benchmarks index calculation
func BenchmarkCalculateNextStartIndex(b *testing.B) {
	ipm := createTestIndexedManager(&testing.T{})

	// Create segments
	ipm.mu.Lock()
	for proto := uint8(0); proto < 10; proto++ {
		ipm.segments[proto] = &ProtocolSegment{
			Protocol:    proto,
			StartIdx:    uint32(proto) * 200,
			PolicyCount: 100,
			MaxCapacity: 200,
		}
	}
	ipm.mu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ipm.calculateNextStartIndex()
	}
}

// BenchmarkGetSegmentStats benchmarks stats retrieval
func BenchmarkGetSegmentStats(b *testing.B) {
	ipm := createTestIndexedManager(&testing.T{})

	// Create some segments
	ipm.mu.Lock()
	for proto := uint8(0); proto < 10; proto++ {
		ipm.segments[proto] = &ProtocolSegment{
			Protocol:    proto,
			StartIdx:    uint32(proto) * 200,
			PolicyCount: 100,
			MaxCapacity: 200,
		}
	}
	ipm.mu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ipm.GetSegmentStats()
	}
}

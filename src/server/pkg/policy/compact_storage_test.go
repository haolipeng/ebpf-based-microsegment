// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"context"
	"testing"

	policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCompactPolicyStorage(t *testing.T) {
	storage := NewCompactPolicyStorage(100)
	assert.NotNil(t, storage)
	assert.Equal(t, 100, storage.maxSlots)
	assert.Equal(t, -1, storage.firstFreeIndex)
	assert.Equal(t, 0, len(storage.slots))
	assert.Equal(t, 0, len(storage.policies))
}

func TestAddPolicy_FirstPolicy(t *testing.T) {
	storage := NewCompactPolicyStorage(100)
	ctx := context.Background()

	policy := &policypb.Policy{
		RuleId:   1001,
		SrcIp:    "10.0.0.0/24",
		DstIp:    "192.168.1.0/24",
		SrcPort:  0,
		DstPort:  80,
		Protocol: commonpb.Protocol_PROTOCOL_TCP,
		Action:   commonpb.PolicyAction_ACTION_ALLOW,
		Priority: 100,
	}

	err := storage.AddPolicy(ctx, policy)
	require.NoError(t, err)

	// Verify policy was added
	assert.Equal(t, 1, len(storage.policies))
	assert.Equal(t, 1, len(storage.slots))
	assert.Equal(t, policy, storage.slots[0])

	// Verify metadata
	slot, exists := storage.policies[1001]
	assert.True(t, exists)
	assert.Equal(t, 0, slot.SlotIndex)
	assert.Equal(t, policy, slot.Policy)
}

func TestAddPolicy_Multiple(t *testing.T) {
	storage := NewCompactPolicyStorage(100)
	ctx := context.Background()

	// Add 3 policies
	for i := 0; i < 3; i++ {
		policy := &policypb.Policy{
			RuleId:   uint32(1000 + i),
			SrcIp:    "10.0.0.0/24",
			DstIp:    "192.168.1.0/24",
			SrcPort:  0,
			DstPort:  uint32(80 + i),
			Protocol: commonpb.Protocol_PROTOCOL_TCP,
			Action:   commonpb.PolicyAction_ACTION_ALLOW,
			Priority: 100,
		}

		err := storage.AddPolicy(ctx, policy)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, len(storage.policies))
	assert.Equal(t, 3, len(storage.slots))
}

func TestAddPolicy_Update(t *testing.T) {
	storage := NewCompactPolicyStorage(100)
	ctx := context.Background()

	// Add initial policy
	policy := &policypb.Policy{
		RuleId:   1001,
		SrcIp:    "10.0.0.0/24",
		DstIp:    "192.168.1.0/24",
		SrcPort:  0,
		DstPort:  80,
		Protocol: commonpb.Protocol_PROTOCOL_TCP,
		Action:   commonpb.PolicyAction_ACTION_ALLOW,
		Priority: 100,
	}

	err := storage.AddPolicy(ctx, policy)
	require.NoError(t, err)

	// Update same policy
	updatedPolicy := &policypb.Policy{
		RuleId:   1001,
		SrcIp:    "10.0.0.0/24",
		DstIp:    "192.168.1.0/24",
		SrcPort:  0,
		DstPort:  443,  // Changed port
		Protocol: commonpb.Protocol_PROTOCOL_TCP,
		Action:   commonpb.PolicyAction_ACTION_DENY,  // Changed action
		Priority: 200,     // Changed priority
	}

	err = storage.AddPolicy(ctx, updatedPolicy)
	require.NoError(t, err)

	// Should still have only 1 policy at same slot
	assert.Equal(t, 1, len(storage.policies))
	assert.Equal(t, 1, len(storage.slots))

	// Verify updated values
	retrieved, err := storage.GetPolicy(1001)
	require.NoError(t, err)
	assert.Equal(t, uint32(443), retrieved.DstPort)
	assert.Equal(t, commonpb.PolicyAction_ACTION_DENY, retrieved.Action)
	assert.Equal(t, uint32(200), retrieved.Priority)
}

func TestDeletePolicy(t *testing.T) {
	storage := NewCompactPolicyStorage(100)
	ctx := context.Background()

	// Add policy
	policy := &policypb.Policy{
		RuleId:   1001,
		SrcIp:    "10.0.0.0/24",
		DstIp:    "192.168.1.0/24",
		SrcPort:  0,
		DstPort:  80,
		Protocol: commonpb.Protocol_PROTOCOL_TCP,
		Action:   commonpb.PolicyAction_ACTION_ALLOW,
		Priority: 100,
	}

	err := storage.AddPolicy(ctx, policy)
	require.NoError(t, err)

	// Delete policy
	err = storage.DeletePolicy(ctx, 1001)
	require.NoError(t, err)

	// Verify deletion
	assert.Equal(t, 0, len(storage.policies))
	assert.Nil(t, storage.slots[0])  // Slot should be marked empty

	// firstFreeIndex should point to deleted slot
	assert.Equal(t, 0, storage.firstFreeIndex)
}

func TestDeletePolicy_NotFound(t *testing.T) {
	storage := NewCompactPolicyStorage(100)
	ctx := context.Background()

	err := storage.DeletePolicy(ctx, 9999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy not found")
}

func TestSlotReuse(t *testing.T) {
	storage := NewCompactPolicyStorage(100)
	ctx := context.Background()

	// Add policies 1, 2, 3
	for i := 0; i < 3; i++ {
		policy := &policypb.Policy{
			RuleId:   uint32(1000 + i),
			SrcIp:    "10.0.0.0/24",
			DstIp:    "192.168.1.0/24",
			SrcPort:  0,
			DstPort:  uint32(80 + i),
			Protocol: commonpb.Protocol_PROTOCOL_TCP,
			Action:   commonpb.PolicyAction_ACTION_ALLOW,
			Priority: 100,
		}

		err := storage.AddPolicy(ctx, policy)
		require.NoError(t, err)
	}

	// Verify initial state: [P1000, P1001, P1002]
	assert.Equal(t, 3, len(storage.slots))

	// Delete policy 1001 (middle slot)
	err := storage.DeletePolicy(ctx, 1001)
	require.NoError(t, err)

	// State should be: [P1000, nil, P1002]
	assert.Equal(t, 3, len(storage.slots))
	assert.NotNil(t, storage.slots[0])
	assert.Nil(t, storage.slots[1])
	assert.NotNil(t, storage.slots[2])

	// firstFreeIndex should point to slot 1
	assert.Equal(t, 1, storage.firstFreeIndex)

	// Add new policy 1003
	newPolicy := &policypb.Policy{
		RuleId:   1003,
		SrcIp:    "10.0.0.0/24",
		DstIp:    "192.168.1.0/24",
		SrcPort:  0,
		DstPort:  8080,
		Protocol: commonpb.Protocol_PROTOCOL_TCP,
		Action:   commonpb.PolicyAction_ACTION_ALLOW,
		Priority: 100,
	}

	err = storage.AddPolicy(ctx, newPolicy)
	require.NoError(t, err)

	// New policy should reuse slot 1: [P1000, P1003, P1002]
	slot, exists := storage.policies[1003]
	assert.True(t, exists)
	assert.Equal(t, 1, slot.SlotIndex)  // Reused slot 1
	assert.Equal(t, newPolicy, storage.slots[1])

	// No empty slots now
	assert.Equal(t, -1, storage.firstFreeIndex)
}

func TestGetCompactPolicies(t *testing.T) {
	storage := NewCompactPolicyStorage(100)
	ctx := context.Background()

	// Add 3 policies
	for i := 0; i < 3; i++ {
		policy := &policypb.Policy{
			RuleId:   uint32(1000 + i),
			SrcIp:    "10.0.0.0/24",
			DstIp:    "192.168.1.0/24",
			SrcPort:  0,
			DstPort:  uint32(80 + i),
			Protocol: commonpb.Protocol_PROTOCOL_TCP,
			Action:   commonpb.PolicyAction_ACTION_ALLOW,
			Priority: 100,
		}

		err := storage.AddPolicy(ctx, policy)
		require.NoError(t, err)
	}

	// Delete middle policy
	err := storage.DeletePolicy(ctx, 1001)
	require.NoError(t, err)

	// GetCompactPolicies should include nil for deleted slot
	compact := storage.GetCompactPolicies()
	assert.Equal(t, 3, len(compact))
	assert.NotNil(t, compact[0])
	assert.Nil(t, compact[1])
	assert.NotNil(t, compact[2])
}

func TestGetActivePolicies(t *testing.T) {
	storage := NewCompactPolicyStorage(100)
	ctx := context.Background()

	// Add 3 policies
	for i := 0; i < 3; i++ {
		policy := &policypb.Policy{
			RuleId:   uint32(1000 + i),
			SrcIp:    "10.0.0.0/24",
			DstIp:    "192.168.1.0/24",
			SrcPort:  0,
			DstPort:  uint32(80 + i),
			Protocol: commonpb.Protocol_PROTOCOL_TCP,
			Action:   commonpb.PolicyAction_ACTION_ALLOW,
			Priority: 100,
		}

		err := storage.AddPolicy(ctx, policy)
		require.NoError(t, err)
	}

	// Delete middle policy
	err := storage.DeletePolicy(ctx, 1001)
	require.NoError(t, err)

	// GetActivePolicies should only return non-nil policies
	active := storage.GetActivePolicies()
	assert.Equal(t, 2, len(active))  // Only P1000 and P1002
}

func TestGetStats(t *testing.T) {
	storage := NewCompactPolicyStorage(100)
	ctx := context.Background()

	// Initially empty
	stats := storage.GetStats()
	assert.Equal(t, 0, stats.TotalSlots)
	assert.Equal(t, 0, stats.ActivePolicies)
	assert.Equal(t, 0, stats.EmptySlots)
	assert.Equal(t, -1, stats.FirstFreeIndex)
	assert.Equal(t, 0.0, stats.Utilization)

	// Add 3 policies
	for i := 0; i < 3; i++ {
		policy := &policypb.Policy{
			RuleId:   uint32(1000 + i),
			SrcIp:    "10.0.0.0/24",
			DstIp:    "192.168.1.0/24",
			SrcPort:  0,
			DstPort:  uint32(80 + i),
			Protocol: commonpb.Protocol_PROTOCOL_TCP,
			Action:   commonpb.PolicyAction_ACTION_ALLOW,
			Priority: 100,
		}

		err := storage.AddPolicy(ctx, policy)
		require.NoError(t, err)
	}

	stats = storage.GetStats()
	assert.Equal(t, 3, stats.TotalSlots)
	assert.Equal(t, 3, stats.ActivePolicies)
	assert.Equal(t, 0, stats.EmptySlots)
	assert.Equal(t, 100.0, stats.Utilization)

	// Delete one policy
	err := storage.DeletePolicy(ctx, 1001)
	require.NoError(t, err)

	stats = storage.GetStats()
	assert.Equal(t, 3, stats.TotalSlots)
	assert.Equal(t, 2, stats.ActivePolicies)
	assert.Equal(t, 1, stats.EmptySlots)
	assert.Equal(t, 1, stats.FirstFreeIndex)
	assert.InDelta(t, 66.67, stats.Utilization, 0.01)
}

func TestCompact(t *testing.T) {
	storage := NewCompactPolicyStorage(100)
	ctx := context.Background()

	// Add 5 policies
	for i := 0; i < 5; i++ {
		policy := &policypb.Policy{
			RuleId:   uint32(1000 + i),
			SrcIp:    "10.0.0.0/24",
			DstIp:    "192.168.1.0/24",
			SrcPort:  0,
			DstPort:  uint32(80 + i),
			Protocol: commonpb.Protocol_PROTOCOL_TCP,
			Action:   commonpb.PolicyAction_ACTION_ALLOW,
			Priority: 100,
		}

		err := storage.AddPolicy(ctx, policy)
		require.NoError(t, err)
	}

	// Delete policies 1001 and 1003 (create gaps)
	storage.DeletePolicy(ctx, 1001)
	storage.DeletePolicy(ctx, 1003)

	// Before compact: [P1000, nil, P1002, nil, P1004]
	assert.Equal(t, 5, len(storage.slots))
	assert.Equal(t, 3, len(storage.policies))

	// Compact
	storage.Compact()

	// After compact: [P1000, P1002, P1004]
	assert.Equal(t, 3, len(storage.slots))
	assert.Equal(t, 3, len(storage.policies))
	assert.Equal(t, -1, storage.firstFreeIndex)

	// Verify all slots are filled
	for i, slot := range storage.slots {
		assert.NotNil(t, slot, "slot %d should not be nil after compact", i)
	}

	// Verify slot indices are updated
	for _, ps := range storage.policies {
		assert.True(t, ps.SlotIndex >= 0 && ps.SlotIndex < len(storage.slots))
		assert.Equal(t, ps.Policy, storage.slots[ps.SlotIndex])
	}
}

func TestClear(t *testing.T) {
	storage := NewCompactPolicyStorage(100)
	ctx := context.Background()

	// Add policies
	for i := 0; i < 3; i++ {
		policy := &policypb.Policy{
			RuleId:   uint32(1000 + i),
			SrcIp:    "10.0.0.0/24",
			DstIp:    "192.168.1.0/24",
			SrcPort:  0,
			DstPort:  uint32(80 + i),
			Protocol: commonpb.Protocol_PROTOCOL_TCP,
			Action:   commonpb.PolicyAction_ACTION_ALLOW,
			Priority: 100,
		}

		err := storage.AddPolicy(ctx, policy)
		require.NoError(t, err)
	}

	// Clear
	storage.Clear()

	// Verify everything is reset
	assert.Equal(t, 0, len(storage.policies))
	assert.Equal(t, 0, len(storage.slots))
	assert.Equal(t, -1, storage.firstFreeIndex)
	assert.Equal(t, 0, len(storage.freeList))
}

func TestMaxSlots(t *testing.T) {
	storage := NewCompactPolicyStorage(3)
	ctx := context.Background()

	// Add 3 policies (max capacity)
	for i := 0; i < 3; i++ {
		policy := &policypb.Policy{
			RuleId:   uint32(1000 + i),
			SrcIp:    "10.0.0.0/24",
			DstIp:    "192.168.1.0/24",
			SrcPort:  0,
			DstPort:  uint32(80 + i),
			Protocol: commonpb.Protocol_PROTOCOL_TCP,
			Action:   commonpb.PolicyAction_ACTION_ALLOW,
			Priority: 100,
		}

		err := storage.AddPolicy(ctx, policy)
		require.NoError(t, err)
	}

	// Try to add 4th policy (should fail)
	policy := &policypb.Policy{
		RuleId:   1003,
		SrcIp:    "10.0.0.0/24",
		DstIp:    "192.168.1.0/24",
		SrcPort:  0,
		DstPort:  8080,
		Protocol: commonpb.Protocol_PROTOCOL_TCP,
		Action:   commonpb.PolicyAction_ACTION_ALLOW,
		Priority: 100,
	}

	err := storage.AddPolicy(ctx, policy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy storage full")
}

func TestConcurrentOperations(t *testing.T) {
	storage := NewCompactPolicyStorage(1000)
	ctx := context.Background()

	// Test concurrent adds
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			policy := &policypb.Policy{
				RuleId:   uint32(1000 + id),
				SrcIp:    "10.0.0.0/24",
				DstIp:    "192.168.1.0/24",
				SrcPort:  0,
				DstPort:  uint32(80 + id),
				Protocol: commonpb.Protocol_PROTOCOL_TCP,
				Action:   commonpb.PolicyAction_ACTION_ALLOW,
				Priority: 100,
			}

			storage.AddPolicy(ctx, policy)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, len(storage.policies))
}

// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"context"
	"fmt"
	"sync"

	policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
)

// CompactPolicyStorage manages compact storage of wildcard policies for efficient eBPF map synchronization.
// It maintains a first_free_index pointer to minimize gaps in the policy array, reducing eBPF scan overhead.
//
// Key features:
//   - Maintains first_free_index for optimal slot allocation
//   - Recycles deleted policy slots immediately
//   - Provides compact policy list for eBPF synchronization
//   - Thread-safe operations with mutex protection
type CompactPolicyStorage struct {
	// policies maps rule_id to its compact storage metadata
	policies map[uint32]*PolicySlot

	// slots is the actual array of policies in compact order
	// Empty slots have nil policy pointers
	slots []*policypb.Policy

	// firstFreeIndex points to the first available empty slot
	// -1 indicates no empty slots (need to append)
	firstFreeIndex int

	// freeList tracks all empty slots for quick reuse
	freeList []int

	// mu protects all fields for concurrent access
	mu sync.RWMutex

	// maxSlots is the maximum number of slots (default: 1000)
	maxSlots int
}

// PolicySlot stores metadata about a policy's storage slot
type PolicySlot struct {
	SlotIndex int                 // Index in the slots array
	Policy    *policypb.Policy    // The actual policy
}

// NewCompactPolicyStorage creates a new compact policy storage manager
func NewCompactPolicyStorage(maxSlots int) *CompactPolicyStorage {
	if maxSlots <= 0 {
		maxSlots = 1000 // Default max from eBPF MAX_ENTRIES_WILDCARD_POLICY
	}

	return &CompactPolicyStorage{
		policies:       make(map[uint32]*PolicySlot),
		slots:          make([]*policypb.Policy, 0, maxSlots),
		firstFreeIndex: -1,  // No slots initially
		freeList:       make([]int, 0),
		maxSlots:       maxSlots,
	}
}

// AddPolicy adds a policy to compact storage, reusing the first available empty slot
func (cs *CompactPolicyStorage) AddPolicy(ctx context.Context, policy *policypb.Policy) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Check if policy already exists (update case)
	if existingSlot, exists := cs.policies[policy.RuleId]; exists {
		// Update existing slot
		cs.slots[existingSlot.SlotIndex] = policy
		existingSlot.Policy = policy
		return nil
	}

	// Check capacity
	if len(cs.policies) >= cs.maxSlots {
		return fmt.Errorf("policy storage full: max=%d", cs.maxSlots)
	}

	var slotIndex int

	// Try to use first free slot
	if cs.firstFreeIndex >= 0 && cs.firstFreeIndex < len(cs.slots) {
		// Reuse empty slot
		slotIndex = cs.firstFreeIndex
		cs.slots[slotIndex] = policy

		// Update firstFreeIndex to next empty slot
		cs.updateFirstFreeIndex()
	} else {
		// No empty slots, append to end
		slotIndex = len(cs.slots)
		cs.slots = append(cs.slots, policy)

		// firstFreeIndex remains -1 (no empty slots)
	}

	// Store metadata
	cs.policies[policy.RuleId] = &PolicySlot{
		SlotIndex: slotIndex,
		Policy:    policy,
	}

	return nil
}

// DeletePolicy removes a policy and recycles its slot
func (cs *CompactPolicyStorage) DeletePolicy(ctx context.Context, ruleID uint32) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Find policy slot
	slot, exists := cs.policies[ruleID]
	if !exists {
		return fmt.Errorf("policy not found: rule_id=%d", ruleID)
	}

	// Mark slot as empty
	cs.slots[slot.SlotIndex] = nil

	// Update firstFreeIndex if this slot is earlier
	if cs.firstFreeIndex < 0 || slot.SlotIndex < cs.firstFreeIndex {
		cs.firstFreeIndex = slot.SlotIndex
	}

	// Add to free list
	cs.freeList = append(cs.freeList, slot.SlotIndex)

	// Remove from policies map
	delete(cs.policies, ruleID)

	return nil
}

// GetCompactPolicies returns policies in compact slot order for eBPF synchronization
// Empty slots are represented as nil entries
func (cs *CompactPolicyStorage) GetCompactPolicies() []*policypb.Policy {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Return copy of slots to prevent external modification
	result := make([]*policypb.Policy, len(cs.slots))
	copy(result, cs.slots)
	return result
}

// GetActivePolicies returns only non-empty policies (no nil slots)
func (cs *CompactPolicyStorage) GetActivePolicies() []*policypb.Policy {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	active := make([]*policypb.Policy, 0, len(cs.policies))
	for _, slot := range cs.slots {
		if slot != nil {
			active = append(active, slot)
		}
	}
	return active
}

// GetPolicy retrieves a specific policy by rule ID
func (cs *CompactPolicyStorage) GetPolicy(ruleID uint32) (*policypb.Policy, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	slot, exists := cs.policies[ruleID]
	if !exists {
		return nil, fmt.Errorf("policy not found: rule_id=%d", ruleID)
	}

	return slot.Policy, nil
}

// GetStats returns storage statistics
func (cs *CompactPolicyStorage) GetStats() CompactStorageStats {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	emptySlots := 0
	for _, policy := range cs.slots {
		if policy == nil {
			emptySlots++
		}
	}

	utilization := 0.0
	if len(cs.slots) > 0 {
		utilization = float64(len(cs.policies)) / float64(len(cs.slots)) * 100.0
	}

	return CompactStorageStats{
		TotalSlots:     len(cs.slots),
		ActivePolicies: len(cs.policies),
		EmptySlots:     emptySlots,
		FirstFreeIndex: cs.firstFreeIndex,
		Utilization:    utilization,
		MaxSlots:       cs.maxSlots,
	}
}

// CompactStorageStats provides storage utilization metrics
type CompactStorageStats struct {
	TotalSlots     int     // Total number of allocated slots
	ActivePolicies int     // Number of active policies
	EmptySlots     int     // Number of empty (recyclable) slots
	FirstFreeIndex int     // Index of first free slot (-1 if none)
	Utilization    float64 // Utilization percentage (0-100)
	MaxSlots       int     // Maximum allowed slots
}

// updateFirstFreeIndex updates the firstFreeIndex pointer to the next available empty slot
// Must be called with lock held
func (cs *CompactPolicyStorage) updateFirstFreeIndex() {
	// Remove used slot from free list
	if len(cs.freeList) > 0 && cs.freeList[0] == cs.firstFreeIndex {
		cs.freeList = cs.freeList[1:]
	}

	// Find next free slot
	if len(cs.freeList) > 0 {
		// Use first item from free list
		cs.firstFreeIndex = cs.freeList[0]
	} else {
		// Scan for next empty slot
		cs.firstFreeIndex = -1
		for i := 0; i < len(cs.slots); i++ {
			if cs.slots[i] == nil {
				cs.firstFreeIndex = i
				cs.freeList = append(cs.freeList, i)
				break
			}
		}
	}
}

// Clear removes all policies from storage
func (cs *CompactPolicyStorage) Clear() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.policies = make(map[uint32]*PolicySlot)
	cs.slots = make([]*policypb.Policy, 0, cs.maxSlots)
	cs.firstFreeIndex = -1
	cs.freeList = make([]int, 0)
}

// Compact removes all empty slots and defragments the storage
// This is useful when there are many gaps and you want to minimize eBPF scan overhead
func (cs *CompactPolicyStorage) Compact() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Build new compact slots array
	newSlots := make([]*policypb.Policy, 0, len(cs.policies))

	// Rebuild slots without gaps
	for _, policy := range cs.slots {
		if policy != nil {
			newSlots = append(newSlots, policy)
		}
	}

	// Update metadata with new indices
	for i, policy := range newSlots {
		if slot, exists := cs.policies[policy.RuleId]; exists {
			slot.SlotIndex = i
		}
	}

	// Replace slots
	cs.slots = newSlots
	cs.firstFreeIndex = -1
	cs.freeList = make([]int, 0)
}

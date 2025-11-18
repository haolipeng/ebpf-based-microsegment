// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"fmt"
	"sync"

	"github.com/cilium/ebpf"
	log "github.com/sirupsen/logrus"
)

// ProtocolSegment represents a contiguous range of wildcard policies for a specific protocol
type ProtocolSegment struct {
	Protocol     uint8  // Protocol number (6=TCP, 17=UDP, 0=ANY)
	StartIdx     uint32 // Starting index in wildcard_policy_map
	PolicyCount  uint32 // Number of policies in this segment
	MaxCapacity  uint32 // Maximum capacity for this segment
	mu           sync.RWMutex
}

// IndexedPolicyManager manages wildcard policies with protocol-based indexing
type IndexedPolicyManager struct {
	*PolicyManager // Embed base policy manager

	protocolOffsetMap *ebpf.Map // eBPF map: protocol -> segment descriptor
	segments          map[uint8]*ProtocolSegment // In-memory index
	mu                sync.RWMutex

	// Configuration
	maxPoliciesPerProtocol uint32
	maxTotalPolicies       uint32
}

// NewIndexedPolicyManager creates a new indexed policy manager
func NewIndexedPolicyManager(dp DataPlaneInterface) (*IndexedPolicyManager, error) {
	baseMgr := NewManager(dp)

	return &IndexedPolicyManager{
		PolicyManager:           baseMgr,
		protocolOffsetMap:       dp.GetProtocolOffsetMap(),
		segments:                make(map[uint8]*ProtocolSegment),
		maxPoliciesPerProtocol: 200, // MAX_POLICIES_PER_PROTOCOL from eBPF
		maxTotalPolicies:       1000, // MAX_ENTRIES_WILDCARD_POLICY
	}, nil
}

// NewIndexedPolicyManagerWithStorage creates a new indexed policy manager with persistence
func NewIndexedPolicyManagerWithStorage(dp DataPlaneInterface, storage Storage) (*IndexedPolicyManager, error) {
	baseMgr := NewManagerWithStorage(dp, storage)

	return &IndexedPolicyManager{
		PolicyManager:           baseMgr,
		protocolOffsetMap:       dp.GetProtocolOffsetMap(),
		segments:                make(map[uint8]*ProtocolSegment),
		maxPoliciesPerProtocol: 200,
		maxTotalPolicies:       1000,
	}, nil
}

// getOrCreateSegment gets or creates a protocol segment
func (ipm *IndexedPolicyManager) getOrCreateSegment(protocol uint8) (*ProtocolSegment, error) {
	ipm.mu.Lock()
	defer ipm.mu.Unlock()

	// Check if segment already exists
	if segment, exists := ipm.segments[protocol]; exists {
		return segment, nil
	}

	// Calculate next available start index
	nextIdx := ipm.calculateNextStartIndex()

	// Check if we have space
	if nextIdx >= ipm.maxTotalPolicies {
		return nil, fmt.Errorf("wildcard policy map is full (max %d policies)", ipm.maxTotalPolicies)
	}

	// Create new segment
	segment := &ProtocolSegment{
		Protocol:    protocol,
		StartIdx:    nextIdx,
		PolicyCount: 0,
		MaxCapacity: ipm.maxPoliciesPerProtocol,
	}

	// Store in memory
	ipm.segments[protocol] = segment

	// Update eBPF map
	if err := ipm.updateSegmentInMap(segment); err != nil {
		delete(ipm.segments, protocol)
		return nil, fmt.Errorf("failed to create segment in eBPF map: %w", err)
	}

	log.Infof("Created protocol segment: protocol=%d, start=%d, capacity=%d",
		protocol, nextIdx, ipm.maxPoliciesPerProtocol)

	return segment, nil
}

// calculateNextStartIndex calculates the next available start index for a new segment
func (ipm *IndexedPolicyManager) calculateNextStartIndex() uint32 {
	maxIdx := uint32(0)

	for _, segment := range ipm.segments {
		endIdx := segment.StartIdx + segment.MaxCapacity
		if endIdx > maxIdx {
			maxIdx = endIdx
		}
	}

	return maxIdx
}

// updateSegmentInMap updates a protocol segment descriptor in the eBPF map
func (ipm *IndexedPolicyManager) updateSegmentInMap(segment *ProtocolSegment) error {
	// Build segment descriptor matching eBPF struct protocol_segment
	descriptor := struct {
		StartIdx    uint32
		PolicyCount uint32
		Reserved    [2]uint32
	}{
		StartIdx:    segment.StartIdx,
		PolicyCount: segment.PolicyCount,
		Reserved:    [2]uint32{0, 0},
	}

	// Map key is protocol number
	key := uint32(segment.Protocol)

	return ipm.protocolOffsetMap.Put(&key, &descriptor)
}

// AddWildcardPolicyIndexed adds a wildcard policy using protocol indexing
func (ipm *IndexedPolicyManager) AddWildcardPolicyIndexed(p *Policy) error {
	// Parse protocol
	proto, err := parseProtocol(p.Protocol)
	if err != nil {
		return fmt.Errorf("invalid protocol: %w", err)
	}

	// Get or create protocol segment
	segment, err := ipm.getOrCreateSegment(proto)
	if err != nil {
		return err
	}

	// Lock segment for modification
	segment.mu.Lock()
	defer segment.mu.Unlock()

	// Check segment capacity
	if segment.PolicyCount >= segment.MaxCapacity {
		return fmt.Errorf("protocol %d segment is full (max %d policies)", proto, segment.MaxCapacity)
	}

	// Parse policy fields
	srcIP, srcMask, err := parseCIDR(p.SrcIP)
	if err != nil {
		return fmt.Errorf("invalid source IP: %w", err)
	}

	dstIP, dstMask, err := parseCIDR(p.DstIP)
	if err != nil {
		return fmt.Errorf("invalid destination IP: %w", err)
	}

	action, err := parseAction(p.Action)
	if err != nil {
		return fmt.Errorf("invalid action: %w", err)
	}

	// Build wildcard policy entry
	wildcard := struct {
		SrcIP      uint32
		SrcIPMask  uint32
		DstIP      uint32
		DstIPMask  uint32
		SrcPort    uint16
		DstPort    uint16
		Protocol   uint8
		Action     uint8
		LogEnabled uint8
		Direction  uint8
		Priority   uint16
		Pad        uint16
		RuleID     uint32
	}{
		SrcIP:      ipToUint32(srcIP),
		SrcIPMask:  maskToUint32(srcMask),
		DstIP:      ipToUint32(dstIP),
		DstIPMask:  maskToUint32(dstMask),
		SrcPort:    htons(p.SrcPort),
		DstPort:    htons(p.DstPort),
		Protocol:   proto,
		Action:     action,
		LogEnabled: boolToUint8(p.Action == "log"),
		Direction:  p.GetDirectionValue(),
		Priority:   p.Priority,
		Pad:        0,
		RuleID:     p.RuleID,
	}

	// Calculate slot index: segment_start + current_count
	slotIdx := segment.StartIdx + segment.PolicyCount

	// Insert into wildcard_policy_map
	if err := ipm.wildcardPolicyMap.Put(&slotIdx, &wildcard); err != nil {
		return fmt.Errorf("failed to add policy to wildcard map at index %d: %w", slotIdx, err)
	}

	// Update segment metadata
	segment.PolicyCount++

	// Update eBPF segment descriptor
	if err := ipm.updateSegmentInMap(segment); err != nil {
		// Rollback: remove policy from wildcard map
		zeroPolicy := struct {
			SrcIP      uint32
			SrcIPMask  uint32
			DstIP      uint32
			DstIPMask  uint32
			SrcPort    uint16
			DstPort    uint16
			Protocol   uint8
			Action     uint8
			LogEnabled uint8
			Direction  uint8
			Priority   uint16
			Pad        uint16
			RuleID     uint32
		}{}
		ipm.wildcardPolicyMap.Put(&slotIdx, &zeroPolicy)
		segment.PolicyCount--

		return fmt.Errorf("failed to update segment metadata: %w", err)
	}

	log.Infof("Wildcard policy added: protocol=%d slot=%d rule_id=%d %s:%d -> %s:%d action=%s priority=%d",
		proto, slotIdx, p.RuleID, p.SrcIP, p.SrcPort, p.DstIP, p.DstPort, p.Action, p.Priority)

	return nil
}

// DeleteWildcardPolicyIndexed deletes a wildcard policy using protocol indexing
func (ipm *IndexedPolicyManager) DeleteWildcardPolicyIndexed(p *Policy) error {
	// Parse protocol
	proto, err := parseProtocol(p.Protocol)
	if err != nil {
		return fmt.Errorf("invalid protocol: %w", err)
	}

	// Get protocol segment
	ipm.mu.RLock()
	segment, exists := ipm.segments[proto]
	ipm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no policies found for protocol %d", proto)
	}

	// Lock segment for modification
	segment.mu.Lock()
	defer segment.mu.Unlock()

	// Search for the policy in this segment
	startIdx := segment.StartIdx
	endIdx := startIdx + segment.PolicyCount

	for idx := startIdx; idx < endIdx; idx++ {
		var existing struct {
			SrcIP      uint32
			SrcIPMask  uint32
			DstIP      uint32
			DstIPMask  uint32
			SrcPort    uint16
			DstPort    uint16
			Protocol   uint8
			Action     uint8
			LogEnabled uint8
			Direction  uint8
			Priority   uint16
			Pad        uint16
			RuleID     uint32
		}

		if err := ipm.wildcardPolicyMap.Lookup(&idx, &existing); err != nil {
			continue
		}

		// Check if this is our target policy
		if existing.RuleID == p.RuleID {
			// Found it! Now we need to compact the segment
			// Move all policies after this one forward by 1 slot
			for moveIdx := idx; moveIdx < endIdx-1; moveIdx++ {
				nextIdx := moveIdx + 1
				var nextPolicy struct {
					SrcIP      uint32
					SrcIPMask  uint32
					DstIP      uint32
					DstIPMask  uint32
					SrcPort    uint16
					DstPort    uint16
					Protocol   uint8
					Action     uint8
					LogEnabled uint8
					Direction  uint8
					Priority   uint16
					Pad        uint16
					RuleID     uint32
				}

				if err := ipm.wildcardPolicyMap.Lookup(&nextIdx, &nextPolicy); err != nil {
					break
				}

				// Move policy forward
				if err := ipm.wildcardPolicyMap.Put(&moveIdx, &nextPolicy); err != nil {
					log.Warnf("Failed to compact policy at index %d: %v", moveIdx, err)
					continue
				}
			}

			// Zero out the last slot
			lastIdx := endIdx - 1
			zeroPolicy := struct {
				SrcIP      uint32
				SrcIPMask  uint32
				DstIP      uint32
				DstIPMask  uint32
				SrcPort    uint16
				DstPort    uint16
				Protocol   uint8
				Action     uint8
				LogEnabled uint8
				Direction  uint8
				Priority   uint16
				Pad        uint16
				RuleID     uint32
			}{}

			if err := ipm.wildcardPolicyMap.Put(&lastIdx, &zeroPolicy); err != nil {
				return fmt.Errorf("failed to zero last slot: %w", err)
			}

			// Update segment metadata
			segment.PolicyCount--

			// Update eBPF segment descriptor
			if err := ipm.updateSegmentInMap(segment); err != nil {
				return fmt.Errorf("failed to update segment metadata: %w", err)
			}

			log.Infof("Wildcard policy deleted: protocol=%d slot=%d rule_id=%d",
				proto, idx, p.RuleID)

			return nil
		}
	}

	return fmt.Errorf("wildcard policy with rule_id=%d not found in protocol %d segment", p.RuleID, proto)
}

// GetSegmentStats returns statistics about protocol segments
func (ipm *IndexedPolicyManager) GetSegmentStats() map[uint8]*ProtocolSegment {
	ipm.mu.RLock()
	defer ipm.mu.RUnlock()

	stats := make(map[uint8]*ProtocolSegment)
	for proto, segment := range ipm.segments {
		// Create a copy to avoid race conditions
		stats[proto] = &ProtocolSegment{
			Protocol:    segment.Protocol,
			StartIdx:    segment.StartIdx,
			PolicyCount: segment.PolicyCount,
			MaxCapacity: segment.MaxCapacity,
		}
	}

	return stats
}

// CompactAllSegments re-organizes all protocol segments to eliminate gaps
// This should be called periodically or after many deletions
func (ipm *IndexedPolicyManager) CompactAllSegments() error {
	ipm.mu.Lock()
	defer ipm.mu.Unlock()

	log.Info("Compacting all protocol segments...")

	// Collect all policies from all segments
	type policyEntry struct {
		Protocol uint8
		Policy   interface{} // wildcard policy struct
	}

	var allPolicies []policyEntry

	for proto, segment := range ipm.segments {
		segment.mu.RLock()
		startIdx := segment.StartIdx
		count := segment.PolicyCount
		segment.mu.RUnlock()

		for i := uint32(0); i < count; i++ {
			idx := startIdx + i
			var policy struct {
				SrcIP      uint32
				SrcIPMask  uint32
				DstIP      uint32
				DstIPMask  uint32
				SrcPort    uint16
				DstPort    uint16
				Protocol   uint8
				Action     uint8
				LogEnabled uint8
				Direction  uint8
				Priority   uint16
				Pad        uint16
				RuleID     uint32
			}

			if err := ipm.wildcardPolicyMap.Lookup(&idx, &policy); err == nil {
				if policy.RuleID != 0 {
					allPolicies = append(allPolicies, policyEntry{
						Protocol: proto,
						Policy:   policy,
					})
				}
			}
		}
	}

	log.Infof("Found %d active policies across all segments", len(allPolicies))

	// Clear all segments
	ipm.segments = make(map[uint8]*ProtocolSegment)

	// Re-insert all policies with compact storage
	nextIdx := uint32(0)
	protocolGroups := make(map[uint8][]interface{})

	// Group policies by protocol
	for _, entry := range allPolicies {
		protocolGroups[entry.Protocol] = append(protocolGroups[entry.Protocol], entry.Policy)
	}

	// Recreate segments
	for proto, policies := range protocolGroups {
		segment := &ProtocolSegment{
			Protocol:    proto,
			StartIdx:    nextIdx,
			PolicyCount: uint32(len(policies)),
			MaxCapacity: ipm.maxPoliciesPerProtocol,
		}

		// Write policies to wildcard map
		for i, policy := range policies {
			slotIdx := nextIdx + uint32(i)
			if err := ipm.wildcardPolicyMap.Put(&slotIdx, &policy); err != nil {
				log.Warnf("Failed to write policy at index %d: %v", slotIdx, err)
			}
		}

		// Update segment metadata
		ipm.segments[proto] = segment
		if err := ipm.updateSegmentInMap(segment); err != nil {
			log.Warnf("Failed to update segment metadata for protocol %d: %v", proto, err)
		}

		nextIdx += segment.MaxCapacity // Reserve space for future policies

		log.Infof("Compacted protocol %d segment: start=%d, count=%d",
			proto, segment.StartIdx, segment.PolicyCount)
	}

	log.Infof("Compaction complete: %d segments, %d policies",
		len(ipm.segments), len(allPolicies))

	return nil
}

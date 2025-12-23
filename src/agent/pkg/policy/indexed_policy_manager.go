// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: policy rules, lookup queries (by ID, source IP, dest IP, workload)
// output: multi-dimensional indexed policy storage, query results
// pos: indexed policy manager with multi-key lookups - if file updated, must sync with this header comment and pkg/policy/CLAUDE.md
package policy

import (
	"fmt"
	"sync"

	"github.com/cilium/ebpf"
	log "github.com/sirupsen/logrus"
)

// ProtocolSegment represents a contiguous range of wildcard policies for a specific protocol
// V3 Layout: [network policies | process policies]
type ProtocolSegment struct {
	Protocol      uint8  // Protocol number (6=TCP, 17=UDP, 0=ANY)
	StartIdx      uint32 // Starting index in wildcard_policy_map
	PolicyCount   uint32 // Total number of policies in this segment
	ProcessCount  uint32 // Number of process-specific policies (at the end)
	MaxCapacity   uint32 // Maximum capacity for this segment
	mu            sync.RWMutex
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

// isProcessPolicy checks if a policy is a process-specific policy
func isProcessPolicy(p *Policy) bool {
	return p.ProcessName != ""
}

// getNetworkPolicyCount returns the number of network-only policies in a segment
func (segment *ProtocolSegment) getNetworkPolicyCount() uint32 {
	return segment.PolicyCount - segment.ProcessCount
}

// updateSegmentInMap updates a protocol segment descriptor in the eBPF map
func (ipm *IndexedPolicyManager) updateSegmentInMap(segment *ProtocolSegment) error {
	// Build segment descriptor matching eBPF struct protocol_segment (V3)
	// struct protocol_segment {
	//     __u32 start_idx;
	//     __u32 policy_count;
	//     __u32 process_count;  // V3: Number of process-specific policies
	//     __u32 reserved;
	// }
	descriptor := struct {
		StartIdx     uint32
		PolicyCount  uint32
		ProcessCount uint32
		Reserved     uint32
	}{
		StartIdx:     segment.StartIdx,
		PolicyCount:  segment.PolicyCount,
		ProcessCount: segment.ProcessCount,
		Reserved:     0,
	}

	// Map key is protocol number
	key := uint32(segment.Protocol)

	return ipm.protocolOffsetMap.Put(&key, &descriptor)
}

// AddWildcardPolicyIndexed adds a wildcard policy using protocol indexing with V3 layout
// V3 Layout: [network policies | process policies]
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

	// Build wildcard policy entry (IPv4-mapped IPv6 format)
	// Must match struct wildcard_policy in common_types.h
	wildcard := struct {
		SrcIP       [4]uint32 // 16 bytes - IPv6 support
		SrcIPMask   [4]uint32 // 16 bytes
		DstIP       [4]uint32 // 16 bytes
		DstIPMask   [4]uint32 // 16 bytes
		SrcPort     uint16    // 2 bytes
		DstPort     uint16    // 2 bytes
		Protocol    uint8     // 1 byte
		Action      uint8     // 1 byte
		LogEnabled  uint8     // 1 byte
		Direction   uint8     // 1 byte
		IPVersion   uint8     // 1 byte - 4 = IPv4, 6 = IPv6
		Pad         [3]uint8  // 3 bytes - padding for alignment
		Priority    uint16    // 2 bytes
		VlanID      uint16    // 2 bytes
		RuleID      uint32    // 4 bytes
		ProcessName [16]byte  // 16 bytes - process name matching
	}{
		// Convert IPv4 to IPv6-mapped format
		SrcIP:       [4]uint32{0, 0, 0, ipToUint32LE(srcIP)},
		SrcIPMask:   [4]uint32{0, 0, 0, maskToUint32(srcMask)},
		DstIP:       [4]uint32{0, 0, 0, ipToUint32LE(dstIP)},
		DstIPMask:   [4]uint32{0, 0, 0, maskToUint32(dstMask)},
		SrcPort:     htons(p.SrcPort),
		DstPort:     htons(p.DstPort),
		Protocol:    proto,
		Action:      action,
		LogEnabled:  boolToUint8(p.Action == "log"),
		Direction:   p.GetDirectionValue(),
		IPVersion:   4, // IPv4 for now
		Pad:         [3]uint8{0, 0, 0},
		Priority:    p.Priority,
		VlanID:      0,
		RuleID:      p.RuleID,
		ProcessName: [16]byte{}, // Will be filled below
	}

	// Copy process name if present
	if len(p.ProcessName) > 0 {
		copy(wildcard.ProcessName[:], []byte(p.ProcessName))
	}

	// V3 Segmented Storage Logic
	var slotIdx uint32
	isProcPolicy := isProcessPolicy(p)

	if isProcPolicy {
		// Process policy: Add at the end of process segment
		slotIdx = segment.StartIdx + segment.PolicyCount
		segment.ProcessCount++
	} else {
		// Network policy: Add at the end of network segment (before process policies)
		networkCount := segment.getNetworkPolicyCount()
		slotIdx = segment.StartIdx + networkCount

		// If there are existing process policies, we need to shift them forward
		if segment.ProcessCount > 0 {
			// Shift all process policies one slot forward
			processStartIdx := segment.StartIdx + networkCount
			for i := int(segment.ProcessCount) - 1; i >= 0; i-- {
				oldIdx := processStartIdx + uint32(i)
				newIdx := oldIdx + 1

				// Read existing process policy
				var existingPolicy struct {
					SrcIP       [4]uint32
					SrcIPMask   [4]uint32
					DstIP       [4]uint32
					DstIPMask   [4]uint32
					SrcPort     uint16
					DstPort     uint16
					Protocol    uint8
					Action      uint8
					LogEnabled  uint8
					Direction   uint8
					IPVersion   uint8
					Pad         [3]uint8
					Priority    uint16
					VlanID      uint16
					RuleID      uint32
					ProcessName [16]byte
				}

				if err := ipm.wildcardPolicyMap.Lookup(&oldIdx, &existingPolicy); err != nil {
					return fmt.Errorf("failed to read policy at index %d during shift: %w", oldIdx, err)
				}

				// Write to new position
				if err := ipm.wildcardPolicyMap.Put(&newIdx, &existingPolicy); err != nil {
					return fmt.Errorf("failed to shift policy from %d to %d: %w", oldIdx, newIdx, err)
				}
			}
		}
	}

	// Insert policy at calculated slot
	if err := ipm.wildcardPolicyMap.Put(&slotIdx, &wildcard); err != nil {
		return fmt.Errorf("failed to add policy to wildcard map at index %d: %w", slotIdx, err)
	}

	// Update segment metadata
	segment.PolicyCount++

	// Update eBPF segment descriptor
	if err := ipm.updateSegmentInMap(segment); err != nil {
		return fmt.Errorf("failed to update segment metadata: %w", err)
	}

	policyType := "network"
	if isProcPolicy {
		policyType = "process"
	}

	log.Infof("Wildcard policy added (%s): protocol=%d slot=%d rule_id=%d %s:%d -> %s:%d action=%s priority=%d proc=%s",
		policyType, proto, slotIdx, p.RuleID, p.SrcIP, p.SrcPort, p.DstIP, p.DstPort, p.Action, p.Priority, p.ProcessName)

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
				SrcIP       [4]uint32
				SrcIPMask   [4]uint32
				DstIP       [4]uint32
				DstIPMask   [4]uint32
				SrcPort     uint16
				DstPort     uint16
				Protocol    uint8
				Action      uint8
				LogEnabled  uint8
				Direction   uint8
				IPVersion   uint8
				Pad         [3]uint8
				Priority    uint16
				VlanID      uint16
				RuleID      uint32
				ProcessName [16]byte
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

	// Recreate segments with V3 layout: [network policies | process policies]
	for proto, policies := range protocolGroups {
		// Separate network and process policies
		var networkPolicies []interface{}
		var processPolicies []interface{}

		for _, policy := range policies {
			// Type assert to check ProcessName field
			if p, ok := policy.(struct {
				SrcIP       [4]uint32
				SrcIPMask   [4]uint32
				DstIP       [4]uint32
				DstIPMask   [4]uint32
				SrcPort     uint16
				DstPort     uint16
				Protocol    uint8
				Action      uint8
				LogEnabled  uint8
				Direction   uint8
				IPVersion   uint8
				Pad         [3]uint8
				Priority    uint16
				VlanID      uint16
				RuleID      uint32
				ProcessName [16]byte
			}); ok {
				// Check if process policy (ProcessName is not empty)
				hasProcessName := false
				for _, b := range p.ProcessName {
					if b != 0 {
						hasProcessName = true
						break
					}
				}

				if hasProcessName {
					processPolicies = append(processPolicies, policy)
				} else {
					networkPolicies = append(networkPolicies, policy)
				}
			}
		}

		segment := &ProtocolSegment{
			Protocol:     proto,
			StartIdx:     nextIdx,
			PolicyCount:  uint32(len(networkPolicies) + len(processPolicies)),
			ProcessCount: uint32(len(processPolicies)),
			MaxCapacity:  ipm.maxPoliciesPerProtocol,
		}

		// Write network policies first
		for i, policy := range networkPolicies {
			slotIdx := nextIdx + uint32(i)
			if err := ipm.wildcardPolicyMap.Put(&slotIdx, &policy); err != nil {
				log.Warnf("Failed to write network policy at index %d: %v", slotIdx, err)
			}
		}

		// Write process policies after network policies
		processStartIdx := nextIdx + uint32(len(networkPolicies))
		for i, policy := range processPolicies {
			slotIdx := processStartIdx + uint32(i)
			if err := ipm.wildcardPolicyMap.Put(&slotIdx, &policy); err != nil {
				log.Warnf("Failed to write process policy at index %d: %v", slotIdx, err)
			}
		}

		// Update segment metadata
		ipm.segments[proto] = segment
		if err := ipm.updateSegmentInMap(segment); err != nil {
			log.Warnf("Failed to update segment metadata for protocol %d: %v", proto, err)
		}

		nextIdx += segment.MaxCapacity // Reserve space for future policies

		log.Infof("Compacted protocol %d segment: start=%d, network=%d, process=%d, total=%d",
			proto, segment.StartIdx, len(networkPolicies), len(processPolicies), segment.PolicyCount)
	}

	log.Infof("Compaction complete: %d segments, %d policies",
		len(ipm.segments), len(allPolicies))

	return nil
}

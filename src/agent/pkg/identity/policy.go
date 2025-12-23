// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: identity-based policy rules
// output: compiled identity policies
// pos: identity-based policy compiler - if file updated, must sync with this header comment and pkg/identity/CLAUDE.md
package identity

import (
	"fmt"
	"sync"

	"github.com/cilium/ebpf"
	log "github.com/sirupsen/logrus"
)

// IdentityPolicyKey represents the key structure for the identity policy BPF map
// Must match struct identity_policy_key in src/bpf/headers/identity_policy.h
type IdentityPolicyKey struct {
	SrcIdentity uint32 // Source security identity
	DstIdentity uint32 // Destination security identity
	DstPort     uint16 // Destination port (0 = any)
	Protocol    uint8  // Protocol (0 = any)
	Pad         uint8  // Padding for alignment
}

// IdentityPolicyValue represents the value structure for the identity policy BPF map
// Must match struct identity_policy_value in src/bpf/headers/identity_policy.h
type IdentityPolicyValue struct {
	Action     uint8  // Policy action (0=allow, 1=deny, 2=log)
	LogEnabled uint8  // Enable logging
	Priority   uint16 // Policy priority
	RuleID     uint32 // Rule identifier
	HitCount   uint64 // Match counter
}

// IdentityPolicyRule represents a high-level identity-based policy rule
type IdentityPolicyRule struct {
	RuleID       uint32            `json:"rule_id"`
	Name         string            `json:"name"`
	SrcLabels    map[string]string `json:"src_labels"`    // Source label selector
	DstLabels    map[string]string `json:"dst_labels"`    // Destination label selector
	DstPorts     []PortRange       `json:"dst_ports"`     // Destination ports (empty = any)
	Protocols    []string          `json:"protocols"`     // Protocols (empty = any)
	Action       string            `json:"action"`        // allow, deny, log
	Priority     uint16            `json:"priority"`      // Higher = more important
	SrcIdentity  NumericIdentity   `json:"-"`             // Resolved source identity
	DstIdentity  NumericIdentity   `json:"-"`             // Resolved destination identity
}

// PortRange represents a port or range of ports
type PortRange struct {
	Start uint16 `json:"start"`
	End   uint16 `json:"end"` // If End == 0 or End == Start, it's a single port
}

// IdentityPolicyManager manages identity-based policies
type IdentityPolicyManager struct {
	mu            sync.RWMutex
	bpfMap        *ebpf.Map
	identityCache *Cache
	rules         map[uint32]*IdentityPolicyRule // RuleID -> Rule
}

// NewIdentityPolicyManager creates a new identity policy manager
func NewIdentityPolicyManager(identityCache *Cache) *IdentityPolicyManager {
	return &IdentityPolicyManager{
		identityCache: identityCache,
		rules:         make(map[uint32]*IdentityPolicyRule),
	}
}

// SetBPFMap sets the BPF map for syncing policies
func (m *IdentityPolicyManager) SetBPFMap(bpfMap *ebpf.Map) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bpfMap = bpfMap
}

// AddRule adds an identity-based policy rule
func (m *IdentityPolicyManager) AddRule(rule *IdentityPolicyRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rule.RuleID == 0 {
		return fmt.Errorf("rule ID cannot be 0")
	}

	// Resolve source identity
	if len(rule.SrcLabels) > 0 {
		srcIdentity, ok := m.identityCache.GetByLabels(rule.SrcLabels)
		if !ok {
			return fmt.Errorf("no identity found for source labels: %v", rule.SrcLabels)
		}
		rule.SrcIdentity = srcIdentity.ID
	}

	// Resolve destination identity
	if len(rule.DstLabels) > 0 {
		dstIdentity, ok := m.identityCache.GetByLabels(rule.DstLabels)
		if !ok {
			return fmt.Errorf("no identity found for destination labels: %v", rule.DstLabels)
		}
		rule.DstIdentity = dstIdentity.ID
	}

	// Store the rule
	m.rules[rule.RuleID] = rule

	// Sync to BPF map
	if m.bpfMap != nil {
		if err := m.syncRuleToBPF(rule); err != nil {
			delete(m.rules, rule.RuleID)
			return fmt.Errorf("failed to sync rule to BPF: %w", err)
		}
	}

	log.WithFields(log.Fields{
		"rule_id":      rule.RuleID,
		"name":         rule.Name,
		"src_identity": rule.SrcIdentity,
		"dst_identity": rule.DstIdentity,
		"action":       rule.Action,
	}).Info("Added identity policy rule")

	return nil
}

// DeleteRule removes an identity-based policy rule
func (m *IdentityPolicyManager) DeleteRule(ruleID uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rule, ok := m.rules[ruleID]
	if !ok {
		return fmt.Errorf("rule %d not found", ruleID)
	}

	// Delete from BPF map
	if m.bpfMap != nil {
		if err := m.deleteRuleFromBPF(rule); err != nil {
			log.WithError(err).Warn("Failed to delete rule from BPF map")
		}
	}

	delete(m.rules, ruleID)

	log.WithField("rule_id", ruleID).Info("Deleted identity policy rule")

	return nil
}

// GetRule retrieves a rule by ID
func (m *IdentityPolicyManager) GetRule(ruleID uint32) (*IdentityPolicyRule, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rule, ok := m.rules[ruleID]
	return rule, ok
}

// ListRules returns all rules
func (m *IdentityPolicyManager) ListRules() []*IdentityPolicyRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*IdentityPolicyRule, 0, len(m.rules))
	for _, rule := range m.rules {
		rules = append(rules, rule)
	}
	return rules
}

// syncRuleToBPF syncs a rule to the BPF map
// Expands the rule into multiple BPF entries if needed (for port ranges)
func (m *IdentityPolicyManager) syncRuleToBPF(rule *IdentityPolicyRule) error {
	if m.bpfMap == nil {
		return nil
	}

	action := parseAction(rule.Action)
	protocols := expandProtocols(rule.Protocols)
	ports := expandPorts(rule.DstPorts)

	for _, proto := range protocols {
		for _, port := range ports {
			key := IdentityPolicyKey{
				SrcIdentity: uint32(rule.SrcIdentity),
				DstIdentity: uint32(rule.DstIdentity),
				DstPort:     port,
				Protocol:    proto,
			}

			value := IdentityPolicyValue{
				Action:     action,
				LogEnabled: boolToUint8(rule.Action == "log"),
				Priority:   rule.Priority,
				RuleID:     rule.RuleID,
				HitCount:   0,
			}

			if err := m.bpfMap.Put(&key, &value); err != nil {
				return fmt.Errorf("failed to put identity policy: %w", err)
			}
		}
	}

	return nil
}

// deleteRuleFromBPF deletes a rule from the BPF map
func (m *IdentityPolicyManager) deleteRuleFromBPF(rule *IdentityPolicyRule) error {
	if m.bpfMap == nil {
		return nil
	}

	protocols := expandProtocols(rule.Protocols)
	ports := expandPorts(rule.DstPorts)

	for _, proto := range protocols {
		for _, port := range ports {
			key := IdentityPolicyKey{
				SrcIdentity: uint32(rule.SrcIdentity),
				DstIdentity: uint32(rule.DstIdentity),
				DstPort:     port,
				Protocol:    proto,
			}

			if err := m.bpfMap.Delete(&key); err != nil {
				// Ignore "not found" errors
				log.WithError(err).Debug("Failed to delete identity policy key")
			}
		}
	}

	return nil
}

// AddDirectPolicy adds a policy directly with resolved identities
// Useful for receiving policies from the server
func (m *IdentityPolicyManager) AddDirectPolicy(
	srcIdentity, dstIdentity NumericIdentity,
	dstPort uint16,
	protocol uint8,
	action uint8,
	priority uint16,
	ruleID uint32,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.bpfMap == nil {
		return fmt.Errorf("BPF map not set")
	}

	key := IdentityPolicyKey{
		SrcIdentity: uint32(srcIdentity),
		DstIdentity: uint32(dstIdentity),
		DstPort:     dstPort,
		Protocol:    protocol,
	}

	value := IdentityPolicyValue{
		Action:     action,
		LogEnabled: 0,
		Priority:   priority,
		RuleID:     ruleID,
		HitCount:   0,
	}

	if err := m.bpfMap.Put(&key, &value); err != nil {
		return fmt.Errorf("failed to put identity policy: %w", err)
	}

	log.WithFields(log.Fields{
		"src_identity": srcIdentity,
		"dst_identity": dstIdentity,
		"dst_port":     dstPort,
		"protocol":     protocol,
		"action":       action,
		"rule_id":      ruleID,
	}).Debug("Added direct identity policy")

	return nil
}

// Clear removes all identity policies from the BPF map
func (m *IdentityPolicyManager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.bpfMap == nil {
		return nil
	}

	// Delete all entries from BPF map
	var key IdentityPolicyKey
	var value IdentityPolicyValue

	iter := m.bpfMap.Iterate()
	keysToDelete := make([]IdentityPolicyKey, 0)

	for iter.Next(&key, &value) {
		keysToDelete = append(keysToDelete, key)
	}

	for _, k := range keysToDelete {
		m.bpfMap.Delete(&k)
	}

	// Clear local rules
	m.rules = make(map[uint32]*IdentityPolicyRule)

	log.WithField("count", len(keysToDelete)).Info("Cleared identity policies")

	return nil
}

// Helper functions

func parseAction(action string) uint8 {
	switch action {
	case "allow":
		return 0
	case "deny":
		return 1
	case "log":
		return 2
	default:
		return 0 // Default to allow
	}
}

func expandProtocols(protocols []string) []uint8 {
	if len(protocols) == 0 {
		return []uint8{0} // Any protocol
	}

	result := make([]uint8, 0, len(protocols))
	for _, p := range protocols {
		switch p {
		case "tcp":
			result = append(result, 6)
		case "udp":
			result = append(result, 17)
		case "icmp":
			result = append(result, 1)
		case "any", "":
			result = append(result, 0)
		}
	}

	if len(result) == 0 {
		return []uint8{0}
	}
	return result
}

func expandPorts(ports []PortRange) []uint16 {
	if len(ports) == 0 {
		return []uint16{0} // Any port
	}

	result := make([]uint16, 0)
	for _, pr := range ports {
		if pr.End == 0 || pr.End == pr.Start {
			// Single port
			result = append(result, pr.Start)
		} else {
			// Port range - expand to individual ports
			// Note: This can create many entries for large ranges
			for p := pr.Start; p <= pr.End; p++ {
				result = append(result, p)
			}
		}
	}

	if len(result) == 0 {
		return []uint16{0}
	}
	return result
}

func boolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

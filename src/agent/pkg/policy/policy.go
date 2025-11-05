// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/cilium/ebpf"
	log "github.com/sirupsen/logrus"
)

// Policy represents a network policy rule
type Policy struct {
	RuleID   uint32
	SrcIP    string // CIDR notation
	DstIP    string // CIDR notation
	SrcPort  uint16
	DstPort  uint16
	Protocol string // "tcp", "udp", "icmp", "any"
	Action   string // "allow", "deny", "log"
	Priority uint16
}

// PolicyManager manages network policies
type PolicyManager struct {
	policyMap         *ebpf.Map
	wildcardPolicyMap *ebpf.Map
	storage           Storage
	compiler          *PolicyCompiler // 策略编译器（用于标签到IP的编译）
}

// DataPlaneInterface defines the interface for data plane operations
type DataPlaneInterface interface {
	GetPolicyMap() *ebpf.Map
	GetWildcardPolicyMap() *ebpf.Map
}

// NewManager creates a new policy manager without persistence
func NewManager(dp DataPlaneInterface) *PolicyManager {
	return &PolicyManager{
		policyMap:         dp.GetPolicyMap(),
		wildcardPolicyMap: dp.GetWildcardPolicyMap(),
		storage:           nil,
	}
}

// NewManagerWithStorage creates a new policy manager with persistence
func NewManagerWithStorage(dp DataPlaneInterface, storage Storage) *PolicyManager {
	return &PolicyManager{
		policyMap:         dp.GetPolicyMap(),
		wildcardPolicyMap: dp.GetWildcardPolicyMap(),
		storage:           storage,
		compiler:          nil,
	}
}

// SetCompiler sets the policy compiler for label-based policy support
func (pm *PolicyManager) SetCompiler(compiler *PolicyCompiler) {
	pm.compiler = compiler
}

// LoadPersisted loads policies from persistent storage and applies them to eBPF map
func (pm *PolicyManager) LoadPersisted() error {
	if pm.storage == nil {
		return fmt.Errorf("no storage configured")
	}

	policies, err := pm.storage.LoadPolicies()
	if err != nil {
		return fmt.Errorf("failed to load policies from storage: %w", err)
	}

	// Apply each policy to eBPF map
	successCount := 0
	for i := range policies {
		if err := pm.addPolicyToMap(&policies[i]); err != nil {
			log.Warnf("Failed to restore policy rule_id=%d: %v", policies[i].RuleID, err)
			continue
		}
		successCount++
	}

	log.Infof("Restored %d/%d policies from storage", successCount, len(policies))
	return nil
}

// AddPolicy adds a new policy rule
func (pm *PolicyManager) AddPolicy(p *Policy) error {
	// Add to eBPF map
	if err := pm.addPolicyToMap(p); err != nil {
		return err
	}

	// Save to persistent storage if configured
	if pm.storage != nil {
		if err := pm.storage.SavePolicy(p); err != nil {
			log.Warnf("Failed to persist policy rule_id=%d: %v", p.RuleID, err)
			// Continue even if persistence fails - eBPF map is the source of truth
		}
	}

	return nil
}

// hasWildcard checks if a policy contains wildcard fields (0 = any)
func hasWildcard(p *Policy) bool {
	// Check for wildcard source port (0 = any)
	if p.SrcPort == 0 {
		return true
	}
	// Check for wildcard CIDR (0.0.0.0/0 or ::/0)
	if p.SrcIP == "0.0.0.0/0" || p.SrcIP == "::/0" {
		return true
	}
	if p.DstIP == "0.0.0.0/0" || p.DstIP == "::/0" {
		return true
	}
	// Check for wildcard protocol
	if strings.ToLower(p.Protocol) == "any" {
		return true
	}
	return false
}

// addPolicyToMap adds a policy to the eBPF map (internal method)
// Routes to exact match map or wildcard map based on policy content
func (pm *PolicyManager) addPolicyToMap(p *Policy) error {
	// Check if this policy has wildcards
	if hasWildcard(p) {
		return pm.addWildcardPolicy(p)
	}
	return pm.addExactPolicy(p)
}

// addExactPolicy adds an exact-match policy to the hash map
func (pm *PolicyManager) addExactPolicy(p *Policy) error {
	// Parse source IP
	srcIP, srcMask, err := parseCIDR(p.SrcIP)
	if err != nil {
		return fmt.Errorf("invalid source IP: %w", err)
	}

	// Parse destination IP
	dstIP, dstMask, err := parseCIDR(p.DstIP)
	if err != nil {
		return fmt.Errorf("invalid destination IP: %w", err)
	}

	// Parse protocol
	proto, err := parseProtocol(p.Protocol)
	if err != nil {
		return fmt.Errorf("invalid protocol: %w", err)
	}

	// Parse action
	action, err := parseAction(p.Action)
	if err != nil {
		return fmt.Errorf("invalid action: %w", err)
	}

	// Build policy key
	key := struct {
		SrcIp    uint32
		DstIp    uint32
		SrcPort  uint16
		DstPort  uint16
		Protocol uint8
		Pad      [3]uint8
	}{
		SrcIp:    ipToUint32(srcIP),
		DstIp:    ipToUint32(dstIP),
		SrcPort:  htons(p.SrcPort),
		DstPort:  htons(p.DstPort),
		Protocol: proto,
	}

	// Build policy value
	value := struct {
		Action     uint8
		LogEnabled uint8
		Priority   uint16
		RuleID     uint32
		HitCount   uint64
	}{
		Action:     action,
		LogEnabled: boolToUint8(p.Action == "log"),
		Priority:   p.Priority,
		RuleID:     p.RuleID,
		HitCount:   0,
	}

	// Insert into eBPF map
	if err := pm.policyMap.Put(&key, &value); err != nil {
		return fmt.Errorf("failed to add policy to map: %w", err)
	}

	log.Infof("Policy added: rule_id=%d %s:%d -> %s:%d proto=%s action=%s",
		p.RuleID, p.SrcIP, p.SrcPort, p.DstIP, p.DstPort, p.Protocol, p.Action)

	// Note: For CIDR matching, we need to implement LPM trie
	// For now, we only support exact IP matching
	_ = srcMask
	_ = dstMask

	return nil
}

// DeletePolicy removes a policy rule based on its 5-tuple
func (pm *PolicyManager) DeletePolicy(p *Policy) error {
	// Parse IPs and protocol
	srcIP, _, err := parseCIDR(p.SrcIP)
	if err != nil {
		return fmt.Errorf("invalid source IP: %w", err)
	}

	dstIP, _, err := parseCIDR(p.DstIP)
	if err != nil {
		return fmt.Errorf("invalid destination IP: %w", err)
	}

	proto, err := parseProtocol(p.Protocol)
	if err != nil {
		return fmt.Errorf("invalid protocol: %w", err)
	}

	// Build policy key
	key := struct {
		SrcIp    uint32
		DstIp    uint32
		SrcPort  uint16
		DstPort  uint16
		Protocol uint8
		Pad      [3]uint8
	}{
		SrcIp:    ipToUint32(srcIP),
		DstIp:    ipToUint32(dstIP),
		SrcPort:  htons(p.SrcPort),
		DstPort:  htons(p.DstPort),
		Protocol: proto,
	}

	// Delete from eBPF map
	if err := pm.policyMap.Delete(&key); err != nil {
		return fmt.Errorf("failed to delete policy from map: %w", err)
	}

	log.Infof("Policy deleted: rule_id=%d %s:%d -> %s:%d proto=%s",
		p.RuleID, p.SrcIP, p.SrcPort, p.DstIP, p.DstPort, p.Protocol)

	// Delete from persistent storage if configured
	if pm.storage != nil {
		if err := pm.storage.DeletePolicy(p.RuleID); err != nil {
			log.Warnf("Failed to delete policy from storage rule_id=%d: %v", p.RuleID, err)
			// Continue even if persistence fails
		}
	}

	return nil
}

// ListPolicies lists all active policies
func (pm *PolicyManager) ListPolicies() ([]Policy, error) {
	var policies []Policy

	// Iterate through eBPF policy map
	var key struct {
		SrcIp    uint32
		DstIp    uint32
		SrcPort  uint16
		DstPort  uint16
		Protocol uint8
		Pad      [3]uint8
	}

	var value struct {
		Action     uint8
		LogEnabled uint8
		Priority   uint16
		RuleID     uint32
		HitCount   uint64
	}

	iter := pm.policyMap.Iterate()
	for iter.Next(&key, &value) {
		// Convert back to Policy struct
		policy := Policy{
			RuleID:   value.RuleID,
			SrcIP:    uint32ToIP(key.SrcIp),
			DstIP:    uint32ToIP(key.DstIp),
			SrcPort:  ntohs(key.SrcPort),
			DstPort:  ntohs(key.DstPort),
			Protocol: protoToString(key.Protocol),
			Action:   actionToString(value.Action),
			Priority: value.Priority,
		}
		policies = append(policies, policy)
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate policies: %w", err)
	}

	return policies, nil
}

// Helper functions

func parseCIDR(cidr string) (net.IP, *net.IPMask, error) {
	if !strings.Contains(cidr, "/") {
		cidr = cidr + "/32"
	}

	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, nil, err
	}

	return ip, &ipnet.Mask, nil
}

func parseProtocol(proto string) (uint8, error) {
	switch strings.ToLower(proto) {
	case "tcp":
		return 6, nil
	case "udp":
		return 17, nil
	case "icmp":
		return 1, nil
	case "any", "":
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown protocol: %s", proto)
	}
}

func parseAction(action string) (uint8, error) {
	switch strings.ToLower(action) {
	case "allow":
		return 0, nil
	case "deny":
		return 1, nil
	case "log":
		return 2, nil
	default:
		return 0, fmt.Errorf("unknown action: %s", action)
	}
}

func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(ip)
}

func htons(v uint16) uint16 {
	return (v<<8)&0xff00 | v>>8
}

func boolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func maskToUint32(mask *net.IPMask) uint32 {
	if mask == nil {
		return 0xFFFFFFFF // Exact match if no mask
	}
	if len(*mask) != 4 {
		return 0xFFFFFFFF
	}
	return binary.BigEndian.Uint32(*mask)
}

// addWildcardPolicy adds a wildcard policy to the array map
func (pm *PolicyManager) addWildcardPolicy(p *Policy) error {
	// Parse source IP
	srcIP, srcMask, err := parseCIDR(p.SrcIP)
	if err != nil {
		return fmt.Errorf("invalid source IP: %w", err)
	}

	// Parse destination IP
	dstIP, dstMask, err := parseCIDR(p.DstIP)
	if err != nil {
		return fmt.Errorf("invalid destination IP: %w", err)
	}

	// Parse protocol
	proto, err := parseProtocol(p.Protocol)
	if err != nil {
		return fmt.Errorf("invalid protocol: %w", err)
	}

	// Parse action
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
		Pad1       uint8
		Priority   uint16
		Pad2       uint16
		RuleID     uint32
	}{
		SrcIP:      ipToUint32(srcIP),
		SrcIPMask:  maskToUint32(srcMask),
		DstIP:      ipToUint32(dstIP),
		DstIPMask:  maskToUint32(dstMask),
		SrcPort:    htons(p.SrcPort), // 0 = wildcard
		DstPort:    htons(p.DstPort), // 0 = wildcard
		Protocol:   proto,             // 0 = wildcard
		Action:     action,
		LogEnabled: boolToUint8(p.Action == "log"),
		Priority:   p.Priority,
		RuleID:     p.RuleID,
	}

	// Find empty slot in wildcard array map
	// Try up to MAX_ENTRIES_WILDCARD_POLICY (1000)
	for i := uint32(0); i < 1000; i++ {
		// Try to read existing entry - must match the exact struct layout in eBPF
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
			Pad1       uint8
			Priority   uint16
			Pad2       uint16
			RuleID     uint32
		}

		// Read the existing entry
		err := pm.wildcardPolicyMap.Lookup(&i, &existing)

		// If lookup fails or RuleID is 0, slot is empty
		if err != nil || existing.RuleID == 0 {
			// Found empty slot, insert here
			if err := pm.wildcardPolicyMap.Put(&i, &wildcard); err != nil {
				return fmt.Errorf("failed to add wildcard policy to map slot %d: %w", i, err)
			}

			log.Infof("Wildcard policy added to slot %d: rule_id=%d %s:%d -> %s:%d proto=%s action=%s (priority=%d)",
				i, p.RuleID, p.SrcIP, p.SrcPort, p.DstIP, p.DstPort, p.Protocol, p.Action, p.Priority)
			return nil
		}

		// Check if this slot already has our rule_id (update case)
		if existing.RuleID == p.RuleID {
			// Update existing entry
			if err := pm.wildcardPolicyMap.Put(&i, &wildcard); err != nil {
				return fmt.Errorf("failed to update wildcard policy at slot %d: %w", i, err)
			}

			log.Infof("Wildcard policy updated at slot %d: rule_id=%d", i, p.RuleID)
			return nil
		}
	}

	return fmt.Errorf("wildcard policy map is full (max 1000 entries)")
}

func uint32ToIP(ip uint32) string {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, ip)
	return net.IPv4(buf[0], buf[1], buf[2], buf[3]).String()
}

func ntohs(v uint16) uint16 {
	return (v<<8)&0xff00 | v>>8
}

func protoToString(proto uint8) string {
	switch proto {
	case 6:
		return "tcp"
	case 17:
		return "udp"
	case 1:
		return "icmp"
	case 0:
		return "any"
	default:
		return fmt.Sprintf("%d", proto)
	}
}

func actionToString(action uint8) string {
	switch action {
	case 0:
		return "allow"
	case 1:
		return "deny"
	case 2:
		return "log"
	default:
		return fmt.Sprintf("%d", action)
	}
}

// ========== Label-Based Policy Methods ==========

// AddPolicyRule adds a label-based policy rule
// It compiles the rule to IP-based policies and loads them into eBPF
func (pm *PolicyManager) AddPolicyRule(r *PolicyRule) error {
	if pm.compiler == nil {
		return fmt.Errorf("policy compiler not configured")
	}

	if pm.storage == nil {
		return fmt.Errorf("storage not configured")
	}

	// 1. Store the source rule
	if err := pm.storage.CreatePolicyRule(r); err != nil {
		return fmt.Errorf("failed to store policy rule: %w", err)
	}

	// 2. Compile the rule to IP-based policies
	result, err := pm.compiler.CompilePolicyRule(r.ID)
	if err != nil {
		// Rollback: delete the source rule
		pm.storage.DeletePolicyRule(r.ID)
		return fmt.Errorf("failed to compile policy rule: %w", err)
	}

	// 3. Load each compiled policy into eBPF
	successCount := 0
	for _, cp := range result.CompiledPolicies {
		if err := pm.AddPolicy(&cp.Policy); err != nil {
			log.Warnf("Failed to load compiled policy %d: %v", cp.RuleID, err)
			continue
		}
		successCount++
	}

	if successCount == 0 && len(result.CompiledPolicies) > 0 {
		// All policies failed to load
		pm.compiler.InvalidateCompiledPolicies(r.ID)
		pm.storage.DeletePolicyRule(r.ID)
		return fmt.Errorf("failed to load any compiled policies")
	}

	log.Infof("Policy rule added: id=%d, name=%s, compiled=%d/%d policies",
		r.ID, r.Name, successCount, len(result.CompiledPolicies))

	return nil
}

// DeletePolicyRule deletes a label-based policy rule
// It removes all compiled policies from eBPF and storage
func (pm *PolicyManager) DeletePolicyRule(id uint32) error {
	if pm.compiler == nil {
		return fmt.Errorf("policy compiler not configured")
	}

	if pm.storage == nil {
		return fmt.Errorf("storage not configured")
	}

	// 1. Get all compiled policies for this rule
	compiledPolicies, err := pm.compiler.GetCompiledPoliciesForRule(id)
	if err != nil {
		log.Warnf("Failed to get compiled policies for rule %d: %v", id, err)
		// Continue anyway to try to delete the source rule
	}

	// 2. Delete each compiled policy from eBPF
	for _, cp := range compiledPolicies {
		if err := pm.DeletePolicy(&cp.Policy); err != nil {
			log.Warnf("Failed to delete compiled policy %d from eBPF: %v", cp.RuleID, err)
			// Continue deleting other policies
		}
	}

	// 3. Invalidate (delete) all compiled policies from storage
	if err := pm.compiler.InvalidateCompiledPolicies(id); err != nil {
		log.Warnf("Failed to invalidate compiled policies for rule %d: %v", id, err)
	}

	// 4. Delete the source policy rule
	if err := pm.storage.DeletePolicyRule(id); err != nil {
		return fmt.Errorf("failed to delete policy rule: %w", err)
	}

	log.Infof("Policy rule deleted: id=%d, removed %d compiled policies", id, len(compiledPolicies))
	return nil
}

// UpdatePolicyRule updates a label-based policy rule
// It recompiles and reloads the policies
func (pm *PolicyManager) UpdatePolicyRule(r *PolicyRule) error {
	if pm.compiler == nil {
		return fmt.Errorf("policy compiler not configured")
	}

	if pm.storage == nil {
		return fmt.Errorf("storage not configured")
	}

	// 1. Delete old compiled policies
	if err := pm.DeletePolicyRule(r.ID); err != nil {
		log.Warnf("Failed to delete old policy rule %d: %v", r.ID, err)
		// Continue with update
	}

	// 2. Add the updated rule (this will compile and load new policies)
	if err := pm.AddPolicyRule(r); err != nil {
		return fmt.Errorf("failed to add updated policy rule: %w", err)
	}

	log.Infof("Policy rule updated: id=%d, name=%s", r.ID, r.Name)
	return nil
}

// GetPolicyRule retrieves a policy rule by ID
func (pm *PolicyManager) GetPolicyRule(id uint32) (*PolicyRule, error) {
	if pm.storage == nil {
		return nil, fmt.Errorf("storage not configured")
	}

	return pm.storage.GetPolicyRule(id)
}

// ListPolicyRules lists all policy rules
func (pm *PolicyManager) ListPolicyRules() ([]*PolicyRule, error) {
	if pm.storage == nil {
		return nil, fmt.Errorf("storage not configured")
	}

	return pm.storage.ListPolicyRules()
}

// GetCompiledPoliciesForRule gets all compiled policies for a source rule
func (pm *PolicyManager) GetCompiledPoliciesForRule(ruleID uint32) ([]*CompiledPolicy, error) {
	if pm.compiler == nil {
		return nil, fmt.Errorf("policy compiler not configured")
	}

	return pm.compiler.GetCompiledPoliciesForRule(ruleID)
}

// RecompileAllPolicyRules recompiles all policy rules
// This is useful when workload membership changes
func (pm *PolicyManager) RecompileAllPolicyRules() error {
	if pm.compiler == nil {
		return fmt.Errorf("policy compiler not configured")
	}

	if pm.storage == nil {
		return fmt.Errorf("storage not configured")
	}

	// Get all enabled policy rules
	rules, err := pm.storage.ListEnabledPolicyRules()
	if err != nil {
		return fmt.Errorf("failed to list policy rules: %w", err)
	}

	log.Infof("Recompiling %d policy rules...", len(rules))

	successCount := 0
	for _, rule := range rules {
		// Delete old compiled policies
		oldPolicies, _ := pm.compiler.GetCompiledPoliciesForRule(rule.ID)
		for _, cp := range oldPolicies {
			pm.DeletePolicy(&cp.Policy)
		}
		pm.compiler.InvalidateCompiledPolicies(rule.ID)

		// Recompile
		result, err := pm.compiler.CompilePolicyRule(rule.ID)
		if err != nil {
			log.Errorf("Failed to recompile rule %d (%s): %v", rule.ID, rule.Name, err)
			continue
		}

		// Load new compiled policies
		loadedCount := 0
		for _, cp := range result.CompiledPolicies {
			if err := pm.AddPolicy(&cp.Policy); err != nil {
				log.Warnf("Failed to load compiled policy %d: %v", cp.RuleID, err)
				continue
			}
			loadedCount++
		}

		if loadedCount > 0 {
			successCount++
			log.Infof("Recompiled rule %d (%s): %d policies", rule.ID, rule.Name, loadedCount)
		}
	}

	log.Infof("Recompilation complete: %d/%d rules recompiled successfully", successCount, len(rules))
	return nil
}

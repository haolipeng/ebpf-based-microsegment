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
	policyMap *ebpf.Map
}

// DataPlaneInterface defines the interface for data plane operations
type DataPlaneInterface interface {
	GetPolicyMap() *ebpf.Map
}

// NewManager creates a new policy manager
func NewManager(dp DataPlaneInterface) *PolicyManager {
	return &PolicyManager{
		policyMap: dp.GetPolicyMap(),
	}
}

// AddPolicy adds a new policy rule
func (pm *PolicyManager) AddPolicy(p *Policy) error {
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

// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/vishvananda/netlink"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -no-strip -target bpf -type policy_key -type policy_value policy ../ebpf/policy.bpf.c -- -I../../../common/headers

const (
	// Policy actions
	PolicyActionAllow = 0
	PolicyActionDeny  = 1

	// Policy direction
	PolicyDirAny = 0
)

type Policy struct {
	SrcIP    net.IP
	DstIP    net.IP
	SrcPort  uint16
	DstPort  uint16
	Protocol uint8
	Action   uint8 // ALLOW or DENY
	RuleID   uint32
}

type Controller struct {
	objs     *policyObjects
	link     link.Link
	ifaceName string
	ifaceIdx int
}

func NewController(ifaceName string) (*Controller, error) {
	// Get interface
	iface, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("getting interface %s: %w", ifaceName, err)
	}

	// Load eBPF objects
	objs := &policyObjects{}
	if err := loadPolicyObjects(objs, nil); err != nil {
		return nil, fmt.Errorf("loading eBPF objects: %w", err)
	}

	// Attach to TC
	l, err := link.AttachTCX(link.TCXOptions{
		Interface: iface.Attrs().Index,
		Program:   objs.TcPolicyMatch,
		Attach:    ebpf.AttachTCXIngress,
	})
	if err != nil {
		// Try legacy TC if TCX fails
		log.Printf("TCX attach failed, trying legacy TC: %v", err)
		objs.Close()
		return nil, fmt.Errorf("attaching TC program: %w", err)
	}

	log.Printf("✓ eBPF program attached to %s", ifaceName)

	return &Controller{
		objs:      objs,
		link:      l,
		ifaceName: ifaceName,
		ifaceIdx:  iface.Attrs().Index,
	}, nil
}

func (c *Controller) Close() error {
	if c.link != nil {
		c.link.Close()
	}
	if c.objs != nil {
		c.objs.Close()
	}
	return nil
}

// AddPolicy adds a policy rule
func (c *Controller) AddPolicy(policy *Policy) error {
	key := policyPolicyKey{
		SrcIp:     ipToUint32(policy.SrcIP),
		DstIp:     ipToUint32(policy.DstIP),
		SrcPort:   htons(policy.SrcPort),
		DstPort:   htons(policy.DstPort),
		Protocol:  policy.Protocol,
		Direction: PolicyDirAny,
	}

	value := policyPolicyValue{
		Action:   policy.Action,
		Priority: 100,
		RuleId:   policy.RuleID,
		HitCount: 0,
	}

	if err := c.objs.PolicyMap.Put(&key, &value); err != nil {
		return fmt.Errorf("adding policy: %w", err)
	}

	action := "ALLOW"
	if policy.Action == PolicyActionDeny {
		action = "DENY"
	}

	log.Printf("✓ Added policy: %s %s:%d → %s:%d proto=%d (rule_id=%d)",
		action,
		policy.SrcIP, policy.SrcPort,
		policy.DstIP, policy.DstPort,
		policy.Protocol, policy.RuleID)

	return nil
}

// ListPolicies lists all policies
func (c *Controller) ListPolicies() error {
	fmt.Println("\n=== Policy Rules ===")

	var key policyPolicyKey
	var value policyPolicyValue
	iter := c.objs.PolicyMap.Iterate()

	count := 0
	for iter.Next(&key, &value) {
		count++
		action := "ALLOW"
		if value.Action == PolicyActionDeny {
			action = "DENY"
		}

		fmt.Printf("[%d] %s: %s:%d → %s:%d proto=%d (hits=%d, rule_id=%d)\n",
			count, action,
			uint32ToIP(key.SrcIp), ntohs(key.SrcPort),
			uint32ToIP(key.DstIp), ntohs(key.DstPort),
			key.Protocol,
			value.HitCount, value.RuleId)
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("iterating policies: %w", err)
	}

	if count == 0 {
		fmt.Println("(no policies configured)")
	}

	return nil
}

// ShowStats displays statistics
func (c *Controller) ShowStats() error {
	fmt.Println("\n=== Statistics ===")

	stats := []struct {
		key  uint32
		name string
	}{
		{0, "Total packets"},
		{1, "Allowed packets"},
		{2, "Denied packets"},
		{6, "Policy hits"},
		{7, "Policy misses"},
	}

	for _, s := range stats {
		var count uint64
		if err := c.objs.StatsMap.Lookup(&s.key, &count); err == nil {
			fmt.Printf("%-20s: %d\n", s.name, count)
		}
	}

	return nil
}

// Helper functions
func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return binary.BigEndian.Uint32(ip)
}

func uint32ToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}

func htons(n uint16) uint16 {
	return (n<<8)&0xff00 | (n>>8)&0x00ff
}

func ntohs(n uint16) uint16 {
	return (n<<8)&0xff00 | (n>>8)&0x00ff
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <interface>\n", os.Args[0])
		fmt.Println("\nExample commands:")
		fmt.Println("  sudo ./policy lo    # Start controller on loopback")
		fmt.Println("  (Interactive commands will be shown after startup)")
		os.Exit(1)
	}

	ifaceName := os.Args[1]

	// Create controller
	ctrl, err := NewController(ifaceName)
	if err != nil {
		log.Fatalf("Failed to create controller: %v", err)
	}
	defer ctrl.Close()

	// Add example policies
	fmt.Println("\n=== Adding Example Policies ===")

	// Policy 1: Allow SSH (port 22)
	ctrl.AddPolicy(&Policy{
		SrcIP:    net.ParseIP("0.0.0.0"),
		DstIP:    net.ParseIP("0.0.0.0"),
		SrcPort:  0,
		DstPort:  22,
		Protocol: 6, // TCP
		Action:   PolicyActionAllow,
		RuleID:   1,
	})

	// Policy 2: Deny HTTP (port 80)
	ctrl.AddPolicy(&Policy{
		SrcIP:    net.ParseIP("0.0.0.0"),
		DstIP:    net.ParseIP("0.0.0.0"),
		SrcPort:  0,
		DstPort:  80,
		Protocol: 6, // TCP
		Action:   PolicyActionDeny,
		RuleID:   2,
	})

	// Show initial state
	ctrl.ListPolicies()
	ctrl.ShowStats()

	fmt.Println("\n=== Controller Running ===")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println("\nView logs with:")
	fmt.Println("  sudo cat /sys/kernel/debug/tracing/trace_pipe")

	// Wait for signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\n\n=== Final Statistics ===")
	ctrl.ShowStats()
	ctrl.ListPolicies()
}

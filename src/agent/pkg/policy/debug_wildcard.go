// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
// +build ignore

package main

import (
	"fmt"
	"log"
	"net"

	"github.com/cilium/ebpf"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/dataplane"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/policy"
)

func main() {
	// 创建 DataPlane
	dp, err := dataplane.New("lo") // 使用 loopback
	if err != nil {
		log.Fatalf("Failed to create dataplane: %v", err)
	}
	defer dp.Close()

	// 创建 PolicyManager
	storage, err := policy.NewSQLiteStorage("/tmp/debug_policy.db")
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	pm := policy.NewManagerWithStorage(dp, storage)

	// 创建测试策略
	testPolicy := &policy.Policy{
		RuleID:    5000,
		SrcIP:     "10.100.0.2",
		DstIP:     "10.100.0.1",
		SrcPort:   0,
		DstPort:   9000,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "egress",
		Priority:  100,
	}

	fmt.Printf("Creating policy:\n")
	fmt.Printf("  RuleID: %d\n", testPolicy.RuleID)
	fmt.Printf("  %s:%d -> %s:%d\n", testPolicy.SrcIP, testPolicy.SrcPort, testPolicy.DstIP, testPolicy.DstPort)
	fmt.Printf("  Protocol: %s, Action: %s, Direction: %s\n", testPolicy.Protocol, testPolicy.Action, testPolicy.Direction)
	fmt.Printf("  Priority: %d\n", testPolicy.Priority)

	err = pm.AddPolicy(testPolicy)
	if err != nil {
		log.Fatalf("Failed to add policy: %v", err)
	}

	fmt.Printf("\n✓ Policy added successfully\n")

	// 获取 maps
	maps, err := dp.GetMaps()
	if err != nil {
		log.Fatalf("Failed to get maps: %v", err)
	}

	// Dump wildcard policy map
	fmt.Printf("\n=== Wildcard Policy Map ===\n")
	wildcardMap := maps.WildcardPolicyMap

	for i := uint32(0); i < 10; i++ {
		var entry struct {
			SrcIP      uint32
			SrcIPMask  uint32
			DstIP      uint32
			DstIPMask  uint32
			SrcPort    uint16
			DstPort    uint16
			Protocol   uint8
			Direction  uint8
			Action     uint8
			LogEnabled uint8
			Priority   uint16
			Pad        uint16
			RuleID     uint32
		}

		err := wildcardMap.Lookup(&i, &entry)
		if err != nil && err != ebpf.ErrKeyNotExist {
			fmt.Printf("Error reading slot %d: %v\n", i, err)
			continue
		}

		if entry.RuleID == 0 {
			continue // Empty slot
		}

		srcIP := uint32ToIP(entry.SrcIP)
		dstIP := uint32ToIP(entry.DstIP)

		fmt.Printf("\nSlot %d:\n", i)
		fmt.Printf("  RuleID: %d\n", entry.RuleID)
		fmt.Printf("  SrcIP: %s (mask: 0x%08x)\n", srcIP, entry.SrcIPMask)
		fmt.Printf("  DstIP: %s (mask: 0x%08x)\n", dstIP, entry.DstIPMask)
		fmt.Printf("  SrcPort: %d (0x%04x)\n", ntohs(entry.SrcPort), entry.SrcPort)
		fmt.Printf("  DstPort: %d (0x%04x)\n", ntohs(entry.DstPort), entry.DstPort)
		fmt.Printf("  Protocol: %d\n", entry.Protocol)
		fmt.Printf("  Direction: %d (0=ANY, 1=INGRESS, 2=EGRESS)\n", entry.Direction)
		fmt.Printf("  Action: %d (0=ALLOW, 1=DENY, 2=LOG)\n", entry.Action)
		fmt.Printf("  Priority: %d\n", entry.Priority)
		fmt.Printf("  LogEnabled: %d\n", entry.LogEnabled)
	}

	fmt.Printf("\n=== End of Dump ===\n")
}

func uint32ToIP(val uint32) net.IP {
	ip := make(net.IP, 4)
	ip[0] = byte(val & 0xff)
	ip[1] = byte((val >> 8) & 0xff)
	ip[2] = byte((val >> 16) & 0xff)
	ip[3] = byte((val >> 24) & 0xff)
	return ip
}

func ntohs(val uint16) uint16 {
	return (val >> 8) | (val << 8)
}

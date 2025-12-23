// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
// +build ignore

// input: policy rules with different directions (ingress/egress/bidirectional)
// output: direction handling test results
// pos: policy direction testing utility - if file updated, must sync with this header comment and pkg/policy/CLAUDE.md
// 独立测试程序,验证 Direction 字段功能
package main

import (
	"fmt"
	"os"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/policy"
)

func main() {
	fmt.Println("=== Policy Direction 字段测试 ===\n")

	// 测试 1: Direction 常量
	fmt.Println("测试 1: Direction 常量定义")
	fmt.Printf("  DirectionAny     = '%s'\n", policy.DirectionAny)
	fmt.Printf("  DirectionIngress = '%s'\n", policy.DirectionIngress)
	fmt.Printf("  DirectionEgress  = '%s'\n", policy.DirectionEgress)
	fmt.Println()

	// 测试 2: GetDirectionValue 转换
	fmt.Println("测试 2: GetDirectionValue 转换")
	testCases := []struct {
		direction string
		expected  uint8
	}{
		{"any", 0},
		{"ingress", 1},
		{"egress", 2},
		{"INGRESS", 1},
		{"Egress", 2},
		{"", 0},
		{"invalid", 0},
	}

	allPassed := true
	for _, tc := range testCases {
		p := &policy.Policy{Direction: tc.direction}
		result := p.GetDirectionValue()
		status := "✓"
		if result != tc.expected {
			status = "✗"
			allPassed = false
		}
		fmt.Printf("  %s '%s' -> %d (expected %d)\n", status, tc.direction, result, tc.expected)
	}
	fmt.Println()

	// 测试 3: NormalizeDirection
	fmt.Println("测试 3: NormalizeDirection 规范化")
	normalizeCases := []struct {
		input    string
		expected string
	}{
		{"ANY", "any"},
		{"INGRESS", "ingress"},
		{"Egress", "egress"},
		{"", "any"},
		{"invalid", "any"},
	}

	for _, tc := range normalizeCases {
		p := &policy.Policy{Direction: tc.input}
		p.NormalizeDirection()
		status := "✓"
		if p.Direction != tc.expected {
			status = "✗"
			allPassed = false
		}
		fmt.Printf("  %s '%s' -> '%s' (expected '%s')\n", status, tc.input, p.Direction, tc.expected)
	}
	fmt.Println()

	// 测试 4: Validate
	fmt.Println("测试 4: Validate 验证")
	validPolicy := policy.Policy{
		RuleID:    1,
		SrcIP:     "192.168.1.0/24",
		DstIP:     "10.0.0.5/32",
		DstPort:   22,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
	}
	err := validPolicy.Validate()
	if err == nil {
		fmt.Println("  ✓ Valid ingress policy passed validation")
	} else {
		fmt.Printf("  ✗ Valid policy failed: %v\n", err)
		allPassed = false
	}

	invalidPolicy := policy.Policy{
		RuleID:    2,
		SrcIP:     "192.168.1.0/24",
		DstIP:     "10.0.0.5/32",
		DstPort:   22,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "invalid_direction",
	}
	err = invalidPolicy.Validate()
	if err != nil {
		fmt.Printf("  ✓ Invalid direction rejected: %v\n", err)
	} else {
		fmt.Println("  ✗ Invalid direction was not rejected")
		allPassed = false
	}
	fmt.Println()

	// 测试 5: 完整的策略创建
	fmt.Println("测试 5: 完整的策略结构")
	completePolicy := policy.Policy{
		RuleID:    100,
		SrcIP:     "192.168.1.100/32",
		DstIP:     "10.0.0.10/32",
		SrcPort:   0,
		DstPort:   80,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "egress",
		Priority:  200,
	}
	fmt.Printf("  RuleID:    %d\n", completePolicy.RuleID)
	fmt.Printf("  SrcIP:     %s\n", completePolicy.SrcIP)
	fmt.Printf("  DstIP:     %s\n", completePolicy.DstIP)
	fmt.Printf("  SrcPort:   %d\n", completePolicy.SrcPort)
	fmt.Printf("  DstPort:   %d\n", completePolicy.DstPort)
	fmt.Printf("  Protocol:  %s\n", completePolicy.Protocol)
	fmt.Printf("  Action:    %s\n", completePolicy.Action)
	fmt.Printf("  Direction: %s (value=%d)\n", completePolicy.Direction, completePolicy.GetDirectionValue())
	fmt.Printf("  Priority:  %d\n", completePolicy.Priority)
	fmt.Println()

	// 最终结果
	fmt.Println("=== 测试结果 ===")
	if allPassed {
		fmt.Println("✓ 所有测试通过!")
		os.Exit(0)
	} else {
		fmt.Println("✗ 部分测试失败")
		os.Exit(1)
	}
}

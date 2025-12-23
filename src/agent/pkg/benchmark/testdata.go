// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: N/A (test data)
// output: benchmark test cases
// pos: benchmark test data generator - if file updated, must sync with this header comment and pkg/benchmark/CLAUDE.md
package benchmark

import (
	"fmt"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/policy"
)

// GenerateTestPolicies generates test wildcard policies for benchmarking
// The last policy is set as a matching policy to test the worst-case scenario
// of linear scanning through all policies.
func GenerateTestPolicies(count int) []*policy.Policy {
	if count <= 0 {
		return nil
	}

	policies := make([]*policy.Policy, count)

	// Generate non-matching policies (except the last one)
	for i := 0; i < count-1; i++ {
		policies[i] = &policy.Policy{
			RuleID:    uint32(10000 + i),
			SrcIP:     fmt.Sprintf("192.168.%d.0/24", i%256),
			DstIP:     "10.0.0.0/8",
			SrcPort:   0, // Wildcard (any port)
			DstPort:   uint16(8000 + i),
			Protocol:  "tcp",
			Action:    "deny",
			Direction: "ingress",
			Priority:  uint16(100),
		}
	}

	// Last policy: matching policy with highest priority
	// This tests the worst-case scenario where we scan all policies
	policies[count-1] = &policy.Policy{
		RuleID:    uint32(10000 + count - 1),
		SrcIP:     "0.0.0.0/0", // Match any source
		DstIP:     "0.0.0.0/0", // Match any destination
		SrcPort:   0,           // Match any source port
		DstPort:   0,           // Match any destination port
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  uint16(200), // Highest priority
	}

	return policies
}

// GenerateTestPoliciesCustom generates test policies with custom configuration
type PolicyConfig struct {
	Count            int
	BaseRuleID       uint32
	Protocol         string
	Action           string
	Direction        string
	BasePriority     uint16
	MatchingAtEnd    bool // If true, place matching policy at end
	DensityFactor    float64 // 0.0-1.0, controls how many policies actually match
}

// GenerateTestPoliciesCustom creates test policies with custom parameters
func GenerateTestPoliciesCustom(config PolicyConfig) []*policy.Policy {
	if config.Count <= 0 {
		return nil
	}

	// Set defaults
	if config.Protocol == "" {
		config.Protocol = "tcp"
	}
	if config.Action == "" {
		config.Action = "deny"
	}
	if config.Direction == "" {
		config.Direction = "ingress"
	}
	if config.BasePriority == 0 {
		config.BasePriority = 100
	}

	policies := make([]*policy.Policy, config.Count)

	for i := 0; i < config.Count; i++ {
		isMatching := false
		if config.MatchingAtEnd && i == config.Count-1 {
			isMatching = true
		}

		if isMatching {
			// Matching policy (catch-all)
			policies[i] = &policy.Policy{
				RuleID:    config.BaseRuleID + uint32(i),
				SrcIP:     "0.0.0.0/0",
				DstIP:     "0.0.0.0/0",
				SrcPort:   0,
				DstPort:   0,
				Protocol:  config.Protocol,
				Action:    "allow",
				Direction: config.Direction,
				Priority:  config.BasePriority + uint16(config.Count),
			}
		} else {
			// Non-matching policy
			policies[i] = &policy.Policy{
				RuleID:    config.BaseRuleID + uint32(i),
				SrcIP:     fmt.Sprintf("192.168.%d.0/24", i%256),
				DstIP:     "10.0.0.0/8",
				SrcPort:   0,
				DstPort:   uint16(8000 + i),
				Protocol:  config.Protocol,
				Action:    config.Action,
				Direction: config.Direction,
				Priority:  config.BasePriority + uint16(i),
			}
		}
	}

	return policies
}

// GenerateBestCasePolicies generates policies where the first policy matches
// This tests the best-case scenario with early match
func GenerateBestCasePolicies(count int) []*policy.Policy {
	if count <= 0 {
		return nil
	}

	policies := make([]*policy.Policy, count)

	// First policy: matching policy
	policies[0] = &policy.Policy{
		RuleID:    10000,
		SrcIP:     "0.0.0.0/0",
		DstIP:     "0.0.0.0/0",
		SrcPort:   0,
		DstPort:   0,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  uint16(200),
	}

	// Rest: non-matching policies
	for i := 1; i < count; i++ {
		policies[i] = &policy.Policy{
			RuleID:    uint32(10000 + i),
			SrcIP:     fmt.Sprintf("192.168.%d.0/24", i%256),
			DstIP:     "10.0.0.0/8",
			SrcPort:   0,
			DstPort:   uint16(8000 + i),
			Protocol:  "tcp",
			Action:    "deny",
			Direction: "ingress",
			Priority:  uint16(100),
		}
	}

	return policies
}

// GenerateAverageCasePolicies generates policies where match occurs in the middle
// This tests the average-case scenario
func GenerateAverageCasePolicies(count int) []*policy.Policy {
	if count <= 0 {
		return nil
	}

	policies := make([]*policy.Policy, count)
	matchIndex := count / 2

	for i := 0; i < count; i++ {
		if i == matchIndex {
			// Matching policy in the middle
			policies[i] = &policy.Policy{
				RuleID:    uint32(10000 + i),
				SrcIP:     "0.0.0.0/0",
				DstIP:     "0.0.0.0/0",
				SrcPort:   0,
				DstPort:   0,
				Protocol:  "tcp",
				Action:    "allow",
				Direction: "ingress",
				Priority:  uint16(200),
			}
		} else {
			// Non-matching policy
			policies[i] = &policy.Policy{
				RuleID:    uint32(10000 + i),
				SrcIP:     fmt.Sprintf("192.168.%d.0/24", i%256),
				DstIP:     "10.0.0.0/8",
				SrcPort:   0,
				DstPort:   uint16(8000 + i),
				Protocol:  "tcp",
				Action:    "deny",
				Direction: "ingress",
				Priority:  uint16(100),
			}
		}
	}

	return policies
}

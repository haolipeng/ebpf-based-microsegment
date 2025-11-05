// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"net"
	"os"
	"testing"

	"github.com/ebpf-microsegment/src/agent/pkg/groups"
	"github.com/ebpf-microsegment/src/agent/pkg/workload"
)

// TestPolicyCompiler_Basic tests basic compiler functionality
func TestPolicyCompiler_Basic(t *testing.T) {
	// Create temporary databases
	policyDB := "/tmp/test_compiler_policy.db"
	workloadDB := "/tmp/test_compiler_workload.db"
	groupDB := "/tmp/test_compiler_group.db"

	os.Remove(policyDB)
	os.Remove(workloadDB)
	os.Remove(groupDB)

	defer os.Remove(policyDB)
	defer os.Remove(workloadDB)
	defer os.Remove(groupDB)

	// Create storages
	policyStorage, err := NewSQLiteStorage(policyDB)
	if err != nil {
		t.Fatalf("Failed to create policy storage: %v", err)
	}
	defer policyStorage.Close()

	workloadStorage, err := workload.NewSQLiteStorage(workloadDB)
	if err != nil {
		t.Fatalf("Failed to create workload storage: %v", err)
	}
	defer workloadStorage.Close()

	groupStorage, err := groups.NewSQLiteStorage(groupDB)
	if err != nil {
		t.Fatalf("Failed to create group storage: %v", err)
	}
	defer groupStorage.Close()

	// Create managers
	workloadMgr := workload.NewManager(workloadStorage)
	groupMgr := groups.NewGroupManager(groupStorage, workloadMgr)
	compiler := NewPolicyCompiler(policyStorage, groupMgr)

	// Create test workloads
	web1 := &workload.Workload{
		ID:     "web-1",
		Name:   "web-1",
		IPs:    []net.IP{net.ParseIP("10.0.1.10")},
		Labels: map[string]string{"role": "web", "app": "frontend"},
	}

	db1 := &workload.Workload{
		ID:     "db-1",
		Name:   "db-1",
		IPs:    []net.IP{net.ParseIP("10.0.2.20")},
		Labels: map[string]string{"role": "db", "app": "backend"},
	}

	// Add workloads
	if err := workloadMgr.AddWorkload(web1); err != nil {
		t.Fatalf("Failed to add web workload: %v", err)
	}
	if err := workloadMgr.AddWorkload(db1); err != nil {
		t.Fatalf("Failed to add db workload: %v", err)
	}

	// Create groups
	err = groupMgr.CreateGroup("web-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "web"},
	})
	if err != nil {
		t.Fatalf("Failed to create web-servers group: %v", err)
	}

	err = groupMgr.CreateGroup("db-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "db"},
	})
	if err != nil {
		t.Fatalf("Failed to create db-servers group: %v", err)
	}

	// Create policy rule
	rule := &PolicyRule{
		Name:        "web-to-db",
		Description: "Allow web to access database",
		FromGroup:   "web-servers",
		ToGroup:     "db-servers",
		Ports: []PortRange{
			{Start: 3306, End: 3306, Protocol: "tcp"},
		},
		Action:   "allow",
		Priority: 100,
		Enabled:  true,
	}

	err = policyStorage.CreatePolicyRule(rule)
	if err != nil {
		t.Fatalf("Failed to create policy rule: %v", err)
	}

	// Test: Compile the rule
	result, err := compiler.CompilePolicyRule(rule.ID)
	if err != nil {
		t.Fatalf("CompilePolicyRule failed: %v", err)
	}

	// Verify basic compilation results
	if result.CompiledCount != 1 {
		t.Errorf("Expected 1 compiled policy, got %d", result.CompiledCount)
	}

	if result.FromGroupSize != 1 {
		t.Errorf("Expected FromGroupSize=1, got %d", result.FromGroupSize)
	}

	if result.ToGroupSize != 1 {
		t.Errorf("Expected ToGroupSize=1, got %d", result.ToGroupSize)
	}

	// Verify compiled policies were saved
	policies, err := policyStorage.ListCompiledPoliciesForRule(rule.ID)
	if err != nil {
		t.Fatalf("Failed to list compiled policies: %v", err)
	}

	if len(policies) != 1 {
		t.Errorf("Expected 1 saved policy, got %d", len(policies))
	}

	// Verify policy details
	if len(policies) > 0 {
		cp := policies[0]
		if cp.SrcIP != "10.0.1.10" {
			t.Errorf("Expected SrcIP=10.0.1.10, got %s", cp.SrcIP)
		}
		if cp.DstIP != "10.0.2.20" {
			t.Errorf("Expected DstIP=10.0.2.20, got %s", cp.DstIP)
		}
		if cp.DstPort != 3306 {
			t.Errorf("Expected DstPort=3306, got %d", cp.DstPort)
		}
		if cp.FromWorkloadID != "web-1" {
			t.Errorf("Expected FromWorkloadID=web-1, got %s", cp.FromWorkloadID)
		}
		if cp.ToWorkloadID != "db-1" {
			t.Errorf("Expected ToWorkloadID=db-1, got %s", cp.ToWorkloadID)
		}
	}

	t.Logf("✅ Basic compilation test passed: %s", result.String())
}

// TestPolicyCompiler_IDAllocation tests unique ID allocation
func TestPolicyCompiler_IDAllocation(t *testing.T) {
	policyDB := "/tmp/test_compiler_id.db"
	workloadDB := "/tmp/test_compiler_id_workload.db"
	groupDB := "/tmp/test_compiler_id_group.db"

	os.Remove(policyDB)
	os.Remove(workloadDB)
	os.Remove(groupDB)

	defer os.Remove(policyDB)
	defer os.Remove(workloadDB)
	defer os.Remove(groupDB)

	policyStorage, _ := NewSQLiteStorage(policyDB)
	defer policyStorage.Close()

	workloadStorage, _ := workload.NewSQLiteStorage(workloadDB)
	defer workloadStorage.Close()

	groupStorage, _ := groups.NewSQLiteStorage(groupDB)
	defer groupStorage.Close()

	workloadMgr := workload.NewManager(workloadStorage)
	groupMgr := groups.NewGroupManager(groupStorage, workloadMgr)
	compiler := NewPolicyCompiler(policyStorage, groupMgr)

	// Test ID allocation
	id1 := compiler.allocateCompiledRuleID()
	id2 := compiler.allocateCompiledRuleID()
	id3 := compiler.allocateCompiledRuleID()

	if id2 != id1+1 {
		t.Errorf("Expected sequential IDs, got %d and %d", id1, id2)
	}

	if id3 != id2+1 {
		t.Errorf("Expected sequential IDs, got %d and %d", id2, id3)
	}

	t.Logf("✅ ID allocation test passed: %d, %d, %d", id1, id2, id3)
}

// TestPolicyCompiler_InvalidateCompiledPolicies tests policy deletion
func TestPolicyCompiler_InvalidateCompiledPolicies(t *testing.T) {
	policyDB := "/tmp/test_compiler_invalidate.db"
	workloadDB := "/tmp/test_compiler_invalidate_workload.db"
	groupDB := "/tmp/test_compiler_invalidate_group.db"

	os.Remove(policyDB)
	os.Remove(workloadDB)
	os.Remove(groupDB)

	defer os.Remove(policyDB)
	defer os.Remove(workloadDB)
	defer os.Remove(groupDB)

	policyStorage, _ := NewSQLiteStorage(policyDB)
	defer policyStorage.Close()

	workloadStorage, _ := workload.NewSQLiteStorage(workloadDB)
	defer workloadStorage.Close()

	groupStorage, _ := groups.NewSQLiteStorage(groupDB)
	defer groupStorage.Close()

	workloadMgr := workload.NewManager(workloadStorage)
	groupMgr := groups.NewGroupManager(groupStorage, workloadMgr)
	compiler := NewPolicyCompiler(policyStorage, groupMgr)

	// Setup workloads and groups
	web := &workload.Workload{
		ID:     "web-1",
		Name:   "web-1",
		IPs:    []net.IP{net.ParseIP("10.0.1.10")},
		Labels: map[string]string{"role": "web"},
	}
	workloadMgr.AddWorkload(web)

	groupMgr.CreateGroup("web-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "web"},
	})

	// Create and compile rule
	rule := &PolicyRule{
		Name:      "test-rule",
		FromGroup: "web-servers",
		ToGroup:   "web-servers",
		Ports:     []PortRange{{Start: 80, End: 80, Protocol: "tcp"}},
		Action:    "allow",
		Priority:  100,
		Enabled:   true,
	}

	policyStorage.CreatePolicyRule(rule)
	compiler.CompilePolicyRule(rule.ID)

	// Verify policies exist
	before, _ := policyStorage.ListCompiledPoliciesForRule(rule.ID)
	if len(before) == 0 {
		t.Fatal("Expected compiled policies before invalidation")
	}

	// Invalidate
	err := compiler.InvalidateCompiledPolicies(rule.ID)
	if err != nil {
		t.Fatalf("InvalidateCompiledPolicies failed: %v", err)
	}

	// Verify policies are deleted
	after, _ := policyStorage.ListCompiledPoliciesForRule(rule.ID)
	if len(after) != 0 {
		t.Errorf("Expected 0 policies after invalidation, got %d", len(after))
	}

	t.Log("✅ Invalidate test passed")
}

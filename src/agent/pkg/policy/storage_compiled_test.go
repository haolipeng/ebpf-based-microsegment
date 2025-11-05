// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"os"
	"testing"
	"time"
)

// setupTestStorage creates a temporary SQLite storage for testing
func setupTestStorage(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()

	// Create temporary database file
	dbPath := "/tmp/test_policy_compiled_" + t.Name() + ".db"

	// Remove if exists
	os.Remove(dbPath)

	storage, err := NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}

	cleanup := func() {
		storage.Close()
		os.Remove(dbPath)
	}

	return storage, cleanup
}

func TestSaveAndGetCompiledPolicy(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a compiled policy
	now := time.Now().Truncate(time.Second)
	cp := &CompiledPolicy{
		Policy: Policy{
			RuleID:   1001,
			SrcIP:    "10.0.1.100",
			DstIP:    "10.0.2.200",
			SrcPort:  0,
			DstPort:  443,
			Protocol: "tcp",
			Action:   "allow",
			Priority: 100,
		},
		SourceRuleID:    1,
		FromGroup:       "web",
		ToGroup:         "db",
		FromWorkloadID:  "workload-web-1",
		ToWorkloadID:    "workload-db-1",
		CompilationTime: now,
		CompilerVersion: CompilerVersionV1,
	}

	// Save the compiled policy
	err := storage.SaveCompiledPolicy(cp)
	if err != nil {
		t.Fatalf("SaveCompiledPolicy failed: %v", err)
	}

	// Retrieve the compiled policy
	retrieved, err := storage.GetCompiledPolicy(1001)
	if err != nil {
		t.Fatalf("GetCompiledPolicy failed: %v", err)
	}

	// Verify fields
	if retrieved.RuleID != cp.RuleID {
		t.Errorf("RuleID mismatch: got %d, want %d", retrieved.RuleID, cp.RuleID)
	}
	if retrieved.SourceRuleID != cp.SourceRuleID {
		t.Errorf("SourceRuleID mismatch: got %d, want %d", retrieved.SourceRuleID, cp.SourceRuleID)
	}
	if retrieved.FromGroup != cp.FromGroup {
		t.Errorf("FromGroup mismatch: got %s, want %s", retrieved.FromGroup, cp.FromGroup)
	}
	if retrieved.ToGroup != cp.ToGroup {
		t.Errorf("ToGroup mismatch: got %s, want %s", retrieved.ToGroup, cp.ToGroup)
	}
	if retrieved.FromWorkloadID != cp.FromWorkloadID {
		t.Errorf("FromWorkloadID mismatch: got %s, want %s", retrieved.FromWorkloadID, cp.FromWorkloadID)
	}
	if retrieved.ToWorkloadID != cp.ToWorkloadID {
		t.Errorf("ToWorkloadID mismatch: got %s, want %s", retrieved.ToWorkloadID, cp.ToWorkloadID)
	}
	if retrieved.CompilerVersion != cp.CompilerVersion {
		t.Errorf("CompilerVersion mismatch: got %s, want %s", retrieved.CompilerVersion, cp.CompilerVersion)
	}
	if !retrieved.CompilationTime.Equal(now) {
		t.Errorf("CompilationTime mismatch: got %v, want %v", retrieved.CompilationTime, now)
	}
}

func TestGetCompiledPolicy_NotFound(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	_, err := storage.GetCompiledPolicy(9999)
	if err == nil {
		t.Error("Expected error for non-existent policy, got nil")
	}
}

func TestListCompiledPolicies(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	now := time.Now().Truncate(time.Second)

	// Create multiple compiled policies
	policies := []*CompiledPolicy{
		{
			Policy: Policy{
				RuleID: 1001, SrcIP: "10.0.1.1", DstIP: "10.0.2.1",
				SrcPort: 0, DstPort: 443, Protocol: "tcp", Action: "allow", Priority: 100,
			},
			SourceRuleID: 1, FromGroup: "web", ToGroup: "db",
			FromWorkloadID: "w1", ToWorkloadID: "d1",
			CompilationTime: now, CompilerVersion: CompilerVersionV1,
		},
		{
			Policy: Policy{
				RuleID: 1002, SrcIP: "10.0.1.2", DstIP: "10.0.2.2",
				SrcPort: 0, DstPort: 443, Protocol: "tcp", Action: "allow", Priority: 100,
			},
			SourceRuleID: 1, FromGroup: "web", ToGroup: "db",
			FromWorkloadID: "w2", ToWorkloadID: "d2",
			CompilationTime: now, CompilerVersion: CompilerVersionV1,
		},
		{
			Policy: Policy{
				RuleID: 2001, SrcIP: "10.0.3.1", DstIP: "10.0.4.1",
				SrcPort: 0, DstPort: 6379, Protocol: "tcp", Action: "allow", Priority: 90,
			},
			SourceRuleID: 2, FromGroup: "app", ToGroup: "cache",
			FromWorkloadID: "a1", ToWorkloadID: "c1",
			CompilationTime: now, CompilerVersion: CompilerVersionV1,
		},
	}

	// Save all policies
	for _, p := range policies {
		if err := storage.SaveCompiledPolicy(p); err != nil {
			t.Fatalf("SaveCompiledPolicy failed: %v", err)
		}
	}

	// List all compiled policies
	retrieved, err := storage.ListCompiledPolicies()
	if err != nil {
		t.Fatalf("ListCompiledPolicies failed: %v", err)
	}

	if len(retrieved) != 3 {
		t.Errorf("Expected 3 policies, got %d", len(retrieved))
	}
}

func TestListCompiledPoliciesForRule(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	now := time.Now().Truncate(time.Second)

	// Create multiple compiled policies from the same source rule
	sourceRuleID := uint32(10)
	policies := []*CompiledPolicy{
		{
			Policy: Policy{
				RuleID: 1001, SrcIP: "10.0.1.1", DstIP: "10.0.2.1",
				SrcPort: 0, DstPort: 443, Protocol: "tcp", Action: "allow", Priority: 100,
			},
			SourceRuleID: sourceRuleID, FromGroup: "web", ToGroup: "db",
			FromWorkloadID: "w1", ToWorkloadID: "d1",
			CompilationTime: now, CompilerVersion: CompilerVersionV1,
		},
		{
			Policy: Policy{
				RuleID: 1002, SrcIP: "10.0.1.2", DstIP: "10.0.2.2",
				SrcPort: 0, DstPort: 443, Protocol: "tcp", Action: "allow", Priority: 100,
			},
			SourceRuleID: sourceRuleID, FromGroup: "web", ToGroup: "db",
			FromWorkloadID: "w2", ToWorkloadID: "d2",
			CompilationTime: now, CompilerVersion: CompilerVersionV1,
		},
		{
			Policy: Policy{
				RuleID: 2001, SrcIP: "10.0.3.1", DstIP: "10.0.4.1",
				SrcPort: 0, DstPort: 6379, Protocol: "tcp", Action: "allow", Priority: 90,
			},
			SourceRuleID: 99, FromGroup: "app", ToGroup: "cache", // Different source rule
			FromWorkloadID: "a1", ToWorkloadID: "c1",
			CompilationTime: now, CompilerVersion: CompilerVersionV1,
		},
	}

	// Save all policies
	for _, p := range policies {
		if err := storage.SaveCompiledPolicy(p); err != nil {
			t.Fatalf("SaveCompiledPolicy failed: %v", err)
		}
	}

	// List compiled policies for source rule 10
	retrieved, err := storage.ListCompiledPoliciesForRule(sourceRuleID)
	if err != nil {
		t.Fatalf("ListCompiledPoliciesForRule failed: %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("Expected 2 policies for rule %d, got %d", sourceRuleID, len(retrieved))
	}

	// Verify all policies have the correct source rule ID
	for _, p := range retrieved {
		if p.SourceRuleID != sourceRuleID {
			t.Errorf("Policy %d has wrong SourceRuleID: got %d, want %d",
				p.RuleID, p.SourceRuleID, sourceRuleID)
		}
	}
}

func TestGetPolicySource(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a source policy rule
	sourceRule := &PolicyRule{
		Name:        "web-to-db",
		Description: "Allow web to access database",
		FromGroup:   "web",
		ToGroup:     "db",
		Ports: []PortRange{
			{Start: 443, End: 443, Protocol: "tcp"},
		},
		Action:   "allow",
		Priority: 100,
		Enabled:  true,
	}

	err := storage.CreatePolicyRule(sourceRule)
	if err != nil {
		t.Fatalf("CreatePolicyRule failed: %v", err)
	}

	// Create a compiled policy
	now := time.Now().Truncate(time.Second)
	compiledRuleID := uint32(2001)
	cp := &CompiledPolicy{
		Policy: Policy{
			RuleID: compiledRuleID, SrcIP: "10.0.1.1", DstIP: "10.0.2.1",
			SrcPort: 0, DstPort: 443, Protocol: "tcp", Action: "allow", Priority: 100,
		},
		SourceRuleID: sourceRule.ID, FromGroup: "web", ToGroup: "db",
		FromWorkloadID: "w1", ToWorkloadID: "d1",
		CompilationTime: now, CompilerVersion: CompilerVersionV1,
	}

	err = storage.SaveCompiledPolicy(cp)
	if err != nil {
		t.Fatalf("SaveCompiledPolicy failed: %v", err)
	}

	// Get the source rule for the compiled policy
	retrieved, err := storage.GetPolicySource(compiledRuleID)
	if err != nil {
		t.Fatalf("GetPolicySource failed: %v", err)
	}

	// Verify the source rule
	if retrieved.ID != sourceRule.ID {
		t.Errorf("ID mismatch: got %d, want %d", retrieved.ID, sourceRule.ID)
	}
	if retrieved.Name != sourceRule.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, sourceRule.Name)
	}
	if retrieved.FromGroup != sourceRule.FromGroup {
		t.Errorf("FromGroup mismatch: got %s, want %s", retrieved.FromGroup, sourceRule.FromGroup)
	}
	if retrieved.ToGroup != sourceRule.ToGroup {
		t.Errorf("ToGroup mismatch: got %s, want %s", retrieved.ToGroup, sourceRule.ToGroup)
	}
}

func TestGetPolicySource_NotFound(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	_, err := storage.GetPolicySource(9999)
	if err == nil {
		t.Error("Expected error for non-existent compiled policy, got nil")
	}
}

func TestDeleteCompiledPoliciesForRule(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	now := time.Now().Truncate(time.Second)
	sourceRuleID := uint32(10)

	// Create multiple compiled policies from the same source rule
	policies := []*CompiledPolicy{
		{
			Policy: Policy{
				RuleID: 1001, SrcIP: "10.0.1.1", DstIP: "10.0.2.1",
				SrcPort: 0, DstPort: 443, Protocol: "tcp", Action: "allow", Priority: 100,
			},
			SourceRuleID: sourceRuleID, FromGroup: "web", ToGroup: "db",
			FromWorkloadID: "w1", ToWorkloadID: "d1",
			CompilationTime: now, CompilerVersion: CompilerVersionV1,
		},
		{
			Policy: Policy{
				RuleID: 1002, SrcIP: "10.0.1.2", DstIP: "10.0.2.2",
				SrcPort: 0, DstPort: 443, Protocol: "tcp", Action: "allow", Priority: 100,
			},
			SourceRuleID: sourceRuleID, FromGroup: "web", ToGroup: "db",
			FromWorkloadID: "w2", ToWorkloadID: "d2",
			CompilationTime: now, CompilerVersion: CompilerVersionV1,
		},
		{
			Policy: Policy{
				RuleID: 2001, SrcIP: "10.0.3.1", DstIP: "10.0.4.1",
				SrcPort: 0, DstPort: 6379, Protocol: "tcp", Action: "allow", Priority: 90,
			},
			SourceRuleID: 99, FromGroup: "app", ToGroup: "cache", // Different source rule
			FromWorkloadID: "a1", ToWorkloadID: "c1",
			CompilationTime: now, CompilerVersion: CompilerVersionV1,
		},
	}

	// Save all policies
	for _, p := range policies {
		if err := storage.SaveCompiledPolicy(p); err != nil {
			t.Fatalf("SaveCompiledPolicy failed: %v", err)
		}
	}

	// Delete compiled policies for source rule 10
	err := storage.DeleteCompiledPoliciesForRule(sourceRuleID)
	if err != nil {
		t.Fatalf("DeleteCompiledPoliciesForRule failed: %v", err)
	}

	// Verify that policies for rule 10 are deleted
	remaining, err := storage.ListCompiledPoliciesForRule(sourceRuleID)
	if err != nil {
		t.Fatalf("ListCompiledPoliciesForRule failed: %v", err)
	}

	if len(remaining) != 0 {
		t.Errorf("Expected 0 policies after deletion, got %d", len(remaining))
	}

	// Verify that policy from rule 99 still exists
	rule99Policies, err := storage.ListCompiledPoliciesForRule(99)
	if err != nil {
		t.Fatalf("ListCompiledPoliciesForRule failed: %v", err)
	}

	if len(rule99Policies) != 1 {
		t.Errorf("Expected 1 policy for rule 99, got %d", len(rule99Policies))
	}

	// Verify that the base policies are also deleted
	_, err = storage.GetCompiledPolicy(1001)
	if err == nil {
		t.Error("Expected error for deleted policy 1001, got nil")
	}

	_, err = storage.GetCompiledPolicy(1002)
	if err == nil {
		t.Error("Expected error for deleted policy 1002, got nil")
	}
}

func TestSaveCompiledPolicy_InvalidPolicy(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create an invalid compiled policy (missing required fields)
	cp := &CompiledPolicy{
		Policy: Policy{
			RuleID: 1001, SrcIP: "10.0.1.1", DstIP: "10.0.2.1",
			SrcPort: 0, DstPort: 443, Protocol: "tcp", Action: "allow", Priority: 100,
		},
		SourceRuleID: 0, // Invalid: must be > 0
		FromGroup:    "web",
		ToGroup:      "db",
	}

	err := storage.SaveCompiledPolicy(cp)
	if err == nil {
		t.Error("Expected validation error for invalid policy, got nil")
	}
}

func TestCompiledPolicyUpdate(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	now := time.Now().Truncate(time.Second)

	// Create initial compiled policy
	cp := &CompiledPolicy{
		Policy: Policy{
			RuleID: 1001, SrcIP: "10.0.1.1", DstIP: "10.0.2.1",
			SrcPort: 0, DstPort: 443, Protocol: "tcp", Action: "allow", Priority: 100,
		},
		SourceRuleID: 1, FromGroup: "web", ToGroup: "db",
		FromWorkloadID: "w1", ToWorkloadID: "d1",
		CompilationTime: now, CompilerVersion: CompilerVersionV1,
	}

	err := storage.SaveCompiledPolicy(cp)
	if err != nil {
		t.Fatalf("SaveCompiledPolicy failed: %v", err)
	}

	// Update the policy with new values
	cp.DstIP = "10.0.2.100"
	cp.Priority = 200
	cp.FromWorkloadID = "w1-updated"

	err = storage.SaveCompiledPolicy(cp)
	if err != nil {
		t.Fatalf("SaveCompiledPolicy (update) failed: %v", err)
	}

	// Retrieve and verify the update
	retrieved, err := storage.GetCompiledPolicy(1001)
	if err != nil {
		t.Fatalf("GetCompiledPolicy failed: %v", err)
	}

	if retrieved.DstIP != "10.0.2.100" {
		t.Errorf("DstIP not updated: got %s, want 10.0.2.100", retrieved.DstIP)
	}
	if retrieved.Priority != 200 {
		t.Errorf("Priority not updated: got %d, want 200", retrieved.Priority)
	}
	if retrieved.FromWorkloadID != "w1-updated" {
		t.Errorf("FromWorkloadID not updated: got %s, want w1-updated", retrieved.FromWorkloadID)
	}
}

// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"os"
	"testing"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/groups"
	"github.com/ebpf-microsegment/src/agent/pkg/workload"
)

// setupCompilerTest creates test fixtures for compiler tests
func setupCompilerTest(t *testing.T) (*SQLiteStorage, *groups.GroupManager, *PolicyCompiler, func()) {
	t.Helper()

	// Create temporary database
	dbPath := "/tmp/test_compiler_" + t.Name() + ".db"
	os.Remove(dbPath)

	storage, err := NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create workload storage
	workloadDBPath := "/tmp/test_workload_" + t.Name() + ".db"
	os.Remove(workloadDBPath)

	workloadStorage, err := workload.NewSQLiteStorage(workloadDBPath)
	if err != nil {
		storage.Close()
		t.Fatalf("Failed to create workload storage: %v", err)
	}

	// Create group manager
	groupMgr := groups.NewGroupManager(workloadStorage)

	// Create compiler
	compiler := NewPolicyCompiler(storage, groupMgr)

	cleanup := func() {
		storage.Close()
		workloadStorage.Close()
		os.Remove(dbPath)
		os.Remove(workloadDBPath)
	}

	return storage, groupMgr, compiler, cleanup
}

func TestCompilePolicyRule_1x1(t *testing.T) {
	storage, groupMgr, compiler, cleanup := setupCompilerTest(t)
	defer cleanup()

	// Create workloads
	web := &workload.Workload{
		ID:     "web-1",
		IP:     "10.0.1.10",
		Labels: map[string]string{"role": "web"},
	}
	db := &workload.Workload{
		ID:     "db-1",
		IP:     "10.0.2.20",
		Labels: map[string]string{"role": "db"},
	}

	groupMgr.AddWorkload(web)
	groupMgr.AddWorkload(db)

	// Create groups
	err := groupMgr.CreateGroup("web-servers", groups.LabelSelector{
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

	err = storage.CreatePolicyRule(rule)
	if err != nil {
		t.Fatalf("Failed to create policy rule: %v", err)
	}

	// Compile the rule
	result, err := compiler.CompilePolicyRule(rule.ID)
	if err != nil {
		t.Fatalf("CompilePolicyRule failed: %v", err)
	}

	// Verify result
	if result.CompiledCount != 1 {
		t.Errorf("Expected 1 compiled policy, got %d", result.CompiledCount)
	}

	if result.FromGroupSize != 1 {
		t.Errorf("Expected FromGroupSize=1, got %d", result.FromGroupSize)
	}

	if result.ToGroupSize != 1 {
		t.Errorf("Expected ToGroupSize=1, got %d", result.ToGroupSize)
	}

	if result.ExpansionRatio != 1.0 {
		t.Errorf("Expected ExpansionRatio=1.0, got %f", result.ExpansionRatio)
	}

	// Verify compiled policy
	if len(result.CompiledPolicies) != 1 {
		t.Fatalf("Expected 1 compiled policy in result, got %d", len(result.CompiledPolicies))
	}

	cp := result.CompiledPolicies[0]
	if cp.SrcIP != "10.0.1.10" {
		t.Errorf("Expected SrcIP=10.0.1.10, got %s", cp.SrcIP)
	}
	if cp.DstIP != "10.0.2.20" {
		t.Errorf("Expected DstIP=10.0.2.20, got %s", cp.DstIP)
	}
	if cp.DstPort != 3306 {
		t.Errorf("Expected DstPort=3306, got %d", cp.DstPort)
	}
	if cp.Protocol != "tcp" {
		t.Errorf("Expected Protocol=tcp, got %s", cp.Protocol)
	}
	if cp.SourceRuleID != rule.ID {
		t.Errorf("Expected SourceRuleID=%d, got %d", rule.ID, cp.SourceRuleID)
	}
}

func TestCompilePolicyRule_NxM(t *testing.T) {
	storage, groupMgr, compiler, cleanup := setupCompilerTest(t)
	defer cleanup()

	// Create 10 web workloads
	for i := 1; i <= 10; i++ {
		web := &workload.Workload{
			ID:     "web-" + string(rune('0'+i)),
			IP:     "10.0.1." + string(rune('0'+i)),
			Labels: map[string]string{"role": "web"},
		}
		groupMgr.AddWorkload(web)
	}

	// Create 2 db workloads
	for i := 1; i <= 2; i++ {
		db := &workload.Workload{
			ID:     "db-" + string(rune('0'+i)),
			IP:     "10.0.2." + string(rune('0'+i)),
			Labels: map[string]string{"role": "db"},
		}
		groupMgr.AddWorkload(db)
	}

	// Create groups
	err := groupMgr.CreateGroup("web-servers", groups.LabelSelector{
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
		Name:      "web-to-db",
		FromGroup: "web-servers",
		ToGroup:   "db-servers",
		Ports: []PortRange{
			{Start: 3306, End: 3306, Protocol: "tcp"},
		},
		Action:   "allow",
		Priority: 100,
		Enabled:  true,
	}

	err = storage.CreatePolicyRule(rule)
	if err != nil {
		t.Fatalf("Failed to create policy rule: %v", err)
	}

	// Compile the rule
	result, err := compiler.CompilePolicyRule(rule.ID)
	if err != nil {
		t.Fatalf("CompilePolicyRule failed: %v", err)
	}

	// Verify result: 10 web × 2 db = 20 policies
	expectedCount := 20
	if result.CompiledCount != expectedCount {
		t.Errorf("Expected %d compiled policies, got %d", expectedCount, result.CompiledCount)
	}

	if result.FromGroupSize != 10 {
		t.Errorf("Expected FromGroupSize=10, got %d", result.FromGroupSize)
	}

	if result.ToGroupSize != 2 {
		t.Errorf("Expected ToGroupSize=2, got %d", result.ToGroupSize)
	}
}

func TestCompilePolicyRule_MultiplePortRanges(t *testing.T) {
	storage, groupMgr, compiler, cleanup := setupCompilerTest(t)
	defer cleanup()

	// Create workloads
	web := &workload.Workload{
		ID:     "web-1",
		IP:     "10.0.1.10",
		Labels: map[string]string{"role": "web"},
	}
	db := &workload.Workload{
		ID:     "db-1",
		IP:     "10.0.2.20",
		Labels: map[string]string{"role": "db"},
	}

	groupMgr.AddWorkload(web)
	groupMgr.AddWorkload(db)

	// Create groups
	groupMgr.CreateGroup("web-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "web"},
	})
	groupMgr.CreateGroup("db-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "db"},
	})

	// Create policy rule with multiple port ranges
	rule := &PolicyRule{
		Name:      "web-to-db-multi-port",
		FromGroup: "web-servers",
		ToGroup:   "db-servers",
		Ports: []PortRange{
			{Start: 3306, End: 3306, Protocol: "tcp"}, // MySQL
			{Start: 5432, End: 5432, Protocol: "tcp"}, // PostgreSQL
			{Start: 6379, End: 6379, Protocol: "tcp"}, // Redis
		},
		Action:   "allow",
		Priority: 100,
		Enabled:  true,
	}

	err := storage.CreatePolicyRule(rule)
	if err != nil {
		t.Fatalf("Failed to create policy rule: %v", err)
	}

	// Compile the rule
	result, err := compiler.CompilePolicyRule(rule.ID)
	if err != nil {
		t.Fatalf("CompilePolicyRule failed: %v", err)
	}

	// Verify result: 1 web × 1 db × 3 ports = 3 policies
	expectedCount := 3
	if result.CompiledCount != expectedCount {
		t.Errorf("Expected %d compiled policies, got %d", expectedCount, result.CompiledCount)
	}

	// Verify all ports are present
	ports := make(map[uint16]bool)
	for _, cp := range result.CompiledPolicies {
		ports[cp.DstPort] = true
	}

	expectedPorts := []uint16{3306, 5432, 6379}
	for _, port := range expectedPorts {
		if !ports[port] {
			t.Errorf("Expected port %d in compiled policies", port)
		}
	}
}

func TestCompilePolicyRule_PortRange(t *testing.T) {
	storage, groupMgr, compiler, cleanup := setupCompilerTest(t)
	defer cleanup()

	// Create workloads
	web := &workload.Workload{
		ID:     "web-1",
		IP:     "10.0.1.10",
		Labels: map[string]string{"role": "web"},
	}
	app := &workload.Workload{
		ID:     "app-1",
		IP:     "10.0.2.20",
		Labels: map[string]string{"role": "app"},
	}

	groupMgr.AddWorkload(web)
	groupMgr.AddWorkload(app)

	// Create groups
	groupMgr.CreateGroup("web-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "web"},
	})
	groupMgr.CreateGroup("app-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "app"},
	})

	// Create policy rule with port range
	rule := &PolicyRule{
		Name:      "web-to-app-range",
		FromGroup: "web-servers",
		ToGroup:   "app-servers",
		Ports: []PortRange{
			{Start: 8080, End: 8082, Protocol: "tcp"}, // 3 ports
		},
		Action:   "allow",
		Priority: 100,
		Enabled:  true,
	}

	err := storage.CreatePolicyRule(rule)
	if err != nil {
		t.Fatalf("Failed to create policy rule: %v", err)
	}

	// Compile the rule
	result, err := compiler.CompilePolicyRule(rule.ID)
	if err != nil {
		t.Fatalf("CompilePolicyRule failed: %v", err)
	}

	// Verify result: 1 web × 1 app × 3 ports = 3 policies
	expectedCount := 3
	if result.CompiledCount != expectedCount {
		t.Errorf("Expected %d compiled policies, got %d", expectedCount, result.CompiledCount)
	}

	// Verify ports 8080, 8081, 8082
	ports := make(map[uint16]bool)
	for _, cp := range result.CompiledPolicies {
		ports[cp.DstPort] = true
	}

	for port := uint16(8080); port <= 8082; port++ {
		if !ports[port] {
			t.Errorf("Expected port %d in compiled policies", port)
		}
	}
}

func TestCompilePolicyRule_Traceability(t *testing.T) {
	storage, groupMgr, compiler, cleanup := setupCompilerTest(t)
	defer cleanup()

	// Create workloads
	web := &workload.Workload{
		ID:     "web-1",
		IP:     "10.0.1.10",
		Labels: map[string]string{"role": "web"},
	}
	db := &workload.Workload{
		ID:     "db-1",
		IP:     "10.0.2.20",
		Labels: map[string]string{"role": "db"},
	}

	groupMgr.AddWorkload(web)
	groupMgr.AddWorkload(db)

	// Create groups
	groupMgr.CreateGroup("web-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "web"},
	})
	groupMgr.CreateGroup("db-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "db"},
	})

	// Create policy rule
	rule := &PolicyRule{
		Name:      "web-to-db",
		FromGroup: "web-servers",
		ToGroup:   "db-servers",
		Ports: []PortRange{
			{Start: 3306, End: 3306, Protocol: "tcp"},
		},
		Action:   "allow",
		Priority: 100,
		Enabled:  true,
	}

	err := storage.CreatePolicyRule(rule)
	if err != nil {
		t.Fatalf("Failed to create policy rule: %v", err)
	}

	// Compile the rule
	result, err := compiler.CompilePolicyRule(rule.ID)
	if err != nil {
		t.Fatalf("CompilePolicyRule failed: %v", err)
	}

	// Verify traceability
	cp := result.CompiledPolicies[0]

	if cp.SourceRuleID != rule.ID {
		t.Errorf("SourceRuleID mismatch: got %d, want %d", cp.SourceRuleID, rule.ID)
	}

	if cp.FromGroup != "web-servers" {
		t.Errorf("FromGroup mismatch: got %s, want web-servers", cp.FromGroup)
	}

	if cp.ToGroup != "db-servers" {
		t.Errorf("ToGroup mismatch: got %s, want db-servers", cp.ToGroup)
	}

	if cp.FromWorkloadID != "web-1" {
		t.Errorf("FromWorkloadID mismatch: got %s, want web-1", cp.FromWorkloadID)
	}

	if cp.ToWorkloadID != "db-1" {
		t.Errorf("ToWorkloadID mismatch: got %s, want db-1", cp.ToWorkloadID)
	}

	if cp.CompilerVersion != CompilerVersionV1 {
		t.Errorf("CompilerVersion mismatch: got %s, want %s", cp.CompilerVersion, CompilerVersionV1)
	}

	if cp.CompilationTime.IsZero() {
		t.Error("CompilationTime should not be zero")
	}

	// Test reverse lookup
	sourceRule, err := storage.GetPolicySource(cp.RuleID)
	if err != nil {
		t.Fatalf("GetPolicySource failed: %v", err)
	}

	if sourceRule.ID != rule.ID {
		t.Errorf("Source rule ID mismatch: got %d, want %d", sourceRule.ID, rule.ID)
	}
}

func TestCompilePolicyRule_DisabledRule(t *testing.T) {
	storage, groupMgr, compiler, cleanup := setupCompilerTest(t)
	defer cleanup()

	// Create workloads and groups
	web := &workload.Workload{
		ID:     "web-1",
		IP:     "10.0.1.10",
		Labels: map[string]string{"role": "web"},
	}
	groupMgr.AddWorkload(web)
	groupMgr.CreateGroup("web-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "web"},
	})

	// Create disabled policy rule
	rule := &PolicyRule{
		Name:      "disabled-rule",
		FromGroup: "web-servers",
		ToGroup:   "web-servers",
		Ports:     []PortRange{{Start: 80, End: 80, Protocol: "tcp"}},
		Action:    "deny",
		Priority:  100,
		Enabled:   false, // Disabled
	}

	err := storage.CreatePolicyRule(rule)
	if err != nil {
		t.Fatalf("Failed to create policy rule: %v", err)
	}

	// Compile the rule
	result, err := compiler.CompilePolicyRule(rule.ID)
	if err != nil {
		t.Fatalf("CompilePolicyRule failed: %v", err)
	}

	// Should compile 0 policies for disabled rule
	if result.CompiledCount != 0 {
		t.Errorf("Expected 0 compiled policies for disabled rule, got %d", result.CompiledCount)
	}
}

func TestCompilePolicyRule_Warnings(t *testing.T) {
	storage, groupMgr, compiler, cleanup := setupCompilerTest(t)
	defer cleanup()

	// Create many workloads to trigger warning threshold
	// We need > 1000 compiled policies, so create 35×35 = 1225 policies
	for i := 1; i <= 35; i++ {
		web := &workload.Workload{
			ID:     "web-" + string(rune('A'+i)),
			IP:     "10.0.1." + string(rune('0'+i)),
			Labels: map[string]string{"role": "web"},
		}
		groupMgr.AddWorkload(web)

		db := &workload.Workload{
			ID:     "db-" + string(rune('A'+i)),
			IP:     "10.0.2." + string(rune('0'+i)),
			Labels: map[string]string{"role": "db"},
		}
		groupMgr.AddWorkload(db)
	}

	groupMgr.CreateGroup("web-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "web"},
	})
	groupMgr.CreateGroup("db-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "db"},
	})

	// Create policy rule
	rule := &PolicyRule{
		Name:      "large-expansion",
		FromGroup: "web-servers",
		ToGroup:   "db-servers",
		Ports:     []PortRange{{Start: 3306, End: 3306, Protocol: "tcp"}},
		Action:    "allow",
		Priority:  100,
		Enabled:   true,
	}

	err := storage.CreatePolicyRule(rule)
	if err != nil {
		t.Fatalf("Failed to create policy rule: %v", err)
	}

	// Compile the rule
	result, err := compiler.CompilePolicyRule(rule.ID)
	if err != nil {
		t.Fatalf("CompilePolicyRule failed: %v", err)
	}

	// Should have warnings for large expansion
	if !result.HasWarnings() {
		t.Error("Expected warnings for large expansion, got none")
	}

	if result.CompiledCount <= CompilationWarningThreshold {
		t.Errorf("Test setup error: expected > %d policies, got %d",
			CompilationWarningThreshold, result.CompiledCount)
	}
}

func TestInvalidateCompiledPolicies(t *testing.T) {
	storage, groupMgr, compiler, cleanup := setupCompilerTest(t)
	defer cleanup()

	// Create workloads and groups
	web := &workload.Workload{
		ID:     "web-1",
		IP:     "10.0.1.10",
		Labels: map[string]string{"role": "web"},
	}
	db := &workload.Workload{
		ID:     "db-1",
		IP:     "10.0.2.20",
		Labels: map[string]string{"role": "db"},
	}

	groupMgr.AddWorkload(web)
	groupMgr.AddWorkload(db)

	groupMgr.CreateGroup("web-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "web"},
	})
	groupMgr.CreateGroup("db-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "db"},
	})

	// Create and compile policy rule
	rule := &PolicyRule{
		Name:      "web-to-db",
		FromGroup: "web-servers",
		ToGroup:   "db-servers",
		Ports:     []PortRange{{Start: 3306, End: 3306, Protocol: "tcp"}},
		Action:    "allow",
		Priority:  100,
		Enabled:   true,
	}

	storage.CreatePolicyRule(rule)
	compiler.CompilePolicyRule(rule.ID)

	// Verify policies exist
	policies, err := storage.ListCompiledPoliciesForRule(rule.ID)
	if err != nil {
		t.Fatalf("ListCompiledPoliciesForRule failed: %v", err)
	}
	if len(policies) == 0 {
		t.Fatal("Expected compiled policies, got none")
	}

	// Invalidate compiled policies
	err = compiler.InvalidateCompiledPolicies(rule.ID)
	if err != nil {
		t.Fatalf("InvalidateCompiledPolicies failed: %v", err)
	}

	// Verify policies are deleted
	policies, err = storage.ListCompiledPoliciesForRule(rule.ID)
	if err != nil {
		t.Fatalf("ListCompiledPoliciesForRule failed: %v", err)
	}
	if len(policies) != 0 {
		t.Errorf("Expected 0 policies after invalidation, got %d", len(policies))
	}
}

func TestCompileAllPolicies(t *testing.T) {
	storage, groupMgr, compiler, cleanup := setupCompilerTest(t)
	defer cleanup()

	// Create workloads
	web := &workload.Workload{
		ID:     "web-1",
		IP:     "10.0.1.10",
		Labels: map[string]string{"role": "web"},
	}
	db := &workload.Workload{
		ID:     "db-1",
		IP:     "10.0.2.20",
		Labels: map[string]string{"role": "db"},
	}

	groupMgr.AddWorkload(web)
	groupMgr.AddWorkload(db)

	groupMgr.CreateGroup("web-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "web"},
	})
	groupMgr.CreateGroup("db-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "db"},
	})

	// Create multiple policy rules
	rule1 := &PolicyRule{
		Name:      "web-to-db",
		FromGroup: "web-servers",
		ToGroup:   "db-servers",
		Ports:     []PortRange{{Start: 3306, End: 3306, Protocol: "tcp"}},
		Action:    "allow",
		Priority:  100,
		Enabled:   true,
	}

	rule2 := &PolicyRule{
		Name:      "db-to-web",
		FromGroup: "db-servers",
		ToGroup:   "web-servers",
		Ports:     []PortRange{{Start: 443, End: 443, Protocol: "tcp"}},
		Action:    "allow",
		Priority:  100,
		Enabled:   true,
	}

	storage.CreatePolicyRule(rule1)
	storage.CreatePolicyRule(rule2)

	// Compile all policies
	err := compiler.CompileAllPolicies()
	if err != nil {
		t.Fatalf("CompileAllPolicies failed: %v", err)
	}

	// Verify both rules are compiled
	policies1, _ := storage.ListCompiledPoliciesForRule(rule1.ID)
	policies2, _ := storage.ListCompiledPoliciesForRule(rule2.ID)

	if len(policies1) == 0 {
		t.Error("Rule 1 should have compiled policies")
	}
	if len(policies2) == 0 {
		t.Error("Rule 2 should have compiled policies")
	}
}

func TestGetCompilationSummary(t *testing.T) {
	storage, groupMgr, compiler, cleanup := setupCompilerTest(t)
	defer cleanup()

	// Create workloads
	web := &workload.Workload{
		ID:     "web-1",
		IP:     "10.0.1.10",
		Labels: map[string]string{"role": "web"},
	}
	db := &workload.Workload{
		ID:     "db-1",
		IP:     "10.0.2.20",
		Labels: map[string]string{"role": "db"},
	}

	groupMgr.AddWorkload(web)
	groupMgr.AddWorkload(db)

	groupMgr.CreateGroup("web-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "web"},
	})
	groupMgr.CreateGroup("db-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "db"},
	})

	// Create and compile policy rule
	rule := &PolicyRule{
		Name:      "web-to-db",
		FromGroup: "web-servers",
		ToGroup:   "db-servers",
		Ports:     []PortRange{{Start: 3306, End: 3306, Protocol: "tcp"}},
		Action:    "allow",
		Priority:  100,
		Enabled:   true,
	}

	storage.CreatePolicyRule(rule)
	compiler.CompilePolicyRule(rule.ID)

	// Get compilation summary
	summary, err := compiler.GetCompilationSummary()
	if err != nil {
		t.Fatalf("GetCompilationSummary failed: %v", err)
	}

	if summary.TotalRules != 1 {
		t.Errorf("Expected TotalRules=1, got %d", summary.TotalRules)
	}

	if summary.CompiledRules != 1 {
		t.Errorf("Expected CompiledRules=1, got %d", summary.CompiledRules)
	}

	if summary.TotalCompiled != 1 {
		t.Errorf("Expected TotalCompiled=1, got %d", summary.TotalCompiled)
	}

	if summary.AverageExpansion != 1.0 {
		t.Errorf("Expected AverageExpansion=1.0, got %f", summary.AverageExpansion)
	}

	if summary.LastCompilation.IsZero() {
		t.Error("LastCompilation should not be zero")
	}
}

func TestCompilePolicyRule_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	storage, groupMgr, compiler, cleanup := setupCompilerTest(t)
	defer cleanup()

	// Create 10×10 workloads
	for i := 1; i <= 10; i++ {
		for j := 1; j <= 10; j++ {
			web := &workload.Workload{
				ID:     "web-" + string(rune('0'+i)) + "-" + string(rune('0'+j)),
				IP:     "10.0." + string(rune('0'+i)) + "." + string(rune('0'+j)),
				Labels: map[string]string{"role": "web"},
			}
			groupMgr.AddWorkload(web)
		}
	}

	groupMgr.CreateGroup("web-servers", groups.LabelSelector{
		MatchLabels: map[string]string{"role": "web"},
	})

	// Create policy rule
	rule := &PolicyRule{
		Name:      "web-to-web",
		FromGroup: "web-servers",
		ToGroup:   "web-servers",
		Ports:     []PortRange{{Start: 8080, End: 8080, Protocol: "tcp"}},
		Action:    "allow",
		Priority:  100,
		Enabled:   true,
	}

	storage.CreatePolicyRule(rule)

	// Measure compilation time
	start := time.Now()
	result, err := compiler.CompilePolicyRule(rule.ID)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("CompilePolicyRule failed: %v", err)
	}

	t.Logf("Compiled %d policies in %v", result.CompiledCount, duration)

	// Performance target: 10×10 expansion should complete in < 500ms
	if duration > 500*time.Millisecond {
		t.Errorf("Compilation took %v, expected < 500ms", duration)
	}
}

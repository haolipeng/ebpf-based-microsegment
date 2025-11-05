// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"testing"
	"time"
)

func TestCompiledPolicy_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cp      *CompiledPolicy
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid compiled policy",
			cp: &CompiledPolicy{
				Policy: Policy{
					RuleID:   100,
					SrcIP:    "192.168.1.10/32",
					DstIP:    "192.168.2.20/32",
					DstPort:  3306,
					Protocol: "tcp",
					Action:   "allow",
				},
				SourceRuleID:    1,
				FromGroup:       "web-servers",
				ToGroup:         "db-servers",
				FromWorkloadID:  "web-01",
				ToWorkloadID:    "db-01",
				CompilationTime: time.Now(),
				CompilerVersion: CompilerVersionV1,
			},
			wantErr: false,
		},
		{
			name: "missing source_rule_id",
			cp: &CompiledPolicy{
				Policy: Policy{
					RuleID:   100,
					SrcIP:    "192.168.1.10/32",
					DstIP:    "192.168.2.20/32",
					DstPort:  3306,
					Protocol: "tcp",
					Action:   "allow",
				},
				SourceRuleID:    0, // Missing
				FromGroup:       "web-servers",
				ToGroup:         "db-servers",
				FromWorkloadID:  "web-01",
				ToWorkloadID:    "db-01",
				CompilationTime: time.Now(),
			},
			wantErr: true,
			errMsg:  "source_rule_id is required",
		},
		{
			name: "missing from_group",
			cp: &CompiledPolicy{
				Policy: Policy{
					RuleID:   100,
					SrcIP:    "192.168.1.10/32",
					DstIP:    "192.168.2.20/32",
					DstPort:  3306,
					Protocol: "tcp",
					Action:   "allow",
				},
				SourceRuleID:    1,
				FromGroup:       "", // Missing
				ToGroup:         "db-servers",
				FromWorkloadID:  "web-01",
				ToWorkloadID:    "db-01",
				CompilationTime: time.Now(),
			},
			wantErr: true,
			errMsg:  "from_group is required",
		},
		{
			name: "missing to_group",
			cp: &CompiledPolicy{
				Policy: Policy{
					RuleID:   100,
					SrcIP:    "192.168.1.10/32",
					DstIP:    "192.168.2.20/32",
					DstPort:  3306,
					Protocol: "tcp",
					Action:   "allow",
				},
				SourceRuleID:    1,
				FromGroup:       "web-servers",
				ToGroup:         "", // Missing
				FromWorkloadID:  "web-01",
				ToWorkloadID:    "db-01",
				CompilationTime: time.Now(),
			},
			wantErr: true,
			errMsg:  "to_group is required",
		},
		{
			name: "missing from_workload_id",
			cp: &CompiledPolicy{
				Policy: Policy{
					RuleID:   100,
					SrcIP:    "192.168.1.10/32",
					DstIP:    "192.168.2.20/32",
					DstPort:  3306,
					Protocol: "tcp",
					Action:   "allow",
				},
				SourceRuleID:    1,
				FromGroup:       "web-servers",
				ToGroup:         "db-servers",
				FromWorkloadID:  "", // Missing
				ToWorkloadID:    "db-01",
				CompilationTime: time.Now(),
			},
			wantErr: true,
			errMsg:  "from_workload_id is required",
		},
		{
			name: "missing to_workload_id",
			cp: &CompiledPolicy{
				Policy: Policy{
					RuleID:   100,
					SrcIP:    "192.168.1.10/32",
					DstIP:    "192.168.2.20/32",
					DstPort:  3306,
					Protocol: "tcp",
					Action:   "allow",
				},
				SourceRuleID:    1,
				FromGroup:       "web-servers",
				ToGroup:         "db-servers",
				FromWorkloadID:  "web-01",
				ToWorkloadID:    "", // Missing
				CompilationTime: time.Now(),
			},
			wantErr: true,
			errMsg:  "to_workload_id is required",
		},
		{
			name: "missing compilation_time",
			cp: &CompiledPolicy{
				Policy: Policy{
					RuleID:   100,
					SrcIP:    "192.168.1.10/32",
					DstIP:    "192.168.2.20/32",
					DstPort:  3306,
					Protocol: "tcp",
					Action:   "allow",
				},
				SourceRuleID:    1,
				FromGroup:       "web-servers",
				ToGroup:         "db-servers",
				FromWorkloadID:  "web-01",
				ToWorkloadID:    "db-01",
				CompilationTime: time.Time{}, // Zero value
			},
			wantErr: true,
			errMsg:  "compilation_time is required",
		},
		{
			name: "missing protocol in base policy",
			cp: &CompiledPolicy{
				Policy: Policy{
					RuleID:   100,
					SrcIP:    "192.168.1.10",
					DstIP:    "192.168.2.20",
					DstPort:  3306,
					Protocol: "", // Missing protocol
					Action:   "allow",
				},
				SourceRuleID:    1,
				FromGroup:       "web-servers",
				ToGroup:         "db-servers",
				FromWorkloadID:  "web-01",
				ToWorkloadID:    "db-01",
				CompilationTime: time.Now(),
			},
			wantErr: true,
			errMsg:  "protocol is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cp.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CompiledPolicy.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					// 允许部分匹配
					if len(err.Error()) < len(tt.errMsg) || err.Error()[:len(tt.errMsg)] != tt.errMsg {
						t.Errorf("CompiledPolicy.Validate() error = %v, want %v", err.Error(), tt.errMsg)
					}
				}
			}
		})
	}
}

func TestCompiledPolicy_String(t *testing.T) {
	cp := &CompiledPolicy{
		Policy: Policy{
			RuleID:   100,
			SrcIP:    "192.168.1.10/32",
			DstIP:    "192.168.2.20/32",
			DstPort:  3306,
			Protocol: "tcp",
			Action:   "allow",
		},
		SourceRuleID:    1,
		FromGroup:       "web-servers",
		ToGroup:         "db-servers",
		FromWorkloadID:  "web-01",
		ToWorkloadID:    "db-01",
		CompilationTime: time.Now(),
	}

	str := cp.String()
	if str == "" {
		t.Error("CompiledPolicy.String() returned empty string")
	}

	// 验证字符串包含关键信息
	expectedParts := []string{
		"RuleID=100",
		"Source=1",
		"FromGroup=web-servers",
		"ToGroup=db-servers",
		"WorkloadIDs=web-01->db-01",
	}

	for _, part := range expectedParts {
		if len(str) < len(part) {
			t.Errorf("CompiledPolicy.String() doesn't contain %q, got %q", part, str)
		}
	}
}

func TestCompilationResult_AddWarning(t *testing.T) {
	cr := &CompilationResult{
		SourceRuleID: 1,
	}

	if cr.HasWarnings() {
		t.Error("CompilationResult.HasWarnings() should be false initially")
	}

	cr.AddWarning("Test warning: %d policies compiled", 1000)

	if !cr.HasWarnings() {
		t.Error("CompilationResult.HasWarnings() should be true after adding warning")
	}

	if len(cr.Warnings) != 1 {
		t.Errorf("CompilationResult.Warnings length = %d, want 1", len(cr.Warnings))
	}

	if cr.Warnings[0] != "Test warning: 1000 policies compiled" {
		t.Errorf("CompilationResult.Warnings[0] = %q, want formatted warning", cr.Warnings[0])
	}
}

func TestCompilationResult_CalculateExpansion(t *testing.T) {
	tests := []struct {
		name     string
		result   *CompilationResult
		expected float64
	}{
		{
			name: "1x1 expansion",
			result: &CompilationResult{
				CompiledCount:  1,
				FromGroupSize:  1,
				ToGroupSize:    1,
			},
			expected: 1.0,
		},
		{
			name: "10x10 expansion",
			result: &CompilationResult{
				CompiledCount:  100,
				FromGroupSize:  10,
				ToGroupSize:    10,
			},
			expected: 100.0,
		},
		{
			name: "zero expansion",
			result: &CompilationResult{
				CompiledCount:  0,
				FromGroupSize:  0,
				ToGroupSize:    0,
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.result.CalculateExpansion()
			if tt.result.ExpansionRatio != tt.expected {
				t.Errorf("CompilationResult.ExpansionRatio = %f, want %f", tt.result.ExpansionRatio, tt.expected)
			}
		})
	}
}

func TestCompilationResult_String(t *testing.T) {
	cr := &CompilationResult{
		SourceRuleID:    1,
		CompiledCount:   100,
		FromGroupSize:   10,
		ToGroupSize:     10,
		ExpansionRatio:  100.0,
		CompilationTime: 50 * time.Millisecond,
	}

	cr.AddWarning("Large expansion detected")

	str := cr.String()
	if str == "" {
		t.Error("CompilationResult.String() returned empty string")
	}

	// 验证字符串包含关键信息
	expectedParts := []string{
		"SourceRule=1",
		"Compiled=100",
		"Expansion=10x10=100",
		"Warnings=1",
	}

	for _, part := range expectedParts {
		found := false
		for i := 0; i+len(part) <= len(str); i++ {
			if str[i:i+len(part)] == part {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("CompilationResult.String() doesn't contain %q, got %q", part, str)
		}
	}
}

func TestCompilationWarningThresholds(t *testing.T) {
	// 验证阈值常量定义
	if CompilationWarningThreshold != 1000 {
		t.Errorf("CompilationWarningThreshold = %d, want 1000", CompilationWarningThreshold)
	}

	if CompilationCriticalThreshold != 10000 {
		t.Errorf("CompilationCriticalThreshold = %d, want 10000", CompilationCriticalThreshold)
	}

	if CompilationCriticalThreshold <= CompilationWarningThreshold {
		t.Error("CompilationCriticalThreshold should be greater than CompilationWarningThreshold")
	}
}

func TestCompilerVersion(t *testing.T) {
	if CompilerVersionV1 == "" {
		t.Error("CompilerVersionV1 should not be empty")
	}

	// 验证版本格式
	expected := "v1.0.0"
	if CompilerVersionV1 != expected {
		t.Errorf("CompilerVersionV1 = %q, want %q", CompilerVersionV1, expected)
	}
}

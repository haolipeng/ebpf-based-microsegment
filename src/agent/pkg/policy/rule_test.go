// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"os"
	"testing"
)

// TestPolicyRuleValidation tests the validation logic for PolicyRule
func TestPolicyRuleValidation(t *testing.T) {
	tests := []struct {
		name    string
		rule    *PolicyRule
		wantErr bool
	}{
		{
			name: "valid rule with single port",
			rule: &PolicyRule{
				Name:      "allow-web-to-db",
				FromGroup: "web",
				ToGroup:   "database",
				Ports: []PortRange{
					{Protocol: "tcp", Start: 3306, End: 3306},
				},
				Action:   "allow",
				Priority: 100,
				Enabled:  true,
			},
			wantErr: false,
		},
		{
			name: "valid rule with port range",
			rule: &PolicyRule{
				Name:      "allow-web-to-api",
				FromGroup: "web",
				ToGroup:   "api",
				Ports: []PortRange{
					{Protocol: "tcp", Start: 8000, End: 8999},
				},
				Action:   "allow",
				Priority: 100,
				Enabled:  true,
			},
			wantErr: false,
		},
		{
			name: "valid rule with multiple ports",
			rule: &PolicyRule{
				Name:      "allow-web-traffic",
				FromGroup: "internet",
				ToGroup:   "web",
				Ports: []PortRange{
					{Protocol: "tcp", Start: 80, End: 80},
					{Protocol: "tcp", Start: 443, End: 443},
				},
				Action:   "allow",
				Priority: 100,
				Enabled:  true,
			},
			wantErr: false,
		},
		{
			name: "valid rule with wildcard port",
			rule: &PolicyRule{
				Name:      "allow-all-ports",
				FromGroup: "admin",
				ToGroup:   "servers",
				Ports: []PortRange{
					{Protocol: "any", Start: 0, End: 0},
				},
				Action:   "allow",
				Priority: 100,
				Enabled:  true,
			},
			wantErr: false,
		},
		{
			name: "invalid - empty name",
			rule: &PolicyRule{
				Name:      "",
				FromGroup: "web",
				ToGroup:   "db",
				Ports: []PortRange{
					{Protocol: "tcp", Start: 3306, End: 3306},
				},
				Action: "allow",
			},
			wantErr: true,
		},
		{
			name: "invalid - missing from_group",
			rule: &PolicyRule{
				Name:      "test-rule",
				FromGroup: "",
				ToGroup:   "db",
				Ports: []PortRange{
					{Protocol: "tcp", Start: 3306, End: 3306},
				},
				Action: "allow",
			},
			wantErr: true,
		},
		{
			name: "invalid - missing to_group",
			rule: &PolicyRule{
				Name:      "test-rule",
				FromGroup: "web",
				ToGroup:   "",
				Ports: []PortRange{
					{Protocol: "tcp", Start: 3306, End: 3306},
				},
				Action: "allow",
			},
			wantErr: true,
		},
		{
			name: "invalid - invalid action",
			rule: &PolicyRule{
				Name:      "test-rule",
				FromGroup: "web",
				ToGroup:   "db",
				Ports: []PortRange{
					{Protocol: "tcp", Start: 3306, End: 3306},
				},
				Action: "reject",
			},
			wantErr: true,
		},
		{
			name: "invalid - no ports",
			rule: &PolicyRule{
				Name:      "test-rule",
				FromGroup: "web",
				ToGroup:   "db",
				Ports:     []PortRange{},
				Action:    "allow",
			},
			wantErr: true,
		},
		{
			name: "invalid - invalid protocol",
			rule: &PolicyRule{
				Name:      "test-rule",
				FromGroup: "web",
				ToGroup:   "db",
				Ports: []PortRange{
					{Protocol: "http", Start: 80, End: 80},
				},
				Action: "allow",
			},
			wantErr: true,
		},
		{
			name: "invalid - start port > end port",
			rule: &PolicyRule{
				Name:      "test-rule",
				FromGroup: "web",
				ToGroup:   "db",
				Ports: []PortRange{
					{Protocol: "tcp", Start: 9000, End: 8000},
				},
				Action: "allow",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPortRangeValidation tests PortRange validation
func TestPortRangeValidation(t *testing.T) {
	tests := []struct {
		name    string
		port    PortRange
		wantErr bool
	}{
		{
			name:    "valid tcp single port",
			port:    PortRange{Protocol: "tcp", Start: 80, End: 80},
			wantErr: false,
		},
		{
			name:    "valid tcp port range",
			port:    PortRange{Protocol: "tcp", Start: 8000, End: 9000},
			wantErr: false,
		},
		{
			name:    "valid udp port",
			port:    PortRange{Protocol: "udp", Start: 53, End: 53},
			wantErr: false,
		},
		{
			name:    "valid icmp",
			port:    PortRange{Protocol: "icmp", Start: 0, End: 0},
			wantErr: false,
		},
		{
			name:    "valid any protocol",
			port:    PortRange{Protocol: "any", Start: 0, End: 0},
			wantErr: false,
		},
		{
			name:    "valid end is zero (single port)",
			port:    PortRange{Protocol: "tcp", Start: 443, End: 0},
			wantErr: false,
		},
		{
			name:    "invalid protocol",
			port:    PortRange{Protocol: "http", Start: 80, End: 80},
			wantErr: true,
		},
		{
			name:    "invalid start > end",
			port:    PortRange{Protocol: "tcp", Start: 9000, End: 8000},
			wantErr: true,
		},
		{
			name:    "invalid port > 65535",
			port:    PortRange{Protocol: "tcp", Start: 65535, End: 65535},
			wantErr: false, // 65535 is actually valid, test boundary
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.port.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPortRangeString tests PortRange string formatting
func TestPortRangeString(t *testing.T) {
	tests := []struct {
		name string
		port PortRange
		want string
	}{
		{
			name: "single port",
			port: PortRange{Protocol: "tcp", Start: 80, End: 80},
			want: "tcp:80",
		},
		{
			name: "port range",
			port: PortRange{Protocol: "tcp", Start: 8000, End: 9000},
			want: "tcp:8000-9000",
		},
		{
			name: "wildcard",
			port: PortRange{Protocol: "any", Start: 0, End: 0},
			want: "any:any",
		},
		{
			name: "end is zero (single port)",
			port: PortRange{Protocol: "udp", Start: 53, End: 0},
			want: "udp:53",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.port.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPortRangeExpandPorts tests port range expansion
func TestPortRangeExpandPorts(t *testing.T) {
	tests := []struct {
		name      string
		port      PortRange
		wantCount int
		wantFirst uint16
		wantLast  uint16
	}{
		{
			name:      "single port",
			port:      PortRange{Protocol: "tcp", Start: 80, End: 80},
			wantCount: 1,
			wantFirst: 80,
			wantLast:  80,
		},
		{
			name:      "small range",
			port:      PortRange{Protocol: "tcp", Start: 8000, End: 8002},
			wantCount: 3,
			wantFirst: 8000,
			wantLast:  8002,
		},
		{
			name:      "wildcard",
			port:      PortRange{Protocol: "any", Start: 0, End: 0},
			wantCount: 1,
			wantFirst: 0,
			wantLast:  0,
		},
		{
			name:      "end is zero (single port)",
			port:      PortRange{Protocol: "udp", Start: 53, End: 0},
			wantCount: 1,
			wantFirst: 53,
			wantLast:  53,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded := tt.port.ExpandPorts()
			if len(expanded) != tt.wantCount {
				t.Errorf("ExpandPorts() count = %d, want %d", len(expanded), tt.wantCount)
			}
			if len(expanded) > 0 {
				if expanded[0].Port != tt.wantFirst {
					t.Errorf("ExpandPorts() first port = %d, want %d", expanded[0].Port, tt.wantFirst)
				}
				if expanded[len(expanded)-1].Port != tt.wantLast {
					t.Errorf("ExpandPorts() last port = %d, want %d", expanded[len(expanded)-1].Port, tt.wantLast)
				}
			}
		})
	}
}

// TestPortRangeContainsPort tests port containment check
func TestPortRangeContainsPort(t *testing.T) {
	tests := []struct {
		name     string
		portRange PortRange
		testPort  uint16
		want      bool
	}{
		{
			name:      "single port - match",
			portRange: PortRange{Protocol: "tcp", Start: 80, End: 80},
			testPort:  80,
			want:      true,
		},
		{
			name:      "single port - no match",
			portRange: PortRange{Protocol: "tcp", Start: 80, End: 80},
			testPort:  443,
			want:      false,
		},
		{
			name:      "range - start",
			portRange: PortRange{Protocol: "tcp", Start: 8000, End: 9000},
			testPort:  8000,
			want:      true,
		},
		{
			name:      "range - middle",
			portRange: PortRange{Protocol: "tcp", Start: 8000, End: 9000},
			testPort:  8500,
			want:      true,
		},
		{
			name:      "range - end",
			portRange: PortRange{Protocol: "tcp", Start: 8000, End: 9000},
			testPort:  9000,
			want:      true,
		},
		{
			name:      "range - before start",
			portRange: PortRange{Protocol: "tcp", Start: 8000, End: 9000},
			testPort:  7999,
			want:      false,
		},
		{
			name:      "range - after end",
			portRange: PortRange{Protocol: "tcp", Start: 8000, End: 9000},
			testPort:  9001,
			want:      false,
		},
		{
			name:      "wildcard - any port",
			portRange: PortRange{Protocol: "any", Start: 0, End: 0},
			testPort:  12345,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.portRange.ContainsPort(tt.testPort)
			if got != tt.want {
				t.Errorf("ContainsPort(%d) = %v, want %v", tt.testPort, got, tt.want)
			}
		})
	}
}

// TestPolicyRulePortsSerialization tests JSON serialization of ports
func TestPolicyRulePortsSerialization(t *testing.T) {
	rule := &PolicyRule{
		ID:          1,
		Name:        "test-rule",
		Description: "Test rule",
		FromGroup:   "web",
		ToGroup:     "db",
		Ports: []PortRange{
			{Protocol: "tcp", Start: 3306, End: 3306},
			{Protocol: "tcp", Start: 3307, End: 3310},
		},
		Action:   "allow",
		Priority: 100,
		Enabled:  true,
	}

	// Serialize
	jsonStr, err := rule.PortsToJSON()
	if err != nil {
		t.Fatalf("PortsToJSON() error = %v", err)
	}

	if jsonStr == "" {
		t.Error("PortsToJSON() returned empty string")
	}

	// Deserialize
	newRule := &PolicyRule{}
	err = newRule.PortsFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("PortsFromJSON() error = %v", err)
	}

	// Compare
	if len(newRule.Ports) != len(rule.Ports) {
		t.Errorf("Port count mismatch: got %d, want %d", len(newRule.Ports), len(rule.Ports))
	}

	for i := range rule.Ports {
		if newRule.Ports[i].Protocol != rule.Ports[i].Protocol {
			t.Errorf("Port %d protocol mismatch: got %s, want %s", i, newRule.Ports[i].Protocol, rule.Ports[i].Protocol)
		}
		if newRule.Ports[i].Start != rule.Ports[i].Start {
			t.Errorf("Port %d start mismatch: got %d, want %d", i, newRule.Ports[i].Start, rule.Ports[i].Start)
		}
		if newRule.Ports[i].End != rule.Ports[i].End {
			t.Errorf("Port %d end mismatch: got %d, want %d", i, newRule.Ports[i].End, rule.Ports[i].End)
		}
	}
}

// TestPolicyRuleClone tests deep cloning of PolicyRule
func TestPolicyRuleClone(t *testing.T) {
	original := &PolicyRule{
		ID:          1,
		Name:        "original-rule",
		Description: "Original description",
		FromGroup:   "web",
		ToGroup:     "db",
		Ports: []PortRange{
			{Protocol: "tcp", Start: 3306, End: 3306},
		},
		Action:   "allow",
		Priority: 100,
		Enabled:  true,
	}

	// Clone
	clone := original.Clone()

	// Verify values match
	if clone.ID != original.ID {
		t.Errorf("ID mismatch: got %d, want %d", clone.ID, original.ID)
	}
	if clone.Name != original.Name {
		t.Errorf("Name mismatch: got %s, want %s", clone.Name, original.Name)
	}
	if len(clone.Ports) != len(original.Ports) {
		t.Errorf("Ports length mismatch: got %d, want %d", len(clone.Ports), len(original.Ports))
	}

	// Modify clone and ensure original is not affected
	clone.Name = "modified-rule"
	clone.Ports[0].Start = 9999

	if original.Name == "modified-rule" {
		t.Error("Modifying clone affected original name")
	}
	if original.Ports[0].Start == 9999 {
		t.Error("Modifying clone ports affected original ports")
	}
}

// TestPolicyRuleString tests string representation
func TestPolicyRuleString(t *testing.T) {
	rule := &PolicyRule{
		ID:       1,
		Name:     "allow-web-to-db",
		FromGroup: "web",
		ToGroup:  "database",
		Ports: []PortRange{
			{Protocol: "tcp", Start: 3306, End: 3306},
		},
		Action:   "allow",
		Priority: 100,
		Enabled:  true,
	}

	str := rule.String()
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Check if key components are in the string
	if !contains(str, "allow-web-to-db") {
		t.Error("String() missing rule name")
	}
	if !contains(str, "web") {
		t.Error("String() missing from_group")
	}
	if !contains(str, "database") {
		t.Error("String() missing to_group")
	}
	if !contains(str, "allow") {
		t.Error("String() missing action")
	}
}

// TestPolicyRuleCRUD tests full CRUD operations with storage
func TestPolicyRuleCRUD(t *testing.T) {
	// Create temporary database
	tmpDB := "/tmp/test_policy_rules.db"
	defer os.Remove(tmpDB)

	storage, err := NewSQLiteStorage(tmpDB)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Test Create
	rule := &PolicyRule{
		Name:        "allow-web-to-db",
		Description: "Allow web servers to access database",
		FromGroup:   "web",
		ToGroup:     "database",
		Ports: []PortRange{
			{Protocol: "tcp", Start: 3306, End: 3306},
		},
		Action:   "allow",
		Priority: 100,
		Enabled:  true,
	}

	err = storage.CreatePolicyRule(rule)
	if err != nil {
		t.Fatalf("CreatePolicyRule() error = %v", err)
	}

	if rule.ID == 0 {
		t.Error("CreatePolicyRule() did not set ID")
	}

	// Test Get
	retrieved, err := storage.GetPolicyRule(rule.ID)
	if err != nil {
		t.Fatalf("GetPolicyRule() error = %v", err)
	}

	if retrieved.Name != rule.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Name, rule.Name)
	}
	if retrieved.FromGroup != rule.FromGroup {
		t.Errorf("FromGroup mismatch: got %s, want %s", retrieved.FromGroup, rule.FromGroup)
	}
	if len(retrieved.Ports) != len(rule.Ports) {
		t.Errorf("Ports length mismatch: got %d, want %d", len(retrieved.Ports), len(rule.Ports))
	}

	// Test Update
	retrieved.Description = "Updated description"
	retrieved.Priority = 200
	err = storage.UpdatePolicyRule(retrieved)
	if err != nil {
		t.Fatalf("UpdatePolicyRule() error = %v", err)
	}

	updated, err := storage.GetPolicyRule(rule.ID)
	if err != nil {
		t.Fatalf("GetPolicyRule() after update error = %v", err)
	}

	if updated.Description != "Updated description" {
		t.Errorf("Description not updated: got %s, want %s", updated.Description, "Updated description")
	}
	if updated.Priority != 200 {
		t.Errorf("Priority not updated: got %d, want %d", updated.Priority, 200)
	}

	// Test List
	rules, err := storage.ListPolicyRules()
	if err != nil {
		t.Fatalf("ListPolicyRules() error = %v", err)
	}

	if len(rules) != 1 {
		t.Errorf("ListPolicyRules() count = %d, want 1", len(rules))
	}

	// Test Delete
	err = storage.DeletePolicyRule(rule.ID)
	if err != nil {
		t.Fatalf("DeletePolicyRule() error = %v", err)
	}

	// Verify deletion
	_, err = storage.GetPolicyRule(rule.ID)
	if err == nil {
		t.Error("GetPolicyRule() should return error for deleted rule")
	}
}

// TestListEnabledPolicyRules tests listing only enabled rules
func TestListEnabledPolicyRules(t *testing.T) {
	// Create temporary database
	tmpDB := "/tmp/test_enabled_rules.db"
	defer os.Remove(tmpDB)

	storage, err := NewSQLiteStorage(tmpDB)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Create enabled rule
	enabledRule := &PolicyRule{
		Name:      "enabled-rule",
		FromGroup: "web",
		ToGroup:   "db",
		Ports: []PortRange{
			{Protocol: "tcp", Start: 3306, End: 3306},
		},
		Action:   "allow",
		Priority: 100,
		Enabled:  true,
	}

	err = storage.CreatePolicyRule(enabledRule)
	if err != nil {
		t.Fatalf("CreatePolicyRule() error = %v", err)
	}

	// Create disabled rule
	disabledRule := &PolicyRule{
		Name:      "disabled-rule",
		FromGroup: "api",
		ToGroup:   "cache",
		Ports: []PortRange{
			{Protocol: "tcp", Start: 6379, End: 6379},
		},
		Action:   "allow",
		Priority: 100,
		Enabled:  false,
	}

	err = storage.CreatePolicyRule(disabledRule)
	if err != nil {
		t.Fatalf("CreatePolicyRule() error = %v", err)
	}

	// List all rules
	allRules, err := storage.ListPolicyRules()
	if err != nil {
		t.Fatalf("ListPolicyRules() error = %v", err)
	}

	if len(allRules) != 2 {
		t.Errorf("ListPolicyRules() count = %d, want 2", len(allRules))
	}

	// List enabled rules only
	enabledRules, err := storage.ListEnabledPolicyRules()
	if err != nil {
		t.Fatalf("ListEnabledPolicyRules() error = %v", err)
	}

	if len(enabledRules) != 1 {
		t.Errorf("ListEnabledPolicyRules() count = %d, want 1", len(enabledRules))
	}

	if enabledRules[0].Name != "enabled-rule" {
		t.Errorf("ListEnabledPolicyRules() returned wrong rule: got %s, want enabled-rule", enabledRules[0].Name)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

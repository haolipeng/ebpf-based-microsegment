// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package groups

import (
	"encoding/json"
	"testing"
)

func TestSelectorOperator(t *testing.T) {
	tests := []struct {
		name     string
		operator SelectorOperator
		isValid  bool
	}{
		{"equal", OpEqual, true},
		{"not equal", OpNotEqual, true},
		{"in", OpIn, true},
		{"not in", OpNotIn, true},
		{"exists", OpExists, true},
		{"not exists", OpNotExists, true},
		{"invalid", SelectorOperator("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidOperator(tt.operator)
			if result != tt.isValid {
				t.Errorf("IsValidOperator(%s) = %v, want %v", tt.operator, result, tt.isValid)
			}
		})
	}
}

func TestAllOperators(t *testing.T) {
	ops := AllOperators()
	if len(ops) != 6 {
		t.Errorf("AllOperators() returned %d operators, want 6", len(ops))
	}

	expected := []SelectorOperator{OpEqual, OpNotEqual, OpIn, OpNotIn, OpExists, OpNotExists}
	for i, op := range expected {
		if ops[i] != op {
			t.Errorf("AllOperators()[%d] = %s, want %s", i, ops[i], op)
		}
	}
}

func TestOperatorString(t *testing.T) {
	tests := []struct {
		operator SelectorOperator
		expected string
	}{
		{OpEqual, "="},
		{OpNotEqual, "!="},
		{OpIn, "in"},
		{OpNotIn, "not-in"},
		{OpExists, "exists"},
		{OpNotExists, "not-exists"},
	}

	for _, tt := range tests {
		t.Run(string(tt.operator), func(t *testing.T) {
			if tt.operator.String() != tt.expected {
				t.Errorf("String() = %s, want %s", tt.operator.String(), tt.expected)
			}
		})
	}
}

func TestOperatorRequiresValues(t *testing.T) {
	tests := []struct {
		operator       SelectorOperator
		requiresValues bool
	}{
		{OpEqual, true},
		{OpNotEqual, true},
		{OpIn, true},
		{OpNotIn, true},
		{OpExists, false},
		{OpNotExists, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.operator), func(t *testing.T) {
			result := tt.operator.RequiresValues()
			if result != tt.requiresValues {
				t.Errorf("RequiresValues() = %v, want %v", result, tt.requiresValues)
			}
		})
	}
}

func TestLabelSelectorValidate(t *testing.T) {
	tests := []struct {
		name     string
		selector LabelSelector
		wantErr  bool
	}{
		{
			name:     "valid equal",
			selector: NewEqualSelector("role", "web"),
			wantErr:  false,
		},
		{
			name:     "valid in",
			selector: NewInSelector("env", []string{"prod", "staging"}),
			wantErr:  false,
		},
		{
			name:     "valid exists",
			selector: NewExistsSelector("version"),
			wantErr:  false,
		},
		{
			name: "empty key",
			selector: LabelSelector{
				Key:      "",
				Operator: OpEqual,
				Values:   []string{"value"},
			},
			wantErr: true,
		},
		{
			name: "invalid operator",
			selector: LabelSelector{
				Key:      "key",
				Operator: SelectorOperator("invalid"),
				Values:   []string{"value"},
			},
			wantErr: true,
		},
		{
			name: "missing values for equal",
			selector: LabelSelector{
				Key:      "key",
				Operator: OpEqual,
				Values:   []string{},
			},
			wantErr: true,
		},
		{
			name: "missing values for in",
			selector: LabelSelector{
				Key:      "key",
				Operator: OpIn,
				Values:   []string{},
			},
			wantErr: true,
		},
		{
			name:     "exists without values",
			selector: NewExistsSelector("key"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.selector.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLabelSelectorString(t *testing.T) {
	tests := []struct {
		name     string
		selector LabelSelector
		expected string
	}{
		{
			name:     "equal",
			selector: NewEqualSelector("role", "web"),
			expected: "role=web",
		},
		{
			name:     "not equal",
			selector: NewNotEqualSelector("env", "dev"),
			expected: "env!=dev",
		},
		{
			name:     "in",
			selector: NewInSelector("app", []string{"api", "web"}),
			expected: "app in [api web]",
		},
		{
			name:     "not in",
			selector: NewNotInSelector("region", []string{"us-east-1"}),
			expected: "region not-in [us-east-1]",
		},
		{
			name:     "exists",
			selector: NewExistsSelector("version"),
			expected: "version exists",
		},
		{
			name:     "not exists",
			selector: NewNotExistsSelector("deprecated"),
			expected: "deprecated not-exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.selector.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNewGroup(t *testing.T) {
	group := NewGroup("test-group")

	if group.Name != "test-group" {
		t.Errorf("Name = %s, want test-group", group.Name)
	}

	if len(group.Selectors) != 0 {
		t.Errorf("New group should have 0 selectors, got %d", len(group.Selectors))
	}

	if group.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	if group.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestGroupAddSelector(t *testing.T) {
	group := NewGroup("test")

	// Add valid selector
	err := group.AddSelector(NewEqualSelector("role", "web"))
	if err != nil {
		t.Errorf("AddSelector() failed: %v", err)
	}

	if len(group.Selectors) != 1 {
		t.Errorf("Expected 1 selector, got %d", len(group.Selectors))
	}

	// Try to add invalid selector
	invalidSelector := LabelSelector{
		Key:      "",
		Operator: OpEqual,
		Values:   []string{"value"},
	}

	err = group.AddSelector(invalidSelector)
	if err == nil {
		t.Error("AddSelector() should fail for invalid selector")
	}
}

func TestGroupSetSelectors(t *testing.T) {
	group := NewGroup("test")

	selectors := []LabelSelector{
		NewEqualSelector("role", "web"),
		NewEqualSelector("env", "prod"),
	}

	err := group.SetSelectors(selectors)
	if err != nil {
		t.Errorf("SetSelectors() failed: %v", err)
	}

	if len(group.Selectors) != 2 {
		t.Errorf("Expected 2 selectors, got %d", len(group.Selectors))
	}

	// Try to set invalid selectors
	invalidSelectors := []LabelSelector{
		NewEqualSelector("role", "web"),
		{Key: "", Operator: OpEqual, Values: []string{"value"}},
	}

	err = group.SetSelectors(invalidSelectors)
	if err == nil {
		t.Error("SetSelectors() should fail for invalid selectors")
	}
}

func TestGroupValidate(t *testing.T) {
	tests := []struct {
		name    string
		group   *Group
		wantErr bool
	}{
		{
			name: "valid group",
			group: func() *Group {
				g := NewGroup("valid")
				g.AddSelector(NewEqualSelector("role", "web"))
				return g
			}(),
			wantErr: false,
		},
		{
			name:    "empty name",
			group:   NewGroup(""),
			wantErr: true,
		},
		{
			name:    "no selectors",
			group:   NewGroup("no-selectors"),
			wantErr: true,
		},
		{
			name: "invalid selector",
			group: func() *Group {
				g := NewGroup("invalid-sel")
				g.Selectors = []LabelSelector{
					{Key: "", Operator: OpEqual, Values: []string{"value"}},
				}
				return g
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.group.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGroupString(t *testing.T) {
	group := NewGroup("test")
	group.AddSelector(NewExistsSelector("version"))

	str := group.String()
	expected := "Group[test]: 1 selectors"

	if str != expected {
		t.Errorf("String() = %q, want %q", str, expected)
	}
}

func TestGroupToSummary(t *testing.T) {
	group := NewGroup("test-group")
	group.Description = "Test description"
	group.AddSelector(NewEqualSelector("role", "web"))
	group.AddSelector(NewEqualSelector("env", "prod"))

	summary := group.ToSummary()

	if summary.Name != "test-group" {
		t.Errorf("Summary.Name = %s, want test-group", summary.Name)
	}

	if summary.Description != "Test description" {
		t.Errorf("Summary.Description = %s, want Test description", summary.Description)
	}

	if summary.SelectorCount != 2 {
		t.Errorf("Summary.SelectorCount = %d, want 2", summary.SelectorCount)
	}

	if summary.CreatedAt.IsZero() {
		t.Error("Summary.CreatedAt should be set")
	}

	if summary.UpdatedAt.IsZero() {
		t.Error("Summary.UpdatedAt should be set")
	}
}

func TestGroupJSONSerialization(t *testing.T) {
	// Create a group with selectors
	group := NewGroup("json-test")
	group.Description = "Test JSON"
	group.AddSelector(NewEqualSelector("role", "web"))
	group.AddSelector(NewInSelector("env", []string{"prod", "staging"}))

	// Marshal to JSON
	data, err := json.Marshal(group)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal from JSON
	var decoded Group
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if decoded.Name != group.Name {
		t.Errorf("Name = %s, want %s", decoded.Name, group.Name)
	}

	if decoded.Description != group.Description {
		t.Errorf("Description = %s, want %s", decoded.Description, group.Description)
	}

	if len(decoded.Selectors) != len(group.Selectors) {
		t.Errorf("Selectors length = %d, want %d", len(decoded.Selectors), len(group.Selectors))
	}

	// Verify selectors
	for i, sel := range decoded.Selectors {
		if sel.Key != group.Selectors[i].Key {
			t.Errorf("Selector[%d].Key = %s, want %s", i, sel.Key, group.Selectors[i].Key)
		}
		if sel.Operator != group.Selectors[i].Operator {
			t.Errorf("Selector[%d].Operator = %s, want %s", i, sel.Operator, group.Selectors[i].Operator)
		}
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("NewEqualSelector", func(t *testing.T) {
		sel := NewEqualSelector("key", "value")
		if sel.Key != "key" || sel.Operator != OpEqual || sel.Values[0] != "value" {
			t.Error("NewEqualSelector() produced incorrect selector")
		}
	})

	t.Run("NewNotEqualSelector", func(t *testing.T) {
		sel := NewNotEqualSelector("key", "value")
		if sel.Key != "key" || sel.Operator != OpNotEqual || sel.Values[0] != "value" {
			t.Error("NewNotEqualSelector() produced incorrect selector")
		}
	})

	t.Run("NewInSelector", func(t *testing.T) {
		sel := NewInSelector("key", []string{"v1", "v2"})
		if sel.Key != "key" || sel.Operator != OpIn || len(sel.Values) != 2 {
			t.Error("NewInSelector() produced incorrect selector")
		}
	})

	t.Run("NewNotInSelector", func(t *testing.T) {
		sel := NewNotInSelector("key", []string{"v1", "v2"})
		if sel.Key != "key" || sel.Operator != OpNotIn || len(sel.Values) != 2 {
			t.Error("NewNotInSelector() produced incorrect selector")
		}
	})

	t.Run("NewExistsSelector", func(t *testing.T) {
		sel := NewExistsSelector("key")
		if sel.Key != "key" || sel.Operator != OpExists {
			t.Error("NewExistsSelector() produced incorrect selector")
		}
	})

	t.Run("NewNotExistsSelector", func(t *testing.T) {
		sel := NewNotExistsSelector("key")
		if sel.Key != "key" || sel.Operator != OpNotExists {
			t.Error("NewNotExistsSelector() produced incorrect selector")
		}
	})
}

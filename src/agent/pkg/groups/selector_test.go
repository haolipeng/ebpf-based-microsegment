// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package groups

import (
	"testing"

	"github.com/ebpf-microsegment/src/agent/pkg/workload"
)

// Helper function to create a workload with labels
func createWorkloadWithLabels(labels map[string]string) *workload.Workload {
	wl := workload.NewWorkload("test-wl-id", "test-wl-name", "test-host")
	wl.Labels = labels
	return wl
}

// TestEvaluateEqual tests the equals (=) operator
func TestEvaluateEqual(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		selector LabelSelector
		want     bool
	}{
		{
			name:     "exact match",
			labels:   map[string]string{"role": "web"},
			selector: NewEqualSelector("role", "web"),
			want:     true,
		},
		{
			name:     "no match - different value",
			labels:   map[string]string{"role": "db"},
			selector: NewEqualSelector("role", "web"),
			want:     false,
		},
		{
			name:     "no match - key doesn't exist",
			labels:   map[string]string{"env": "prod"},
			selector: NewEqualSelector("role", "web"),
			want:     false,
		},
		{
			name:     "no match - empty labels",
			labels:   map[string]string{},
			selector: NewEqualSelector("role", "web"),
			want:     false,
		},
		{
			name:     "case sensitive match",
			labels:   map[string]string{"env": "Prod"},
			selector: NewEqualSelector("env", "prod"),
			want:     false,
		},
		{
			name:     "case sensitive exact",
			labels:   map[string]string{"env": "Prod"},
			selector: NewEqualSelector("env", "Prod"),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := createWorkloadWithLabels(tt.labels)
			result := EvaluateSelector(wl, tt.selector)
			if result != tt.want {
				t.Errorf("EvaluateSelector() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestEvaluateNotEqual tests the not-equals (!=) operator
func TestEvaluateNotEqual(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		selector LabelSelector
		want     bool
	}{
		{
			name:     "different value",
			labels:   map[string]string{"env": "prod"},
			selector: NewNotEqualSelector("env", "dev"),
			want:     true,
		},
		{
			name:     "same value",
			labels:   map[string]string{"env": "prod"},
			selector: NewNotEqualSelector("env", "prod"),
			want:     false,
		},
		{
			name:     "key doesn't exist - considered not equal",
			labels:   map[string]string{"role": "web"},
			selector: NewNotEqualSelector("env", "dev"),
			want:     true,
		},
		{
			name:     "empty labels - key doesn't exist",
			labels:   map[string]string{},
			selector: NewNotEqualSelector("env", "dev"),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := createWorkloadWithLabels(tt.labels)
			result := EvaluateSelector(wl, tt.selector)
			if result != tt.want {
				t.Errorf("EvaluateSelector() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestEvaluateIn tests the in operator
func TestEvaluateIn(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		selector LabelSelector
		want     bool
	}{
		{
			name:     "value in list - first",
			labels:   map[string]string{"role": "web"},
			selector: NewInSelector("role", []string{"web", "api", "worker"}),
			want:     true,
		},
		{
			name:     "value in list - middle",
			labels:   map[string]string{"role": "api"},
			selector: NewInSelector("role", []string{"web", "api", "worker"}),
			want:     true,
		},
		{
			name:     "value in list - last",
			labels:   map[string]string{"role": "worker"},
			selector: NewInSelector("role", []string{"web", "api", "worker"}),
			want:     true,
		},
		{
			name:     "value not in list",
			labels:   map[string]string{"role": "db"},
			selector: NewInSelector("role", []string{"web", "api", "worker"}),
			want:     false,
		},
		{
			name:     "key doesn't exist",
			labels:   map[string]string{"env": "prod"},
			selector: NewInSelector("role", []string{"web", "api"}),
			want:     false,
		},
		{
			name:     "single value in list",
			labels:   map[string]string{"env": "prod"},
			selector: NewInSelector("env", []string{"prod"}),
			want:     true,
		},
		{
			name:     "empty list - no match",
			labels:   map[string]string{"role": "web"},
			selector: LabelSelector{Key: "role", Operator: OpIn, Values: []string{}},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := createWorkloadWithLabels(tt.labels)
			result := EvaluateSelector(wl, tt.selector)
			if result != tt.want {
				t.Errorf("EvaluateSelector() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestEvaluateNotIn tests the not-in operator
func TestEvaluateNotIn(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		selector LabelSelector
		want     bool
	}{
		{
			name:     "value not in list",
			labels:   map[string]string{"role": "db"},
			selector: NewNotInSelector("role", []string{"web", "api"}),
			want:     true,
		},
		{
			name:     "value in list",
			labels:   map[string]string{"role": "web"},
			selector: NewNotInSelector("role", []string{"web", "api"}),
			want:     false,
		},
		{
			name:     "key doesn't exist - considered not in",
			labels:   map[string]string{"env": "prod"},
			selector: NewNotInSelector("role", []string{"web", "api"}),
			want:     true,
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			selector: NewNotInSelector("role", []string{"web", "api"}),
			want:     true,
		},
		{
			name:     "single value not in list",
			labels:   map[string]string{"env": "dev"},
			selector: NewNotInSelector("env", []string{"prod"}),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := createWorkloadWithLabels(tt.labels)
			result := EvaluateSelector(wl, tt.selector)
			if result != tt.want {
				t.Errorf("EvaluateSelector() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestEvaluateExists tests the exists operator
func TestEvaluateExists(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		selector LabelSelector
		want     bool
	}{
		{
			name:     "key exists with value",
			labels:   map[string]string{"version": "1.0"},
			selector: NewExistsSelector("version"),
			want:     true,
		},
		{
			name:     "key exists with empty value",
			labels:   map[string]string{"version": ""},
			selector: NewExistsSelector("version"),
			want:     true,
		},
		{
			name:     "key doesn't exist",
			labels:   map[string]string{"role": "web"},
			selector: NewExistsSelector("version"),
			want:     false,
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			selector: NewExistsSelector("version"),
			want:     false,
		},
		{
			name:     "multiple keys - check one exists",
			labels:   map[string]string{"role": "web", "env": "prod", "version": "1.0"},
			selector: NewExistsSelector("version"),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := createWorkloadWithLabels(tt.labels)
			result := EvaluateSelector(wl, tt.selector)
			if result != tt.want {
				t.Errorf("EvaluateSelector() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestEvaluateNotExists tests the not-exists operator
func TestEvaluateNotExists(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		selector LabelSelector
		want     bool
	}{
		{
			name:     "key doesn't exist",
			labels:   map[string]string{"role": "web"},
			selector: NewNotExistsSelector("deprecated"),
			want:     true,
		},
		{
			name:     "key exists",
			labels:   map[string]string{"deprecated": "true"},
			selector: NewNotExistsSelector("deprecated"),
			want:     false,
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			selector: NewNotExistsSelector("deprecated"),
			want:     true,
		},
		{
			name:     "key exists with empty value",
			labels:   map[string]string{"deprecated": ""},
			selector: NewNotExistsSelector("deprecated"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := createWorkloadWithLabels(tt.labels)
			result := EvaluateSelector(wl, tt.selector)
			if result != tt.want {
				t.Errorf("EvaluateSelector() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestEvaluateInvalidSelector tests handling of invalid selectors
func TestEvaluateInvalidSelector(t *testing.T) {
	wl := createWorkloadWithLabels(map[string]string{"role": "web"})

	tests := []struct {
		name     string
		selector LabelSelector
	}{
		{
			name:     "empty key",
			selector: LabelSelector{Key: "", Operator: OpEqual, Values: []string{"web"}},
		},
		{
			name:     "invalid operator",
			selector: LabelSelector{Key: "role", Operator: SelectorOperator("invalid"), Values: []string{"web"}},
		},
		{
			name:     "missing values for equal",
			selector: LabelSelector{Key: "role", Operator: OpEqual, Values: []string{}},
		},
		{
			name:     "missing values for in",
			selector: LabelSelector{Key: "role", Operator: OpIn, Values: []string{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluateSelector(wl, tt.selector)
			if result != false {
				t.Errorf("EvaluateSelector() with invalid selector = %v, want false", result)
			}
		})
	}
}

// TestEvaluateSelectors tests evaluating multiple selectors (AND logic)
func TestEvaluateSelectors(t *testing.T) {
	tests := []struct {
		name      string
		labels    map[string]string
		selectors []LabelSelector
		want      bool
	}{
		{
			name:   "single selector - match",
			labels: map[string]string{"role": "web"},
			selectors: []LabelSelector{
				NewEqualSelector("role", "web"),
			},
			want: true,
		},
		{
			name:   "single selector - no match",
			labels: map[string]string{"role": "db"},
			selectors: []LabelSelector{
				NewEqualSelector("role", "web"),
			},
			want: false,
		},
		{
			name:   "multiple selectors - all match",
			labels: map[string]string{"role": "web", "env": "prod"},
			selectors: []LabelSelector{
				NewEqualSelector("role", "web"),
				NewEqualSelector("env", "prod"),
			},
			want: true,
		},
		{
			name:   "multiple selectors - one doesn't match",
			labels: map[string]string{"role": "web", "env": "dev"},
			selectors: []LabelSelector{
				NewEqualSelector("role", "web"),
				NewEqualSelector("env", "prod"),
			},
			want: false,
		},
		{
			name:   "complex selectors - all match",
			labels: map[string]string{"role": "web", "env": "prod", "version": "1.0", "region": "us-east-1"},
			selectors: []LabelSelector{
				NewEqualSelector("role", "web"),
				NewInSelector("env", []string{"prod", "staging"}),
				NewExistsSelector("version"),
				NewNotInSelector("region", []string{"us-west-1", "us-west-2"}),
			},
			want: true,
		},
		{
			name:   "complex selectors - one fails",
			labels: map[string]string{"role": "web", "env": "dev", "version": "1.0"},
			selectors: []LabelSelector{
				NewEqualSelector("role", "web"),
				NewInSelector("env", []string{"prod", "staging"}), // This will fail
				NewExistsSelector("version"),
			},
			want: false,
		},
		{
			name:      "empty selectors - no match",
			labels:    map[string]string{"role": "web"},
			selectors: []LabelSelector{},
			want:      false,
		},
		{
			name:   "mix of exists and value checks",
			labels: map[string]string{"role": "api", "version": "2.0"},
			selectors: []LabelSelector{
				NewInSelector("role", []string{"web", "api", "worker"}),
				NewExistsSelector("version"),
				NewNotExistsSelector("deprecated"),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := createWorkloadWithLabels(tt.labels)
			result := EvaluateSelectors(wl, tt.selectors)
			if result != tt.want {
				t.Errorf("EvaluateSelectors() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestMatchesGroup tests the MatchesGroup convenience function
func TestMatchesGroup(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		group  *Group
		want   bool
	}{
		{
			name:   "matches group",
			labels: map[string]string{"role": "web", "env": "prod"},
			group: func() *Group {
				g := NewGroup("web-prod")
				g.AddSelector(NewEqualSelector("role", "web"))
				g.AddSelector(NewEqualSelector("env", "prod"))
				return g
			}(),
			want: true,
		},
		{
			name:   "doesn't match group",
			labels: map[string]string{"role": "db", "env": "prod"},
			group: func() *Group {
				g := NewGroup("web-prod")
				g.AddSelector(NewEqualSelector("role", "web"))
				g.AddSelector(NewEqualSelector("env", "prod"))
				return g
			}(),
			want: false,
		},
		{
			name:   "nil group",
			labels: map[string]string{"role": "web"},
			group:  nil,
			want:   false,
		},
		{
			name:   "empty workload labels",
			labels: map[string]string{},
			group: func() *Group {
				g := NewGroup("web-servers")
				g.AddSelector(NewEqualSelector("role", "web"))
				return g
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := createWorkloadWithLabels(tt.labels)
			result := MatchesGroup(wl, tt.group)
			if result != tt.want {
				t.Errorf("MatchesGroup() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestSelectorEdgeCases tests edge cases and boundary conditions
func TestSelectorEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		selector LabelSelector
		want     bool
	}{
		{
			name:     "nil labels map",
			labels:   nil,
			selector: NewEqualSelector("role", "web"),
			want:     false,
		},
		{
			name:     "empty string value - equals",
			labels:   map[string]string{"status": ""},
			selector: NewEqualSelector("status", ""),
			want:     true,
		},
		{
			name:     "empty string value - not equals",
			labels:   map[string]string{"status": ""},
			selector: NewNotEqualSelector("status", "active"),
			want:     true,
		},
		{
			name:     "whitespace in value",
			labels:   map[string]string{"name": "web server"},
			selector: NewEqualSelector("name", "web server"),
			want:     true,
		},
		{
			name:     "special characters in value",
			labels:   map[string]string{"app": "my-app_v1.0"},
			selector: NewEqualSelector("app", "my-app_v1.0"),
			want:     true,
		},
		{
			name:     "unicode characters",
			labels:   map[string]string{"team": "开发团队"},
			selector: NewEqualSelector("team", "开发团队"),
			want:     true,
		},
		{
			name:     "very long value",
			labels:   map[string]string{"long": "a very long value that exceeds normal expectations"},
			selector: NewEqualSelector("long", "a very long value that exceeds normal expectations"),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl := createWorkloadWithLabels(tt.labels)
			result := EvaluateSelector(wl, tt.selector)
			if result != tt.want {
				t.Errorf("EvaluateSelector() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestAllOperatorsCombined tests all operators in a single comprehensive test
func TestAllOperatorsCombined(t *testing.T) {
	// Create a workload with various labels
	wl := createWorkloadWithLabels(map[string]string{
		"role":    "web",
		"env":     "prod",
		"version": "1.0",
		"region":  "us-east-1",
	})

	tests := []struct {
		name     string
		selector LabelSelector
		want     bool
	}{
		// OpEqual
		{"equal match", NewEqualSelector("role", "web"), true},
		{"equal no match", NewEqualSelector("role", "db"), false},

		// OpNotEqual
		{"not equal match", NewNotEqualSelector("role", "db"), true},
		{"not equal no match", NewNotEqualSelector("role", "web"), false},

		// OpIn
		{"in match", NewInSelector("env", []string{"dev", "prod", "staging"}), true},
		{"in no match", NewInSelector("env", []string{"dev", "staging"}), false},

		// OpNotIn
		{"not in match", NewNotInSelector("region", []string{"us-west-1", "eu-west-1"}), true},
		{"not in no match", NewNotInSelector("region", []string{"us-east-1", "us-west-1"}), false},

		// OpExists
		{"exists match", NewExistsSelector("version"), true},
		{"exists no match", NewExistsSelector("nonexistent"), false},

		// OpNotExists
		{"not exists match", NewNotExistsSelector("deprecated"), true},
		{"not exists no match", NewNotExistsSelector("version"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluateSelector(wl, tt.selector)
			if result != tt.want {
				t.Errorf("EvaluateSelector() = %v, want %v for selector %v", result, tt.want, tt.selector.String())
			}
		})
	}
}

// BenchmarkEvaluateSelector benchmarks selector evaluation performance
func BenchmarkEvaluateSelector(b *testing.B) {
	wl := createWorkloadWithLabels(map[string]string{
		"role": "web",
		"env":  "prod",
	})

	selector := NewEqualSelector("role", "web")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EvaluateSelector(wl, selector)
	}
}

// BenchmarkEvaluateSelectors benchmarks multiple selector evaluation
func BenchmarkEvaluateSelectors(b *testing.B) {
	wl := createWorkloadWithLabels(map[string]string{
		"role":    "web",
		"env":     "prod",
		"version": "1.0",
	})

	selectors := []LabelSelector{
		NewEqualSelector("role", "web"),
		NewEqualSelector("env", "prod"),
		NewExistsSelector("version"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EvaluateSelectors(wl, selectors)
	}
}

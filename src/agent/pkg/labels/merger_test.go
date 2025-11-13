// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package labels

import (
	"testing"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/runtime"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/workload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRuntimeDetector mocks the runtime detector for testing
type MockRuntimeDetector struct {
	mock.Mock
}

func (m *MockRuntimeDetector) GetContainerLabels(containerID string) (map[string]string, error) {
	args := m.Called(containerID)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockRuntimeDetector) ListContainersWithLabels() (map[string]map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]map[string]string), args.Error(1)
}

func (m *MockRuntimeDetector) WatchContainerEvents(callback func(runtime.ContainerEvent)) error {
	args := m.Called(callback)
	return args.Error(0)
}

func (m *MockRuntimeDetector) Close() error {
	args := m.Called()
	return args.Error(0)
}

// TestGetEffectiveLabels_AllLayers tests 4-layer label merging
func TestGetEffectiveLabels_AllLayers(t *testing.T) {
	// Create mock runtime detector
	mockRuntime := new(MockRuntimeDetector)
	mockRuntime.On("GetContainerLabels", "container-123").Return(map[string]string{
		"app.kubernetes.io/name":      "k8s-app",
		"app.kubernetes.io/component": "k8s-role",
		"environment":                 "k8s-env",
		"k8s-custom":                  "from-k8s",
	}, nil)

	// Create auto tagger
	autoTagger := NewAutoTagger()

	// Create label merger
	merger := NewLabelMerger(mockRuntime, autoTagger)

	// Create workload
	wl := &workload.Workload{
		ID:          "wl-123",
		HostID:      "host-001",
		Namespace:   "production",
		ContainerID: "container-123",
		Image:       "nginx:latest",
		Labels: map[string]string{
			"app": "user-app", // User-defined, should override all
			"env": "user-env", // User-defined, should override all
		},
	}

	// Get effective labels
	effectiveLabels := merger.GetEffectiveLabels(wl)

	// Layer 1: System metadata
	assert.Equal(t, "host-001", effectiveLabels["host"], "Should have system host label")
	assert.Equal(t, "production", effectiveLabels["namespace"], "Should have system namespace label")

	// Layer 2: AutoTagger inferred labels
	// Note: nginx image is inferred as "webserver", but k8s runtime label overrides it
	// The runtime label "k8s-role" is mapped from "app.kubernetes.io/component"

	// Layer 3: Runtime labels (mapped from K8s labels)
	assert.Equal(t, "from-k8s", effectiveLabels["k8s-custom"], "Should have runtime label")

	// Layer 4: User-defined labels (highest priority)
	assert.Equal(t, "user-app", effectiveLabels["app"], "User label should override K8s label")
	assert.Equal(t, "user-env", effectiveLabels["env"], "User label should override K8s label")

	mockRuntime.AssertExpectations(t)
}

// TestGetEffectiveLabels_PriorityOverride tests that higher priority labels override lower ones
func TestGetEffectiveLabels_PriorityOverride(t *testing.T) {
	mockRuntime := new(MockRuntimeDetector)
	mockRuntime.On("GetContainerLabels", "container-123").Return(map[string]string{
		"app":  "runtime-app",  // Should be overridden by user
		"tier": "runtime-tier", // Should be mapped to "role" and kept
	}, nil)

	autoTagger := NewAutoTagger()
	merger := NewLabelMerger(mockRuntime, autoTagger)

	wl := &workload.Workload{
		ID:          "wl-123",
		ContainerID: "container-123",
		Image:       "redis:alpine",
		Labels: map[string]string{
			"app": "user-app", // Highest priority
		},
	}

	effectiveLabels := merger.GetEffectiveLabels(wl)

	// User-defined "app" should win
	assert.Equal(t, "user-app", effectiveLabels["app"],
		"User-defined label should override runtime label")

	// Runtime "tier" mapped to "role" should be present
	assert.Equal(t, "runtime-tier", effectiveLabels["role"],
		"Runtime label should be mapped to role")

	// AutoTagger should not override runtime labels
	// (redis-server would be auto-tagged, but runtime takes precedence)

	mockRuntime.AssertExpectations(t)
}

// TestGetEffectiveLabels_NoRuntime tests label merging without runtime detector
func TestGetEffectiveLabels_NoRuntime(t *testing.T) {
	autoTagger := NewAutoTagger()
	merger := NewLabelMerger(nil, autoTagger) // No runtime detector

	wl := &workload.Workload{
		ID:        "wl-123",
		HostID:    "host-001",
		Namespace: "production",
		Image:     "postgres:13",
		Labels: map[string]string{
			"app": "user-app",
		},
	}

	effectiveLabels := merger.GetEffectiveLabels(wl)

	// Should have system labels
	assert.Equal(t, "host-001", effectiveLabels["host"])
	assert.Equal(t, "production", effectiveLabels["namespace"])

	// Should have auto-inferred labels
	assert.Equal(t, "db", effectiveLabels["role"])

	// Should have user labels
	assert.Equal(t, "user-app", effectiveLabels["app"])

	// Should NOT have runtime labels
	assert.NotContains(t, effectiveLabels, "k8s.pod")
}

// TestGetEffectiveLabels_NoAutoTagger tests label merging without auto tagger
func TestGetEffectiveLabels_NoAutoTagger(t *testing.T) {
	mockRuntime := new(MockRuntimeDetector)
	mockRuntime.On("GetContainerLabels", "container-123").Return(map[string]string{
		"app": "runtime-app",
	}, nil)

	merger := NewLabelMerger(mockRuntime, nil) // No auto tagger

	wl := &workload.Workload{
		ID:          "wl-123",
		HostID:      "host-001",
		ContainerID: "container-123",
		Image:       "postgres:13",
		Labels: map[string]string{
			"env": "user-env",
		},
	}

	effectiveLabels := merger.GetEffectiveLabels(wl)

	// Should have system labels
	assert.Equal(t, "host-001", effectiveLabels["host"])

	// Should have runtime labels
	assert.Equal(t, "runtime-app", effectiveLabels["app"])

	// Should have user labels
	assert.Equal(t, "user-env", effectiveLabels["env"])

	// Should NOT have auto-inferred labels
	// (Image "postgres:13" would normally be auto-tagged as "db", but no auto tagger)
	assert.NotEqual(t, "db", effectiveLabels["role"])

	mockRuntime.AssertExpectations(t)
}

// TestGetEffectiveLabels_RuntimeError tests handling of runtime errors
func TestGetEffectiveLabels_RuntimeError(t *testing.T) {
	mockRuntime := new(MockRuntimeDetector)
	mockRuntime.On("GetContainerLabels", "container-123").Return(
		map[string]string{}, assert.AnError)

	autoTagger := NewAutoTagger()
	merger := NewLabelMerger(mockRuntime, autoTagger)

	wl := &workload.Workload{
		ID:          "wl-123",
		ContainerID: "container-123",
		Image:       "nginx:latest",
		Labels: map[string]string{
			"app": "user-app",
		},
	}

	effectiveLabels := merger.GetEffectiveLabels(wl)

	// Should still have auto-inferred and user labels
	assert.Equal(t, "web", effectiveLabels["role"])
	assert.Equal(t, "user-app", effectiveLabels["app"])

	// Runtime error should be silently handled
	mockRuntime.AssertExpectations(t)
}

// TestGetEffectiveLabels_NoContainerID tests behavior without container ID
func TestGetEffectiveLabels_NoContainerID(t *testing.T) {
	mockRuntime := new(MockRuntimeDetector)
	// Should NOT call GetContainerLabels when ContainerID is empty

	autoTagger := NewAutoTagger()
	merger := NewLabelMerger(mockRuntime, autoTagger)

	wl := &workload.Workload{
		ID:          "wl-123",
		HostID:      "host-001",
		ContainerID: "", // Empty container ID
		Image:       "nginx:latest",
		Labels: map[string]string{
			"app": "user-app",
		},
	}

	effectiveLabels := merger.GetEffectiveLabels(wl)

	// Should have system and user labels
	assert.Equal(t, "host-001", effectiveLabels["host"])
	assert.Equal(t, "user-app", effectiveLabels["app"])

	// Should have auto-inferred labels
	assert.Equal(t, "web", effectiveLabels["role"])

	// Should NOT call runtime detector
	mockRuntime.AssertNotCalled(t, "GetContainerLabels", mock.Anything)
}

// TestMapLabelKey tests Kubernetes label mapping
func TestMapLabelKey(t *testing.T) {
	merger := NewLabelMerger(nil, nil)

	tests := []struct {
		name     string
		k8sKey   string
		expected string
	}{
		// Kubernetes recommended labels
		{
			name:     "app.kubernetes.io/name",
			k8sKey:   "app.kubernetes.io/name",
			expected: "app",
		},
		{
			name:     "app.kubernetes.io/component",
			k8sKey:   "app.kubernetes.io/component",
			expected: "role",
		},
		{
			name:     "app.kubernetes.io/version",
			k8sKey:   "app.kubernetes.io/version",
			expected: "version",
		},
		{
			name:     "app.kubernetes.io/instance",
			k8sKey:   "app.kubernetes.io/instance",
			expected: "instance",
		},
		{
			name:     "app.kubernetes.io/part-of",
			k8sKey:   "app.kubernetes.io/part-of",
			expected: "app",
		},

		// Common labels
		{
			name:     "app",
			k8sKey:   "app",
			expected: "app",
		},
		{
			name:     "application",
			k8sKey:   "application",
			expected: "app",
		},
		{
			name:     "role",
			k8sKey:   "role",
			expected: "role",
		},
		{
			name:     "tier",
			k8sKey:   "tier",
			expected: "role",
		},
		{
			name:     "component",
			k8sKey:   "component",
			expected: "role",
		},
		{
			name:     "environment",
			k8sKey:   "environment",
			expected: "env",
		},
		{
			name:     "env",
			k8sKey:   "env",
			expected: "env",
		},

		// Location labels
		{
			name:     "topology.kubernetes.io/zone",
			k8sKey:   "topology.kubernetes.io/zone",
			expected: "loc",
		},
		{
			name:     "topology.kubernetes.io/region",
			k8sKey:   "topology.kubernetes.io/region",
			expected: "region",
		},
		{
			name:     "failure-domain.beta.kubernetes.io/zone",
			k8sKey:   "failure-domain.beta.kubernetes.io/zone",
			expected: "loc",
		},

		// Team/ownership
		{
			name:     "team",
			k8sKey:   "team",
			expected: "team",
		},
		{
			name:     "owner",
			k8sKey:   "owner",
			expected: "owner",
		},

		// Unknown label (should pass through)
		{
			name:     "custom-label",
			k8sKey:   "custom.company.com/label",
			expected: "custom.company.com/label",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := merger.mapLabelKey(tt.k8sKey)
			assert.Equal(t, tt.expected, result,
				"Label %s should map to %s", tt.k8sKey, tt.expected)
		})
	}
}

// TestMergeLabelsInPlace tests in-place label merging
func TestMergeLabelsInPlace(t *testing.T) {
	mockRuntime := new(MockRuntimeDetector)
	mockRuntime.On("GetContainerLabels", "container-123").Return(map[string]string{
		"app": "runtime-app",
	}, nil)

	autoTagger := NewAutoTagger()
	merger := NewLabelMerger(mockRuntime, autoTagger)

	wl := &workload.Workload{
		ID:          "wl-123",
		HostID:      "host-001",
		ContainerID: "container-123",
		Image: "nginx:latest",
		Labels: map[string]string{
			"original": "label",
		},
	}

	// Store original labels
	originalLabels := make(map[string]string)
	for k, v := range wl.Labels {
		originalLabels[k] = v
	}

	// Merge in place
	merger.MergeLabelsInPlace(wl)

	// Verify labels were updated
	assert.NotEqual(t, originalLabels, wl.Labels, "Labels should be updated")
	assert.Contains(t, wl.Labels, "host", "Should contain system labels")
	assert.Contains(t, wl.Labels, "app", "Should contain runtime labels")
	assert.Contains(t, wl.Labels, "original", "Should contain original labels")

	mockRuntime.AssertExpectations(t)
}

// TestGetLabelSource tests label source tracking
func TestGetLabelSource(t *testing.T) {
	mockRuntime := new(MockRuntimeDetector)
	mockRuntime.On("GetContainerLabels", "container-123").Return(map[string]string{
		"app": "runtime-app",
	}, nil)

	autoTagger := NewAutoTagger()
	merger := NewLabelMerger(mockRuntime, autoTagger)

	wl := &workload.Workload{
		ID:          "wl-123",
		HostID:      "host-001",
		Namespace:   "production",
		ContainerID: "container-123",
		Image: "nginx:latest",
		Labels: map[string]string{
			"user-label": "user-value",
		},
	}

	// Test different label sources
	tests := []struct {
		name           string
		labelKey       string
		expectedSource string
	}{
		{
			name:           "system label - host",
			labelKey:       "host",
			expectedSource: "system",
		},
		{
			name:           "system label - namespace",
			labelKey:       "namespace",
			expectedSource: "system",
		},
		{
			name:           "runtime label",
			labelKey:       "app",
			expectedSource: "k8s",
		},
		{
			name:           "user label",
			labelKey:       "user-label",
			expectedSource: "user",
		},
		{
			name:           "auto-inferred label",
			labelKey:       "role",
			expectedSource: "inferred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := merger.GetLabelSource(wl, tt.labelKey)
			assert.Equal(t, tt.expectedSource, source,
				"Label %s should have source %s", tt.labelKey, tt.expectedSource)
		})
	}

	mockRuntime.AssertExpectations(t)
}

// TestGetLabelSource_UserOverridesRuntime tests that user labels are detected even if runtime has the same key
func TestGetLabelSource_UserOverridesRuntime(t *testing.T) {
	mockRuntime := new(MockRuntimeDetector)
	// Note: GetContainerLabels will NOT be called because GetLabelSource checks user labels first

	merger := NewLabelMerger(mockRuntime, nil)

	wl := &workload.Workload{
		ID:          "wl-123",
		ContainerID: "container-123",
		Labels: map[string]string{
			"app": "user-app", // User-defined, overrides runtime
		},
	}

	source := merger.GetLabelSource(wl, "app")
	assert.Equal(t, "user", source,
		"Should detect user label even though runtime also has 'app'")

	// Runtime detector should NOT be called since user label has highest priority
	mockRuntime.AssertNotCalled(t, "GetContainerLabels", mock.Anything)
}

// TestGetAllLabelSources tests getting all label sources
func TestGetAllLabelSources(t *testing.T) {
	mockRuntime := new(MockRuntimeDetector)
	mockRuntime.On("GetContainerLabels", "container-123").Return(map[string]string{
		"app": "runtime-app",
	}, nil)

	autoTagger := NewAutoTagger()
	merger := NewLabelMerger(mockRuntime, autoTagger)

	wl := &workload.Workload{
		ID:          "wl-123",
		HostID:      "host-001",
		Namespace:   "production",
		ContainerID: "container-123",
		Image: "nginx:latest",
		Labels: map[string]string{
			"user-label": "user-value",
		},
	}

	sources := merger.GetAllLabelSources(wl)

	// Verify sources are tracked
	assert.Equal(t, "system", sources["host"])
	assert.Equal(t, "system", sources["namespace"])
	assert.Equal(t, "k8s", sources["app"])
	assert.Equal(t, "user", sources["user-label"])
	assert.Equal(t, "inferred", sources["role"])

	mockRuntime.AssertExpectations(t)
}

// TestGetEffectiveLabels_EmptyWorkload tests handling of empty workload
func TestGetEffectiveLabels_EmptyWorkload(t *testing.T) {
	merger := NewLabelMerger(nil, nil)

	wl := &workload.Workload{
		ID: "wl-empty",
	}

	effectiveLabels := merger.GetEffectiveLabels(wl)

	// Should return empty map, not nil
	assert.NotNil(t, effectiveLabels, "Should return non-nil map")
	assert.Empty(t, effectiveLabels, "Should be empty for empty workload")
}

// TestGetEffectiveLabels_NilWorkload tests handling of nil workload
func TestGetEffectiveLabels_NilWorkload(t *testing.T) {
	merger := NewLabelMerger(nil, nil)

	// This should not panic
	effectiveLabels := merger.GetEffectiveLabels(nil)

	// Should return empty map, not nil
	assert.NotNil(t, effectiveLabels, "Should return non-nil map")
	assert.Empty(t, effectiveLabels, "Should be empty for nil workload")
}

// TestLabelMerger_ComplexScenario tests a complex real-world scenario
func TestLabelMerger_ComplexScenario(t *testing.T) {
	// Simulate a complex Kubernetes environment
	mockRuntime := new(MockRuntimeDetector)
	mockRuntime.On("GetContainerLabels", "container-nginx-123").Return(map[string]string{
		"app.kubernetes.io/name":       "nginx",
		"app.kubernetes.io/component":  "webserver",
		"app.kubernetes.io/version":    "1.21.0",
		"topology.kubernetes.io/zone":  "us-west-1a",
		"topology.kubernetes.io/region": "us-west-1",
		"team":                         "platform",
	}, nil)

	autoTagger := NewAutoTagger()
	merger := NewLabelMerger(mockRuntime, autoTagger)

	wl := &workload.Workload{
		ID:          "wl-nginx-prod",
		HostID:      "node-prod-001",
		Namespace:   "production",
		ContainerID: "container-nginx-123",
		Image: "nginx:latest",
		Labels: map[string]string{
			"env":         "production", // User override
			"owner":       "john.doe",   // Additional user label
			"cost-center": "engineering", // Additional user label
		},
	}

	effectiveLabels := merger.GetEffectiveLabels(wl)

	// Verify system labels
	assert.Equal(t, "node-prod-001", effectiveLabels["host"])
	assert.Equal(t, "production", effectiveLabels["namespace"])

	// Verify K8s labels are mapped correctly
	assert.Equal(t, "nginx", effectiveLabels["app"])
	assert.Equal(t, "webserver", effectiveLabels["role"])
	assert.Equal(t, "1.21.0", effectiveLabels["version"])
	assert.Equal(t, "us-west-1a", effectiveLabels["loc"])
	assert.Equal(t, "us-west-1", effectiveLabels["region"])
	assert.Equal(t, "platform", effectiveLabels["team"])

	// Verify user labels
	assert.Equal(t, "production", effectiveLabels["env"])
	assert.Equal(t, "john.doe", effectiveLabels["owner"])
	assert.Equal(t, "engineering", effectiveLabels["cost-center"])

	// Verify label count is reasonable
	assert.GreaterOrEqual(t, len(effectiveLabels), 10,
		"Should have labels from all sources")

	mockRuntime.AssertExpectations(t)
}

// BenchmarkGetEffectiveLabels benchmarks label merging performance
func BenchmarkGetEffectiveLabels(b *testing.B) {
	mockRuntime := new(MockRuntimeDetector)
	mockRuntime.On("GetContainerLabels", "container-123").Return(map[string]string{
		"app.kubernetes.io/name": "test-app",
		"app":                    "runtime-app",
		"env":                    "runtime-env",
	}, nil)

	autoTagger := NewAutoTagger()
	merger := NewLabelMerger(mockRuntime, autoTagger)

	wl := &workload.Workload{
		ID:          "wl-123",
		HostID:      "host-001",
		Namespace:   "production",
		ContainerID: "container-123",
		Image: "nginx:latest",
		Labels: map[string]string{
			"user-label": "user-value",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = merger.GetEffectiveLabels(wl)
	}
}

// BenchmarkMapLabelKey benchmarks label key mapping performance
func BenchmarkMapLabelKey(b *testing.B) {
	merger := NewLabelMerger(nil, nil)

	testKeys := []string{
		"app.kubernetes.io/name",
		"app.kubernetes.io/component",
		"topology.kubernetes.io/zone",
		"environment",
		"custom.label.com/foo",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := testKeys[i%len(testKeys)]
		_ = merger.mapLabelKey(key)
	}
}

// TestNewLabelMerger tests creating a new label merger
func TestNewLabelMerger(t *testing.T) {
	mockRuntime := new(MockRuntimeDetector)
	autoTagger := NewAutoTagger()

	merger := NewLabelMerger(mockRuntime, autoTagger)

	assert.NotNil(t, merger, "Merger should not be nil")
	assert.Equal(t, mockRuntime, merger.runtimeDetector, "Runtime detector should be set")
	assert.Equal(t, autoTagger, merger.autoTagger, "Auto tagger should be set")
}

// TestNewLabelMerger_NilComponents tests creating merger with nil components
func TestNewLabelMerger_NilComponents(t *testing.T) {
	// Should not panic with nil components
	merger := NewLabelMerger(nil, nil)
	assert.NotNil(t, merger, "Merger should not be nil even with nil components")

	// Should still work with empty workload
	wl := &workload.Workload{
		ID:     "wl-123",
		Labels: map[string]string{"app": "test"},
	}

	effectiveLabels := merger.GetEffectiveLabels(wl)
	assert.Equal(t, "test", effectiveLabels["app"], "Should still merge user labels")
}

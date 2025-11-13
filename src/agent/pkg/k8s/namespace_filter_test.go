package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNamespaceFilter(t *testing.T) {
	tests := []struct {
		name       string
		config     *NamespaceFilterConfig
		includeAll bool
		includeLen int
		excludeLen int
	}{
		{
			name:       "nil config",
			config:     nil,
			includeAll: true,
			includeLen: 0,
			excludeLen: 0,
		},
		{
			name:       "empty config",
			config:     &NamespaceFilterConfig{},
			includeAll: true,
			includeLen: 0,
			excludeLen: 0,
		},
		{
			name: "only include",
			config: &NamespaceFilterConfig{
				Include: []string{"default", "production"},
			},
			includeAll: false,
			includeLen: 2,
			excludeLen: 0,
		},
		{
			name: "only exclude",
			config: &NamespaceFilterConfig{
				Exclude: []string{"kube-system", "kube-public"},
			},
			includeAll: true,
			includeLen: 0,
			excludeLen: 2,
		},
		{
			name: "both include and exclude",
			config: &NamespaceFilterConfig{
				Include: []string{"production", "staging"},
				Exclude: []string{"kube-system"},
			},
			includeAll: false,
			includeLen: 2,
			excludeLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewNamespaceFilter(tt.config)
			assert.NotNil(t, filter)
			assert.Equal(t, tt.includeAll, filter.IsIncludeAll())
			assert.Len(t, filter.GetIncludedNamespaces(), tt.includeLen)
			assert.Len(t, filter.GetExcludedNamespaces(), tt.excludeLen)
		})
	}
}

func TestNamespaceFilter_ShouldInclude(t *testing.T) {
	tests := []struct {
		name      string
		config    *NamespaceFilterConfig
		namespace string
		expected  bool
	}{
		// Case 1: Include all (empty include list)
		{
			name:      "include all - default namespace",
			config:    &NamespaceFilterConfig{},
			namespace: "default",
			expected:  true,
		},
		{
			name:      "include all - any namespace",
			config:    &NamespaceFilterConfig{},
			namespace: "production",
			expected:  true,
		},

		// Case 2: Include all with exclusions
		{
			name: "include all except kube-system",
			config: &NamespaceFilterConfig{
				Exclude: []string{"kube-system"},
			},
			namespace: "default",
			expected:  true,
		},
		{
			name: "exclude kube-system",
			config: &NamespaceFilterConfig{
				Exclude: []string{"kube-system"},
			},
			namespace: "kube-system",
			expected:  false,
		},

		// Case 3: Specific include list
		{
			name: "include specific namespaces - match",
			config: &NamespaceFilterConfig{
				Include: []string{"production", "staging"},
			},
			namespace: "production",
			expected:  true,
		},
		{
			name: "include specific namespaces - no match",
			config: &NamespaceFilterConfig{
				Include: []string{"production", "staging"},
			},
			namespace: "development",
			expected:  false,
		},

		// Case 4: Include with exclusions (exclude takes precedence)
		{
			name: "include with exclusions - included",
			config: &NamespaceFilterConfig{
				Include: []string{"production", "staging", "development"},
				Exclude: []string{"staging"},
			},
			namespace: "production",
			expected:  true,
		},
		{
			name: "include with exclusions - excluded takes precedence",
			config: &NamespaceFilterConfig{
				Include: []string{"production", "staging"},
				Exclude: []string{"staging"},
			},
			namespace: "staging",
			expected:  false,
		},

		// Case 5: Multiple exclusions
		{
			name: "multiple exclusions - excluded",
			config: &NamespaceFilterConfig{
				Exclude: []string{"kube-system", "kube-public", "kube-node-lease"},
			},
			namespace: "kube-public",
			expected:  false,
		},
		{
			name: "multiple exclusions - not excluded",
			config: &NamespaceFilterConfig{
				Exclude: []string{"kube-system", "kube-public", "kube-node-lease"},
			},
			namespace: "default",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewNamespaceFilter(tt.config)
			result := filter.ShouldInclude(tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNamespaceFilter_GetIncludedNamespaces(t *testing.T) {
	tests := []struct {
		name     string
		config   *NamespaceFilterConfig
		expected []string
	}{
		{
			name:     "include all - returns empty",
			config:   &NamespaceFilterConfig{},
			expected: []string{},
		},
		{
			name: "specific namespaces",
			config: &NamespaceFilterConfig{
				Include: []string{"production", "staging"},
			},
			expected: []string{"production", "staging"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewNamespaceFilter(tt.config)
			result := filter.GetIncludedNamespaces()
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestNamespaceFilter_GetExcludedNamespaces(t *testing.T) {
	tests := []struct {
		name     string
		config   *NamespaceFilterConfig
		expected []string
	}{
		{
			name:     "no exclusions",
			config:   &NamespaceFilterConfig{},
			expected: []string{},
		},
		{
			name: "with exclusions",
			config: &NamespaceFilterConfig{
				Exclude: []string{"kube-system", "kube-public"},
			},
			expected: []string{"kube-system", "kube-public"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewNamespaceFilter(tt.config)
			result := filter.GetExcludedNamespaces()
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestNamespaceFilter_String(t *testing.T) {
	tests := []struct {
		name     string
		config   *NamespaceFilterConfig
		expected string
	}{
		{
			name:     "all namespaces",
			config:   &NamespaceFilterConfig{},
			expected: "all namespaces",
		},
		{
			name: "all except excluded",
			config: &NamespaceFilterConfig{
				Exclude: []string{"kube-system"},
			},
			expected: "all namespaces except excluded",
		},
		{
			name: "specified namespaces only",
			config: &NamespaceFilterConfig{
				Include: []string{"production"},
			},
			expected: "specified namespaces only",
		},
		{
			name: "specified with exclusions",
			config: &NamespaceFilterConfig{
				Include: []string{"production", "staging"},
				Exclude: []string{"staging"},
			},
			expected: "specified namespaces (with exclusions)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewNamespaceFilter(tt.config)
			result := filter.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNamespaceFilter_RealWorldScenarios(t *testing.T) {
	t.Run("default system namespace exclusion", func(t *testing.T) {
		// 模拟默认配置: 排除 Kubernetes 系统 namespace
		filter := NewNamespaceFilter(&NamespaceFilterConfig{
			Exclude: []string{"kube-system", "kube-public", "kube-node-lease"},
		})

		// 系统 namespace 应该被排除
		assert.False(t, filter.ShouldInclude("kube-system"))
		assert.False(t, filter.ShouldInclude("kube-public"))
		assert.False(t, filter.ShouldInclude("kube-node-lease"))

		// 用户 namespace 应该被包含
		assert.True(t, filter.ShouldInclude("default"))
		assert.True(t, filter.ShouldInclude("production"))
		assert.True(t, filter.ShouldInclude("staging"))
	})

	t.Run("production only monitoring", func(t *testing.T) {
		// 只监控生产环境
		filter := NewNamespaceFilter(&NamespaceFilterConfig{
			Include: []string{"production", "prod-backup"},
		})

		// 只有生产 namespace 被包含
		assert.True(t, filter.ShouldInclude("production"))
		assert.True(t, filter.ShouldInclude("prod-backup"))

		// 其他环境被排除
		assert.False(t, filter.ShouldInclude("staging"))
		assert.False(t, filter.ShouldInclude("development"))
		assert.False(t, filter.ShouldInclude("default"))
	})

	t.Run("multi-tenant with exclusions", func(t *testing.T) {
		// 多租户场景: 监控所有租户 namespace,但排除特定租户
		filter := NewNamespaceFilter(&NamespaceFilterConfig{
			Exclude: []string{"kube-system", "kube-public", "tenant-archived"},
		})

		// 活跃租户应该被包含
		assert.True(t, filter.ShouldInclude("tenant-a"))
		assert.True(t, filter.ShouldInclude("tenant-b"))

		// 系统和归档租户应该被排除
		assert.False(t, filter.ShouldInclude("kube-system"))
		assert.False(t, filter.ShouldInclude("tenant-archived"))
	})
}

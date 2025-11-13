package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)

	assert.Equal(t, "eth0", cfg.Interface)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 30, cfg.StatsInterval)
	assert.Equal(t, "agent-server", cfg.Mode)
}

func TestKubernetesConfigDefaults(t *testing.T) {
	// Test that Kubernetes is disabled by default
	cfg := DefaultConfig()
	assert.Nil(t, cfg.Kubernetes)
}

func TestKubernetesConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		modifyFunc  func(*Config)
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with kubernetes disabled",
			modifyFunc: func(c *Config) {
				c.DataPlane.Mode = "auto" // Initialize required field
				c.Kubernetes = &KubernetesConfig{
					Enabled: false,
				}
			},
			expectError: false,
		},
		{
			name: "valid config with kubernetes enabled - auto mode",
			modifyFunc: func(c *Config) {
				c.Kubernetes = &KubernetesConfig{
					Enabled:    true,
					ConfigMode: "auto",
				}
			},
			expectError: false,
		},
		{
			name: "valid config with kubernetes enabled - in-cluster mode",
			modifyFunc: func(c *Config) {
				c.Kubernetes = &KubernetesConfig{
					Enabled:    true,
					ConfigMode: "in-cluster",
				}
			},
			expectError: false,
		},
		{
			name: "valid config with kubernetes enabled - kubeconfig mode",
			modifyFunc: func(c *Config) {
				c.Kubernetes = &KubernetesConfig{
					Enabled:        true,
					ConfigMode:     "kubeconfig",
					KubeconfigPath: "/path/to/kubeconfig",
				}
			},
			expectError: false,
		},
		{
			name: "invalid config_mode",
			modifyFunc: func(c *Config) {
				c.Kubernetes = &KubernetesConfig{
					Enabled:    true,
					ConfigMode: "invalid",
				}
			},
			expectError: true,
			errorMsg:    "invalid kubernetes.config_mode",
		},
		{
			name: "health check timeout >= interval",
			modifyFunc: func(c *Config) {
				c.Kubernetes = &KubernetesConfig{
					Enabled:    true,
					ConfigMode: "auto",
					HealthCheck: KubernetesHealthCheckConfig{
						Enabled:  true,
						Interval: 10,
						Timeout:  10, // Equal to interval
					},
				}
			},
			expectError: true,
			errorMsg:    "health_check.timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			// Initialize DataPlane mode
			cfg.DataPlane.Mode = "auto"
			// Call the modify function
			tt.modifyFunc(cfg)

			err := cfg.Validate()
			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestKubernetesConfigDefaultsApplied(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DataPlane.Mode = "auto" // Initialize required field
	cfg.Kubernetes = &KubernetesConfig{
		Enabled: true,
		// Leave other fields at zero values
	}

	err := cfg.Validate()
	require.NoError(t, err)

	// Verify defaults were applied
	assert.Equal(t, "auto", cfg.Kubernetes.ConfigMode)
	assert.Equal(t, float32(5.0), cfg.Kubernetes.QPS)
	assert.Equal(t, 10, cfg.Kubernetes.Burst)
	assert.Equal(t, 30, cfg.Kubernetes.Timeout)
	assert.Equal(t, 30, cfg.Kubernetes.HealthCheck.Interval)
	assert.Equal(t, 5, cfg.Kubernetes.HealthCheck.Timeout)
}

func TestToK8sConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    *KubernetesConfig
		expected *K8sConfig
	}{
		{
			name:     "nil config returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name: "disabled config returns nil",
			input: &KubernetesConfig{
				Enabled: false,
			},
			expected: nil,
		},
		{
			name: "enabled config with auto mode",
			input: &KubernetesConfig{
				Enabled:    true,
				ConfigMode: "auto",
				QPS:        10.0,
				Burst:      20,
				Timeout:    60,
			},
			expected: &K8sConfig{
				Enabled:    true,
				ConfigMode: "auto",
				QPS:        10.0,
				Burst:      20,
				Timeout:    60,
			},
		},
		{
			name: "enabled config with in-cluster mode",
			input: &KubernetesConfig{
				Enabled:    true,
				ConfigMode: "in-cluster",
				QPS:        5.0,
				Burst:      10,
				Timeout:    30,
			},
			expected: &K8sConfig{
				Enabled:    true,
				ConfigMode: "in-cluster",
				QPS:        5.0,
				Burst:      10,
				Timeout:    30,
			},
		},
		{
			name: "enabled config with kubeconfig mode",
			input: &KubernetesConfig{
				Enabled:        true,
				ConfigMode:     "kubeconfig",
				KubeconfigPath: "/tmp/kubeconfig",
				APIServer:      "https://localhost:6443",
				QPS:            8.0,
				Burst:          15,
				Timeout:        45,
			},
			expected: &K8sConfig{
				Enabled:        true,
				ConfigMode:     "kubeconfig",
				KubeconfigPath: "/tmp/kubeconfig",
				APIServer:      "https://localhost:6443",
				QPS:            8.0,
				Burst:          15,
				Timeout:        45,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.ToK8sConfig()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Enabled, result.Enabled)
				assert.Equal(t, tt.expected.ConfigMode, result.ConfigMode)
				assert.Equal(t, tt.expected.KubeconfigPath, result.KubeconfigPath)
				assert.Equal(t, tt.expected.APIServer, result.APIServer)
				assert.Equal(t, tt.expected.QPS, result.QPS)
				assert.Equal(t, tt.expected.Burst, result.Burst)
				assert.Equal(t, tt.expected.Timeout, result.Timeout)
			}
		})
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	configContent := `
interface: veth0
log_level: debug
mode: standalone

kubernetes:
  enabled: true
  config_mode: kubeconfig
  kubeconfig_path: /tmp/test-kubeconfig
  qps: 10
  burst: 20
  timeout: 45
  health_check:
    enabled: true
    interval: 60
    timeout: 10
  namespaces:
    include:
      - production
      - staging
    exclude:
      - kube-system
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify basic config
	assert.Equal(t, "veth0", cfg.Interface)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "standalone", cfg.Mode)

	// Verify Kubernetes config
	require.NotNil(t, cfg.Kubernetes)
	assert.True(t, cfg.Kubernetes.Enabled)
	assert.Equal(t, "kubeconfig", cfg.Kubernetes.ConfigMode)
	assert.Equal(t, "/tmp/test-kubeconfig", cfg.Kubernetes.KubeconfigPath)
	assert.Equal(t, float32(10), cfg.Kubernetes.QPS)
	assert.Equal(t, 20, cfg.Kubernetes.Burst)
	assert.Equal(t, 45, cfg.Kubernetes.Timeout)

	// Verify health check config
	assert.True(t, cfg.Kubernetes.HealthCheck.Enabled)
	assert.Equal(t, 60, cfg.Kubernetes.HealthCheck.Interval)
	assert.Equal(t, 10, cfg.Kubernetes.HealthCheck.Timeout)

	// Verify namespace filtering
	assert.Equal(t, []string{"production", "staging"}, cfg.Kubernetes.Namespaces.Include)
	assert.Equal(t, []string{"kube-system"}, cfg.Kubernetes.Namespaces.Exclude)
}

package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds the agent configuration
type Config struct {
	// Interface is the network interface to attach eBPF program
	Interface string `mapstructure:"interface"`

	// LogLevel sets the log level (debug, info, warn, error)
	LogLevel string `mapstructure:"log_level"`

	// StatsInterval is the interval for printing statistics (seconds)
	StatsInterval int `mapstructure:"stats_interval"`

	// Mode specifies the agent operation mode: "agent-server" or "standalone"
	// - agent-server: Agent connects to Server, reports flows, syncs policies
	// - standalone: Agent runs independently without Server connection
	Mode string `mapstructure:"mode"`

	// API server configuration
	API APIConfig `mapstructure:"api"`

	// AgentServer configuration (required in agent-server mode)
	AgentServer *AgentServerConfig `mapstructure:"server"`

	// Flow collection configuration
	Flow FlowConfig `mapstructure:"flow"`

	// DataPlane configuration
	DataPlane DataPlaneConfig `mapstructure:"dataplane"`

	// Kubernetes integration configuration
	Kubernetes *KubernetesConfig `mapstructure:"kubernetes"`
}

// APIConfig holds API server configuration
type APIConfig struct {
	// Enabled controls whether API server is started
	Enabled bool `mapstructure:"enabled"`

	// Host is the address to bind the API server to
	Host string `mapstructure:"host"`

	// Port is the HTTP port to listen on
	Port int `mapstructure:"port"`

	// EnableCORS enables Cross-Origin Resource Sharing
	EnableCORS bool `mapstructure:"enable_cors"`
}

// FlowConfig holds flow collection configuration
// Flow collection is always enabled as it's a core system functionality
type FlowConfig struct {
	// StoragePath is the path to SQLite database for flow storage (standalone mode only)
	StoragePath string `mapstructure:"storage_path"`

	// FlowTimeout is the duration after which inactive flows are considered closed
	FlowTimeout time.Duration `mapstructure:"flow_timeout"`

	// CleanupInterval is the interval for cleaning up old flows
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`

	// RetentionDays is the number of days to retain flow data
	RetentionDays int `mapstructure:"retention_days"`
}

// AgentServerConfig holds configuration for control plane server connection
type AgentServerConfig struct {
	// ServerAddr is the gRPC server address (host:port)
	ServerAddr string `mapstructure:"server_addr"`

	// AgentID is the unique identifier for this agent
	// If empty, will be auto-generated from hostname
	AgentID string `mapstructure:"agent_id"`

	// BatchSize is the number of flows to batch before sending
	BatchSize int `mapstructure:"batch_size"`

	// BatchTimeout is the maximum time to wait before sending a partial batch
	BatchTimeout time.Duration `mapstructure:"batch_timeout"`

	// ReconnectInterval is the time to wait before reconnecting on failure
	ReconnectInterval time.Duration `mapstructure:"reconnect_interval"`

	// Retry configuration for flow reporting
	MaxRetries     int           `mapstructure:"max_retries"`      // Maximum number of retries (default: 3)
	RetryBaseDelay time.Duration `mapstructure:"retry_base_delay"` // Base delay for exponential backoff (default: 1s)
	RetryMaxDelay  time.Duration `mapstructure:"retry_max_delay"`  // Maximum retry delay (default: 30s)
}

// DataPlaneConfig holds data plane (eBPF) configuration
type DataPlaneConfig struct {
	// Mode specifies the dataplane mode:
	// - "auto": Automatically select best mode (default)
	// - "xdp-native": Force Native XDP (requires driver support)
	// - "xdp-generic": Force Generic XDP (kernel fallback)
	// - "tc": Force TC (TCX or Legacy)
	Mode string `mapstructure:"mode"`

	// PreferXDP controls whether to prefer XDP over TC in auto mode
	PreferXDP bool `mapstructure:"prefer_xdp"`

	// AllowGenericXDP controls whether Generic XDP fallback is allowed
	AllowGenericXDP bool `mapstructure:"allow_generic_xdp"`
}

// KubernetesConfig holds Kubernetes integration configuration
type KubernetesConfig struct {
	// Enabled controls whether Kubernetes integration is enabled
	Enabled bool `mapstructure:"enabled"`

	// ConfigMode specifies how to connect to Kubernetes API server:
	// - "auto": Auto-detect (in-cluster first, then kubeconfig) [default]
	// - "in-cluster": Use ServiceAccount (for running in K8s Pod)
	// - "kubeconfig": Use kubeconfig file (for development)
	ConfigMode string `mapstructure:"config_mode"`

	// KubeconfigPath is the path to kubeconfig file (only for "kubeconfig" mode)
	// If empty, uses ~/.kube/config
	KubeconfigPath string `mapstructure:"kubeconfig_path"`

	// APIServer is the Kubernetes API server address (optional, overrides kubeconfig)
	APIServer string `mapstructure:"api_server"`

	// QPS is the maximum queries per second to the API server (default: 5)
	QPS float32 `mapstructure:"qps"`

	// Burst is the maximum burst for throttle (default: 10)
	Burst int `mapstructure:"burst"`

	// Timeout is the request timeout in seconds (default: 30)
	Timeout int `mapstructure:"timeout"`

	// HealthCheck configuration
	HealthCheck KubernetesHealthCheckConfig `mapstructure:"health_check"`

	// Namespaces configuration
	Namespaces KubernetesNamespacesConfig `mapstructure:"namespaces"`
}

// KubernetesHealthCheckConfig holds Kubernetes health check configuration
type KubernetesHealthCheckConfig struct {
	// Enabled controls whether periodic health checks are enabled
	Enabled bool `mapstructure:"enabled"`

	// Interval is the health check interval in seconds (default: 30)
	Interval int `mapstructure:"interval"`

	// Timeout is the health check timeout in seconds (default: 5)
	Timeout int `mapstructure:"timeout"`
}

// KubernetesNamespacesConfig holds namespace filtering configuration
type KubernetesNamespacesConfig struct {
	// Include is a list of namespaces to monitor (empty = all)
	Include []string `mapstructure:"include"`

	// Exclude is a list of namespaces to exclude (takes precedence over Include)
	Exclude []string `mapstructure:"exclude"`
}

// LoadConfig loads configuration from file or returns defaults
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Allow environment variable overrides
	v.AutomaticEnv()
	v.SetEnvPrefix("MICROSEGMENT")

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Basic defaults
	v.SetDefault("interface", "eth0")
	v.SetDefault("log_level", "info")
	v.SetDefault("stats_interval", 30)
	v.SetDefault("mode", "agent-server") // Default to agent-server mode

	// API defaults
	v.SetDefault("api.enabled", true)
	v.SetDefault("api.host", "127.0.0.1")
	v.SetDefault("api.port", 8080)
	v.SetDefault("api.enable_cors", true)

	// Server defaults
	v.SetDefault("server.batch_size", 100)
	v.SetDefault("server.batch_timeout", "5s")
	v.SetDefault("server.reconnect_interval", "30s")
	v.SetDefault("server.max_retries", 3)
	v.SetDefault("server.retry_base_delay", "1s")
	v.SetDefault("server.retry_max_delay", "30s")

	// Flow collection defaults (always enabled)
	v.SetDefault("flow.storage_path", "./data/flows.db")
	v.SetDefault("flow.flow_timeout", "5m")
	v.SetDefault("flow.cleanup_interval", "1m")
	v.SetDefault("flow.retention_days", 7)

	// DataPlane defaults
	v.SetDefault("dataplane.mode", "auto")                // Auto-select best mode
	v.SetDefault("dataplane.prefer_xdp", true)            // Prefer XDP for better performance
	v.SetDefault("dataplane.allow_generic_xdp", true)     // Allow Generic XDP fallback

	// Kubernetes defaults
	v.SetDefault("kubernetes.enabled", false)             // Disabled by default
	v.SetDefault("kubernetes.config_mode", "auto")        // Auto-detect (in-cluster then kubeconfig)
	v.SetDefault("kubernetes.qps", 5.0)                   // 5 QPS
	v.SetDefault("kubernetes.burst", 10)                  // 10 burst
	v.SetDefault("kubernetes.timeout", 30)                // 30 seconds timeout
	v.SetDefault("kubernetes.health_check.enabled", true) // Enable health checks
	v.SetDefault("kubernetes.health_check.interval", 30)  // Check every 30 seconds
	v.SetDefault("kubernetes.health_check.timeout", 5)    // 5 seconds timeout
	// Default namespace filtering: exclude system namespaces
	v.SetDefault("kubernetes.namespaces.exclude", []string{"kube-system", "kube-public", "kube-node-lease"})
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate interface
	if c.Interface == "" {
		return fmt.Errorf("interface is required")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log_level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	// Validate mode
	if c.Mode == "" {
		c.Mode = "agent-server" // Default mode
	}
	if c.Mode != "agent-server" && c.Mode != "standalone" {
		return fmt.Errorf("invalid mode: %s (must be 'agent-server' or 'standalone')", c.Mode)
	}

	// Validate server configuration (only required in agent-server mode)
	if c.Mode == "agent-server" {
		if c.AgentServer == nil {
			return fmt.Errorf("server configuration is required in agent-server mode")
		}

		if c.AgentServer.ServerAddr == "" {
			return fmt.Errorf("server.server_addr is required in agent-server mode")
		}

		// Auto-generate agent ID if not provided
		if c.AgentServer.AgentID == "" {
			c.AgentServer.AgentID = generateAgentID()
		}

		// Set defaults if not specified
		if c.AgentServer.BatchSize == 0 {
			c.AgentServer.BatchSize = 100
		}
		if c.AgentServer.BatchTimeout == 0 {
			c.AgentServer.BatchTimeout = 5 * time.Second
		}
		if c.AgentServer.ReconnectInterval == 0 {
			c.AgentServer.ReconnectInterval = 30 * time.Second
		}
		if c.AgentServer.MaxRetries == 0 {
			c.AgentServer.MaxRetries = 3
		}
		if c.AgentServer.RetryBaseDelay == 0 {
			c.AgentServer.RetryBaseDelay = 1 * time.Second
		}
		if c.AgentServer.RetryMaxDelay == 0 {
			c.AgentServer.RetryMaxDelay = 30 * time.Second
		}
	}

	// Validate DataPlane configuration
	validModes := map[string]bool{
		"auto":        true,
		"xdp-native":  true,
		"xdp-generic": true,
		"tc":          true,
	}
	if !validModes[c.DataPlane.Mode] {
		return fmt.Errorf("invalid dataplane.mode: %s (must be auto, xdp-native, xdp-generic, or tc)", c.DataPlane.Mode)
	}

	// Validate Kubernetes configuration (if enabled)
	if c.Kubernetes != nil && c.Kubernetes.Enabled {
		// Validate ConfigMode
		validK8sConfigModes := map[string]bool{
			"auto":       true,
			"in-cluster": true,
			"kubeconfig": true,
		}
		if c.Kubernetes.ConfigMode == "" {
			c.Kubernetes.ConfigMode = "auto" // Default to auto
		}
		if !validK8sConfigModes[c.Kubernetes.ConfigMode] {
			return fmt.Errorf("invalid kubernetes.config_mode: %s (must be auto, in-cluster, or kubeconfig)", c.Kubernetes.ConfigMode)
		}

		// Set defaults for optional fields
		if c.Kubernetes.QPS == 0 {
			c.Kubernetes.QPS = 5.0
		}
		if c.Kubernetes.Burst == 0 {
			c.Kubernetes.Burst = 10
		}
		if c.Kubernetes.Timeout == 0 {
			c.Kubernetes.Timeout = 30
		}
		if c.Kubernetes.HealthCheck.Interval == 0 {
			c.Kubernetes.HealthCheck.Interval = 30
		}
		if c.Kubernetes.HealthCheck.Timeout == 0 {
			c.Kubernetes.HealthCheck.Timeout = 5
		}

		// Validate health check timeout < interval
		if c.Kubernetes.HealthCheck.Timeout >= c.Kubernetes.HealthCheck.Interval {
			return fmt.Errorf("kubernetes.health_check.timeout (%d) must be less than interval (%d)",
				c.Kubernetes.HealthCheck.Timeout, c.Kubernetes.HealthCheck.Interval)
		}
	}

	return nil
}

// generateAgentID generates a unique agent ID from hostname and timestamp
func generateAgentID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s-%d", hostname, time.Now().Unix())
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Interface:     "eth0",
		LogLevel:      "info",
		StatsInterval: 30,
		Mode:          "agent-server", // Default to agent-server mode
		API: APIConfig{
			Enabled:    true,
			Host:       "127.0.0.1",
			Port:       8080,
			EnableCORS: true,
		},
		AgentServer: &AgentServerConfig{
			ServerAddr:        "10.107.12.201:9090",
			AgentID:           "",
			BatchSize:         100,
			BatchTimeout:      5 * time.Second,
			ReconnectInterval: 30 * time.Second,
			MaxRetries:        3,
			RetryBaseDelay:    1 * time.Second,
			RetryMaxDelay:     30 * time.Second,
		},
		Flow: FlowConfig{
			StoragePath:     "./data/flows.db",
			FlowTimeout:     5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
			RetentionDays:   7,
		},
		DataPlane: DataPlaneConfig{
			Mode:            "auto",
			PreferXDP:       true,
			AllowGenericXDP: true,
		},
	}
}

// IsStandaloneMode returns true if agent is in standalone mode
func (c *Config) IsStandaloneMode() bool {
	return c.Mode == "standalone"
}

// IsAgentServerMode returns true if agent is in agent-server mode
func (c *Config) IsAgentServerMode() bool {
	return c.Mode == "agent-server"
}

// ToK8sConfig converts KubernetesConfig to k8s.Config
// Returns nil if Kubernetes integration is disabled
func (kc *KubernetesConfig) ToK8sConfig() *K8sConfig {
	if kc == nil || !kc.Enabled {
		return nil
	}

	cfg := &K8sConfig{
		Enabled:        kc.Enabled,
		KubeconfigPath: kc.KubeconfigPath,
		APIServer:      kc.APIServer,
		QPS:            kc.QPS,
		Burst:          kc.Burst,
		Timeout:        kc.Timeout,
	}

	// Map config_mode to ConfigMode type
	switch kc.ConfigMode {
	case "auto":
		cfg.ConfigMode = "auto"
	case "in-cluster":
		cfg.ConfigMode = "in-cluster"
	case "kubeconfig":
		cfg.ConfigMode = "kubeconfig"
	default:
		cfg.ConfigMode = "auto"
	}

	return cfg
}

// K8sConfig is an alias for the k8s package Config type to avoid import cycles
// This will be imported from k8s package when used
type K8sConfig struct {
	Enabled        bool
	ConfigMode     string
	KubeconfigPath string
	APIServer      string
	QPS            float32
	Burst          int
	Timeout        int
}

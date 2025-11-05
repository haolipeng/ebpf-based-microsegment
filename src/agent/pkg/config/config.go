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

	// API server configuration
	API APIConfig `mapstructure:"api"`

	// AgentServer configuration (required)
	AgentServer *AgentServerConfig `mapstructure:"server"`

	// Flow collection configuration
	Flow FlowConfig `mapstructure:"flow"`
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
type FlowConfig struct {
	// Enabled controls whether flow collection is enabled
	Enabled bool `mapstructure:"enabled"`

	// StoragePath is the path to SQLite database for flow storage
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

	// API defaults
	v.SetDefault("api.enabled", true)
	v.SetDefault("api.host", "127.0.0.1")
	v.SetDefault("api.port", 8080)
	v.SetDefault("api.enable_cors", true)

	// Server defaults
	v.SetDefault("server.batch_size", 100)
	v.SetDefault("server.batch_timeout", "5s")
	v.SetDefault("server.reconnect_interval", "30s")

	// Flow collection defaults
	v.SetDefault("flow.enabled", true)
	v.SetDefault("flow.storage_path", "./data/flows.db")
	v.SetDefault("flow.flow_timeout", "5m")
	v.SetDefault("flow.cleanup_interval", "1m")
	v.SetDefault("flow.retention_days", 7)
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

	// Validate server configuration
	if c.AgentServer == nil {
		return fmt.Errorf("server configuration is required")
	}

	if c.AgentServer.ServerAddr == "" {
		return fmt.Errorf("server.server_addr is required")
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
		API: APIConfig{
			Enabled:    true,
			Host:       "127.0.0.1",
			Port:       8080,
			EnableCORS: true,
		},
		AgentServer: &AgentServerConfig{
			ServerAddr:        "localhost:9090",
			AgentID:           "",
			BatchSize:         100,
			BatchTimeout:      5 * time.Second,
			ReconnectInterval: 30 * time.Second,
		},
		Flow: FlowConfig{
			Enabled:         true,
			StoragePath:     "./data/flows.db",
			FlowTimeout:     5 * time.Minute,
			CleanupInterval: 1 * time.Minute,
			RetentionDays:   7,
		},
	}
}

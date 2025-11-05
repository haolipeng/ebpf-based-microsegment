package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds the agent configuration
type Config struct {
	// Mode determines agent operation mode: "standalone" or "agent-server"
	Mode string `mapstructure:"mode"`

	// Interface is the network interface to attach eBPF program
	Interface string `mapstructure:"interface"`

	// LogLevel sets the log level (debug, info, warn, error)
	LogLevel string `mapstructure:"log_level"`

	// StatsInterval is the interval for printing statistics (seconds)
	StatsInterval int `mapstructure:"stats_interval"`

	// Storage configuration for standalone mode
	Storage StorageConfig `mapstructure:"storage"`

	// API server configuration
	API APIConfig `mapstructure:"api"`

	// AgentServer configuration for agent-server mode
	AgentServer *AgentServerConfig `mapstructure:"agent_server,omitempty"`
}

// StorageConfig holds SQLite storage configuration
type StorageConfig struct {
	// Path to SQLite database file
	Path string `mapstructure:"path"`
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

// AgentServerConfig holds configuration for agent-server mode
type AgentServerConfig struct {
	// Enabled controls whether agent-server mode is active
	Enabled bool `mapstructure:"enabled"`

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
	// Mode defaults
	v.SetDefault("mode", "standalone")
	v.SetDefault("interface", "lo")
	v.SetDefault("log_level", "info")
	v.SetDefault("stats_interval", 30)

	// Storage defaults
	v.SetDefault("storage.path", "/var/lib/microsegment/flows.db")

	// API defaults
	v.SetDefault("api.enabled", true)
	v.SetDefault("api.host", "127.0.0.1")
	v.SetDefault("api.port", 8080)
	v.SetDefault("api.enable_cors", true)

	// Agent-server defaults
	v.SetDefault("agent_server.batch_size", 100)
	v.SetDefault("agent_server.batch_timeout", "5s")
	v.SetDefault("agent_server.reconnect_interval", "30s")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate mode
	if c.Mode != "standalone" && c.Mode != "agent-server" {
		return fmt.Errorf("invalid mode: %s (must be 'standalone' or 'agent-server')", c.Mode)
	}

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

	// Validate standalone mode
	if c.Mode == "standalone" {
		if c.Storage.Path == "" {
			return fmt.Errorf("storage.path is required in standalone mode")
		}
	}

	// Validate agent-server mode
	if c.Mode == "agent-server" {
		if c.AgentServer == nil {
			return fmt.Errorf("agent_server configuration is required for agent-server mode")
		}

		if c.AgentServer.ServerAddr == "" {
			return fmt.Errorf("agent_server.server_addr is required")
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
		Mode:          "standalone",
		Interface:     "lo",
		LogLevel:      "info",
		StatsInterval: 30,
		Storage: StorageConfig{
			Path: "/var/lib/microsegment/flows.db",
		},
		API: APIConfig{
			Enabled:    true,
			Host:       "127.0.0.1",
			Port:       8080,
			EnableCORS: true,
		},
	}
}

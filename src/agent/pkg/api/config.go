package api

import "time"

// Config holds API server configuration
type Config struct {
	// Host is the address to bind the API server to
	Host string `json:"host" yaml:"host"`

	// Port is the HTTP port to listen on
	Port int `json:"port" yaml:"port"`

	// ReadTimeout is the maximum duration for reading the entire request
	ReadTimeout time.Duration `json:"read_timeout" yaml:"read_timeout"`

	// WriteTimeout is the maximum duration before timing out writes
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`

	// IdleTimeout is the maximum amount of time to wait for the next request
	IdleTimeout time.Duration `json:"idle_timeout" yaml:"idle_timeout"`

	// EnableCORS enables Cross-Origin Resource Sharing
	EnableCORS bool `json:"enable_cors" yaml:"enable_cors"`

	// LogLevel sets the log level for API server (debug, info, warn, error)
	LogLevel string `json:"log_level" yaml:"log_level"`

	// Interface is the network interface name (for config endpoint display)
	Interface string `json:"interface" yaml:"interface"`

	// StatsInterval is the statistics reporting interval in seconds
	StatsInterval int `json:"stats_interval" yaml:"stats_interval"`

	// Version is the agent version string
	Version string `json:"version" yaml:"version"`
}

// DefaultConfig returns default API configuration
func DefaultConfig() *Config {
	return &Config{
		Host:         "127.0.0.1",
		Port:         8080,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		EnableCORS:   true,
		LogLevel:     "info",
		Version:      "0.1.0",
	}
}

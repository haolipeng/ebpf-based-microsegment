package models

// ConfigResponse represents the current system configuration
type ConfigResponse struct {
	Interface     string `json:"interface"`
	LogLevel      string `json:"log_level"`
	StatsInterval int    `json:"stats_interval"`
	APIHost       string `json:"api_host"`
	APIPort       int    `json:"api_port"`
}

// ConfigUpdateRequest represents a configuration update request
type ConfigUpdateRequest struct {
	LogLevel      *string `json:"log_level,omitempty" binding:"omitempty,oneof=debug info warn error"`
	StatsInterval *int    `json:"stats_interval,omitempty" binding:"omitempty,min=1,max=300"`
}


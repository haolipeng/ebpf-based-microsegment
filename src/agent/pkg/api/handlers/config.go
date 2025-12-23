
// input: runtime config API requests (GET/PUT)
// output: agent configuration JSON
// pos: runtime config API handlers - if file updated, must sync with this header comment and pkg/api/CLAUDE.md
package handlers

import (
	"net/http"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/api/models"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// ConfigHandler handles configuration management requests
type ConfigHandler struct {
	currentConfig *RuntimeConfig
}

// RuntimeConfig holds the current runtime configuration
// This is a simplified version for the configuration endpoint
type RuntimeConfig struct {
	Interface     string
	LogLevel      string
	StatsInterval int
	APIHost       string
	APIPort       int
}

// NewConfigHandler creates a new configuration handler
func NewConfigHandler(iface, logLevel string, statsInterval int, apiHost string, apiPort int) *ConfigHandler {
	return &ConfigHandler{
		currentConfig: &RuntimeConfig{
			Interface:     iface,
			LogLevel:      logLevel,
			StatsInterval: statsInterval,
			APIHost:       apiHost,
			APIPort:       apiPort,
		},
	}
}

// GetConfig handles GET /api/v1/config
// Returns the current system configuration
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	response := models.ConfigResponse{
		Interface:     h.currentConfig.Interface,
		LogLevel:      h.currentConfig.LogLevel,
		StatsInterval: h.currentConfig.StatsInterval,
		APIHost:       h.currentConfig.APIHost,
		APIPort:       h.currentConfig.APIPort,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateConfig handles PUT /api/v1/config
// Updates the runtime configuration (only certain fields are mutable)
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	var req models.ConfigUpdateRequest

	// Bind and validate JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Track if any changes were made
	updated := false

	// Update log level if provided
	if req.LogLevel != nil {
		oldLevel := h.currentConfig.LogLevel
		h.currentConfig.LogLevel = *req.LogLevel

		// Apply log level change to the logger
		level, err := log.ParseLevel(*req.LogLevel)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid log level",
				Message: err.Error(),
			})
			// Rollback
			h.currentConfig.LogLevel = oldLevel
			return
		}
		log.SetLevel(level)
		log.Infof("Log level changed from %s to %s", oldLevel, *req.LogLevel)
		updated = true
	}

	// Update stats interval if provided
	if req.StatsInterval != nil {
		oldInterval := h.currentConfig.StatsInterval
		h.currentConfig.StatsInterval = *req.StatsInterval
		log.Infof("Statistics interval changed from %d to %d seconds", oldInterval, *req.StatsInterval)
		updated = true
		// Note: The actual ticker interval change would need to be implemented
		// in the main agent loop, which is outside the scope of this handler
	}

	if !updated {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "No configuration fields provided",
			Message: "At least one configuration field must be specified",
		})
		return
	}

	// Return the updated configuration
	response := models.ConfigResponse{
		Interface:     h.currentConfig.Interface,
		LogLevel:      h.currentConfig.LogLevel,
		StatsInterval: h.currentConfig.StatsInterval,
		APIHost:       h.currentConfig.APIHost,
		APIPort:       h.currentConfig.APIPort,
	}

	c.JSON(http.StatusOK, response)
}

package api

import (
	"github.com/ebpf-microsegment/src/agent/pkg/api/handlers"
)

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Create handlers
	healthHandler := handlers.NewHealthHandler(s.dataPlane, s.policyManager)
	policyHandler := handlers.NewPolicyHandler(s.policyManager)
	statsHandler := handlers.NewStatisticsHandler(s.dataPlane)
	configHandler := handlers.NewConfigHandler(
		"lo",                // TODO: Get from actual config
		s.config.LogLevel,
		5,                   // TODO: Get from actual config
		s.config.Host,
		s.config.Port,
	)

	// API v1 group
	v1 := s.router.Group("/api/v1")
	{
		// Health and status endpoints
		v1.GET("/health", healthHandler.GetHealth)
		v1.GET("/status", healthHandler.GetStatus)

		// Policy management endpoints
		policies := v1.Group("/policies")
		{
			policies.POST("", policyHandler.CreatePolicy)
			policies.GET("", policyHandler.ListPolicies)
			policies.GET("/:id", policyHandler.GetPolicy)
			policies.PUT("/:id", policyHandler.UpdatePolicy)
			policies.DELETE("/:id", policyHandler.DeletePolicy)
		}

		// Statistics endpoints
		stats := v1.Group("/stats")
		{
			stats.GET("", statsHandler.GetAllStats)
			stats.GET("/packets", statsHandler.GetPacketStats)
			stats.GET("/sessions", statsHandler.GetSessionStats)
			stats.GET("/policies", statsHandler.GetPolicyStats)
		}

		// Configuration endpoints
		config := v1.Group("/config")
		{
			config.GET("", configHandler.GetConfig)
			config.PUT("", configHandler.UpdateConfig)
		}
	}
}


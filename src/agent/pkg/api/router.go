package api

import (
	"github.com/ebpf-microsegment/src/agent/pkg/api/handlers"
	"github.com/gin-gonic/gin"
)

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Create handlers
	healthHandler := handlers.NewHealthHandler(s.dataPlane, s.policyManager)
	policyHandler := handlers.NewPolicyHandler(s.policyManager)
	statsHandler := handlers.NewStatisticsHandler(s.dataPlane)

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

		// Configuration endpoints (to be implemented)
		config := v1.Group("/config")
		{
			config.GET("", s.handleGetConfig)
			config.PUT("", s.handleUpdateConfig)
		}
	}
}

// Placeholder handlers (will be implemented in separate files)

func (s *Server) handleGetConfig(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (s *Server) handleUpdateConfig(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}


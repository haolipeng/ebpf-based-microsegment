package api

import (
	"github.com/ebpf-microsegment/src/agent/pkg/api/handlers"
	"github.com/ebpf-microsegment/src/agent/pkg/flow"
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

		// Flow collection endpoints (conditionally enabled)
		// Note: Flow components are set via SetFlowComponents() if flow collection is enabled
		if s.flowCollector != nil && s.flowStorage != nil {
			// Type assertions to convert interfaces to concrete types
			collector, ok1 := s.flowCollector.(*flow.Collector)
			storage, ok2 := s.flowStorage.(flow.Storage)

			if ok1 && ok2 {
				flowHandler := handlers.NewFlowHandler(collector, storage)
				flows := v1.Group("/flows")
				{
					flows.GET("", flowHandler.ListFlows)                      // List/query flows
					flows.GET("/:id", flowHandler.GetFlow)                    // Get single flow
					flows.GET("/summary", flowHandler.GetFlowSummary)         // Flow statistics summary
					flows.GET("/active", flowHandler.GetActiveFlows)          // Active flows from collector
					flows.GET("/metrics", flowHandler.GetCollectorMetrics)    // Collector metrics
					flows.GET("/dependencies", flowHandler.GetDependencies)   // Workload dependencies
					flows.GET("/top-talkers", flowHandler.GetTopTalkers)      // Top talkers analysis
				}
			}
		}
	}
}

